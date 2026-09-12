package sources

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	checksv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
)

var errPortabilityUnwired = errors.New("the capability grid reader is not configured")

// CheckPlatformsSourceID identifies the check-platform-declaration source in
// availability reporting.
const CheckPlatformsSourceID = "vrooli-autoheal/check-platforms"

// CheckPlatforms is one check's declared host OS applicability, read from the
// owner's typed surface rather than derived by parsing its Go source. A parsed
// declaration goes stale silently the moment the check changes and cannot be
// read from a deployed binary at all.
type CheckPlatforms struct {
	CheckID  string
	Category string
	// Platforms is the declared host OS list. An EMPTY list means the check
	// applies to every platform. It is not "unknown", and a consumer that
	// treats it as unknown turns a universally applicable check into a gap.
	Platforms []string
}

// AppliesTo reports whether the check applies on one host OS.
func (c CheckPlatforms) AppliesTo(hostOS string) bool {
	if len(c.Platforms) == 0 {
		return true
	}
	for _, declared := range c.Platforms {
		if declared == hostOS {
			return true
		}
	}
	return false
}

// CheckPlatformsReader reads the autoheal check registry's platform
// declarations over its typed surface.
type CheckPlatformsReader struct {
	Resolver *discovery.Resolver
	HTTP     *http.Client
}

func (r CheckPlatformsReader) ReadPlatforms(ctx context.Context) ([]CheckPlatforms, error) {
	resolver := r.Resolver
	if resolver == nil {
		resolver = discovery.NewResolver(discovery.ResolverConfig{})
	}
	httpClient := r.HTTP
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	base, err := resolver.ResolveScenarioURLDefault(ctx, "vrooli-autoheal")
	if err != nil {
		return nil, err
	}
	client := checksconnect.NewChecksServiceClient(httpClient, base)
	response, err := client.ListChecks(ctx, connect.NewRequest(&checksv1.ListChecksRequest{}))
	if err != nil {
		return nil, err
	}
	items := response.Msg.GetChecks()
	out := make([]CheckPlatforms, 0, len(items))
	for _, info := range items {
		out = append(out, CheckPlatforms{
			CheckID:   info.GetId(),
			Category:  info.GetCategory(),
			Platforms: append([]string(nil), info.GetPlatforms()...),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CheckID < out[j].CheckID })
	return out, nil
}

// ReadCheckPlatforms runs the declaration read under the standard per-source
// deadline.
func ReadCheckPlatforms(ctx context.Context, reader CheckPlatformsReader, timeout time.Duration) TypedResult[[]CheckPlatforms] {
	return ReadTyped(ctx, CheckPlatformsSourceID, reader.ReadPlatforms, timeout)
}
