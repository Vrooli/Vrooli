package executionwriter

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/services/evidence"
	"github.com/vrooli/browser-automation-studio/storage"
	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

const maxEmbeddedExternalArtifactBytes = 5 * 1024 * 1024

// RecordExecutionArtifacts persists available evidence and returns every failed
// import. A requested path or digest alone cannot acknowledge retained bytes.
func (r *FileWriter) RecordExecutionArtifacts(ctx context.Context, plan contracts.ExecutionPlan, artifacts []ExternalArtifact) error {
	if r == nil || len(artifacts) == 0 {
		return nil
	}
	result, timeline := r.getOrCreateResult(plan), r.getOrCreateTimeline(plan)
	var failures error
	for _, item := range artifacts {
		artifact, err := r.prepareExternalArtifact(ctx, plan.ExecutionID, item)
		if err != nil {
			failures = errors.Join(failures, fmt.Errorf("import %s artifact: %w", item.ArtifactType, err))
			continue
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, artifact)
		result.mu.Unlock()
	}
	return errors.Join(failures, r.writeResultFile(ctx, plan.ExecutionID, result, timeline))
}

// prepareExternalArtifact applies disclosure policy before publishing any bytes.
// Non-inline captures must have a successful storage receipt, including traces.
func (r *FileWriter) prepareExternalArtifact(ctx context.Context, executionID uuid.UUID, item ExternalArtifact) (ArtifactData, error) {
	path := strings.TrimSpace(item.Path)
	info, err := os.Stat(path)
	if err != nil {
		return ArtifactData{}, err
	}
	if !info.Mode().IsRegular() {
		return ArtifactData{}, fmt.Errorf("artifact is not a regular file: %s", path)
	}
	kind := strings.TrimSpace(item.ArtifactType)
	if kind == "" {
		kind = "custom"
	}
	label := strings.TrimSpace(item.Label)
	if label == "" {
		label = kind
	}
	payload := make(map[string]any)
	for key, value := range item.Payload {
		switch strings.ToLower(key) {
		case "path", "source_path", "storage_path":
		default:
			payload[key] = value
		}
	}
	isHAR := evidence.KindFor(kind) == basevidence.ArtifactKind_ARTIFACT_KIND_HAR
	inline := !isNonInlineArtifactType(kind) && info.Size() <= maxEmbeddedExternalArtifactBytes
	inlineKey, contentType := "base64", strings.TrimSpace(item.ContentType)
	var data []byte
	var descriptor evidence.Descriptor
	if isHAR || inline {
		data, err = os.ReadFile(path)
		if err != nil {
			return ArtifactData{}, err
		}
		if isHAR {
			data, err = evidence.SanitizeHAR(data, evidence.DefaultPolicy())
			if err != nil {
				return ArtifactData{}, err
			}
			inline = len(data) <= maxEmbeddedExternalArtifactBytes
			inlineKey, contentType = "sanitized_base64", "application/json"
		}
		descriptor = evidence.Describe(kind, contentType, data, evidence.DefaultPolicy())
	} else {
		descriptor, err = evidence.DescribeFile(kind, contentType, path, evidence.DefaultPolicy())
		if err != nil {
			return ArtifactData{}, err
		}
	}
	payload["size_bytes"] = descriptor.SizeBytes
	if inline {
		payload[inlineKey], payload["inline"] = base64.StdEncoding.EncodeToString(data), true
	}
	storageURL := ""
	if !inline || (isHAR && r.storage != nil) {
		if r.storage == nil {
			return ArtifactData{}, errors.New("artifact storage unavailable for non-inline evidence")
		}
		var stored *storage.ArtifactInfo
		if isHAR {
			objectName := executionID.String() + "/artifacts/har/" + uuid.NewString() + ".sanitized.har"
			stored, err = r.storage.StoreArtifact(ctx, objectName, data, contentType)
		} else {
			stored, err = r.storage.StoreArtifactFromFile(ctx, executionID, label, path, contentType)
		}
		if err != nil {
			return ArtifactData{}, err
		}
		if stored == nil || strings.TrimSpace(stored.ObjectName) == "" || strings.TrimSpace(stored.URL) == "" {
			return ArtifactData{}, errors.New("artifact storage returned an incomplete receipt")
		}
		if stored.SizeBytes != descriptor.SizeBytes {
			return ArtifactData{}, errors.New("artifact storage size differs from described bytes")
		}
		storageURL, payload["storage_object"] = stored.URL, stored.ObjectName
		if stored.ContentType != "" {
			contentType = stored.ContentType
		}
		if isHAR {
			payload["sanitized_available"] = true
		}
	}
	return ArtifactData{
		ArtifactID: uuid.NewString(), ArtifactType: kind, Label: label,
		Payload: payload, StorageURL: storageURL, ContentType: contentType,
		SizeBytes: &descriptor.SizeBytes, SHA256: descriptor.SHA256,
		Classification: descriptor.Classification.String(), RetentionClass: descriptor.Retention.String(),
		AccessPolicy: descriptor.Access.String(), Redacted: descriptor.Redacted,
	}, nil
}

func isNonInlineArtifactType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "video_meta", "trace", "trace_meta", "har", "har_meta":
		return true
	}
	return false
}
