package variantspace_test

import (
	"bytes"
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	landingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1"

	vsH "landing-page-react-vite-api/handlers/variantspace"
	internalvariantspace "landing-page-react-vite-api/internal/variantspace"
)

func TestGetVariantSpaceServesVerbatimJSON(t *testing.T) {
	data := []byte(`{"_name":"T","_schemaVersion":1,"axes":{"persona":{"variants":[{"id":"ops","label":"Ops"}]}}}`)
	space, err := internalvariantspace.Parse(data)
	require.NoError(t, err)

	h := vsH.NewConnectHandler(vsH.Deps{Space: space})
	resp, err := h.GetVariantSpace(context.Background(), connect.NewRequest(&landingv1.GetVariantSpaceRequest{}))
	require.NoError(t, err)
	require.True(t, bytes.Equal(data, resp.Msg.RawJson))
}

func TestGetVariantSpaceDefaultsWhenNil(t *testing.T) {
	h := vsH.NewConnectHandler(vsH.Deps{})
	resp, err := h.GetVariantSpace(context.Background(), connect.NewRequest(&landingv1.GetVariantSpaceRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, resp.Msg.RawJson)
}
