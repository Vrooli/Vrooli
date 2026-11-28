package adoptions

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	adoptionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/adoptions"
	"react-component-library/internal/adoptions"
	"react-component-library/internal/components"
)

type resolveLibrary struct {
	adoptions.LibraryReader
	component components.Component
	err       error
}

func (l resolveLibrary) Get(context.Context, string) (components.Component, error) {
	return l.component, l.err
}

type resolveSlot struct {
	slot string
	err  error
}

func (s resolveSlot) Slot(context.Context, string) (string, error) { return s.slot, s.err }

func TestResolveAdoptionPathRequiresConfiguredResolverAndSlotReader(t *testing.T) {
	h := NewConnectHandler(Deps{})
	_, err := h.ResolveAdoptionPath(context.Background(), connect.NewRequest(&adoptionsv1.ResolveAdoptionPathRequest{}))
	require.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
}

func TestResolveAdoptionPathOverrideUsesSlugAndSurvivesSlotLookupFailure(t *testing.T) {
	h := NewConnectHandler(Deps{
		Resolver:   adoptions.NewResolver(nil, ""),
		SlotReader: resolveSlot{err: errors.New("index unavailable")},
		Library:    resolveLibrary{component: components.Component{ID: "cmp", Slug: "Button", DisplayName: "A Button"}},
	})
	got, err := h.ResolveAdoptionPath(context.Background(), connect.NewRequest(&adoptionsv1.ResolveAdoptionPathRequest{
		ComponentId: "cmp", Scenario: "demo", OverridePath: "ui/custom/Button.tsx",
	}))
	require.NoError(t, err)
	require.Equal(t, "ui/custom/Button.tsx", got.Msg.Path)
	require.Equal(t, adoptionsv1.ResolveSource_RESOLVE_SOURCE_EXPLICIT, got.Msg.Source)
	require.Empty(t, got.Msg.Slot)
}

func TestResolveAdoptionPathRejectsTraversalOverride(t *testing.T) {
	h := NewConnectHandler(Deps{
		Resolver: adoptions.NewResolver(nil, ""), SlotReader: resolveSlot{},
		Library: resolveLibrary{component: components.Component{Slug: "Button"}},
	})
	_, err := h.ResolveAdoptionPath(context.Background(), connect.NewRequest(&adoptionsv1.ResolveAdoptionPathRequest{
		ComponentId: "cmp", Scenario: "demo", OverridePath: "../outside.tsx",
	}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestResolveAdoptionPathRequiresScenarioOrTemplate(t *testing.T) {
	h := NewConnectHandler(Deps{
		Resolver: adoptions.NewResolver(nil, ""), SlotReader: resolveSlot{},
		Library: resolveLibrary{component: components.Component{DisplayName: "Button"}},
	})
	_, err := h.ResolveAdoptionPath(context.Background(), connect.NewRequest(&adoptionsv1.ResolveAdoptionPathRequest{ComponentId: "cmp"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestResolveAdoptionPathReturnsLibraryError(t *testing.T) {
	want := errors.New("component index unavailable")
	h := NewConnectHandler(Deps{Resolver: adoptions.NewResolver(nil, ""), SlotReader: resolveSlot{}, Library: resolveLibrary{err: want}})
	_, err := h.ResolveAdoptionPath(context.Background(), connect.NewRequest(&adoptionsv1.ResolveAdoptionPathRequest{ComponentId: "cmp", Scenario: "demo", OverridePath: "Button.tsx"}))
	require.Equal(t, connect.CodeInternal, connect.CodeOf(err))
}
