package artifacts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"vrooli-bridge/internal/artifacts"
	"vrooli-bridge/internal/registry"

	"github.com/vrooli/api-core/discovery"
	"google.golang.org/protobuf/types/known/timestamppb"

	transferv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-sync-hub/v1/transfer"
	artifactsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/artifacts"
	"google.golang.org/protobuf/encoding/protojson"
)

// This file is the single translation point between the proto-free artifacts
// domain (its seams + DTOs) and the concrete registry service, the
// device-sync-hub directed-delivery client, and the proto wire types. The
// artifacts domain never imports a sibling domain or proto; these adapters do.

// ---- proto <-> domain translations (api-steer §7) ----

func domainToProto(d artifacts.Distribution) *artifactsv1.Distribution {
	out := &artifactsv1.Distribution{
		Id:              d.ID,
		NodeId:          d.NodeID,
		Name:            d.Name,
		SourceRef:       d.SourceRef,
		DestinationPath: d.DestinationPath,
		Status:          statusToProto(d.Status),
		DeliveryRef:     d.DeliveryRef,
		Detail:          d.Detail,
		CreatedAt:       timestamppb.New(d.CreatedAt),
	}
	if !d.UpdatedAt.IsZero() {
		out.UpdatedAt = timestamppb.New(d.UpdatedAt)
	}
	return out
}

func statusToProto(s artifacts.DeliveryStatus) artifactsv1.DeliveryStatus {
	switch s {
	case artifacts.StatusPending:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_PENDING
	case artifacts.StatusDelivered:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_DELIVERED
	case artifacts.StatusFailed:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_FAILED
	default:
		return artifactsv1.DeliveryStatus_DELIVERY_STATUS_UNSPECIFIED
	}
}

// ---- seam adapters (proto-free domain <-> concrete services) ----

// nodeReaderAdapter projects a registry node down to the artifacts TargetNode.
type nodeReaderAdapter struct {
	svc registry.Service
}

var _ artifacts.NodeReader = nodeReaderAdapter{}

func (a nodeReaderAdapter) GetTarget(ctx context.Context, id string) (artifacts.TargetNode, error) {
	n, err := a.svc.Get(ctx, id)
	if err != nil {
		var notFound registry.ErrNodeNotFound
		if errors.As(err, &notFound) {
			return artifacts.TargetNode{}, artifacts.ErrNodeNotFound{ID: id}
		}
		return artifacts.TargetNode{}, err
	}
	return artifacts.TargetNode{ID: n.ID, Revoked: n.Revoked()}, nil
}

// deviceSyncDelivery is the production DirectedDelivery. It streams the
// operator-provided source into device-sync-hub's multipart upload edge and
// records the returned item reference. Bridge does not retain the bytes, but it
// is necessarily a transient stream proxy because device-sync-hub's public
// upload contract accepts bytes, not a source URI.
//
// The hub's device-token trust model is intentionally fail-closed. The token is
// supplied through BRIDGE_DEVICE_SYNC_TOKEN, while BRIDGE_DEVICE_SYNC_TARGETS
// is a JSON object mapping Bridge node ids to the corresponding hub device ids.
// The identities are separate by design: a Bridge registry node id is not
// silently treated as a device-sync-hub device id.
type deviceSyncDelivery struct {
	endpoint   string
	token      string
	targets    map[string]string
	resolver   *discovery.Resolver
	httpClient *http.Client
}

var _ artifacts.DirectedDelivery = deviceSyncDelivery{}

func newDeviceSyncDelivery() deviceSyncDelivery {
	var targets map[string]string
	if raw := strings.TrimSpace(os.Getenv("BRIDGE_DEVICE_SYNC_TARGETS")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &targets)
	}
	return deviceSyncDelivery{
		endpoint:   strings.TrimRight(strings.TrimSpace(os.Getenv("BRIDGE_DEVICE_SYNC_URL")), "/"),
		token:      strings.TrimSpace(os.Getenv("BRIDGE_DEVICE_SYNC_TOKEN")),
		targets:    targets,
		resolver:   discovery.NewResolver(discovery.ResolverConfig{}),
		httpClient: &http.Client{},
	}
}

func (d deviceSyncDelivery) Deliver(ctx context.Context, req artifacts.DeliveryRequest) (artifacts.DeliveryResult, error) {
	targetID := strings.TrimSpace(d.targets[req.NodeID])
	if targetID == "" {
		return artifacts.DeliveryResult{}, fmt.Errorf("device-sync-hub target mapping is not configured for bridge node %q", req.NodeID)
	}
	if d.token == "" {
		return artifacts.DeliveryResult{}, errors.New("device-sync-hub device token is not configured")
	}
	endpoint, err := d.endpointURL(ctx)
	if err != nil {
		return artifacts.DeliveryResult{}, err
	}
	client := d.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	source, filename, err := openArtifactSource(ctx, req.SourceRef, req.Name, client)
	if err != nil {
		return artifacts.DeliveryResult{}, err
	}
	defer source.Close()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+"/api/v1/transfer/items", nil)
	if err != nil {
		return artifacts.DeliveryResult{}, fmt.Errorf("create device-sync-hub upload request: %w", err)
	}
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)
	httpReq.Body = io.NopCloser(pr)
	httpReq.ContentLength = -1
	go func() {
		var writeErr error
		defer func() { _ = pw.CloseWithError(writeErr) }()
		if writeErr = writer.WriteField("name", strings.TrimSpace(req.Name)); writeErr != nil {
			return
		}
		if writeErr = writer.WriteField("retention", "pinned"); writeErr != nil {
			return
		}
		if writeErr = writer.WriteField("target_device_id", targetID); writeErr != nil {
			return
		}
		var part io.Writer
		part, writeErr = writer.CreateFormFile("file", filename)
		if writeErr != nil {
			return
		}
		_, writeErr = io.Copy(part, source)
		if writeErr != nil {
			return
		}
		writeErr = writer.Close()
	}()

	httpReq.Header.Set("Content-Type", writer.FormDataContentType())
	httpReq.Header.Set("X-Device-Token", d.token)
	resp, err := client.Do(httpReq)
	if err != nil {
		return artifacts.DeliveryResult{}, fmt.Errorf("device-sync-hub upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return artifacts.DeliveryResult{}, fmt.Errorf("device-sync-hub upload returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return artifacts.DeliveryResult{}, fmt.Errorf("read device-sync-hub upload response: %w", err)
	}
	var uploaded transferv1.UploadItemResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(body, &uploaded); err != nil {
		return artifacts.DeliveryResult{}, fmt.Errorf("decode device-sync-hub upload response: %w", err)
	}
	if uploaded.Item == nil || strings.TrimSpace(uploaded.Item.Id) == "" {
		return artifacts.DeliveryResult{}, errors.New("device-sync-hub upload response did not contain an item id")
	}
	return artifacts.DeliveryResult{
		Ref:       "dsh://item/" + url.PathEscape(uploaded.Item.Id),
		Delivered: false,
		Detail:    "accepted by device-sync-hub; target receipt remains pending",
	}, nil
}

func (d deviceSyncDelivery) endpointURL(ctx context.Context) (string, error) {
	if d.endpoint != "" {
		return d.endpoint, nil
	}
	if d.resolver == nil {
		return "", errors.New("device-sync-hub URL is not configured")
	}
	endpoint, err := d.resolver.ResolveScenarioURLDefault(ctx, "device-sync-hub")
	if err != nil {
		return "", fmt.Errorf("resolve device-sync-hub URL: %w", err)
	}
	return strings.TrimRight(endpoint, "/"), nil
}

func openArtifactSource(ctx context.Context, sourceRef, requestedName string, client *http.Client) (io.ReadCloser, string, error) {
	parsed, err := url.Parse(strings.TrimSpace(sourceRef))
	if err != nil {
		return nil, "", fmt.Errorf("parse artifact source reference: %w", err)
	}
	name := filepath.Base(strings.TrimSpace(requestedName))
	if name == "." || name == "" || name == string(filepath.Separator) {
		name = "artifact"
	}
	switch parsed.Scheme {
	case "":
		file, err := os.Open(parsed.Path)
		if err != nil {
			return nil, "", fmt.Errorf("open artifact source %q: %w", sourceRef, err)
		}
		if requestedName == "" {
			name = filepath.Base(parsed.Path)
		}
		return file, name, nil
	case "file":
		if parsed.Host != "" && parsed.Host != "localhost" {
			return nil, "", fmt.Errorf("artifact file source host %q is not allowed", parsed.Host)
		}
		path, err := url.PathUnescape(parsed.Path)
		if err != nil {
			return nil, "", fmt.Errorf("decode artifact file source: %w", err)
		}
		file, err := os.Open(path)
		if err != nil {
			return nil, "", fmt.Errorf("open artifact source %q: %w", sourceRef, err)
		}
		if requestedName == "" {
			name = filepath.Base(path)
		}
		return file, name, nil
	case "http", "https":
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
		if err != nil {
			return nil, "", fmt.Errorf("create artifact source request: %w", err)
		}
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, "", fmt.Errorf("fetch artifact source: %w", err)
		}
		if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
			resp.Body.Close()
			return nil, "", fmt.Errorf("artifact source returned %s", resp.Status)
		}
		if requestedName == "" && parsed.Path != "" {
			name = filepath.Base(parsed.Path)
		}
		return resp.Body, name, nil
	default:
		return nil, "", fmt.Errorf("artifact source scheme %q is unsupported; use a local path, file://, or HTTP(S)", parsed.Scheme)
	}
}
