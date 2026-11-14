package ramp

import (
	"net/http"

	buildsh "scenario-to-android/handlers/builds"
	distributionh "scenario-to-android/handlers/distribution"
	journeyh "scenario-to-android/handlers/journeys"
	readinessh "scenario-to-android/handlers/readiness"
	targetsh "scenario-to-android/handlers/targets"
	"scenario-to-android/internal/module"

	"github.com/gorilla/mux"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

func Module(matrixHandlers []*validationmatrix.Handler, surfaces ...buildsh.Surface) module.Module {
	var surface buildsh.Surface
	if len(surfaces) > 0 {
		surface = surfaces[0]
	}
	return module.Module{
		Name: "ramp",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/android/targets", targetsh.Handler).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/conformance-plan", journeyh.ConformancePlan).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/readiness", readinessh.Handler).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/distribution", distributionh.Handler).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/generate", buildsh.GenerateHandler(surface)).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/android/build", buildsh.BuilderHandler(surface)).Methods(http.MethodPost)
			r.HandleFunc("/api/v1/android/artifacts", buildsh.ArtifactListHandler(surface)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/artifacts/metadata", buildsh.ArtifactMetadataHandler(surface)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/artifacts/apk", buildsh.APKArtifactHandler(surface)).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/signing/provision", buildsh.ProvisionSigningHandler(surface)).Methods(http.MethodPost)
			matrixCreateRegistered := false
			for _, handler := range matrixHandlers {
				if handler != nil {
					if !matrixCreateRegistered {
						r.HandleFunc("/api/v1/android/matrices", MatrixCreateHandler(surface, handler)).Methods(http.MethodPost)
						matrixCreateRegistered = true
					}
					handler.RegisterRoutes(r)
				}
			}
		},
		Endpoints: []module.EndpointDescriptor{
			rampEndpoint("android_targets", "/api/v1/android/targets", http.MethodGet, "Probe local and device-control Android targets"),
			rampEndpoint("android_conformance_plan", "/api/v1/android/conformance-plan", http.MethodGet, "Show the generated-app Android conformance contract"),
			rampEndpoint("android_readiness", "/api/v1/android/readiness", http.MethodGet, "Show Google Android release readiness rungs"),
			rampEndpoint("android_distribution", "/api/v1/android/distribution", http.MethodGet, "Show independent Android distribution channels"),
			{ID: "android_generate", Path: "/api/v1/android/generate", Method: http.MethodPost, Summary: "Render a generated Android project without building artifacts", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Operator generation control surface accepts a JSON request and returns a generated-project descriptor."}},
			{ID: "android_build", Path: "/api/v1/android/build", Method: http.MethodPost, Summary: "Build a debug APK and AAB from a scenario web bundle", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Operator build control surface accepts a JSON request and returns an immutable artifact descriptor."}},
			{ID: "android_artifacts", Path: "/api/v1/android/artifacts", Method: http.MethodGet, Summary: "List recent path-free Android artifact candidates", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "The producer returns immutable refs and package/version/signing metadata; matrix creation verifies selected bytes before use."}},
			{ID: "android_artifact_metadata", Path: "/api/v1/android/artifacts/metadata", Method: http.MethodGet, Summary: "Resolve immutable Android artifact metadata", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "The artifact producer resolves an exact immutable ref and returns package/version/checksum metadata without host paths."}},
			{ID: "android_artifact_apk", Path: "/api/v1/android/artifacts/apk", Method: http.MethodGet, Summary: "Fetch verified APK bytes by immutable artifact ref", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "The artifact producer integrity-checks the retained APK/AAB pair and streams only the APK selected by exact immutable_ref."}},
			rampEndpoint("android_matrix_create", "/api/v1/android/matrices", http.MethodPost, "Create an Android validation matrix from a producer-owned immutable artifact ref"),
			{ID: "android_signing_provision", Path: "/api/v1/android/signing/provision", Method: http.MethodPost, Summary: "Generate and store the Android upload key through secrets-manager", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Provisioning returns metadata only; key material is never returned."}},
		},
	}
}

func rampEndpoint(id, path, method, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: method, Summary: summary, Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Android delivery-ramp operator surface is exposed as a plain JSON control endpoint."}}
}
