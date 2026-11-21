package programlibrary

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	libraryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/library"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/program-runtime/v1/shared"
)

type fakeLister struct {
	entries []*sharedv1.LibraryProgram
	err     error
}

func (f fakeLister) ListLibrary(context.Context, *connect.Request[libraryv1.ListLibraryRequest]) (*connect.Response[libraryv1.ListLibraryResponse], error) {
	return connect.NewResponse(&libraryv1.ListLibraryResponse{Programs: f.entries}), f.err
}

// Declared contracts that call bindings are the library; invalid contracts,
// entries without bindings and older duplicates of a name are not.
func TestLibraryKeepsValidBindingProgramsOncePerName(t *testing.T) {
	source := &Source{lister: fakeLister{entries: []*sharedv1.LibraryProgram{
		{Name: "test-genie.iterate", Scenario: "test-genie", Kind: "contract", CalledBindingIds: []string{"test-genie/validation/create"}},
		{Name: "probe", Scenario: "x", ValidationError: "bad contract", CalledBindingIds: []string{"x/y/z"}},
		{Name: "promoted-no-bindings"},
		{Name: "test-genie.iterate", CalledBindingIds: []string{"old/b/c"}},
	}}}
	library := source.Library(context.Background())
	if library.Unavailable != "" || len(library.Programs) != 1 {
		t.Fatalf("library = %+v", library)
	}
	if p := library.Programs[0]; p.Name != "test-genie.iterate" || p.Scenario != "test-genie" || p.Bindings[0] != "test-genie/validation/create" {
		t.Fatalf("program = %+v", p)
	}
}

// A failing program-runtime is an unavailable library with a reason, so the
// sequences read as unmatched instead of as having no program.
func TestLibraryUnavailableCarriesTheReason(t *testing.T) {
	library := (&Source{lister: fakeLister{err: errors.New("down")}}).Library(context.Background())
	if library.Unavailable == "" || len(library.Programs) != 0 {
		t.Fatalf("library = %+v", library)
	}
	unresolved := &Source{resolve: func(context.Context) (string, error) { return "", errors.New("not running") }}
	if got := unresolved.Library(context.Background()); got.Unavailable == "" {
		t.Fatalf("unresolved library = %+v", got)
	}
}
