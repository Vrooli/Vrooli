package ramp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"scenario-to-android/internal/androidbuild"
	"scenario-to-android/internal/androidprobe"
	"scenario-to-android/internal/androidrelease"
	"scenario-to-android/internal/conformance"
	"scenario-to-android/internal/module"

	"github.com/gorilla/mux"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
	validationmatrix "github.com/vrooli/vrooli/packages/delivery-ramp-go/validationmatrix"
)

func Module(matrixHandlers []*validationmatrix.Handler, builders ...androidbuild.Builder) module.Module {
	var builder androidbuild.Builder
	if len(builders) > 0 {
		builder = builders[0]
	}
	return module.Module{
		Name: "ramp",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/api/v1/android/targets", targets).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/conformance-plan", conformancePlan).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/readiness", readiness).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/distribution", distribution).Methods(http.MethodGet)
			r.HandleFunc("/api/v1/android/build", builderHandler(builder)).Methods(http.MethodPost)
			for _, handler := range matrixHandlers {
				if handler != nil {
					handler.RegisterRoutes(r)
				}
			}
		},
		Endpoints: []module.EndpointDescriptor{
			rampEndpoint("android_targets", "/api/v1/android/targets", http.MethodGet, "Probe local and device-control Android targets"),
			rampEndpoint("android_conformance_plan", "/api/v1/android/conformance-plan", http.MethodGet, "Show the generated-app Android conformance contract"),
			rampEndpoint("android_readiness", "/api/v1/android/readiness", http.MethodGet, "Show Google Android release readiness rungs"),
			rampEndpoint("android_distribution", "/api/v1/android/distribution", http.MethodGet, "Show independent Android distribution channels"),
			{ID: "android_build", Path: "/api/v1/android/build", Method: http.MethodPost, Summary: "Build a debug APK and AAB from a scenario web bundle", Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Operator build control surface accepts a JSON request and returns an immutable artifact descriptor."}},
		},
	}
}

func rampEndpoint(id, path, method, summary string) module.EndpointDescriptor {
	return module.EndpointDescriptor{ID: id, Path: path, Method: method, Summary: summary, Category: "ramp", RESTException: &module.RESTException{Reason: module.RESTReasonOpsProbe, Note: "Android delivery-ramp operator surface is exposed as a plain JSON control endpoint."}}
}

type buildRequest struct {
	SourceRef   string `json:"source_ref"`
	PackageName string `json:"package_name,omitempty"`
	AppName     string `json:"app_name,omitempty"`
	VersionName string `json:"version_name,omitempty"`
	VersionCode string `json:"version_code,omitempty"`
	TargetSDK   string `json:"target_sdk,omitempty"`
}

func builderHandler(builder androidbuild.Builder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request buildRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, "decode Android build request: "+err.Error(), http.StatusBadRequest)
			return
		}
		parameters := map[string]string{
			"package_name": request.PackageName,
			"app_name":     request.AppName,
			"version_name": request.VersionName,
			"version_code": request.VersionCode,
			"target_sdk":   request.TargetSDK,
		}
		artifact, err := builder.Build(r.Context(), deliveryramp.BuildRequest{SourceRef: request.SourceRef, Parameters: parameters})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, artifact)
	}
}

func targets(w http.ResponseWriter, r *http.Request) {
	prober := androidprobe.Prober{Devices: androidprobe.NewDeviceControlInventory()}
	var bridgeSources []deliveryramp.BridgeSource
	if bridge := validationmatrix.NewClientFromEnv(); bridge != nil {
		bridgeSources = append(bridgeSources, bridge)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	inventory, err := deliveryramp.Discover(ctx, prober, bridgeSources...)
	writeJSON(w, inventory, err)
}

func conformancePlan(w http.ResponseWriter, _ *http.Request) { writeJSON(w, conformance.AndroidPlan()) }

func readiness(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, androidrelease.GoogleReadiness(false, false, false, true, false, false))
}

func distribution(w http.ResponseWriter, r *http.Request) {
	result, err := (androidrelease.Distributor{}).Distribute(r.Context(), deliveryramp.DistributionRequest{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, result)
}

func writeJSON(w http.ResponseWriter, value any, err ...error) {
	if len(err) > 0 && err[0] != nil {
		http.Error(w, err[0].Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
