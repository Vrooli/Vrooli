package export

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/services/evidence"
	"github.com/vrooli/browser-automation-studio/storage"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

// ExecutionArtifact is the public metadata projection for a durable execution
// artifact. The local path is intentionally kept out of this type.
type ExecutionArtifact struct {
	ArtifactID  string
	StorageURL  string
	ContentType string
	Label       string
	SizeBytes   int64
	Payload     map[string]any
}

// ArtifactCatalog is the canonical read owner for execution artifact files.
// The writer owns admission and durable bytes; this catalog owns the safe
// metadata and URL projection consumed by BAS execution APIs.
type ArtifactCatalog struct {
	root string
}

// ExecutionExportability is the safe existence summary exposed by the
// execution-list API. Filesystem probing stays with the export artifact owner;
// handlers only project this value into the transport contract.
type ExecutionExportability struct {
	HasTimeline      bool
	HasScreenshots   bool
	HasRecordedVideo bool
	IsExportable     bool
}

func (c *ArtifactCatalog) InspectExportability(ctx context.Context, resultPath string, executionID uuid.UUID) (ExecutionExportability, error) {
	var out ExecutionExportability
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if strings.TrimSpace(resultPath) == "" {
		return out, nil
	}
	resultDir := filepath.Dir(resultPath)
	if info, err := os.Stat(filepath.Join(resultDir, "timeline.proto.json")); err == nil && !info.IsDir() && info.Size() > 0 {
		out.HasTimeline = true
	}
	if entries, err := os.ReadDir(filepath.Join(resultDir, "screenshots")); err == nil {
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return ExecutionExportability{}, err
			}
			if !entry.IsDir() {
				out.HasScreenshots = true
				break
			}
		}
	}
	if c != nil && strings.TrimSpace(c.root) != "" && executionID != uuid.Nil {
		if entries, err := os.ReadDir(filepath.Join(c.root, executionID.String(), "artifacts", "videos")); err == nil {
			for _, entry := range entries {
				if err := ctx.Err(); err != nil {
					return ExecutionExportability{}, err
				}
				if !entry.IsDir() {
					out.HasRecordedVideo = true
					break
				}
			}
		}
	}
	out.IsExportable = out.HasTimeline || out.HasRecordedVideo
	return out, nil
}

func (e ExecutionExportability) Proto() *basapi.ExecutionExportability {
	return &basapi.ExecutionExportability{HasTimeline: e.HasTimeline, HasScreenshots: e.HasScreenshots, HasRecordedVideo: e.HasRecordedVideo, IsExportable: e.IsExportable}
}

func NewArtifactCatalog(root string) *ArtifactCatalog {
	return &ArtifactCatalog{root: strings.TrimSpace(root)}
}

func (c *ArtifactCatalog) List(ctx context.Context, executionID uuid.UUID, kind string) ([]ExecutionArtifact, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if c == nil || c.root == "" {
		return []ExecutionArtifact{}, nil
	}
	if kind != "videos" && kind != "traces" && kind != "har" {
		return nil, fmt.Errorf("unsupported execution artifact kind %q", kind)
	}
	base := filepath.Join(c.root, executionID.String(), "artifacts", kind)
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []ExecutionArtifact{}, nil
		}
		return nil, fmt.Errorf("read artifacts directory: %w", err)
	}

	items := make([]ExecutionArtifact, 0, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		path := filepath.Join(base, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}
		contentType := storage.DetectContentTypeFromFile(path, "")
		descriptor, err := evidence.DescribeFile(kind, contentType, path, evidence.DefaultPolicy())
		if err != nil {
			return nil, fmt.Errorf("describe %s artifact %q: %w", kind, name, err)
		}
		url := ""
		if descriptor.Access != basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY {
			url = fmt.Sprintf("/api/v1/recordings/assets/%s/artifacts/%s/%s", executionID, kind, name)
		}
		size := info.Size()
		items = append(items, ExecutionArtifact{
			ArtifactID: name, StorageURL: url, ContentType: contentType,
			Label: strings.TrimSuffix(name, filepath.Ext(name)), SizeBytes: size,
			Payload: map[string]any{
				"size_bytes": size, "sha256": descriptor.SHA256,
				"classification":  descriptor.Classification.String(),
				"retention_class": descriptor.Retention.String(),
				"access_policy":   descriptor.Access.String(),
				"redacted":        descriptor.Redacted,
			},
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ArtifactID < items[j].ArtifactID })
	return items, nil
}
