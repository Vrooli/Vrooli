package artifacts

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/api-core/coreset"
	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

const MaxClosureShipmentBytes int64 = 2 << 30

type ClosureShipmentFramePusher interface {
	PushClosureShipment(context.Context, string, *channelv1.ClosureShipmentCommand) (int, error)
}
type ClosureShipmentSourceOpener func(context.Context, string) (io.ReadCloser, error)

type ClosureShipmentCoordinator struct {
	service *ClosureShipmentService
	pusher  ClosureShipmentFramePusher
	open    ClosureShipmentSourceOpener
}

func NewClosureShipmentCoordinator(service *ClosureShipmentService, pusher ClosureShipmentFramePusher, open ClosureShipmentSourceOpener) *ClosureShipmentCoordinator {
	return &ClosureShipmentCoordinator{service: service, pusher: pusher, open: open}
}

func (c *ClosureShipmentCoordinator) Start(ctx context.Context, req ClosureShipmentRequest) (ClosureShipment, error) {
	if c.service == nil || c.pusher == nil || c.open == nil {
		return ClosureShipment{}, errors.New("closure shipment coordinator is not configured")
	}
	if err := validateClosureShipment(req); err != nil {
		return ClosureShipment{}, err
	}
	if prior, err := c.service.Get(ctx, req.IdempotencyKey); err == nil {
		if !shipmentMatchesRequest(prior, req) {
			return prior, errors.New("idempotency key already belongs to a different closure shipment")
		}
		switch prior.State {
		case ShipmentActivated, ShipmentRolledBack, ShipmentRejected:
			return prior, nil
		case ShipmentUncertain:
			if err := c.checkTarget(ctx, prior); err != nil {
				return prior, err
			}
			return prior, c.push(ctx, prior.TargetID, closureCommandFromShipment(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_RESUME, prior))
		case ShipmentVerified:
			if err := c.checkTarget(ctx, prior); err != nil {
				return prior, err
			}
			return prior, c.push(ctx, prior.TargetID, closureCommandFromShipment(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, prior))
		case ShipmentPrepared, ShipmentStaged:
			// Continue below after re-reading and validating the source archive.
		default:
			return prior, fmt.Errorf("unsupported durable closure shipment state %q", prior.State)
		}
	}
	paths := map[string]ClosureFile{}
	for _, f := range req.Files {
		paths[f.Path] = f
	}
	source, err := c.open(ctx, req.SourceRef)
	if err != nil {
		return ClosureShipment{}, fmt.Errorf("open closure source: %w", err)
	}
	tmp, err := os.MkdirTemp("", "bridge-closure-source-*")
	if err != nil {
		source.Close()
		return ClosureShipment{}, err
	}
	defer os.RemoveAll(tmp)
	defer source.Close()
	files, err := stageArchive(source, tmp, paths, req.InventoryDigest)
	if err != nil {
		return ClosureShipment{}, err
	}
	req.Files = files
	shipment, err := c.service.Prepare(ctx, req)
	if err != nil {
		return shipment, err
	}
	if shipment.State == ShipmentActivated {
		return shipment, nil
	}
	if shipment.State == ShipmentRolledBack || shipment.State == ShipmentRejected {
		return shipment, nil
	}
	if err := c.checkTarget(ctx, shipment); err != nil {
		return shipment, err
	}
	pb := closureShipmentProto(shipment, req)
	if shipment.State == ShipmentPrepared {
		base := closureCommand(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_BEGIN, pb)
		base.Request = pb
		if err := c.push(ctx, shipment.TargetID, base); err != nil {
			return shipment, err
		}
		// BEGIN is a separate idempotent transition. Wait for the authenticated
		// target receipt before sending content so a lost receipt cannot make a
		// retry race a target that has already advanced.
		return shipment, nil
	}
	if shipment.State == ShipmentStaged {
		if err := c.pushChunks(ctx, shipment, pb, files, tmp); err != nil {
			return shipment, err
		}
		if err := c.push(ctx, shipment.TargetID, closureCommand(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_VERIFY, pb)); err != nil {
			return shipment, err
		}
		return shipment, c.push(ctx, shipment.TargetID, closureCommand(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ACTIVATE, pb))
	}
	return shipment, fmt.Errorf("closure shipment %s is not resumable from %s", shipment.ID, shipment.State)
}

func (c *ClosureShipmentCoordinator) pushChunks(ctx context.Context, shipment ClosureShipment, pb *channelv1.ClosureShipmentRequest, files []ClosureFile, tempDir string) error {
	for index, file := range files {
		f, err := os.Open(filepath.Join(tempDir, filepath.FromSlash(file.Path)))
		if err != nil {
			return err
		}
		// ServerFrame is base64 encoded in the signed SSE envelope; keep each
		// binary chunk below the reader's 1 MiB JSON line limit.
		buf := make([]byte, 1<<19)
		var offset uint64
		for {
			n, readErr := f.Read(buf)
			if n > 0 {
				part := append([]byte(nil), buf[:n]...)
				sum := sha256.Sum256(part)
				cmd := closureCommand(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_FILE_CHUNK, pb)
				cmd.FileIndex = uint32(index)
				cmd.OffsetBytes = offset
				cmd.Content = part
				cmd.ChunkSha256 = hex.EncodeToString(sum[:])
				if err := c.push(ctx, shipment.TargetID, cmd); err != nil {
					f.Close()
					return err
				}
				offset += uint64(n)
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				f.Close()
				return readErr
			}
		}
		if err := f.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (c *ClosureShipmentCoordinator) Reconcile(ctx context.Context, id string) (ClosureShipment, error) {
	shipment, err := c.Get(ctx, id)
	if err != nil {
		return ClosureShipment{}, err
	}
	if err := c.checkTarget(ctx, shipment); err != nil {
		return shipment, err
	}
	return shipment, c.push(ctx, shipment.TargetID, closureCommandFromShipment(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_RESUME, shipment))
}

func (c *ClosureShipmentCoordinator) Rollback(ctx context.Context, id string) (ClosureShipment, error) {
	shipment, err := c.Get(ctx, id)
	if err != nil {
		return ClosureShipment{}, err
	}
	if shipment.State != ShipmentStaged && shipment.State != ShipmentVerified && shipment.State != ShipmentActivated && shipment.State != ShipmentUncertain {
		return shipment, fmt.Errorf("closure shipment cannot roll back from %s", shipment.State)
	}
	if err := c.checkTarget(ctx, shipment); err != nil {
		return shipment, err
	}
	return shipment, c.push(ctx, shipment.TargetID, closureCommandFromShipment(channelv1.ClosureShipmentAction_CLOSURE_SHIPMENT_ACTION_ROLLBACK, shipment))
}

func (c *ClosureShipmentCoordinator) checkTarget(ctx context.Context, shipment ClosureShipment) error {
	if c.service.TargetPolicy == nil {
		return errors.New("closure target policy is not configured")
	}
	return c.service.TargetPolicy(ctx, shipment)
}

func closureCommandFromShipment(action channelv1.ClosureShipmentAction, x ClosureShipment) *channelv1.ClosureShipmentCommand {
	return &channelv1.ClosureShipmentCommand{Action: action, ShipmentId: x.ID, IdempotencyKey: x.IdempotencyKey, TargetId: x.TargetID, SourceRef: x.SourceRef, Destination: x.Destination, ClosureDigest: x.ClosureDigest, InventoryDigest: x.InventoryDigest, PriorRevision: x.PriorRevision, NewRevision: x.NewRevision, SelectionReceiptRef: x.SelectionReceiptRef, SelectionDigest: x.SelectionDigest, DeploymentId: x.DeploymentID, MachineId: x.MachineID, NodeId: x.NodeID, EnrollmentGeneration: x.EnrollmentGeneration, DesiredRevision: x.DesiredRevision}
}

func (c *ClosureShipmentCoordinator) Get(ctx context.Context, id string) (ClosureShipment, error) {
	if c.service == nil {
		return ClosureShipment{}, errors.New("closure shipment service is not configured")
	}
	return c.service.Get(ctx, id)
}

func (c *ClosureShipmentCoordinator) push(ctx context.Context, target string, cmd *channelv1.ClosureShipmentCommand) error {
	count, err := c.pusher.PushClosureShipment(ctx, target, cmd)
	if err != nil {
		return err
	}
	if count < 1 {
		return errors.New("target node is not connected to accept closure shipment command")
	}
	return nil
}

func stageArchive(source io.Reader, tempDir string, manifest map[string]ClosureFile, expectedDigest string) ([]ClosureFile, error) {
	br := bufio.NewReader(source)
	peek, err := br.Peek(2)
	if err != nil && err != io.EOF {
		return nil, err
	}
	var reader io.Reader = br
	var gz *gzip.Reader
	if len(peek) == 2 && peek[0] == 0x1f && peek[1] == 0x8b {
		gz, err = gzip.NewReader(br)
		if err != nil {
			return nil, fmt.Errorf("open compressed closure archive: %w", err)
		}
		defer gz.Close()
		reader = gz
	}
	tarReader := tar.NewReader(reader)
	seen := map[string]bool{}
	var files []ClosureFile
	var total int64
	for {
		h, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read closure archive: %w", err)
		}
		name := path.Clean(h.Name)
		if name == "." || name != h.Name || strings.HasPrefix(name, "../") || path.IsAbs(name) || strings.Contains(name, "\\") {
			return nil, fmt.Errorf("unsafe archive path %q", h.Name)
		}
		if h.Typeflag == tar.TypeDir {
			if !isManifestDirectory(name, manifest) {
				return nil, fmt.Errorf("unexpected archive directory %q", name)
			}
			continue
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("unsupported archive member type for %q", name)
		}
		entry, ok := manifest[name]
		if !ok {
			return nil, fmt.Errorf("extra closure archive member %q", name)
		}
		if seen[name] {
			return nil, fmt.Errorf("duplicate closure archive member %q", name)
		}
		seen[name] = true
		if h.Size < 0 || h.Size > MaxClosureShipmentBytes-total {
			return nil, errors.New("closure archive exceeds shipment size limit")
		}
		total += h.Size
		if h.Size != entry.Size || coreset.NormalizeFileMode(uint32(h.Mode)) != coreset.NormalizeFileMode(entry.Mode) {
			return nil, fmt.Errorf("closure archive metadata mismatch for %q", name)
		}
		out := filepath.Join(tempDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(coreset.NormalizeFileMode(entry.Mode)))
		if err != nil {
			return nil, err
		}
		hash := sha256.New()
		n, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(tarReader, h.Size+1))
		closeErr := file.Close()
		if copyErr != nil {
			return nil, copyErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if n != h.Size || hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(entry.SHA256) {
			return nil, fmt.Errorf("closure archive content mismatch for %q", name)
		}
		entry.Data = nil
		files = append(files, entry)
	}
	if len(seen) != len(manifest) {
		var missing []string
		for name := range manifest {
			if !seen[name] {
				missing = append(missing, name)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("closure archive missing members: %s", strings.Join(missing, ", "))
	}
	entries := make([]coreset.FileInventoryEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, coreset.FileInventoryEntry{Path: f.Path, SHA256: f.SHA256, Mode: f.Mode, Size: f.Size})
	}
	digest, err := coreset.FileInventoryDigest(entries)
	if err != nil {
		return nil, err
	}
	if digest != expectedDigest {
		return nil, errors.New("closure source inventory digest mismatch")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func isManifestDirectory(dir string, manifest map[string]ClosureFile) bool {
	for p := range manifest {
		if strings.HasPrefix(p, strings.TrimSuffix(dir, "/")+"/") {
			return true
		}
	}
	return false
}

func closureShipmentProto(x ClosureShipment, req ClosureShipmentRequest) *channelv1.ClosureShipmentRequest {
	c := &channelv1.DependencyClosure{SchemaVersion: int32(req.Closure.SchemaVersion), Digest: req.Closure.Digest}
	for _, m := range req.Closure.Members {
		cm := &channelv1.ClosureMember{Component: componentProto(m.Component), Selected: m.Selected, Root: componentProto(m.Root)}
		for _, r := range m.Path {
			cm.Path = append(cm.Path, reasonProto(r))
		}
		for _, r := range m.Reasons {
			cm.Reasons = append(cm.Reasons, reasonProto(r))
		}
		c.Members = append(c.Members, cm)
	}
	p := &channelv1.ClosureShipmentRequest{ShipmentId: x.ID, IdempotencyKey: x.IdempotencyKey, TargetId: x.TargetID, SourceRef: x.SourceRef, Destination: x.Destination, Closure: c, InventoryDigest: x.InventoryDigest, PriorRevision: x.PriorRevision, NewRevision: x.NewRevision}
	p.SelectionReceiptRef = x.SelectionReceiptRef
	p.SelectionDigest = x.SelectionDigest
	p.DeploymentId = x.DeploymentID
	p.MachineId = x.MachineID
	p.NodeId = x.NodeID
	p.EnrollmentGeneration = x.EnrollmentGeneration
	p.DesiredRevision = x.DesiredRevision
	for _, f := range req.Files {
		size := f.Size
		if size == 0 {
			size = int64(len(f.Data))
		}
		p.Files = append(p.Files, &channelv1.ClosureFile{Path: f.Path, Sha256: f.SHA256, Mode: coreset.NormalizeFileMode(f.Mode), SizeBytes: uint64(size)})
	}
	return p
}

func closureCommand(action channelv1.ClosureShipmentAction, req *channelv1.ClosureShipmentRequest) *channelv1.ClosureShipmentCommand {
	if req == nil {
		return &channelv1.ClosureShipmentCommand{Action: action}
	}
	return &channelv1.ClosureShipmentCommand{
		Action: action, Request: req, ShipmentId: req.GetShipmentId(),
		IdempotencyKey: req.GetIdempotencyKey(), TargetId: req.GetTargetId(),
		SourceRef: req.GetSourceRef(), Destination: req.GetDestination(),
		ClosureDigest: req.GetClosure().GetDigest(), InventoryDigest: req.GetInventoryDigest(),
		PriorRevision: req.GetPriorRevision(), NewRevision: req.GetNewRevision(),
		SelectionReceiptRef: req.GetSelectionReceiptRef(), SelectionDigest: req.GetSelectionDigest(), DeploymentId: req.GetDeploymentId(), MachineId: req.GetMachineId(), NodeId: req.GetNodeId(), EnrollmentGeneration: req.GetEnrollmentGeneration(), DesiredRevision: req.GetDesiredRevision(),
	}
}

func componentProto(id coreset.ComponentID) *channelv1.ClosureComponentID {
	return &channelv1.ClosureComponentID{Kind: id.Kind, Name: id.Name}
}
func reasonProto(r coreset.ClosureReason) *channelv1.ClosureReason {
	return &channelv1.ClosureReason{From: componentProto(r.From), To: componentProto(r.To), Relation: r.Relation, DeclaredBy: r.DeclaredBy, Detail: r.Detail, Required: r.Required, Policy: r.Policy}
}

// OpenClosureSource opens local file and HTTP(S) source references only. It is
// injected into the coordinator so tests can use a bounded in-memory archive.
func OpenClosureSource(ctx context.Context, source string, client *http.Client) (io.ReadCloser, error) {
	u, err := url.Parse(strings.TrimSpace(source))
	if err != nil {
		return nil, err
	}
	if u.User != nil {
		return nil, errors.New("closure source credentials in URL are not allowed")
	}
	switch u.Scheme {
	case "", "file":
		p := u.Path
		if u.Scheme == "" {
			p = source
		}
		if u.Scheme == "file" && u.Host != "" && u.Host != "localhost" {
			return nil, errors.New("remote file hosts are not allowed")
		}
		p, err = url.PathUnescape(p)
		if err != nil {
			return nil, err
		}
		return os.Open(p)
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, err
		}
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			return nil, fmt.Errorf("closure source returned %s", resp.Status)
		}
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unsupported closure source scheme %q", u.Scheme)
	}
}
