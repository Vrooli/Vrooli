// Package programlibrary reads the program library from program-runtime for
// opportunity mining: which programs exist, who owns them and which governed
// bindings they call.
package programlibrary

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"agent-manager/internal/opportunities"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	libraryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library/library_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

// listLimit bounds the stored programs requested; declared contracts are
// always returned in full.
const listLimit = 500

type lister interface {
	ListLibrary(context.Context, *connect.Request[libraryv1.ListLibraryRequest]) (*connect.Response[libraryv1.ListLibraryResponse], error)
}

// Source reads the library from the program-runtime that discovery resolves.
type Source struct {
	resolve func(context.Context) (string, error)
	client  *http.Client
	lister  lister
}

// New is a Source over the live program-runtime.
func New() *Source {
	return &Source{
		resolve: func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "program-runtime")
		},
		client: &http.Client{Timeout: 20 * time.Second},
	}
}

// Library is the current library, or a library marked unavailable with the
// reason when program-runtime cannot be reached or answers with an error:
// sequences are then reported as unmatched, never as having no program.
func (s *Source) Library(ctx context.Context) opportunities.Library {
	l := s.lister
	if l == nil {
		base, err := s.resolve(ctx)
		if err != nil {
			return opportunities.Library{Unavailable: fmt.Sprintf("resolve program-runtime: %v", err)}
		}
		l = libraryconnect.NewLibraryServiceClient(s.client, strings.TrimRight(base, "/"))
	}
	response, err := l.ListLibrary(ctx, connect.NewRequest(&libraryv1.ListLibraryRequest{Limit: listLimit}))
	if err != nil {
		return opportunities.Library{Unavailable: fmt.Sprintf("list program library: %v", err)}
	}
	return opportunities.Library{Programs: programs(response.Msg.GetPrograms())}
}

// programs are the valid library entries that call at least one governed
// binding, one per name (the first listed, which is the current version).
func programs(entries []*sharedv1.LibraryProgram) []opportunities.Program {
	seen := map[string]bool{}
	var out []opportunities.Program
	for _, e := range entries {
		if e.GetName() == "" || e.GetValidationError() != "" || len(e.GetCalledBindingIds()) == 0 || seen[e.GetName()] {
			continue
		}
		seen[e.GetName()] = true
		out = append(out, opportunities.Program{Name: e.GetName(), Scenario: e.GetScenario(), Purpose: e.GetPurpose(), Bindings: e.GetCalledBindingIds()})
	}
	return out
}
