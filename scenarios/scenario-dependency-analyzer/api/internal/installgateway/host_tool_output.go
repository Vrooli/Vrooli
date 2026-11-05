package installgateway

import "sync"

const hostToolOutputBytes = 1 << 20

// hostToolOutput drains the command streams while retaining at most their last
// MiB. Truncation is explicit and never converts a command failure to success.
type hostToolOutput struct {
	mu        sync.Mutex
	tail      []byte
	position  int
	size      int
	truncated bool
}

func (o *hostToolOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	n := len(p)
	if n > 0 && o.tail == nil {
		o.tail = make([]byte, hostToolOutputBytes)
	}
	if n >= hostToolOutputBytes {
		o.truncated = o.truncated || o.size > 0 || n > hostToolOutputBytes
		copy(o.tail, p[n-hostToolOutputBytes:])
		o.position, o.size = 0, hostToolOutputBytes
		return n, nil
	}
	// A ring avoids copying the entire retained MiB for each short progress
	// write after the budget fills. Every incoming byte is copied once.
	first := min(n, hostToolOutputBytes-o.position)
	copy(o.tail[o.position:], p[:first])
	copy(o.tail, p[first:])
	o.position = (o.position + n) % hostToolOutputBytes
	if n > hostToolOutputBytes-o.size {
		o.truncated = true
		o.size = hostToolOutputBytes
	} else {
		o.size += n
	}
	return n, nil
}

func (o *hostToolOutput) String() string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.size < hostToolOutputBytes {
		return string(o.tail[:o.size])
	}
	tail := string(o.tail[o.position:]) + string(o.tail[:o.position])
	if o.truncated {
		return "[earlier package-manager output truncated; retaining last 1 MiB]\n" + tail
	}
	return tail
}
