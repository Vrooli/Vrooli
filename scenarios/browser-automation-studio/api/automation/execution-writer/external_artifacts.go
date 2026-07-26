package executionwriter

import (
	"context"
	"encoding/base64"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/services/evidence"
)

const maxEmbeddedExternalArtifactBytes = 5 * 1024 * 1024

// RecordExecutionArtifacts owns execution-scoped files such as videos, traces,
// and HAR captures. Step-level outcome persistence remains separate.
func (r *FileWriter) RecordExecutionArtifacts(ctx context.Context, plan contracts.ExecutionPlan, artifacts []ExternalArtifact) error {
	if r == nil || len(artifacts) == 0 {
		return nil
	}
	result, timeline := r.getOrCreateResult(plan), r.getOrCreateTimeline(plan)
	for _, item := range artifacts {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("path", path).Debug("external artifact not readable")
			}
			continue
		}
		kind := strings.TrimSpace(item.ArtifactType)
		if kind == "" {
			kind = "custom"
		}
		// A result file is consumed by API/UI/CLI clients. It must never expose a
		// local capture path, particularly for protected HAR material.
		descriptor, err := evidence.DescribeFile(kind, item.ContentType, path, evidence.DefaultPolicy())
		if err != nil {
			if r.log != nil {
				r.log.WithError(err).WithField("path", path).Warn("Failed to describe external artifact")
			}
			continue
		}
		payload := map[string]any{"size_bytes": info.Size()}
		for key, value := range item.Payload {
			if strings.EqualFold(key, "path") || strings.EqualFold(key, "storage_path") {
				continue
			}
			payload[key] = value
		}
		if descriptor.Kind.String() == "ARTIFACT_KIND_HAR" {
			// Raw HAR stays at its protected capture location. Only a sanitized
			// derivative can enter the replay/result boundary.
			if info.Size() > 0 && info.Size() <= maxEmbeddedExternalArtifactBytes {
				if raw, readErr := os.ReadFile(path); readErr == nil {
					if sanitized, sanitizeErr := evidence.SanitizeHAR(raw, evidence.DefaultPolicy()); sanitizeErr == nil {
						payload["sanitized_base64"], payload["inline"] = base64.StdEncoding.EncodeToString(sanitized), true
					}
				}
			}
		} else if !isNonInlineArtifactType(kind) && info.Size() > 0 && info.Size() <= maxEmbeddedExternalArtifactBytes {
			if data, err := os.ReadFile(path); err == nil {
				payload["base64"], payload["inline"] = base64.StdEncoding.EncodeToString(data), true
			}
		}
		label := strings.TrimSpace(item.Label)
		if label == "" {
			label = kind
		}
		contentType, storageURL := strings.TrimSpace(item.ContentType), ""
		var sizeBytes *int64
		if r.storage != nil && isVideoArtifactType(kind) {
			stored, err := r.storage.StoreArtifactFromFile(ctx, plan.ExecutionID, label, path, item.ContentType)
			if err != nil {
				if r.log != nil {
					r.log.WithError(err).WithField("path", path).Warn("Failed to store video artifact")
				}
			} else if stored != nil {
				storageURL, payload["storage_object"] = stored.URL, stored.ObjectName
				if strings.TrimSpace(stored.ContentType) != "" {
					contentType = stored.ContentType
				}
				size := stored.SizeBytes
				sizeBytes = &size
			}
		}
		if sizeBytes == nil {
			size := descriptor.SizeBytes
			sizeBytes = &size
		}
		result.mu.Lock()
		result.Artifacts = append(result.Artifacts, ArtifactData{ArtifactID: uuid.New().String(), ArtifactType: kind, Label: label, Payload: payload, StorageURL: storageURL, ContentType: contentType, SizeBytes: sizeBytes, SHA256: descriptor.SHA256, Classification: descriptor.Classification.String(), RetentionClass: descriptor.Retention.String(), AccessPolicy: descriptor.Access.String(), Redacted: descriptor.Redacted})
		result.mu.Unlock()
	}
	return r.writeResultFile(plan.ExecutionID, result, timeline)
}

func isVideoArtifactType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "video_meta":
		return true
	}
	return false
}
func isNonInlineArtifactType(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "video_meta", "trace", "trace_meta", "har", "har_meta":
		return true
	}
	return false
}
