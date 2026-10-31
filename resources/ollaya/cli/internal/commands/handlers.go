// Package commands holds the resource-ollaya command groups: `policy` (the
// contract ai-gateway's adapter reads), `gateway` (the decision call) and
// `models` (digest-pinned pulls into the managed daemon).
package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// Handlers carries the injectable seams. Tests point HTTP at httptest servers
// and GetEnv at a fixture map.
type Handlers struct {
	GetEnv func(string) string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	HTTP   *http.Client
	Now    func() time.Time
}

func Default() *Handlers {
	return &Handlers{GetEnv: os.Getenv, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr, HTTP: &http.Client{}, Now: time.Now}
}

func (h *Handlers) policy() (policy.Policy, error) {
	return policy.Load(policy.PolicyPath(h.GetEnv))
}

func (h *Handlers) client() upstream.Client {
	return upstream.Client{HTTP: h.HTTP, GetEnv: h.GetEnv, Now: h.Now}
}

// fail writes a classified failure's marker line before returning it, so the
// gateway recovers the code while a human still sees the message.
func (h *Handlers) fail(err error) error {
	var providerErr *upstream.ProviderError
	if errors.As(err, &providerErr) {
		fmt.Fprintln(h.Stderr, providerErr.MarkerLine())
	}
	return err
}

// localModelReady reports whether the daemon is reachable and holds the
// model; a router also needs every target it routes to.
func (h *Handlers) localModelReady(ctx context.Context, p policy.Policy, model string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	installed, err := h.client().LocalModels(ctx)
	if err != nil {
		return err
	}
	for _, name := range append([]string{model}, p.Models[model].Routes...) {
		if _, ok := installed[upstream.InstalledName(name)]; !ok {
			return &upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: fmt.Sprintf("model %s is not pulled; run `resource-ollaya models pull --model %s`", name, model)}
		}
	}
	return nil
}

func writeJSON(out io.Writer, payload any) error {
	return cliout.NewEncoder(out).Encode(payload)
}
