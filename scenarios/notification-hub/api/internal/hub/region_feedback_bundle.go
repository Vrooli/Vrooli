package hub

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	WarnRegionFeedbackInvalid    = "region_feedback_invalid"
	WarnRegionFeedbackIncomplete = "region_feedback_incomplete"
	regionFeedbackSchema         = "ux-region-feedback/v1"
	regionFeedbackManifestName   = "ux-region-feedback.json"
	regionManifestReadLimit      = 1 << 20
	regionScreenshotLimit        = 20 << 20
	regionCropLimit              = 5 << 20
)

var regionIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)

type regionFeedbackManifest struct {
	Schema  string                 `json:"schema"`
	ID      string                 `json:"id"`
	Route   string                 `json:"route"`
	StateID string                 `json:"state_id"`
	DataID  string                 `json:"data_id"`
	Before  regionFeedbackCapture  `json:"before"`
	After   regionFeedbackCapture  `json:"after"`
	Regions []regionFeedbackRegion `json:"regions"`
}

type regionFeedbackCapture struct {
	ID       string `json:"id"`
	BuildID  string `json:"build_id"`
	Route    string `json:"route"`
	StateID  string `json:"state_id"`
	DataID   string `json:"data_id"`
	Fidelity string `json:"fidelity"`
	Viewport struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	} `json:"viewport"`
	Image struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"image"`
}

type regionFeedbackRegion struct {
	ID         string `json:"id"`
	NodeID     string `json:"node_id"`
	Label      string `json:"label"`
	Component  string `json:"component"`
	EvidenceID string `json:"evidence_id"`
	Bounds     struct {
		X      float64 `json:"x"`
		Y      float64 `json:"y"`
		Width  float64 `json:"width"`
		Height float64 `json:"height"`
	} `json:"bounds"`
	Crop struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	} `json:"crop"`
}

// checkRegionFeedbackBundles checks every EX12 region manifest at the packet
// boundary. It reads only the Hub's stored attachment bytes, never source paths.
func checkRegionFeedbackBundles(ctx context.Context, spec AskSpec, read BlobReader) []PacketWarning {
	var warnings []PacketWarning
	for index, attachment := range spec.Attachments {
		if attachment.Kind != AttachmentFolder {
			continue
		}
		manifestEntry := findRegionEntry(attachment, regionFeedbackManifestName)
		if manifestEntry == nil {
			continue
		}
		field := fmt.Sprintf("attachments[%d]", index)
		if !strings.EqualFold(manifestEntry.MediaType, "application/json") {
			warnings = append(warnings, advice(WarnRegionFeedbackInvalid, field, "the capture compare manifest must be application/json inside the evidence folder"))
			continue
		}
		data, err := read(ctx, manifestEntry.SHA256, regionManifestReadLimit)
		if err != nil {
			warnings = append(warnings, advice(WarnRegionFeedbackIncomplete, field, "the capture compare manifest cannot be read from the stored packet; attach its complete evidence folder"))
			continue
		}
		var manifest regionFeedbackManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			warnings = append(warnings, advice(WarnRegionFeedbackInvalid, field, "the capture compare manifest is not valid JSON"))
			continue
		}
		problems := validateRegionFeedbackManifest(ctx, manifest, attachment, read)
		if len(problems) > 0 {
			warnings = append(warnings, advice(WarnRegionFeedbackIncomplete, field, "the capture compare packet is incomplete: "+strings.Join(problems, "; ")+"."))
		}
	}
	return warnings
}

func findRegionEntry(folder AskAttachment, path string) *AskAttachmentEntry {
	for index := range folder.Entries {
		if folder.Entries[index].Path == path {
			return &folder.Entries[index]
		}
	}
	return nil
}

func validateRegionFeedbackManifest(ctx context.Context, manifest regionFeedbackManifest, folder AskAttachment, read BlobReader) []string {
	var problems []string
	add := func(ok bool, message string) {
		if !ok {
			problems = append(problems, message)
		}
	}
	add(manifest.Schema == regionFeedbackSchema, "schema must be ux-region-feedback/v1")
	add(strings.TrimSpace(manifest.ID) != "", "capture-set id is required")
	add(strings.TrimSpace(manifest.Route) != "" && strings.TrimSpace(manifest.StateID) != "" && strings.TrimSpace(manifest.DataID) != "", "route, state and data identities are required")
	validCapture := func(label string, capture regionFeedbackCapture) bool {
		valid := true
		fail := func(ok bool, message string) {
			if !ok {
				problems = append(problems, label+" "+message)
				valid = false
			}
		}
		fail(strings.TrimSpace(capture.ID) != "", "capture id is required")
		fail(featureBuildID.MatchString(capture.BuildID), "build identity must be a full sha256 id")
		fail(strings.TrimSpace(capture.Route) != "" && strings.TrimSpace(capture.StateID) != "" && strings.TrimSpace(capture.DataID) != "", "route, state and data identities are required")
		fail(capture.Viewport.Width > 0 && capture.Viewport.Height > 0, "viewport size must be positive")
		fail(capture.Fidelity == "captured" || capture.Fidelity == "composited" || capture.Fidelity == "rendered" || capture.Fidelity == "sketch", "fidelity must be captured, composited, rendered or sketch")
		fail(strings.TrimSpace(capture.Image.Path) != "" && validDigest(capture.Image.SHA256), "image needs a folder path and sha256 digest")
		if capture.Image.Path != "" && validDigest(capture.Image.SHA256) {
			if err := validateFeedbackImage(ctx, folder, capture.Image.Path, capture.Image.SHA256, regionScreenshotLimit, read); err != nil {
				problems = append(problems, label+" image is missing or invalid: "+err.Error())
				valid = false
			}
		}
		return valid
	}
	validCapture("before", manifest.Before)
	validCapture("after", manifest.After)
	add(manifest.Before.ID != manifest.After.ID, "before and after must be distinct captures")
	add(manifest.Before.BuildID != manifest.After.BuildID, "before and after must name distinct builds")
	add(manifest.Before.Route == manifest.After.Route && manifest.Before.Route == manifest.Route, "before and after must use the manifest route")
	add(manifest.Before.StateID == manifest.After.StateID && manifest.Before.StateID == manifest.StateID, "before and after must use the manifest interaction state")
	add(manifest.Before.DataID == manifest.After.DataID && manifest.Before.DataID == manifest.DataID, "before and after must use the manifest data identity")
	add(manifest.Before.Viewport == manifest.After.Viewport, "before and after must use the same viewport")
	add(len(manifest.Regions) > 0, "at least one selectable region is required")
	seen := map[string]bool{}
	for index, region := range manifest.Regions {
		field := fmt.Sprintf("region %d", index+1)
		add(regionIDPattern.MatchString(region.ID), field+" id must be a safe stable token")
		if regionIDPattern.MatchString(region.ID) {
			add(!seen[region.ID], "region ids must be unique")
			seen[region.ID] = true
			entryPath := "regions/" + region.ID + ".json"
			add(findRegionEntry(folder, entryPath) != nil, "region "+region.ID+" needs an answerable evidence entry")
		}
		add(strings.TrimSpace(region.NodeID) != "" && strings.TrimSpace(region.Label) != "" && strings.TrimSpace(region.Component) != "" && strings.TrimSpace(region.EvidenceID) != "", field+" must name node, label, component and crop evidence")
		box := region.Bounds
		add(box.X >= 0 && box.Y >= 0 && box.Width > 0 && box.Height > 0 && box.X+box.Width <= 1 && box.Y+box.Height <= 1, field+" bounds must stay within the image")
		add(strings.TrimSpace(region.Crop.Path) != "" && validDigest(region.Crop.SHA256), field+" crop needs a folder path and sha256 digest")
		if region.Crop.Path != "" && validDigest(region.Crop.SHA256) {
			if err := validateFeedbackImage(ctx, folder, region.Crop.Path, region.Crop.SHA256, regionCropLimit, read); err != nil {
				problems = append(problems, field+" crop is missing or invalid: "+err.Error())
			}
		}
	}
	return problems
}

func validDigest(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validateFeedbackImage(ctx context.Context, folder AskAttachment, path, digest string, limit int64, read BlobReader) error {
	entry := findRegionEntry(folder, path)
	if entry == nil || entry.SHA256 != digest {
		return fmt.Errorf("folder entry or sha256 does not match")
	}
	if entry.MediaType != "image/png" && entry.MediaType != "image/jpeg" && entry.MediaType != "image/webp" {
		return fmt.Errorf("only PNG, JPEG and WebP evidence is supported")
	}
	if entry.SizeBytes <= 0 || entry.SizeBytes > limit {
		return fmt.Errorf("image bytes exceed the %d byte limit", limit)
	}
	data, err := read(ctx, entry.SHA256, limit)
	if err != nil {
		return fmt.Errorf("stored image bytes are unavailable")
	}
	if int64(len(data)) != entry.SizeBytes {
		return fmt.Errorf("stored image size does not match")
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != digest {
		return fmt.Errorf("stored image sha256 does not match")
	}
	if !validFeedbackImageBytes(entry.MediaType, data) {
		return fmt.Errorf("image bytes do not match their declared type")
	}
	return nil
}

func validFeedbackImageBytes(media string, data []byte) bool {
	switch media {
	case "image/png":
		return len(data) >= 8 && bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n"))
	case "image/jpeg":
		return len(data) >= 3 && bytes.HasPrefix(data, []byte{0xff, 0xd8, 0xff})
	case "image/webp":
		return len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP"
	default:
		return false
	}
}
