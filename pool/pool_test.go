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

// Tests the connectionExpiredFunc and the whole inactivity aging
func TestInactivity(t *testing.T) {
	duration := 5 * time.Second

	var address Address = "localhost:38888"
	go startTcpServer(string(address))
	config := poolConfig{
		Address:              address,
		MinPoolSize:          1,
		MaxPoolSize:          1,
		PoolMonitor:          nil,
		ConnectionInactivity: duration,
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
		assert.Equal(t, ReasonOld, conn.expireReason)
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
		ConnectionInactivity: duration,
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
		assert.Equal(t, ReasonOld, conn.expireReason)
		conn, err = p.get(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "", conn.expireReason)
		assert.Equal(t, 1, conn.connected)
	}
}
