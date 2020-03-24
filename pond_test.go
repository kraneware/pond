package pond_test

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alitto/pond"
	"github.com/stretchr/testify/assert"
)

func TestSubmitAndStopWaiting(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(1, 5)

	// Submit tasks
	var doneCount int32
	for i := 0; i < 17; i++ {
		pool.Submit(func() {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&doneCount, 1)
		})
	}

	// Wait until all submitted tasks complete
	pool.StopAndWait()

	assert.Equal(int32(17), atomic.LoadInt32(&doneCount))
}

func TestSubmitAndStopWaitingWithMoreWorkersThanTasks(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(18, 5)

	// Submit tasks
	var doneCount int32
	for i := 0; i < 17; i++ {
		pool.Submit(func() {
			time.Sleep(1 * time.Millisecond)
			atomic.AddInt32(&doneCount, 1)
		})
	}

	// Wait until all submitted tasks complete
	pool.StopAndWait()

	assert.Equal(int32(17), atomic.LoadInt32(&doneCount))
}

func TestSubmitAndStopWithoutWaiting(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(1, 5)

	// Submit tasks
	started := make(chan bool)
	completed := make(chan bool)
	var doneCount int32
	for i := 0; i < 5; i++ {
		pool.Submit(func() {
			started <- true
			time.Sleep(5 * time.Millisecond)
			atomic.AddInt32(&doneCount, 1)
			<-completed
		})
	}

	// Make sure the first task started
	<-started

	// Stop without waiting for the rest of the tasks to start
	pool.Stop()

	// Let the first task complete now
	completed <- true

	// Only the first task should have been completed, the rest are discarded
	assert.Equal(int32(1), atomic.LoadInt32(&doneCount))

	// Make sure the exit lines in the worker pool are executed and covered
	time.Sleep(6 * time.Millisecond)
}

func TestSubmitWithNilTask(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(2, 5)

	// Submit nil task
	pool.Submit(nil)

	// Wait until all submitted tasks complete
	pool.StopAndWait()

	assert.Equal(0, pool.Running())
}

func TestSubmitAndWait(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(1, 5)
	defer pool.StopAndWait()

	// Submit a task and wait for it to complete
	var doneCount int32
	pool.SubmitAndWait(func() {
		time.Sleep(5 * time.Millisecond)
		atomic.AddInt32(&doneCount, 1)
	})

	assert.Equal(int32(1), atomic.LoadInt32(&doneCount))
}

func TestSubmitAndWaitWithNilTask(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(2, 5)

	// Submit nil task
	pool.SubmitAndWait(nil)

	// Wait until all submitted tasks complete
	pool.StopAndWait()

	assert.Equal(0, pool.Running())
}

func TestRunning(t *testing.T) {

	assert := assert.New(t)

	workerCount := 5
	taskCount := 10
	pool := pond.New(workerCount, taskCount)

	assert.Equal(0, pool.Running())

	// Submit tasks
	var started = make(chan struct{}, workerCount)
	var completed = make(chan struct{}, workerCount)
	for i := 0; i < taskCount; i++ {
		pool.Submit(func() {
			started <- struct{}{}
			time.Sleep(1 * time.Millisecond)
			<-completed
		})
	}

	// Wait until half the tasks have started
	for i := 0; i < taskCount/2; i++ {
		<-started
	}

	assert.Equal(workerCount, pool.Running())
	time.Sleep(1 * time.Millisecond)

	// Make sure half the tasks tasks complete
	for i := 0; i < taskCount/2; i++ {
		completed <- struct{}{}
	}

	// Wait until the rest of the tasks have started
	for i := 0; i < taskCount/2; i++ {
		<-started
	}

	// Make sure all tasks complete
	for i := 0; i < taskCount/2; i++ {
		completed <- struct{}{}
	}

	pool.StopAndWait()

	assert.Equal(0, pool.Running())
}

func TestSubmitWithPanic(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(1, 5)
	assert.Equal(0, pool.Running())

	// Submit a task that panics
	var doneCount int32
	pool.Submit(func() {
		arr := make([]string, 0)
		fmt.Printf("Out of range value: %s", arr[1])
		atomic.AddInt32(&doneCount, 1)
	})

	// Submit a task that completes normally
	pool.Submit(func() {
		time.Sleep(2 * time.Millisecond)
		atomic.AddInt32(&doneCount, 1)
	})

	pool.StopAndWait()
	assert.Equal(0, pool.Running())
	assert.Equal(int32(1), atomic.LoadInt32(&doneCount))
}

func TestSubmitWithIdleTimeout(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(1, 5, pond.IdleTimeout(2*time.Millisecond))

	// Submit a task
	started := make(chan bool)
	completed := make(chan bool)
	pool.Submit(func() {
		<-started
		time.Sleep(3 * time.Millisecond)
		<-completed
	})

	// Make sure the first task has started
	started <- true

	// There should be 1 worker running
	assert.Equal(1, pool.Running())

	// Let the task complete
	completed <- true

	// Wait for idle timeout + 1ms
	time.Sleep(3 * time.Millisecond)

	// Worker should have been killed
	assert.Equal(0, pool.Running())

	pool.StopAndWait()
}

func TestSubmitWithPanicHandler(t *testing.T) {

	assert := assert.New(t)

	var capturedPanic interface{} = nil
	panicHandler := func(panic interface{}) {
		capturedPanic = panic
	}

	pool := pond.New(1, 5, pond.PanicHandler(panicHandler))

	// Submit a task that panics
	pool.Submit(func() {
		panic("panic now!")
	})

	pool.StopAndWait()

	// Panic should have been captured
	assert.Equal("panic now!", capturedPanic)
}

func TestGroupSubmit(t *testing.T) {

	assert := assert.New(t)

	pool := pond.New(5, 5)
	assert.Equal(0, pool.Running())

	// Submit groups of tasks
	var doneCount, taskCount int32
	var groups []*pond.TaskGroup
	for i := 0; i < 5; i++ {
		group := pool.Group()
		for j := 0; j < i+5; j++ {
			group.Submit(func() {
				time.Sleep(1 * time.Millisecond)
				atomic.AddInt32(&doneCount, 1)
			})
			taskCount++
		}
		groups = append(groups, group)
	}

	// Wait for all groups to complete
	for _, group := range groups {
		group.Wait()
	}

	assert.Equal(int32(taskCount), atomic.LoadInt32(&doneCount))
}
