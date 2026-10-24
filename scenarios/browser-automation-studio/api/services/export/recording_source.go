package export

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	executionwriter "github.com/vrooli/browser-automation-studio/automation/execution-writer"
	"github.com/vrooli/browser-automation-studio/services/export/source"
	"github.com/vrooli/browser-automation-studio/storage"
)

// ResolveExecutionRecording uses typed execution artifacts first. Old recordings
// without typed video artifacts have a constrained local-file fallback.
func ResolveExecutionRecording(ctx context.Context, repo ExecutionRepository, root string, id uuid.UUID, store storage.StorageInterface) (*source.VideoSource, error) {
	timeline, err := NewTimelineLoader(repo).LoadTimeline(ctx, id)
	if err != nil {
		return nil, err
	}
	var artifacts []executionwriter.ArtifactData
	for _, frame := range timeline.Frames {
		for _, artifact := range frame.Artifacts {
			kind := strings.ToLower(strings.TrimPrefix(artifact.Type, "ARTIFACT_TYPE_"))
			if kind != "video" && kind != "video_meta" && !strings.HasPrefix(strings.ToLower(artifact.ContentType), "video/") && !strings.Contains(artifact.StorageURL, "/artifacts/videos/") {
				continue
			}
			payload := artifact.Payload
			if payload == nil {
				payload = map[string]any{}
			}
			// A public recording asset URL is resolved only beneath this execution's
			// existing local owner, never as an arbitrary remote download.
			parsed, parseErr := url.Parse(artifact.StorageURL)
			prefix := "/api/v1/recordings/assets/" + id.String() + "/"
			if parseErr == nil && strings.HasPrefix(parsed.Path, prefix) {
				relative := strings.TrimPrefix(parsed.Path, prefix)
				if !filepath.IsLocal(relative) {
					return nil, fmt.Errorf("recording artifact escapes its execution")
				}
				copied := make(map[string]any, len(payload)+1)
				for key, value := range payload {
					copied[key] = value
				}
				copied["path"] = filepath.Join(root, id.String(), filepath.FromSlash(relative))
				payload = copied
			}
			declaredHash, _ := payload["sha256"].(string)
			artifacts = append(artifacts, executionwriter.ArtifactData{ArtifactID: artifact.ID, ArtifactType: "video", SHA256: declaredHash, StorageURL: artifact.StorageURL, ContentType: artifact.ContentType, SizeBytes: artifact.SizeBytes, Payload: payload})
		}
	}

	if len(artifacts) > 0 {
		return uniqueTypedRecording(ctx, artifacts, store)
	}
	if strings.TrimSpace(root) == "" {
		return nil, source.ErrVideoNotFound
	}
	directory := filepath.Join(root, id.String(), "artifacts", "videos")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, source.ErrVideoNotFound
	}
	if err != nil {
		return nil, err
	}
	var path string
	for _, entry := range entries {
		switch strings.ToLower(filepath.Ext(entry.Name())) {
		case ".webm", ".mp4", ".mov", ".mkv":
		default:
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if path != "" {
			return nil, fmt.Errorf("multiple legacy recordings require an explicit artifact association; export the desired recording separately")
		}
		path = filepath.Join(directory, entry.Name())
	}
	if path == "" {
		return nil, source.ErrVideoNotFound
	}
	return source.ResolveVideoSourceContext(ctx, []executionwriter.ArtifactData{{ArtifactID: filepath.Base(path), ArtifactType: "video", ContentType: source.DetectVideoContentType(path), Payload: map[string]any{"path": path}}}, nil)
}

// uniqueTypedRecording is stricter than the generic first-valid artifact resolver.
// Repeated references and byte-identical aliases represent one recording; distinct
// verified contents require an explicit composition/association rather than order.
func uniqueTypedRecording(ctx context.Context, artifacts []executionwriter.ArtifactData, store storage.StorageInterface) (*source.VideoSource, error) {
	sort.SliceStable(artifacts, func(i, j int) bool { return artifacts[i].ArtifactID < artifacts[j].ArtifactID })
	seen := map[string]bool{}
	var selected *source.VideoSource
	cleanup := func() {
		if selected != nil && selected.Cleanup != nil {
			selected.Cleanup()
		}
	}
	for _, artifact := range artifacts {
		path, _ := artifact.Payload["path"].(string)
		size := int64(-1)
		if artifact.SizeBytes != nil {
			size = *artifact.SizeBytes
		}
		key := fmt.Sprintf("%s\x00%s\x00%s\x00%d", artifact.StorageURL, path, strings.ToLower(artifact.SHA256), size)
		// Inline payloads have no stable external locator and must be measured.
		// Their missing URL/path must never collapse distinct embedded recordings.
		inline, _ := artifact.Payload["inline"].(bool)
		if !inline && (path != "" || artifact.StorageURL != "") {
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		resolved, err := source.ResolveVideoSourceContext(ctx, []executionwriter.ArtifactData{artifact}, store)
		if err != nil {
			cleanup()
			return nil, fmt.Errorf("resolve typed recording artifact %q: %w", artifact.ArtifactID, err)
		}
		if selected == nil {
			selected = resolved
			continue
		}
		same := selected.SHA256 == resolved.SHA256 && selected.SizeBytes != nil && resolved.SizeBytes != nil && *selected.SizeBytes == *resolved.SizeBytes
		if resolved.Cleanup != nil {
			resolved.Cleanup()
		}
		if !same {
			cleanup()
			return nil, fmt.Errorf("multiple distinct typed recordings require an explicit artifact association or composition; refusing arbitrary selection")
		}
	}
	if selected == nil {
		return nil, source.ErrVideoNotFound
	}
	return selected, nil
}
