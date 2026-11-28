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

	sharedpolicy "github.com/vrooli/ai-provider-go/policy"
	"github.com/vrooli/vrooli/packages/capacity/modelretention"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/policy"
	"github.com/vrooli/vrooli/resources/ollaya/cli/internal/upstream"
)

// Handlers carries the injectable seams. Tests point HTTP at httptest servers
// and GetEnv at a fixture map.
type Handlers struct {
	GetEnv          func(string) string
	Stdin           io.Reader
	Stdout          io.Writer
	Stderr          io.Writer
	HTTP            *http.Client
	Now             func() time.Time
	RightsContext   func(context.Context) (sharedpolicy.RightsContext, error)
	RightsEvaluator sharedpolicy.RightsEvaluator
	// ModelDiskAdmit is a test seam. Production defaults always create the
	// shared capacity broker before any missing-model pull.
	ModelDiskAdmit func(context.Context, []string, policy.Policy) (*modelDiskLease, func() error, error)
	// PrepareModelPull binds traffic and disk accounting to one managed service
	// owner while holding its lifecycle lock. Tests inject a fixture endpoint.
	PrepareModelPull func(context.Context) (string, *upstream.Client, func(context.Context) error, func(), error)
	// Residents lists the models the daemon holds in memory and where.
	Residents func(context.Context) ([]upstream.ResidentModel, error)
	// Unload releases a resident model's memory.
	Unload func(context.Context, string) error
	// DaemonDevice reports the device the managed daemon is pinned to.
	DaemonDevice func(context.Context) (string, error)
	// CapacityAdmit asks the shared broker ("vram" or "ram") whether a
	// footprint fits without recording a claim.
	CapacityAdmit func(context.Context, string, int64) (bool, string, error)
	// LoadSlot takes the per-daemon model-load slot (tests inject a recorder).
	LoadSlot func(context.Context) (func(), error)
	// RunControlPlane runs a control-plane command (tests inject a recorder).
	RunControlPlane func(context.Context, string, ...string) ([]byte, error)
	// RetentionPaths and ModelRoot locate the eviction fence, usage, receipts
	// and manifests (tests inject temporary directories).
	RetentionPaths func() (modelretention.Paths, error)
	ModelRoot      func() (string, error)
}

func Default() *Handlers {
	rightsContext := func(context.Context) (sharedpolicy.RightsContext, error) {
		path := policy.RightsContextPath(os.Getenv)
		if path == "" {
			return sharedpolicy.RightsContext{}, errors.New("trusted local model rights context path is unavailable")
		}
		return sharedpolicy.LoadRightsContextFile(path)
	}
	return &Handlers{
		GetEnv: os.Getenv, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
		HTTP: &http.Client{}, Now: time.Now,
		RightsContext: rightsContext, RightsEvaluator: sharedpolicy.NewRightsEvaluator(rightsContext),
	}
}

func (h *Handlers) rightsContext(ctx context.Context) (sharedpolicy.RightsContext, error) {
	if h.RightsContext == nil {
		return sharedpolicy.RightsContext{}, errors.New("trusted local model rights context source is unavailable")
	}
	return h.RightsContext(ctx)
}

func (h *Handlers) rightsEvaluator() sharedpolicy.RightsEvaluator {
	if h.RightsEvaluator != nil {
		return h.RightsEvaluator
	}
	return sharedpolicy.NewRightsEvaluator(h.rightsContext)
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
func (h *Handlers) localModelIdentity(ctx context.Context, p policy.Policy, modelName string) (*sharedpolicy.RuntimeArtifactIdentity, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	installed, err := h.client().LocalModels(ctx)
	if err != nil {
		return nil, "", err
	}
	model, ok := p.Models[modelName]
	if !ok {
		return nil, "", &upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: fmt.Sprintf("model %s has no policy identity", modelName)}
	}
	for _, name := range append([]string{modelName}, model.Routes...) {
		installedName := upstream.InstalledName(name)
		observedDigest, ok := installed[installedName]
		if !ok {
			return nil, "", &upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: fmt.Sprintf("model %s is not pulled; run `resource-ollaya models pull --model %s`", name, modelName)}
		}
		pinnedDigest := p.Models[name].Digest
		if observedDigest != pinnedDigest {
			return nil, "", &upstream.ProviderError{Code: upstream.CodeUnconfigured, Message: fmt.Sprintf("model %s runtime manifest %s does not match pinned manifest %s", name, observedDigest, pinnedDigest)}
		}
	}
	digest := installed[upstream.InstalledName(modelName)]
	identity := &sharedpolicy.RuntimeArtifactIdentity{
		Provider: sharedpolicy.RuntimeProviderOllaya, Model: modelName,
		ArtifactID: "ollaya/" + modelName, ArtifactRevision: model.Digest,
		DigestKind: sharedpolicy.RuntimeDigestOllayaManifest, Digest: digest,
		Source: sharedpolicy.RuntimeSourceOllayaTags, ObservedAt: h.now().UTC().Format("2006-01-02"),
	}
	return identity, digest, nil
}

func (h *Handlers) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func writeJSON(out io.Writer, payload any) error {
	return cliout.NewEncoder(out).Encode(payload)
}
