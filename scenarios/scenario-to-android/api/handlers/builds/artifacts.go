package builds

import (
	"net/http"
	"strconv"
	"strings"

	buildsdomain "scenario-to-android/internal/builds"
)

const scenarioUIProxyHeader = "X-Vrooli-Scenario-UI-Proxy"
const scenarioUIProxyValue = "scenario-to-android"

// ArtifactMetadataHandler resolves an immutable Android reference to a
// path-free descriptor. Artifact bytes remain in the producer's data root.
func ArtifactMetadataHandler(surface Surface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if fromScenarioUIProxy(r) {
			http.Error(w, "Android artifact downloads are available only on the local producer API", http.StatusForbidden)
			return
		}
		resolved, ok := resolveAPK(w, r, surface)
		if !ok {
			return
		}
		_ = resolved.File.Close()
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		writeJSON(w, resolved.Descriptor)
	}
}

// APKArtifactHandler streams only the APK selected by its exact immutable ref.
// The response carries the owner-verified package/version and APK digest so a
// consumer can verify bytes before installation without receiving host paths.
func APKArtifactHandler(surface Surface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if fromScenarioUIProxy(r) {
			http.Error(w, "Android artifact downloads are available only on the local producer API", http.StatusForbidden)
			return
		}
		resolved, ok := resolveAPK(w, r, surface)
		if !ok {
			return
		}
		defer resolved.File.Close()
		descriptor := resolved.Descriptor
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Type", "application/vnd.android.package-archive")
		w.Header().Set("Content-Disposition", `attachment; filename="app.apk"`)
		w.Header().Set("ETag", `"`+strings.TrimPrefix(descriptor.APKChecksum, "sha256:")+`"`)
		w.Header().Set("X-Vrooli-Immutable-Ref", descriptor.ImmutableRef)
		w.Header().Set("X-Vrooli-Artifact-Checksum", descriptor.Checksum)
		w.Header().Set("X-Vrooli-APK-Checksum", descriptor.APKChecksum)
		w.Header().Set("X-Vrooli-Android-Package", descriptor.PackageName)
		w.Header().Set("X-Vrooli-Android-Version-Name", descriptor.VersionName)
		w.Header().Set("X-Vrooli-Android-Version-Code", strconv.FormatInt(descriptor.VersionCode, 10))
		http.ServeContent(w, r, "app.apk", descriptor.CreatedAt, resolved.File)
	}
}

func fromScenarioUIProxy(r *http.Request) bool {
	return r.Header.Get(scenarioUIProxyHeader) == scenarioUIProxyValue
}

func resolveAPK(w http.ResponseWriter, r *http.Request, surface Surface) (buildsdomain.ResolvedAPK, bool) {
	ref := r.URL.Query().Get("immutable_ref")
	if strings.TrimSpace(ref) == "" {
		http.Error(w, "immutable_ref is required", http.StatusBadRequest)
		return buildsdomain.ResolvedAPK{}, false
	}
	if surface.ArtifactRoot == nil {
		http.Error(w, "Android artifact resolution unavailable", http.StatusServiceUnavailable)
		return buildsdomain.ResolvedAPK{}, false
	}
	root, err := surface.ArtifactRoot(r.Context())
	if err != nil || root == "" {
		http.Error(w, "Android artifact storage unavailable", http.StatusServiceUnavailable)
		return buildsdomain.ResolvedAPK{}, false
	}
	resolved, err := buildsdomain.ResolveAPK(root, ref)
	if err != nil {
		http.Error(w, "Android artifact not found or failed integrity verification", http.StatusNotFound)
		return buildsdomain.ResolvedAPK{}, false
	}
	return resolved, true
}
