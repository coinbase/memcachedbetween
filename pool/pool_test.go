package pool

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"time"
)

func startTcpServer(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	defer l.Close()
	for {
		conn, err := l.Accept()
		if err != nil {
			panic(err)
		}
		go func(conn net.Conn) {
			fmt.Println("request arrived")
			conn.Close()
		}(conn)
	}
}

func TestPoolExpiredFn(t *testing.T) {
	p := &pool{
    address:    "localhost:8000",
		monitor:    nil,
		connected:  disconnected,
    opened:     nil,
		opts:       nil,
		sem:        nil,
	}
  conn, err := newConnection("localhost:0000")
  assert.NoError(t, err)
  conn.pool = p
  assert.Empty(t, conn.expireReason)
  // first the disconnected case
  conn.connected = disconnected
  assert.True(t, connectionExpiredFunc(conn))
  assert.Equal(t, ReasonPoolClosed, conn.expireReason)

  // the connection staleness
  p.generation++
  p.connected = connected
  conn.connected = connected
  assert.True(t, connectionExpiredFunc(conn))
  assert.Equal(t, ReasonStale, conn.expireReason)

  // the connection staleness
  conn.generation = p.generation
  conn.expiresAfter = time.Now()
  time.Sleep(100 * time.Millisecond)
  assert.True(t, connectionExpiredFunc(conn))
  assert.Equal(t, ReasonConnectionExpired, conn.expireReason)
}
// Tests the connectionExpiredFunc and the whole connection expiry
func TestConnectionExpiry(t *testing.T) {
	duration := 5 * time.Second

	var address Address = "localhost:38888"
	go startTcpServer(string(address))
	config := poolConfig{
		Address:              address,
		MinPoolSize:          1,
		MaxPoolSize:          1,
		PoolMonitor:          nil,
		ConnectionLifeSpan: duration,
		MaintainInterval:     1 * time.Second,
	}

	p, e := newPool(config)
	assert.NoError(t, e)
	e = p.connect()
	assert.NoError(t, e)

	// get a reference to a connection
	ctx := context.TODO()
	defer ctx.Done()
	conn, err := p.get(ctx)
	assert.NoError(t, err)
	if err == nil {
		time.Sleep(duration)
		assert.True(t, connectionExpiredFunc(conn))
		assert.Equal(t, ReasonConnectionExpired, conn.expireReason)
	}
}

// Ensures we always get a good connection even if there's inactivity there
func TestGoodConnectionDespiteInactivity(t *testing.T) {
	duration := 5 * time.Second

	var address Address = "localhost:38889"
	go startTcpServer(string(address))
	config := poolConfig{
		Address:              address,
		MinPoolSize:          1,
		MaxPoolSize:          1,
		PoolMonitor:          nil,
		ConnectionLifeSpan: duration,
		MaintainInterval:     1 * time.Second,
	}

	p, e := newPool(config)
	assert.NoError(t, e)
	e = p.connect()
	assert.NoError(t, e)

	// get a reference to a connection
	ctx := context.TODO()
	defer ctx.Done()
	conn, err := p.get(ctx)
	p.put(conn)
	assert.NoError(t, err)
	assert.False(t, connectionExpiredFunc(conn))
	if err == nil {
		time.Sleep(duration)
		conn, err = p.get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "", conn.expireReason)
		assert.Equal(t, connected, conn.connected)
	}
}
