package ramp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	buildsh "scenario-to-android/handlers/builds"
	buildsdomain "scenario-to-android/internal/builds"

	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

type matrixCreator interface {
	Create(validationmatrix.MatrixSelection) (*validationmatrix.MatrixRun, error)
}

type matrixCreateRequest struct {
	ArtifactRef string                           `json:"artifact_ref"`
	Selection   validationmatrix.MatrixSelection `json:"selection"`
}

// MatrixCreateHandler resolves an immutable Android artifact reference inside
// the producer before creating a matrix. The host path never crosses the API.
func MatrixCreateHandler(surface buildsh.Surface, creator matrixCreator) http.HandlerFunc {
	return matrixCreateHandler(surface, creator, buildsdomain.ResolveAPK)
}

func matrixCreateHandler(surface buildsh.Surface, creator matrixCreator, resolve func(string, string) (buildsdomain.ResolvedAPK, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vrooli-Scenario-UI-Proxy") == "scenario-to-android" {
			http.Error(w, "Android matrix creation is available only on the local producer API", http.StatusForbidden)
			return
		}
		var input matrixCreateRequest
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "decode Android matrix selection: "+err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(input.ArtifactRef) == "" {
			http.Error(w, "artifact_ref is required", http.StatusBadRequest)
			return
		}
		if resolve == nil {
			http.Error(w, "Android artifact resolver unavailable", http.StatusServiceUnavailable)
			return
		}
		if creator == nil {
			http.Error(w, "Android matrix service unavailable", http.StatusServiceUnavailable)
			return
		}
		if strings.TrimSpace(input.Selection.ArtifactPath) != "" || len(input.Selection.ArtifactPaths) > 0 {
			http.Error(w, "artifact paths are resolved by the Android producer", http.StatusBadRequest)
			return
		}
		if surface.ArtifactRoot == nil {
			http.Error(w, "Android artifact resolution unavailable", http.StatusServiceUnavailable)
			return
		}
		root, err := surface.ArtifactRoot(r.Context())
		if err != nil || root == "" {
			http.Error(w, "Android artifact storage unavailable", http.StatusServiceUnavailable)
			return
		}
		resolved, err := resolve(root, input.ArtifactRef)
		if err != nil {
			http.Error(w, "Android artifact not found or failed integrity verification", http.StatusUnprocessableEntity)
			return
		}
		if resolved.File == nil || strings.TrimSpace(resolved.Path) == "" || resolved.Descriptor.ImmutableRef != input.ArtifactRef {
			if resolved.File != nil {
				_ = resolved.File.Close()
			}
			http.Error(w, "Android artifact resolver returned an incomplete identity", http.StatusUnprocessableEntity)
			return
		}
		defer resolved.File.Close()
		if input.Selection.ArtifactDigest != resolved.Descriptor.Checksum {
			http.Error(w, "artifact digest does not match the immutable reference", http.StatusBadRequest)
			return
		}
		if got := strings.TrimSpace(input.Selection.Metadata["immutable_ref"]); got != input.ArtifactRef {
			http.Error(w, "artifact metadata does not match the immutable reference", http.StatusBadRequest)
			return
		}
		if err := bindVerifiedArtifactMetadata(&input.Selection, resolved.Descriptor); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		input.Selection.ArtifactPath = resolved.Path
		run, err := creator.Create(input.Selection)
		if err != nil {
			http.Error(w, fmt.Sprintf("create Android matrix: %v", err), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "private, no-store")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(validationmatrix.PublicRun(run))
	}
}

func bindVerifiedArtifactMetadata(selection *validationmatrix.MatrixSelection, descriptor buildsdomain.APKDescriptor) error {
	supplied := selection.Metadata
	verified := map[string]string{
		"immutable_ref":    descriptor.ImmutableRef,
		"package_name":     descriptor.PackageName,
		"version_name":     descriptor.VersionName,
		"version_code":     strconv.FormatInt(descriptor.VersionCode, 10),
		"signing":          descriptor.Signing,
		"signing_identity": descriptor.SigningIdentity,
	}
	allowed := map[string]struct{}{
		"scenario_name": {}, "auth_profile_id": {}, "install_mode": {},
		"immutable_ref": {}, "package_name": {}, "version_name": {},
		"version_code": {}, "signing": {}, "signing_identity": {},
	}
	for key := range supplied {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("artifact metadata key %q is not supported", key)
		}
	}
	metadata := make(map[string]string, len(supplied)+len(verified))
	for key, value := range verified {
		if suppliedValue := strings.TrimSpace(supplied[key]); suppliedValue != "" && suppliedValue != value {
			return fmt.Errorf("artifact metadata %q does not match the verified immutable artifact", key)
		}
		metadata[key] = value
	}
	for _, key := range []string{"scenario_name", "auth_profile_id", "install_mode"} {
		if value := strings.TrimSpace(supplied[key]); value != "" {
			metadata[key] = value
		}
	}
	selection.Metadata = metadata
	return nil
}
