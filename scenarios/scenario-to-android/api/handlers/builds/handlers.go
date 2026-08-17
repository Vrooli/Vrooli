package builds

import (
	"context"
	"encoding/json"
	"net/http"

	"scenario-to-android/internal/builds"

	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

type Surface struct {
	Builder          builds.Builder
	Generate         func(context.Context, deliveryramp.BuildRequest) (builds.GeneratedProject, error)
	SigningIdentity  string
	ProvisionSigning func(context.Context, string) error
}

type request struct {
	SourceRef       string `json:"source_ref"`
	ScenarioName    string `json:"scenario_name,omitempty"`
	PackageName     string `json:"package_name,omitempty"`
	AppName         string `json:"app_name,omitempty"`
	VersionName     string `json:"version_name,omitempty"`
	VersionCode     string `json:"version_code,omitempty"`
	TargetSDK       string `json:"target_sdk,omitempty"`
	Signing         string `json:"signing,omitempty"`
	SigningIdentity string `json:"signing_identity,omitempty"`
}

func BuilderHandler(surface Surface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input request
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "decode Android build request: "+err.Error(), http.StatusBadRequest)
			return
		}
		artifact, err := surface.Builder.Build(r.Context(), deliveryramp.BuildRequest{SourceRef: input.SourceRef, Parameters: parameters(input)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, artifact)
	}
}

func GenerateHandler(surface Surface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input request
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "decode Android generation request: "+err.Error(), http.StatusBadRequest)
			return
		}
		if surface.Generate == nil {
			http.Error(w, "Android generation unavailable: configured builder does not expose generation", http.StatusServiceUnavailable)
			return
		}
		project, err := surface.Generate(r.Context(), deliveryramp.BuildRequest{SourceRef: input.SourceRef, Parameters: parameters(input)})
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, project)
	}
}

func ProvisionSigningHandler(surface Surface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identity := surface.SigningIdentity
		var input struct {
			Identity string `json:"identity,omitempty"`
		}
		if r.Body != nil {
			if err := json.NewDecoder(r.Body).Decode(&input); err != nil && err.Error() != "EOF" {
				http.Error(w, "decode Android signing request: "+err.Error(), http.StatusBadRequest)
				return
			}
		}
		if input.Identity != "" {
			identity = input.Identity
		}
		if surface.ProvisionSigning == nil {
			http.Error(w, "Android signing unavailable: secrets-manager credential client is not configured", http.StatusServiceUnavailable)
			return
		}
		if err := surface.ProvisionSigning(r.Context(), identity); err != nil {
			http.Error(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeJSON(w, map[string]string{"identity": identity, "status": "configured", "provider": "secrets-manager", "material": "not-returned"})
	}
}

func parameters(input request) map[string]string {
	return map[string]string{"scenario_name": input.ScenarioName, "package_name": input.PackageName, "app_name": input.AppName, "version_name": input.VersionName, "version_code": input.VersionCode, "target_sdk": input.TargetSDK, "signing": input.Signing, "signing_identity": input.SigningIdentity}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
