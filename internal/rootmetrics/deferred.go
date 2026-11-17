package rootmetrics

// Deferred records a bounded observation without doing observation filesystem
// I/O on the caller. A crash or process exit may lose any queued evidence.
// It has one producer and must be finished once that producer is done.
type Deferred struct {
	commands chan func(*Attempt)
	done     chan struct{}
}

func BeginDeferred(home, kind string) *Deferred {
	return beginDeferred(home, kind, Begin)
}

func beginDeferred(home, kind string, begin func(string, string) *Attempt) *Deferred {
	d := &Deferred{commands: make(chan func(*Attempt), 64), done: make(chan struct{})}
	go func() {
		defer close(d.done)
		a := begin(home, kind)
		for command := range d.commands {
			command(a)
		}
	}()
	return d
}

// enqueue never waits for the observation writer. Overflow loses evidence,
// never the operation; callers keep observation sequences bounded.
func (d *Deferred) enqueue(command func(*Attempt)) {
	if d == nil {
		return
	}
	select {
	case d.commands <- command:
	default:
	}
}

func (d *Deferred) Bundle(manifest string, images map[string]string) {
	copyImages := make(map[string]string, len(images))
	for k, v := range images {
		copyImages[k] = v
	}
	d.enqueue(func(a *Attempt) { a.Bundle(manifest, copyImages) })
}

func (d *Deferred) Event(stage, reason string) {
	d.enqueue(func(a *Attempt) { a.Event(stage, reason) })
}

// Finish closes the queue without waiting for filesystem I/O. The returned
// channel closes when the observer exits; a production caller under a safety
// lock must not wait on it.
func (d *Deferred) Finish() <-chan struct{} {
	if d == nil {
		ch := make(chan struct{})
		close(ch)
		return ch
	}
	close(d.commands)
	return d.done
}

// Done allows tests and non-lock-holding callers to observe worker completion.
func (d *Deferred) Done() <-chan struct{} { return d.done }
