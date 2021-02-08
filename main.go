package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/coinbase/mongobetween/util"

	"github.com/coinbase/memcachedbetween/config"
	"github.com/coinbase/memcachedbetween/elasticache"
	"github.com/coinbase/memcachedbetween/handlers"
	"github.com/coinbase/memcachedbetween/listener"
	"github.com/coinbase/memcachedbetween/pool"
)

const disconnectTimeout = 10 * time.Second

func main() {
	c := config.ParseFlags()
	log := newLogger(c.Level, c.Pretty)
	err := run(log, c)
	if err != nil {
		log.Panic("Error", zap.Error(err))
	}
}

func run(log *zap.Logger, cfg *config.Config) error {
	sd, err := statsd.New(cfg.Statsd, statsd.WithNamespace("memcachedbetween"))
	if err != nil {
		return err
	}

	nodes, err := elasticache.ClusterNodes(log, cfg.UpstreamConfigHost)
	if err != nil {
		return err
	}
	log.Info("Config read", zap.Strings("servers", nodes))

	listeners, err := createListeners(log, sd, cfg, nodes)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	defer func() {
		wg.Wait()
	}()

	for _, l := range listeners {
		l := l
		wg.Add(1)
		go func() {
			defer wg.Done()

			err = l.Run()
			if err != nil {
				log.Error("Error", zap.Error(err))
			}
		}()
	}

	shutdown := func() {
		for _, l := range listeners {
			l.Shutdown()
		}
	}
	kill := func() {
		for _, l := range listeners {
			l.Kill()
		}
	}
	shutdownOnSignal(log, shutdown, kill)

	log.Info("Running")
	return nil
}

func createListeners(log *zap.Logger, sd *statsd.Client, cfg *config.Config, upstreams []string) ([]*listener.Listener, error) {
	var configs []string
	var listeners []*listener.Listener

	for index, upstream := range upstreams {
		var local string
		if strings.Contains(cfg.Network, "unix") {
			local = fmt.Sprintf("%s%d%s", cfg.LocalSocketPrefix, index, cfg.LocalSocketSuffix)
			configs = append(configs, fmt.Sprintf("%s||", local))
		} else {
			port := cfg.LocalPortStart + index
			local = fmt.Sprintf(":%d", port)
			configs = append(configs, fmt.Sprintf("localhost|127.0.0.1|%d", port))
		}

		logWith := log.With(zap.String("upstream", upstream), zap.String("local", local))
		sdWith, err := util.StatsdWithTags(sd, []string{fmt.Sprintf("upstream:%s", upstream), fmt.Sprintf("local:%s", local)})
		if err != nil {
			return nil, err
		}

		m, err := pool.ConnectServer(
			pool.Address(upstream),
			pool.WithMinConnections(func(uint64) uint64 { return cfg.MinPoolSize }),
			pool.WithMaxConnections(func(uint64) uint64 { return cfg.MaxPoolSize }),
			pool.WithConnectionPoolMonitor(func(*pool.Monitor) *pool.Monitor { return poolMonitor(sdWith) }),
		)
		if err != nil {
			return nil, err
		}

		connectionHandler := func(log *zap.Logger, conn net.Conn, id uint64, kill chan interface{}) {
			handlers.CommandConnection(log, sd, cfg, conn, local, id, m, kill)
		}
		shutdownHandler := func() {
			ctx, cancel := context.WithTimeout(context.Background(), disconnectTimeout)
			defer cancel()
			_ = m.Disconnect(ctx)
		}
		l, err := listener.New(logWith, sdWith, cfg.Network, local, cfg.Unlink, connectionHandler, shutdownHandler)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, l)
	}

	configsJoined := strings.Join(configs, " ")
	connectionHandler := func(log *zap.Logger, conn net.Conn, id uint64, kill chan interface{}) {
		handlers.ConfigConnection(log, conn, kill, configsJoined)
	}
	l, err := listener.New(log, sd, "tcp4", cfg.LocalConfigHost, cfg.Unlink, connectionHandler, func() {})
	if err != nil {
		return nil, err
	}
	listeners = append(listeners, l)

	return listeners, nil
}

func poolMonitor(sd *statsd.Client) *pool.Monitor {
	checkedOut, checkedIn := util.StatsdBackgroundGauge(sd, "pool.checked_out_connections", []string{})
	opened, closed := util.StatsdBackgroundGauge(sd, "pool.open_connections", []string{})

	return &pool.Monitor{
		Event: func(e *pool.Event) {
			snake := strings.ToLower(regexp.MustCompile("([a-z0-9])([A-Z])").ReplaceAllString(e.Type, "${1}_${2}"))
			name := fmt.Sprintf("pool_event.%s", snake)
			tags := []string{
				fmt.Sprintf("address:%s", e.Address),
				fmt.Sprintf("reason:%s", e.Reason),
			}
			switch e.Type {
			case pool.ConnectionCreated:
				opened(name, tags)
			case pool.ConnectionClosed:
				closed(name, tags)
			case pool.GetSucceeded:
				checkedOut(name, tags)
			case pool.ConnectionReturned:
				checkedIn(name, tags)
			default:
				_ = sd.Incr(name, tags, 1)
			}
		},
	}
}

func newLogger(level zapcore.Level, pretty bool) *zap.Logger {
	var c zap.Config
	if pretty {
		c = zap.NewDevelopmentConfig()
		c.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		c = zap.NewProductionConfig()
	}

	c.EncoderConfig.MessageKey = "message"
	c.Level.SetLevel(level)

	log, err := c.Build(zap.AddStacktrace(zap.FatalLevel))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	return log
}

func shutdownOnSignal(log *zap.Logger, shutdownFunc func(), killFunc func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		shutdownAttempted := false
		for sig := range c {
			log.Info("Signal", zap.String("signal", sig.String()))

			if !shutdownAttempted {
				log.Info("Shutting down")
				go shutdownFunc()
				shutdownAttempted = true

				if sig == os.Interrupt {
					time.AfterFunc(1*time.Second, func() {
						fmt.Println("Ctrl-C again to kill incoming connections")
					})
				}
			} else if sig == os.Interrupt {
				log.Warn("Terminating")
				_ = log.Sync() // #nosec
				killFunc()
			}
		}
	}()
}
