package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	"git-control-tower/internal/proposals"
	proposalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/proposals"
	repov1 "github.com/vrooli/vrooli/packages/proto/gen/go/git-control-tower/v1/repo"
)

func (h proposalConnectServer) CombineProposals(ctx context.Context, req *connect.Request[proposalsv1.CombineProposalsRequest]) (*connect.Response[proposalsv1.GetProposalResponse], error) {
	if len(req.Msg.GetSources()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("select source proposals"))
	}
	_, repo, err := h.server.proposalRepoByID(ctx, req.Msg.GetSources()[0].GetId())
	if err != nil {
		return nil, err
	}
	sources := make([]proposals.RevisionReference, 0, len(req.Msg.GetSources()))
	for _, source := range req.Msg.GetSources() {
		sources = append(sources, proposals.RevisionReference{ID: source.GetId(), Revision: int(source.GetRevision())})
	}
	view, err := h.server.proposalService.Combine(ctx, repo, proposals.CombineRequest{Sources: sources, Subject: req.Msg.GetSubject(), Body: req.Msg.GetBody(), Actor: proposalActor(ctx, req.Header(), "")})
	if err != nil {
		return nil, proposalError(err)
	}
	return connect.NewResponse(&proposalsv1.GetProposalResponse{Proposal: h.server.proposalProto(ctx, repo, view)}), nil
}

// capturedProposalDiff delegates hunk parsing to the established diff owner.
// Temporary inputs contain only retained bytes, so drift and index state are
// irrelevant. External diff/text-conversion drivers are never executed.
func capturedProposalDiff(ctx context.Context, file proposals.File, before, after []byte) (*repov1.GetRepoDiffResponse, error) {
	result := &repov1.GetRepoDiffResponse{Path: file.Path, Base: file.BaseBlobID, ContentHash: file.BlobID, Mode: "diff", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), HasDiff: !bytes.Equal(before, after) || file.BaseMode != file.Mode}
	if bytes.IndexByte(before, 0) >= 0 || bytes.IndexByte(after, 0) >= 0 || !utf8.Valid(before) || !utf8.Valid(after) {
		result.Stats = &repov1.DiffStats{Files: 1, IsBinary: true, IsNewFile: file.BaseBlobID == "", IsDeletedFile: file.Deleted}
		result.Raw = "Binary file content or metadata changed. Captured blob identities: " + file.BaseBlobID + " → " + file.ContentID()
		return result, nil
	}
	dir, err := os.MkdirTemp("", "gct-review-diff-")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	defer os.RemoveAll(dir)
	oldPath, newPath := filepath.Join(dir, "before"), filepath.Join(dir, "after")
	if err = os.WriteFile(oldPath, before, 0600); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if err = os.WriteFile(newPath, after, 0600); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	cmd := exec.CommandContext(ctx, "git", "diff", "--no-index", "--no-ext-diff", "--no-textconv", "--no-color", "--", oldPath, newPath)
	cmd.Dir = dir
	var output bytes.Buffer
	cmd.Stdout = &boundedCaptureWriter{target: &output, remaining: 2 << 20}
	err = cmd.Run()
	var exit *exec.ExitError
	if err != nil && (!errors.As(err, &exit) || exit.ExitCode() != 1) {
		return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("captured diff exceeded its 2 MiB output limit or failed: %w", err))
	}
	raw := output.String()
	if raw == "" && file.BaseBlobID == "" && !file.Deleted {
		raw = "New empty file: " + file.Path + "\n"
	}
	if raw == "" && file.Deleted && file.BaseBlobID != "" {
		raw = "Deleted empty file: " + file.Path + "\n"
	}
	raw = strings.ReplaceAll(raw, "a"+oldPath, "a/"+file.Path)
	raw = strings.ReplaceAll(raw, "b"+newPath, "b/"+file.Path)
	if file.BaseMode != file.Mode {
		raw = fmt.Sprintf("File mode: %s → %s\n", file.BaseMode, file.Mode) + raw
	}
	parsed := ParseDiffOutput(raw)
	parsed.Stats.IsNewFile, parsed.Stats.IsDeletedFile = file.BaseBlobID == "", file.Deleted
	parsed.Stats.Files = 1
	enrichCommentStats(parsed, file.Path)
	result.Raw, result.Hunks, result.Stats = raw, diffHunksProto(parsed.Hunks), diffStatsProto(parsed.Stats)
	return result, nil
}
