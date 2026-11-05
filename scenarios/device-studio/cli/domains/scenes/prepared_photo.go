package scenes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"context"
	"github.com/vrooli/cli-core/cliapp"
	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes"
	api "github.com/vrooli/vrooli/packages/proto/gen/go/device-studio/v1/scenes/scenes_v1connect"
)

type preparedPhotoResult struct {
	SceneID            string  `json:"scene_id"`
	SceneRevision      uint32  `json:"scene_revision"`
	SceneSHA256        string  `json:"scene_sha256"`
	LibraryBundleID    string  `json:"library_bundle_id"`
	LibraryRevision    uint32  `json:"library_revision"`
	LibraryManifestSHA string  `json:"library_manifest_sha256"`
	ScaleStatus        string  `json:"scale_status"`
	EstimatedScaleM    float64 `json:"estimated_scale_m"`
}

func preparedPhotoAction(client api.ScenesServiceClient) func(cliapp.OperationContext) (preparedPhotoResult, error) {
	return func(ctx cliapp.OperationContext) (preparedPhotoResult, error) {
		preparationID := strings.TrimSpace(ctx.Flag("preparation-id"))
		if !validPreparationID(preparationID) {
			return preparedPhotoResult{}, errors.New("preparation-id must be a lowercase SHA-256 digest")
		}
		expectedSHA := strings.TrimSpace(ctx.Flag("expected-bundle-sha256"))
		if !validPreparationID(expectedSHA) || expectedSHA != preparationID {
			return preparedPhotoResult{}, errors.New("expected-bundle-sha256 must equal the exact preparation content ID")
		}
		licenseID := strings.TrimSpace(ctx.Flag("license-id"))
		attribution := strings.TrimSpace(ctx.Flag("attribution"))
		sourceLabel := strings.TrimSpace(ctx.Flag("source-label"))
		allowLocalRender, err := parseAllowLocalRender(ctx.Flag("allow-local-render"))
		if err != nil {
			return preparedPhotoResult{}, err
		}
		sourceUseRationale := strings.TrimSpace(ctx.Flag("source-use-rationale"))
		if licenseID == "" || len(licenseID) > 128 || attribution == "" || len(attribution) > 1024 || sourceLabel == "" || len(sourceLabel) > 512 {
			return preparedPhotoResult{}, errors.New("license-id, attribution, and source-label claims are required and must fit their documented limits")
		}
		if len(sourceUseRationale) > 1024 || (allowLocalRender && sourceUseRationale == "") || (!allowLocalRender && sourceUseRationale != "") {
			return preparedPhotoResult{}, errors.New("source-use-rationale is required only with --allow-local-render=true and must be at most 1024 bytes")
		}
		response, err := client.ImportPreparedPhotograph(context.Background(), connect.NewRequest(&v1.ImportPreparedPhotographRequest{PreparationId: preparationID, ExpectedBundleSha256: expectedSHA, LicenseId: licenseID, Attribution: attribution, SourceLabel: sourceLabel, AllowLocalRender: allowLocalRender, SourceUseRationale: sourceUseRationale}))
		if err != nil {
			return preparedPhotoResult{}, cliapp.WrapAPIError("convert accepted photograph preparation", err, nil)
		}
		msg := response.Msg
		result := preparedPhotoResult{SceneID: msg.GetRevision().GetSceneId(), SceneRevision: msg.GetRevision().GetRevision(), SceneSHA256: msg.GetRevision().GetSha256(), LibraryBundleID: msg.GetLibraryBundleId(), LibraryRevision: msg.GetLibraryRevision(), LibraryManifestSHA: msg.GetLibraryManifestSha256(), ScaleStatus: msg.GetScaleStatus(), EstimatedScaleM: msg.GetEstimatedScaleM()}
		if !validPreparationID(result.SceneSHA256) || !validPreparationID(result.LibraryManifestSHA) || result.SceneID == "" || result.LibraryBundleID == "" || result.SceneRevision == 0 || result.LibraryRevision == 0 || result.ScaleStatus != "estimated_monocular_scale_not_measured" {
			return result, errors.New("server returned incomplete or unqualified photographic scene candidate metadata")
		}
		return result, nil
	}
}

func parseAllowLocalRender(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, errors.New("allow-local-render must be true or false")
	}
}

func preparedPhotoReport(_ cliapp.OperationContext, result preparedPhotoResult) cliapp.MutationReport {
	return cliapp.MutationReport{
		Result:  []string{fmt.Sprintf("Created scene candidate %s@%d and library bundle %s@%d.", result.SceneID, result.SceneRevision, result.LibraryBundleID, result.LibraryRevision), "Local render permission is separate from candidate qualification and publication. Monocular scale is an estimate, not measured camera calibration."},
		Changes: []string{fmt.Sprintf("scene_sha256=%s library_manifest_sha256=%s scale_status=%s estimated_scale_m=%.4f", result.SceneSHA256, result.LibraryManifestSHA, result.ScaleStatus, result.EstimatedScaleM)},
	}
}

func validPreparationID(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

type geometryQAResult struct {
	PreparationID  string `json:"preparation_id"`
	OutputPath     string `json:"output_path"`
	Bytes          int    `json:"bytes"`
	InspectionOnly bool   `json:"inspection_only"`
}

func geometryQAAction(core *cliapp.ScenarioApp) func(cliapp.OperationContext) (geometryQAResult, error) {
	return func(ctx cliapp.OperationContext) (geometryQAResult, error) {
		id := strings.TrimSpace(ctx.Flag("preparation-id"))
		out := strings.TrimSpace(ctx.Flag("out"))
		if !validPreparationID(id) {
			return geometryQAResult{}, errors.New("preparation-id must be a lowercase SHA-256 digest")
		}
		if out == "" || strings.ToLower(filepath.Ext(out)) != ".zip" {
			return geometryQAResult{}, errors.New("out must name a new .zip file")
		}
		if _, err := os.Lstat(out); !os.IsNotExist(err) {
			if err == nil {
				return geometryQAResult{}, fmt.Errorf("output path already exists: %s", out)
			}
			return geometryQAResult{}, err
		}
		data, err := core.Request("GET", "/photographic-scenes/preparations/"+id+"/geometry-qa", nil, nil)
		if err != nil {
			return geometryQAResult{}, cliapp.WrapAPIError("inspect pending photograph geometry", err, nil)
		}
		if len(data) < 4 || len(data) > 32<<20 || string(data[:2]) != "PK" {
			return geometryQAResult{}, errors.New("geometry QA response is not a bounded ZIP archive")
		}
		if err := os.MkdirAll(filepath.Dir(out), 0o700); err != nil {
			return geometryQAResult{}, err
		}
		file, err := os.CreateTemp(filepath.Dir(out), ".device-studio-geometry-qa-*")
		if err != nil {
			return geometryQAResult{}, err
		}
		defer os.Remove(file.Name())
		if _, err := file.Write(data); err != nil {
			_ = file.Close()
			return geometryQAResult{}, err
		}
		if err := file.Sync(); err != nil {
			_ = file.Close()
			return geometryQAResult{}, err
		}
		if err := file.Close(); err != nil {
			return geometryQAResult{}, err
		}
		if err := os.Link(file.Name(), out); err != nil {
			return geometryQAResult{}, fmt.Errorf("write QA archive without overwriting: %w", err)
		}
		return geometryQAResult{PreparationID: id, OutputPath: out, Bytes: len(data), InspectionOnly: true}, nil
	}
}

func geometryQAReport(_ cliapp.OperationContext, result geometryQAResult) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Wrote pending-only, inspection-only geometry QA for %s to %s.", result.PreparationID, result.OutputPath)}, Changes: []string{fmt.Sprintf("bytes=%d accepted=false; archive contains geometry-qa.json and geometry-qa.png", result.Bytes)}}
}
