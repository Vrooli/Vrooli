//go:build linux

package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
	offersconnect "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers/offers_v1connect"
	"google.golang.org/protobuf/encoding/protojson"
)

type cf7CatalogObservation struct {
	NodeCount int
	BodyBytes int
	Names     []string
	Digest    string
}

// observeCF7Catalog reads the Offer Desk graph that the paid endpoint serves
// after settlement. The paid handler and CatalogService read the same store;
// this test-only read gives each receipt a measured, non-secret result record.
func observeCF7Catalog(ctx context.Context, catalogURL string) (cf7CatalogObservation, error) {
	var observation cf7CatalogObservation
	parsed, err := url.ParseRequestURI(catalogURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return observation, fmt.Errorf("invalid CF7 Offer Desk resource URL %q", catalogURL)
	}
	baseURL := parsed.Scheme + "://" + parsed.Host
	client := offersconnect.NewCatalogServiceClient(&http.Client{Timeout: 15 * time.Second}, baseURL)
	response, err := client.ListNodes(ctx, connect.NewRequest(&offerspb.ListNodesRequest{}))
	if err != nil {
		return observation, fmt.Errorf("read Offer Desk catalog result: %w", err)
	}
	if len(response.Msg.GetNodes()) == 0 {
		return observation, fmt.Errorf("Offer Desk returned an empty research catalog")
	}
	canonical, err := protojson.Marshal(response.Msg)
	if err != nil {
		return observation, fmt.Errorf("canonicalize Offer Desk catalog result: %w", err)
	}
	digest := sha256.Sum256(canonical)
	for _, node := range response.Msg.GetNodes() {
		if name := strings.TrimSpace(node.GetName()); name != "" {
			observation.Names = append(observation.Names, name)
		}
	}
	sort.Strings(observation.Names)
	observation.NodeCount = len(response.Msg.GetNodes())
	observation.BodyBytes = len(canonical)
	observation.Digest = "sha256:" + hex.EncodeToString(digest[:])
	return observation, nil
}
