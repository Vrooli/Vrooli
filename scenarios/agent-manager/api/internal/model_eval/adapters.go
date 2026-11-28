package model_eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/discovery"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing/routing_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	capacitypb "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity"
	capacityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/system-monitor/v1/capacity/capacityconnect"
)

// DiscoveredGateway uses the public AI Gateway Connect service. It does not
// call a provider directly and does not persist returned output.
type DiscoveredGateway struct {
	Client interface {
		PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error)
		ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
	}
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
}

func (g *DiscoveredGateway) Preview(ctx context.Context, request *sharedv1.GatewayRequest) (*routingv1.PreviewRouteResponse, error) {
	client, err := g.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.PreviewRoute(ctx, connect.NewRequest(&routingv1.PreviewRouteRequest{Request: request}))
	if err != nil || response == nil || response.Msg == nil {
		return nil, errors.New("AI Gateway route preview is unavailable")
	}
	return response.Msg, nil
}

func (g *DiscoveredGateway) Execute(ctx context.Context, request *routingv1.ExecuteRouteRequest) (*routingv1.ExecuteRouteResponse, error) {
	client, err := g.client(ctx)
	if err != nil {
		return nil, err
	}
	response, err := client.ExecuteRoute(ctx, connect.NewRequest(request))
	if err != nil || response == nil || response.Msg == nil {
		return nil, errors.New("AI Gateway execution returned no metadata receipt")
	}
	return response.Msg, nil
}

func (g *DiscoveredGateway) client(ctx context.Context) (interface {
	PreviewRoute(context.Context, *connect.Request[routingv1.PreviewRouteRequest]) (*connect.Response[routingv1.PreviewRouteResponse], error)
	ExecuteRoute(context.Context, *connect.Request[routingv1.ExecuteRouteRequest]) (*connect.Response[routingv1.ExecuteRouteResponse], error)
}, error) {
	if g != nil && g.Client != nil {
		return g.Client, nil
	}
	resolver := g.ResolveURL
	if resolver == nil {
		resolver = func(ctx context.Context) (string, error) {
			return discovery.ResolveScenarioURLDefault(ctx, "ai-gateway")
		}
	}
	baseURL, err := resolver(ctx)
	if err != nil || strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("AI Gateway service discovery failed")
	}
	httpClient := connect.HTTPClient(&http.Client{Timeout: 3 * time.Minute})
	if g.HTTPClient != nil {
		httpClient = g.HTTPClient
	}
	return routingconnect.NewRoutingServiceClient(httpClient, baseURL), nil
}

// SystemMonitorCapacityGate reads the fresh measured host GPU inventory. The
// operation-scoped, no-reclaim claim is still taken by AI Gateway on each call.
type SystemMonitorCapacityGate struct {
	Client     capacityconnect.CapacityServiceClient
	HTTPClient connect.HTTPClient
	ResolveURL func(context.Context) (string, error)
	Now        func() time.Time
}

func (g SystemMonitorCapacityGate) MeasureAndCheck(ctx context.Context, required int64) (CapacityObservation, error) {
	client := g.Client
	if client == nil {
		resolver := g.ResolveURL
		if resolver == nil {
			resolver = func(ctx context.Context) (string, error) {
				return discovery.ResolveScenarioURLDefault(ctx, "system-monitor")
			}
		}
		baseURL, err := resolver(ctx)
		if err != nil || strings.TrimSpace(baseURL) == "" {
			return CapacityObservation{}, errors.New("System Monitor capacity discovery failed")
		}
		httpClient := connect.HTTPClient(&http.Client{Timeout: 20 * time.Second})
		if g.HTTPClient != nil {
			httpClient = g.HTTPClient
		}
		client = capacityconnect.NewCapacityServiceClient(httpClient, baseURL)
	}
	response, err := client.GetCapacityOverview(ctx, connect.NewRequest(&capacitypb.GetCapacityOverviewRequest{}))
	if err != nil || response == nil || response.Msg == nil || !response.Msg.GetSuccess() || !response.Msg.GetSensingAvailable() {
		return CapacityObservation{}, errors.New("live GPU sensing is unavailable")
	}
	var available int64
	for _, gpu := range response.Msg.GetGpus() {
		if gpu.GetFreeBytes() > available {
			available = gpu.GetFreeBytes()
		}
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	return CapacityObservation{ObservedAt: now, Available: available, Fits: available >= required}, nil
}

// FilesystemDiskGate measures the filesystem that holds the LM2 research
// workspace. No prompts, outputs, or per-case artifacts are written there.
type FilesystemDiskGate struct {
	Path         string
	ArtifactPath string
	Now          func() time.Time
}

func (g FilesystemDiskGate) Measure(context.Context) (DiskObservation, error) {
	if strings.TrimSpace(g.Path) == "" {
		return DiskObservation{}, errors.New("disk measurement path is not configured")
	}
	usage, err := readFilesystemBytes(g.Path)
	if err != nil {
		return DiskObservation{}, errors.New("model-eval filesystem measurement failed")
	}
	artifactBytes, artifactHash, err := measureArtifactTree(g.ArtifactPath)
	if err != nil {
		return DiskObservation{}, errors.New("model-eval task artifact measurement failed")
	}
	now := time.Now().UTC()
	if g.Now != nil {
		now = g.Now().UTC()
	}
	return DiskObservation{ObservedAt: now, FreeBytes: usage, ArtifactBytes: artifactBytes, ArtifactHash: artifactHash}, nil
}

type artifactFile struct {
	Path   string `json:"path"`
	Bytes  int64  `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// measureArtifactTree hashes the exact task-owned packet tree without retaining
// file contents. Symlinks and special files are refused so the bound cannot
// silently include data outside the packet directory.
func measureArtifactTree(root string) (int64, string, error) {
	if strings.TrimSpace(root) == "" {
		return 0, "", errors.New("task artifact path is not configured")
	}
	entries := make([]artifactFile, 0, 8)
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return errors.New("task artifact tree contains a non-regular file")
		}
		info, err := entry.Info()
		if err != nil || info.Size() < 0 {
			return errors.New("task artifact metadata is unavailable")
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return errors.New("task artifact path is invalid")
		}
		if total+info.Size() > MaximumArtifactSize || len(entries) >= 4096 {
			return errors.New("task artifact tree exceeds the measurable bound")
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		openedInfo, statErr := file.Stat()
		if statErr != nil || !os.SameFile(info, openedInfo) || openedInfo.Size() != info.Size() || !openedInfo.ModTime().Equal(info.ModTime()) {
			_ = file.Close()
			return errors.New("task artifact changed before content measurement")
		}
		hasher := sha256.New()
		copied, copyErr := io.Copy(hasher, io.LimitReader(file, MaximumArtifactSize+1))
		finalInfo, finalStatErr := file.Stat()
		closeErr := file.Close()
		if copyErr != nil || closeErr != nil || finalStatErr != nil || !os.SameFile(openedInfo, finalInfo) ||
			copied != info.Size() || finalInfo.Size() != info.Size() || !finalInfo.ModTime().Equal(openedInfo.ModTime()) {
			return errors.New("task artifact changed while measured")
		}
		entries = append(entries, artifactFile{Path: filepath.ToSlash(relative), Bytes: copied, SHA256: hex.EncodeToString(hasher.Sum(nil))})
		total += copied
		return nil
	})
	if err != nil {
		return total, "", err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	manifest, err := json.Marshal(entries)
	if err != nil {
		return 0, "", fmt.Errorf("encode task artifact manifest: %w", err)
	}
	digest := sha256.Sum256(manifest)
	return total, hex.EncodeToString(digest[:]), nil
}
