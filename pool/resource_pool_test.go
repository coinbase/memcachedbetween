package pool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func newRp(expiredElements ...int) (*resourcePool, error) {
  expired := FakeExpiredElements {
    Expired: make(map[int]bool),
  }
  for i := range expiredElements {
    expired.Expired[i] = true
  }
  var config = resourcePoolConfig{
    MaxSize:          5,
    MinSize:          0,
    MaintainInterval: 300 * time.Second,
    ExpiredFn:        expired.rpExpiredFunc,
    CloseFn:          rpCloseFunc,
    InitFn:           rpInitFunc,
  }
  rp, e := newResourcePool(config)
	return rp, e
}

var counter = 0
func rpInitFunc() interface{} {
	counter++
	return counter
}
type FakeExpiredElements struct {
  Expired     map[int]bool
}

func (fe *FakeExpiredElements) rpExpiredFunc(item interface{}) bool {
  i := item.(int)
  if fe.Expired[i] {
    return true
  }
  return false
}
func rpCloseFunc(interface{}) {

}

func TestList(t *testing.T) {
	rp, e := newRp()
	assert.NoError(t, e)
	assert.Equal(t, uint64(0), rp.totalSize)
	assert.Nil(t, rp.start)
	assert.Nil(t, rp.end)

  // Put uses add which increments size
	newItem := 1
	assert.True(t, rp.Put(newItem))
	assert.Equal(t, uint64(1), rp.size)
	assert.NotNil(t, rp.start)
	assert.NotNil(t, rp.end)
	assert.Nil(t, rp.start.prev)
	assert.Nil(t, rp.start.next)
	assert.Nil(t, rp.end.next)
	assert.Nil(t, rp.end.prev)

  // this should reduce the size to 0
	entry := rp.Get()
	assert.NotNil(t, entry)
	assert.Equal(t, uint64(0), rp.size)
	assert.Equal(t, newItem, entry)
	assert.Nil(t, rp.start)
	assert.Nil(t, rp.end)

  // it's able to recover from going all the way to zero
	assert.True(t, rp.Put(newItem))
	assert.Equal(t, uint64(1), rp.size)
	assert.Equal(t, newItem, entry)
	assert.NotNil(t, rp.start)
	assert.NotNil(t, rp.end)
	assert.Nil(t, rp.start.prev)
	assert.Nil(t, rp.start.next)
}

// ensures that when add() is used, the item is added to the end of the list
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

// ensures that when get() is used, the item is taken from the beginning of the list
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
