package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var counter = 0
var config = resourcePoolConfig{
	MaxSize:          5,
	MinSize:          0,
	MaintainInterval: 300 * time.Second,
	ExpiredFn:        rpExpiredFunc,
	CloseFn:          rpCloseFunc,
	InitFn:           rpInitFunc,
}

func newRp() (*resourcePool, error) {
	rp, e := newResourcePool(config)
	return rp, e
}

func rpInitFunc() interface{} {
	counter++
	return counter
}
func rpExpiredFunc(interface{}) bool {
	return false
}
func rpCloseFunc(interface{}) {

}

func TestList(t *testing.T) {
	rp, e := newRp()
	assert.NoError(t, e)
	assert.Equal(t, uint64(0), rp.totalSize)
	newItem := 1
	assert.True(t, rp.Put(newItem))
	assert.Equal(t, uint64(1), rp.size)
	assert.NotNil(t, rp.start)
	assert.NotNil(t, rp.end)
	assert.Nil(t, rp.start.prev)
	assert.Nil(t, rp.start.next)
	assert.Nil(t, rp.end.next)
	assert.Nil(t, rp.end.prev)

	entry := rp.Get()
	assert.NotNil(t, entry)
	assert.Equal(t, uint64(0), rp.size)
	assert.Equal(t, newItem, entry)
	assert.Nil(t, rp.start)
	assert.Nil(t, rp.end)

	assert.True(t, rp.Put(newItem))
	assert.Equal(t, uint64(1), rp.size)
	assert.Equal(t, newItem, entry)
	assert.NotNil(t, rp.start)
	assert.NotNil(t, rp.end)
	assert.Nil(t, rp.start.prev)
	assert.Nil(t, rp.start.next)
}

func TestAddedToEnd(t *testing.T) {
	rp, e := newRp()
	assert.NoError(t, e)
	newItem := 1
	assert.True(t, rp.Put(newItem))
	newItem = 2
	assert.True(t, rp.Put(newItem))
	// Start should be 1; end should be 2
	assert.Equal(t, 1, rp.start.value.(int))
	assert.Equal(t, 2, rp.end.value.(int))
}

func TestTakenFromStart(t *testing.T) {
	rp, e := newRp()
	assert.NoError(t, e)
	newItem := 1
	assert.True(t, rp.Put(newItem))
	newItem = 2
	assert.True(t, rp.Put(newItem))
	// We'll get two elements; in order should 1 and 2
	item1 := rp.Get().(int)
	item2 := rp.Get().(int)
	assert.Equal(t, 1, item1)
	assert.Equal(t, 2, item2)
}
