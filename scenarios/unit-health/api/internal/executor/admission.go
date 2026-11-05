package executor

import (
	"context"
	"errors"
	"sync"
)

var ErrAdmissionTooLarge = errors.New("executor: command exceeds admission capacity")

// Admission is a small weighted reservation gate. It bounds aggregate
// declared CPU and memory demand independently of worker count. A zero
// capacity disables that dimension; callers can therefore opt into only the
// budgets they can measure on a host.
type Admission struct {
	mu             sync.Mutex
	notify         chan struct{}
	cpuCapacity    int
	memoryCapacity int64
	cpuUsed        int
	memoryUsed     int64
	waiters        []*admissionWaiter
}

type admissionWaiter struct {
	cpu    int
	memory int64
}

func NewAdmission(cpuCapacity int, memoryCapacity int64) *Admission {
	return &Admission{notify: make(chan struct{}), cpuCapacity: cpuCapacity, memoryCapacity: memoryCapacity}
}

func (a *Admission) Acquire(ctx context.Context, command Command) error {
	if a == nil {
		return nil
	}
	cpu, memory := command.Resources.CPUWeight, command.Resources.MemoryBytes
	if cpu < 0 || memory < 0 || (a.cpuCapacity > 0 && cpu > a.cpuCapacity) || (a.memoryCapacity > 0 && memory > a.memoryCapacity) {
		return ErrAdmissionTooLarge
	}
	waiter := &admissionWaiter{cpu: cpu, memory: memory}
	a.mu.Lock()
	a.waiters = append(a.waiters, waiter)
	for {
		if err := ctx.Err(); err != nil {
			a.removeWaiterLocked(waiter)
			a.signalLocked()
			a.mu.Unlock()
			return err
		}
		if len(a.waiters) > 0 && a.waiters[0] == waiter && a.canReserveLocked(cpu, memory) {
			a.waiters = a.waiters[1:]
			a.cpuUsed += cpu
			a.memoryUsed += memory
			a.mu.Unlock()
			return nil
		}
		notify := a.notify
		a.mu.Unlock()
		select {
		case <-notify:
			a.mu.Lock()
		case <-ctx.Done():
			a.mu.Lock()
		}
	}
}

func (a *Admission) Release(command Command) {
	if a == nil {
		return
	}
	a.mu.Lock()
	a.cpuUsed -= command.Resources.CPUWeight
	a.memoryUsed -= command.Resources.MemoryBytes
	if a.cpuUsed < 0 {
		a.cpuUsed = 0
	}
	if a.memoryUsed < 0 {
		a.memoryUsed = 0
	}
	a.signalLocked()
	a.mu.Unlock()
}

func (a *Admission) canReserveLocked(cpu int, memory int64) bool {
	return (a.cpuCapacity == 0 || a.cpuUsed+cpu <= a.cpuCapacity) &&
		(a.memoryCapacity == 0 || a.memoryUsed+memory <= a.memoryCapacity)
}

func (a *Admission) removeWaiterLocked(target *admissionWaiter) {
	for index, waiter := range a.waiters {
		if waiter != target {
			continue
		}
		copy(a.waiters[index:], a.waiters[index+1:])
		a.waiters[len(a.waiters)-1] = nil
		a.waiters = a.waiters[:len(a.waiters)-1]
		return
	}
}

// signalLocked wakes every waiter and replaces the notification channel. The
// mutex makes close-and-replace atomic with respect to Acquire, so a canceled
// head waiter cannot strand later FIFO waiters.
func (a *Admission) signalLocked() {
	close(a.notify)
	a.notify = make(chan struct{})
}

func RunAllWithAdmission(ctx context.Context, runner Runner, commands []Command, maxConcurrency int, admission *Admission) []Result {
	if admission == nil {
		return RunAll(ctx, runner, commands, maxConcurrency)
	}
	if maxConcurrency < 1 {
		maxConcurrency = 1
	}
	if maxConcurrency > len(commands) {
		maxConcurrency = len(commands)
	}
	results := make([]Result, len(commands))
	if len(commands) == 0 {
		return results
	}
	type job struct {
		index   int
		command Command
	}
	jobs := make(chan job)
	var wg sync.WaitGroup
	for worker := 0; worker < maxConcurrency; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if err := admission.Acquire(ctx, item.command); err != nil {
					results[item.index] = Result{WorkspaceID: item.command.WorkspaceID, Name: item.command.Name, Command: formatCommand(item.command.Executable, item.command.Args), Status: StatusError, FailureClass: ClassSystem, FailureReason: err.Error()}
					continue
				}
				results[item.index] = runner.Run(ctx, item.command)
				admission.Release(item.command)
			}
		}()
	}
	for index, command := range commands {
		select {
		case jobs <- job{index: index, command: command}:
		case <-ctx.Done():
			results[index] = Result{WorkspaceID: command.WorkspaceID, Name: command.Name, Command: formatCommand(command.Executable, command.Args), Status: StatusError, FailureClass: ClassSystem, FailureReason: ctx.Err().Error()}
		}
	}
	close(jobs)
	wg.Wait()
	return results
}
