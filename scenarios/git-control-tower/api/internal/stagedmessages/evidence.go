package stagedmessages

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"connectrpc.com/connect"
	"git-control-tower/internal/reviewjobs"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/advisory"
	"google.golang.org/protobuf/encoding/protojson"
)

const MaxEvidencePageBytes = 49152

type EvidenceSnapshotter interface {
	EvidenceFile(context.Context, string, *v1.StagedMessageFile) (*v1.StagedMessageEvidenceFile, error)
}

type evidenceReceipt struct {
	Execution    string `json:"execution"`
	Snapshot     string `json:"snapshot"`
	IDs          []int  `json:"ids"`
	InventoryIDs []int  `json:"inventory_ids,omitempty"`
}

func (g GitSnapshotter) EvidenceFile(ctx context.Context, dir string, file *v1.StagedMessageFile) (*v1.StagedMessageEvidenceFile, error) {
	out := &v1.StagedMessageEvidenceFile{Path: file.Path, OldPath: file.OldPath, OldBlobId: file.OldBlobId, NewBlobId: file.NewBlobId}
	if file.NewMode == "160000" || file.OldMode == "160000" {
		out.OmissionReason = "submodule content is unavailable as a file patch"
		return out, nil
	}
	var before, after []byte
	for _, item := range []struct {
		id  string
		dst *[]byte
	}{{file.OldBlobId, &before}, {file.NewBlobId, &after}} {
		if item.id == "" {
			continue
		}
		if len(item.id) != 40 && len(item.id) != 64 {
			return nil, errors.New("immutable blob identity is invalid")
		}
		for _, c := range item.id {
			if !strings.ContainsRune("0123456789abcdef", c) {
				return nil, errors.New("immutable blob identity is invalid")
			}
		}
		size, err := gitRead(ctx, dir, "cat-file", "-s", item.id)
		if err != nil {
			out.OmissionReason = "recorded immutable blob is no longer available"
			return out, nil
		}
		n, err := strconv.Atoi(strings.TrimSpace(string(size)))
		if err != nil || n > maxFileBytes {
			out.OmissionReason = "immutable file exceeds 128 KiB read bound"
			return out, nil
		}
		*item.dst, err = gitRead(ctx, dir, "cat-file", "blob", item.id)
		if err != nil {
			return nil, err
		}
	}
	if bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 || !utf8.Valid(before) || !utf8.Valid(after) {
		out.OmissionReason = "binary content cannot be supplied as a text patch"
		return out, nil
	}
	patch, err := renderPatch(ctx, file.Path, before, after)
	if err != nil {
		return nil, err
	}
	if file.OldMode != file.NewMode {
		patch = fmt.Sprintf("old mode %s\nnew mode %s\n", file.OldMode, file.NewMode) + patch
	}
	if patch == "" {
		patch = "No textual content change; immutable metadata retained."
	}
	out.Patch = patch
	out.CompletePatch = true
	return out, nil
}

func priorCovered(input DraftInput) map[string]bool {
	paths := map[string]bool{}
	for _, path := range input.PriorResult.GetCoverage().GetCoveredPaths() {
		paths[path] = true
	}
	return paths
}

// ReadEvidence always reads named immutable objects. It neither recollects the
// index nor reads current working files, and page reads never dispatch work.
func (s *Service) ReadEvidence(ctx context.Context, repo, dir string, req *v1.ReadStagedMessageEvidenceRequest, execution string) (*v1.ReadStagedMessageEvidenceResponse, error) {
	job, err := s.store.Get(ctx, req.JobId)
	if err != nil {
		return nil, err
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return nil, ErrNotFound
	}
	var input DraftInput
	if err = json.Unmarshal([]byte(job.InputJSON), &input); err != nil {
		return nil, err
	}
	if (input.FormatVersion != 3 && input.FormatVersion != 4) || input.Snapshot == nil || input.Snapshot.Digest != req.SnapshotDigest {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("deep evidence requires its exact immutable draft subject"))
	}
	if execution != "" {
		if terminal(job.State) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("draft execution is terminal"))
		}
		if job.ExecutionID != "" && job.ExecutionID != execution {
			return nil, connect.NewError(connect.CodePermissionDenied, errors.New("evidence belongs to another workflow execution"))
		}
		if job.ExecutionID == "" {
			authorizer, ok := s.provider.(EvidenceAuthorizer)
			if !ok {
				return nil, connect.NewError(connect.CodeUnavailable, errors.New("execution evidence binding is unavailable"))
			}
			if err = authorizer.AuthorizeEvidence(ctx, input, job.IdempotencyKey, execution); err != nil {
				return nil, err
			}
		}
	}
	budget := int(req.MaxBytes)
	if budget == 0 {
		budget = 24576
	}
	if budget < 1024 || budget > MaxEvidencePageBytes {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("evidence page budget must be 1024..49152 bytes"))
	}
	if req.Cursor > uint32(len(input.Snapshot.Files)) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("evidence cursor exceeds immutable manifest"))
	}
	targeted := len(req.FileIds) > 0
	if (req.InventoryOnly || targeted) && input.FormatVersion != 4 {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("inventory and targeted reads require synthesis format 4"))
	}
	if len(req.FileIds) > 64 || (targeted && (req.InventoryOnly || req.Cursor != 0)) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target at most 64 unique IDs without inventory or cursor"))
	}
	indices := make([]int, 0, len(req.FileIds))
	seen := map[uint32]bool{}
	for _, id := range req.FileIds {
		if id == 0 || id > uint32(len(input.Snapshot.Files)) || seen[id] {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("target file ID is absent or duplicated"))
		}
		seen[id] = true
		indices = append(indices, int(id)-1)
	}
	reader, readable := s.snapshots.(EvidenceSnapshotter)
	if !req.InventoryOnly && !readable {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("immutable blob evidence reader unavailable"))
	}
	out := &v1.ReadStagedMessageEvidenceResponse{SnapshotDigest: input.Snapshot.Digest, TotalFiles: uint32(len(input.Snapshot.Files)), Receipt: strings.Repeat("0", 71)}
	covered := priorCovered(input)
	cursor := int(req.Cursor)
	position := 0
	entryLimit := 64
	if req.InventoryOnly {
		entryLimit = 256
	}
	for len(out.Files) < entryLimit {
		index := cursor
		if targeted {
			if position == len(indices) {
				break
			}
			index = indices[position]
		} else if cursor == len(input.Snapshot.Files) {
			break
		}
		file := input.Snapshot.Files[index]
		if !targeted && !req.InventoryOnly && covered[file.Path] {
			cursor++
			continue
		}
		var detail *v1.StagedMessageEvidenceFile
		if req.InventoryOnly {
			detail = &v1.StagedMessageEvidenceFile{Path: file.Path, Status: file.Status, EditCount: file.EditCount}
			if file.OldPath != file.Path {
				detail.OldPath = file.OldPath
			}
			if file.OldMode != file.NewMode {
				detail.OldMode = file.OldMode
				detail.NewMode = file.NewMode
			}
		} else {
			detail, err = reader.EvidenceFile(ctx, dir, file)
			if err != nil {
				return nil, err
			}
		}
		detail.FileId = uint32(index + 1)
		out.Files = append(out.Files, detail)
		if !targeted {
			out.NextCursor = uint32(index + 1)
		}
		raw, err := protojson.Marshal(out)
		if err != nil {
			return nil, err
		}
		if len(raw)+32 > budget {
			if len(out.Files) > 1 {
				out.Files = out.Files[:len(out.Files)-1]
				break
			}
			if !req.InventoryOnly {
				detail.Patch = ""
				detail.CompletePatch = false
				detail.OmissionReason = "whole patch exceeds requested evidence page bound; larger pages allow up to 49152 bytes"
			}
			raw, err = protojson.Marshal(out)
			if err != nil || len(raw)+32 > budget {
				return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("file metadata exceeds requested evidence page bound"))
			}
		}
		if targeted {
			position++
		} else {
			cursor++
		}
	}
	if targeted {
		out.Complete = position == len(indices)
	} else {
		out.NextCursor = uint32(cursor)
		out.Complete = cursor == len(input.Snapshot.Files)
	}
	out.Receipt = ""
	raw, err := protojson.Marshal(out)
	if err != nil {
		return nil, err
	}
	out.Receipt = digest(append([]byte(job.ID+"\x00"), raw...))
	if execution != "" {
		receipt := evidenceReceipt{Execution: execution, Snapshot: input.Snapshot.Digest}
		for _, file := range out.Files {
			if req.InventoryOnly {
				receipt.InventoryIDs = append(receipt.InventoryIDs, int(file.FileId))
			}
			if file.CompletePatch {
				receipt.IDs = append(receipt.IDs, int(file.FileId))
			}
		}
		detail, _ := json.Marshal(receipt)
		if err = s.store.RecordAdvisoryEvidence(ctx, job.ID, repo, execution, reviewjobs.Check{Name: out.Receipt, Availability: "supplied", Verdict: "immutable_staged_evidence", Detail: string(detail)}); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *Service) EvidenceEligibility(ctx context.Context, jobID, repo, snapshot, execution string) (map[int]bool, error) {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return nil, ErrNotFound
	}
	return suppliedEvidence(job.Checks, snapshot, execution), nil
}

func suppliedEvidence(checks []reviewjobs.Check, snapshot, execution string) map[int]bool {
	ids := map[int]bool{}
	for _, check := range checks {
		if check.Verdict != "immutable_staged_evidence" {
			continue
		}
		var receipt evidenceReceipt
		if json.Unmarshal([]byte(check.Detail), &receipt) != nil || receipt.Snapshot != snapshot || receipt.Execution != execution {
			continue
		}
		for _, id := range receipt.IDs {
			ids[id] = true
		}
	}
	return ids
}

// InventoryEligibility records metadata delivery independently from patch inspection.
func (s *Service) InventoryEligibility(ctx context.Context, jobID, repo, snapshot, execution string) (map[int]bool, error) {
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job.Operation != operation || job.RepositoryID != repo {
		return nil, ErrNotFound
	}
	return suppliedInventory(job.Checks, snapshot, execution), nil
}
func suppliedInventory(checks []reviewjobs.Check, snapshot, execution string) map[int]bool {
	ids := map[int]bool{}
	for _, check := range checks {
		if check.Verdict != "immutable_staged_evidence" {
			continue
		}
		var receipt evidenceReceipt
		if json.Unmarshal([]byte(check.Detail), &receipt) != nil || receipt.Snapshot != snapshot || receipt.Execution != execution {
			continue
		}
		for _, id := range receipt.InventoryIDs {
			ids[id] = true
		}
	}
	return ids
}
