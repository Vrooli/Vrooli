package assessment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliutil"
	scenariovalidationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DefaultContract is the shared validation contract identifier a provider
// speaks when its descriptor does not name one explicitly.
const DefaultContract = "scenario-validation/v1"

// ProviderDescription is the provider-owned, target-independent identity that a
// readiness consumer needs: who this provider is, which phase it backs, which
// contract it speaks, and what it can do.
//
// Every field is a property of the provider process. None of them can depend on
// a target scenario, which is precisely why answering these questions must not
// require running a target analysis.
type ProviderDescription struct {
	Provider          string
	Phase             string
	SpecVersion       string
	Contract          string
	DeliveryMode      string
	SupportsExecution bool
	SupportsFixes     bool
	Build             ProviderBuild
}

// ProviderBuild carries best-effort build provenance. An unset field means
// "unknown" and must never be read as "stale".
type ProviderBuild struct {
	Revision         string
	BuiltAt          time.Time
	BinaryModifiedAt time.Time
	// FreshnessDigest is the build-input digest recorded next to this binary,
	// read once at startup. A consumer comparing it against the digest now on
	// disk learns whether the binary was rebuilt without this process being
	// restarted. Empty means no manifest was found, which consumers must read
	// as "unknown", never as "stale".
	FreshnessDigest string
}

// describeDescriptor is the subset of `.vrooli/test-genie.json` that describes
// the provider itself. It deliberately ignores the maturity block, which
// LoadSpecFromScenario already owns.
// Provider and phase identity are deliberately absent here: they come from the
// validated Spec so the two RPCs cannot disagree about who this provider is.
type describeDescriptor struct {
	Validation struct {
		Contract     string `json:"contract"`
		DeliveryMode string `json:"deliveryMode"`
		Execution    bool   `json:"execution"`
		// IncludeExecution is the retired inline delivery flag, still present on
		// descriptors that have not migrated to `execution`. Treat either as
		// "this provider has a real execution mode".
		IncludeExecution bool `json:"includeExecution"`
	} `json:"validation"`
}

// LoadProviderDescription reads the provider's own descriptor and assembles the
// facts DescribeProvider reports. scenarioDir is the provider scenario's root
// directory (the parent of `.vrooli`), the same argument LoadSpecFromScenario
// takes.
//
// The maturity spec is loaded through the existing path so provider/phase
// identity has exactly one source of truth and cannot drift between the two
// RPCs.
func LoadProviderDescription(scenarioDir string) (*ProviderDescription, error) {
	spec, err := LoadSpecFromScenario(scenarioDir)
	if err != nil {
		return nil, err
	}
	cleanScenarioDir, err := filepath.Abs(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("resolve scenario dir %s: %w", scenarioDir, err)
	}
	path := filepath.Join(cleanScenarioDir, filepath.FromSlash(TestGenieDescriptorRelPath))
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var descriptor describeDescriptor
	if err := json.Unmarshal(raw, &descriptor); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	contract := strings.TrimSpace(descriptor.Validation.Contract)
	if contract == "" {
		contract = DefaultContract
	}
	deliveryMode := strings.TrimSpace(descriptor.Validation.DeliveryMode)
	if deliveryMode == "" {
		deliveryMode = "inline"
	}

	return &ProviderDescription{
		Provider:    spec.Provider,
		Phase:       spec.Phase,
		SpecVersion: spec.Version,
		Contract:    contract,
		// A durable-run provider always executes; an inline provider executes
		// only when it declares one of the execution flags.
		DeliveryMode:      deliveryMode,
		SupportsExecution: descriptor.Validation.Execution || descriptor.Validation.IncludeExecution || deliveryMode == "durable-run",
		Build:             CurrentBuild(),
	}, nil
}

// CurrentBuild resolves this process's build provenance once. Every lookup is
// best-effort: a value that cannot be determined is left unset rather than
// guessed, because a wrong build stamp would make a freshness gate lie.
func CurrentBuild() ProviderBuild {
	var out ProviderBuild
	if info, ok := debug.ReadBuildInfo(); ok && info != nil {
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				out.Revision = setting.Value
			case "vcs.time":
				if ts, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					out.BuiltAt = ts.UTC()
				}
			}
		}
	}
	if exe, err := os.Executable(); err == nil {
		if stat, err := os.Stat(exe); err == nil {
			out.BinaryModifiedAt = stat.ModTime().UTC()
		}
		// Read the digest this binary was stamped with. Any failure leaves it
		// empty, which is the fail-open direction: a consumer must not be able
		// to conclude "stale" from a stamp we could not read.
		if manifest, ok, err := cliutil.ReadFreshnessManifest(cliutil.FreshnessManifestPath(exe)); err == nil && ok {
			out.FreshnessDigest = strings.TrimSpace(manifest.Digest)
		}
	}
	return out
}

// ToProto renders the description on the wire. It is safe to call on a nil
// receiver, which yields nil so callers can treat "not configured" uniformly.
func (d *ProviderDescription) ToProto() *scenariovalidationv1.DescribeProviderResponse {
	if d == nil {
		return nil
	}
	out := &scenariovalidationv1.DescribeProviderResponse{
		Provider:    d.Provider,
		Phase:       d.Phase,
		SpecVersion: d.SpecVersion,
		Contract:    d.Contract,
		Capabilities: &scenariovalidationv1.ProviderCapabilities{
			SupportsExecution: d.SupportsExecution,
			DeliveryMode:      d.DeliveryMode,
			SupportsFixes:     d.SupportsFixes,
		},
	}
	build := &scenariovalidationv1.ProviderBuild{
		Revision:        d.Build.Revision,
		FreshnessDigest: d.Build.FreshnessDigest,
	}
	if !d.Build.BuiltAt.IsZero() {
		build.BuiltAt = timestamppb.New(d.Build.BuiltAt)
	}
	if !d.Build.BinaryModifiedAt.IsZero() {
		build.BinaryModifiedAt = timestamppb.New(d.Build.BinaryModifiedAt)
	}
	out.Build = build
	return out
}

// Describer is an embeddable implementation of
// ScenarioValidationService.DescribeProvider.
//
// Its zero value is deliberately useful: a provider that embeds Describer
// without configuring it compiles and returns Unimplemented, which consumers
// already treat as "fall back to the legacy readiness probe". That makes
// adopting this RPC a per-provider decision rather than a fleet-wide flag day.
type Describer struct {
	response *scenariovalidationv1.DescribeProviderResponse
}

// NewDescriber builds a Describer from already-resolved provider facts.
func NewDescriber(description ProviderDescription) Describer {
	return Describer{response: description.ToProto()}
}

// LoadDescriber resolves the provider's description from its descriptor and
// returns a ready Describer. Providers that already call LoadSpecFromScenario
// can swap to this and drop a redundant read.
func LoadDescriber(scenarioDir string) (Describer, error) {
	description, err := LoadProviderDescription(scenarioDir)
	if err != nil {
		return Describer{}, err
	}
	return Describer{response: description.ToProto()}, nil
}

// WithFixes reports whether the provider implements PreviewFix/ApplyFix. Only
// the provider knows this, so it is set explicitly rather than inferred.
func (d Describer) WithFixes(supported bool) Describer {
	if d.response == nil {
		return d
	}
	cloned, _ := proto.Clone(d.response).(*scenariovalidationv1.DescribeProviderResponse)
	if cloned.GetCapabilities() == nil {
		cloned.Capabilities = &scenariovalidationv1.ProviderCapabilities{}
	}
	cloned.Capabilities.SupportsFixes = supported
	return Describer{response: cloned}
}

// Configured reports whether this Describer will answer rather than return
// Unimplemented. Useful in provider tests and conformance checks.
func (d Describer) Configured() bool { return d.response != nil }

// DescribeProvider answers the readiness contract from facts resolved at
// construction. It performs no target work — that is the entire point of the
// RPC — so it stays O(1) no matter how large the scenario being tested is.
func (d Describer) DescribeProvider(
	_ context.Context,
	_ *connect.Request[scenariovalidationv1.DescribeProviderRequest],
) (*connect.Response[scenariovalidationv1.DescribeProviderResponse], error) {
	if d.response == nil {
		return nil, connect.NewError(
			connect.CodeUnimplemented,
			errors.New("provider has not adopted DescribeProvider"),
		)
	}
	cloned, ok := proto.Clone(d.response).(*scenariovalidationv1.DescribeProviderResponse)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, errors.New("clone provider description"))
	}
	return connect.NewResponse(cloned), nil
}

// ValidationServer is the provider-implemented subset of
// ScenarioValidationService: every RPC except DescribeProvider, which the
// shared Describer answers from provider-owned facts.
//
// Splitting the interface this way is what keeps adoption to a one-line change
// at each provider's mount site. A provider never writes a DescribeProvider
// method; it composes one in.
type ValidationServer interface {
	ValidateScenario(context.Context, *connect.Request[scenariovalidationv1.ValidateScenarioRequest]) (*connect.Response[scenariovalidationv1.ValidateScenarioResponse], error)
	PreviewFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
	ApplyFix(context.Context, *connect.Request[scenariovalidationv1.FixRequest]) (*connect.Response[scenariovalidationv1.FixResponse], error)
}

// describedServer composes a provider's validation implementation with the
// shared Describer. Both are embedded, so the promoted method sets add up to
// the full generated handler interface.
type describedServer struct {
	ValidationServer
	Describer
}

// Serve composes a provider's validation implementation with a Describer into
// the full ScenarioValidationService handler.
//
// Pass the zero Describer to keep DescribeProvider unimplemented; consumers
// fall back to the legacy readiness probe, so a provider can adopt the fast
// path whenever it is ready rather than on a fleet-wide schedule.
func Serve(impl ValidationServer, describer Describer) scenariovalidationv1connect.ScenarioValidationServiceHandler {
	return describedServer{ValidationServer: impl, Describer: describer}
}
