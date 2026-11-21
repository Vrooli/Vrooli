package localworkbridge

import (
	"context"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"
)

// Lifecycle owns one registered host and its tick goroutine. The shared server
// authentication middleware remains outside this handler. Events authenticates
// with native HMAC in the worker, not a synthesized human/service principal.
type Lifecycle struct {
	host   *Host
	cancel context.CancelFunc
	done   chan error
	once   sync.Once
	runErr error
}

// Register binds only the exact route without starting work. Disabled config
// registers 404 and performs no dependency I/O. Active use requires an approved
// provider and worker-side HMAC verification before storage access.
func Register(mux *http.ServeMux, cfg Config) (*Lifecycle, error) {
	host, err := New(cfg)
	if err != nil {
		return nil, err
	}
	lifecycle := &Lifecycle{host: host}
	var slots chan struct{}
	if cfg.Enabled {
		slots = make(chan struct{}, 1)
	}
	mux.Handle(Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !cfg.Enabled {
			host.ServeHTTP(w, r)
			return
		}
		peer, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(peer)
		if err != nil || ip == nil || !ip.IsLoopback() {
			http.Error(w, "native bridge requires loopback", 403)
			return
		}
		// Forwarding headers cannot establish loopback identity. The shared server
		// has finite transport header/time bounds; this adds route admission caps.
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		default:
			http.Error(w, "bridge busy", 503)
			return
		}
		headerBytes := 0
		for key, values := range r.Header {
			headerBytes += len(key)
			for _, value := range values {
				headerBytes += len(value)
			}
		}
		if headerBytes > 8192 {
			http.Error(w, "headers too large", 431)
			return
		}
		host.ServeHTTP(w, r)
	}))
	return lifecycle, nil
}

// Start is idempotent. Disabled startup allocates no goroutine.
func (l *Lifecycle) Start(ctx context.Context) {
	l.once.Do(func() {
		if !l.host.cfg.Enabled {
			return
		}
		ctx, l.cancel = context.WithCancel(ctx)
		l.done = make(chan error, 1)
		go func() {
			err := l.host.Run(ctx)
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, ErrStopped) {
				// Do not expose runner error text or allow a failed loop to
				// keep accepting deliveries without owner supervision.
				err = errors.New("local work bridge lifecycle failed")
				stopCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
				defer cancel()
				// Report first so a concurrently stuck receipt cannot hide failure.
				if l.host.cfg.OnFailure != nil {
					l.host.cfg.OnFailure(err)
				}
				l.host.Stop(stopCtx)
			}
			l.done <- err
		}()
	})
}

// Stop is called serially by the API lifecycle owner and joins the tick loop.
func (l *Lifecycle) Stop(ctx context.Context) error {
	if l.cancel != nil {
		l.cancel()
	}
	if err := l.host.Stop(ctx); err != nil {
		return err
	}
	if l.done != nil {
		select {
		case err := <-l.done:
			if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, ErrStopped) {
				l.runErr = errors.New("local work bridge lifecycle failed")
			}
			l.done = nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return l.runErr
}
