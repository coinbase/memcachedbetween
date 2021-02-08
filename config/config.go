package config

import (
	"errors"
	"flag"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

const defaultStatsdAddress = "localhost:8125"

var validNetworks = []string{"tcp", "tcp4", "tcp6", "unix", "unixpacket"}

type Config struct {
	UpstreamConfigHost string
	LocalConfigHost    string

	Network           string
	LocalSocketPrefix string
	LocalSocketSuffix string
	LocalPortStart    int
	Unlink            bool

	MinPoolSize  uint64
	MaxPoolSize  uint64
	ReadTimeout  time.Duration
	WriteTimeout time.Duration

	Pretty bool
	Statsd string
	Level  zapcore.Level
}

func ParseFlags() *Config {
	config, err := parseFlags()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		flag.Usage()
		os.Exit(2)
	}
	return config
}

func parseFlags() (*Config, error) {
	flag.Usage = func() {
		fmt.Printf("Usage: %s [OPTIONS] upstreamconfig\n", os.Args[0])
		flag.PrintDefaults()
	}

	var network, localConfigHost, localSocketPrefix, localSocketSuffix, stats, loglevel string
	var localPortStart int
	var minPoolSize, maxPoolSize uint64
	var readTimeout, writeTimeout time.Duration
	var pretty, unlink bool
	flag.StringVar(&network, "network", "unix", "One of: tcp, tcp4, tcp6, unix or unixpacket")
	flag.StringVar(&localConfigHost, "localconfig", ":11210", "Address to listen on for elasticache-like config server responses")
	flag.StringVar(&localSocketPrefix, "localsocketprefix", "/var/tmp/memcachedbetween-", "Prefix to use for unix socket filenames")
	flag.StringVar(&localSocketSuffix, "localsocketsuffix", ".sock", "Suffix to use for unix socket filenames")
	flag.IntVar(&localPortStart, "localportstart", 11220, "Port number to start from for local proxies")
	flag.BoolVar(&unlink, "unlink", false, "Unlink existing unix sockets before listening")
	flag.Uint64Var(&minPoolSize, "minpoolsize", 0, "Min connection pool size")
	flag.Uint64Var(&maxPoolSize, "maxpoolsize", 10, "Max connection pool size")
	flag.DurationVar(&readTimeout, "readtimeout", 1*time.Second, "Read timeout")
	flag.DurationVar(&writeTimeout, "writetimeout", 1*time.Second, "Write timeout")
	flag.StringVar(&stats, "statsd", defaultStatsdAddress, "Statsd address")
	flag.BoolVar(&pretty, "pretty", false, "Pretty print logging")
	flag.StringVar(&loglevel, "loglevel", "info", "One of: debug, info, warn, error, dpanic, panic, fatal")

	flag.Parse()

	level := zap.InfoLevel
	if loglevel != "" {
		err := level.Set(loglevel)
		if err != nil {
			return nil, fmt.Errorf("invalid loglevel: %s", loglevel)
		}
	}

	upstreamConfigHost := flag.Arg(0)
	if len(upstreamConfigHost) == 0 {
		return nil, errors.New("missing upstream config address")
	}

	if !validNetwork(network) {
		return nil, fmt.Errorf("invalid network: %s", network)
	}

	return &Config{
		UpstreamConfigHost: upstreamConfigHost,
		LocalConfigHost:    localConfigHost,

		Network:           network,
		LocalSocketPrefix: localSocketPrefix,
		LocalSocketSuffix: localSocketSuffix,
		LocalPortStart:    localPortStart,
		Unlink:            unlink,

		MinPoolSize:  minPoolSize,
		MaxPoolSize:  maxPoolSize,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,

		Pretty: pretty,
		Statsd: stats,
		Level:  level,
	}, nil
}

func validNetwork(network string) bool {
	for _, n := range validNetworks {
		if n == network {
			return true
		}
	}
	return false
}
