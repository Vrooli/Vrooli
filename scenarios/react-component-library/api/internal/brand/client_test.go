package brand

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	assignmentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments"
	assignmentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/assignments/assignments_v1connect"
	brandsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands"
	brandsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/brand-manager/v1/brands/brands_v1connect"
)

// fakeBrandManager speaks just enough of Brand Manager's Connect surface for
// the client contract. The generated Connect-Go client negotiates protobuf by
// default, so responses are encoded the same way the real API encodes them.
func fakeBrandManager(t *testing.T, status *assignmentsv1.ScenarioStatus, brand *brandsv1.Brand, mark string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case assignmentsconnect.AssignmentsServiceGetScenarioStatusProcedure:
			writeProto(t, w, &assignmentsv1.GetScenarioStatusResponse{Status: status})
		case brandsconnect.BrandsServiceGetBrandProcedure:
			if brand == nil {
				http.Error(w, "boom", http.StatusInternalServerError)
				return
			}
			writeProto(t, w, &brandsv1.GetBrandResponse{Brand: brand})
		case "/api/v1/brand-manager/brands/mark":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte(mark))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func writeProto(t *testing.T, w http.ResponseWriter, message proto.Message) {
	t.Helper()
	raw, err := proto.Marshal(message)
	require.NoError(t, err)
	w.Header().Set("Content-Type", "application/proto")
	_, _ = w.Write(raw)
}

func clientFor(server *httptest.Server) *Client {
	return &Client{
		Scenario:   ScenarioName,
		HTTPClient: &http.Client{Timeout: time.Second},
		resolveBaseURL: func(context.Context) (string, error) {
			return server.URL, nil
		},
	}
}

func TestResolveAssignedBrand(t *testing.T) {
	server := fakeBrandManager(t,
		&assignmentsv1.ScenarioStatus{Scenario: ScenarioName, HasBrand: true, BrandId: "brand-1"},
		&brandsv1.Brand{
			Id:          "brand-1",
			Name:        "Aperture",
			Identity:    &brandsv1.Identity{DisplayName: "Aperture Labs"},
			Colors:      &brandsv1.Colors{Accent: "#C0FFEE"},
			MarkAssetId: "asset-1",
		},
		`<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
	)

	brand, err := clientFor(server).Resolve(context.Background())
	require.NoError(t, err)
	require.True(t, brand.Assigned)
	require.Equal(t, "Aperture", brand.Name)
	require.NotNil(t, brand.Accent)
	require.Equal(t, "#C0FFEE", *brand.Accent)
	require.NotNil(t, brand.Wordmark)
	require.Equal(t, "Aperture Labs", *brand.Wordmark)
	require.NotNil(t, brand.Mark)
	require.Contains(t, *brand.Mark, "data:image/svg+xml;base64,")
}

func TestResolveWithoutAssignment(t *testing.T) {
	server := fakeBrandManager(t, &assignmentsv1.ScenarioStatus{Scenario: ScenarioName, HasBrand: false}, nil, "")
	brand, err := clientFor(server).Resolve(context.Background())
	require.NoError(t, err)
	require.Equal(t, Brand{}, brand)
}

func TestResolveDegradesWhenBrandManagerIsUnreachable(t *testing.T) {
	client := &Client{Scenario: ScenarioName, HTTPClient: http.DefaultClient, resolveBaseURL: func(context.Context) (string, error) {
		return "", errors.New("discovery failed")
	}}

	brand, err := client.Resolve(context.Background())
	require.NoError(t, err, "an unavailable Brand Manager is a normal unassigned state")
	require.Equal(t, Brand{}, brand)
}

func TestResolveDegradesWhenBrandReadFails(t *testing.T) {
	server := fakeBrandManager(t,
		&assignmentsv1.ScenarioStatus{HasBrand: true, BrandId: "brand-1"},
		nil,
		"",
	)
	brand, err := clientFor(server).Resolve(context.Background())
	require.NoError(t, err)
	require.Equal(t, Brand{}, brand)
}

func TestResolveIgnoresRelativeLogoPath(t *testing.T) {
	server := fakeBrandManager(t,
		&assignmentsv1.ScenarioStatus{HasBrand: true, BrandId: "brand-1"},
		&brandsv1.Brand{Id: "brand-1", Name: "Acme", Identity: &brandsv1.Identity{LogoPath: "ui/public/logo.svg"}},
		"",
	)
	brand, err := clientFor(server).Resolve(context.Background())
	require.NoError(t, err)
	require.True(t, brand.Assigned)
	require.Nil(t, brand.Mark, "a repo-relative logo path is not browser-reachable")
}

func TestResolveUsesAbsoluteLogoPath(t *testing.T) {
	logo := "https://cdn.example.com/logo.svg"
	server := fakeBrandManager(t,
		&assignmentsv1.ScenarioStatus{HasBrand: true, BrandId: "brand-1"},
		&brandsv1.Brand{Id: "brand-1", Name: "Acme", Identity: &brandsv1.Identity{LogoPath: logo}},
		"",
	)
	brand, err := clientFor(server).Resolve(context.Background())
	require.NoError(t, err)
	require.NotNil(t, brand.Mark)
	require.Equal(t, logo, *brand.Mark)
}

func TestBrandJSONShape(t *testing.T) {
	accent := "#123456"
	payload := Brand{Assigned: true, Name: "Acme", Accent: &accent}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"assigned":true,"name":"Acme","mark":null,"accent":"#123456","wordmark":null}`, string(raw),
		"the shell contract uses explicit nulls, not omitted keys")

	raw, err = json.Marshal(Brand{})
	require.NoError(t, err)
	require.JSONEq(t, `{"assigned":false,"name":"","mark":null,"accent":null,"wordmark":null}`, string(raw))
}
