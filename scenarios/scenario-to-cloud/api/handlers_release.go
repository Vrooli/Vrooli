package main

import (
	"net/http"

	"github.com/gorilla/mux"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/release"
	"scenario-to-cloud/releasesvc"
)

func (s *Server) releaseService() (*releasesvc.Service, error) {
	if s == nil {
		return nil, apierrors.Internal("release service unavailable", nil)
	}
	return s.releaseSvc, s.releaseErr
}

// newDefaultReleaseService binds the repository root, the bundle store and
// the control-plane release authority (argv seam) as the production signer.
func newDefaultReleaseService() (*releasesvc.Service, error) {
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		return nil, err
	}
	storeDir, err := bundle.GetLocalBundlesDir()
	if err != nil {
		return nil, err
	}
	return releasesvc.New(releasesvc.Config{RepoRoot: repoRoot, StoreDir: storeDir, Signer: release.ArgvSigner{RepoRoot: repoRoot}}), nil
}

// registerReleaseRoutes mounts the REST release surface beside the Connect
// ReleasesService. Mount line for main.go setupRoutes:
//
//	s.registerReleaseRoutes(api)
func (s *Server) registerReleaseRoutes(api *mux.Router) {
	api.HandleFunc("/releases/build", s.handleBuildRelease).Methods("POST")
	api.HandleFunc("/releases/{digest}", s.handleGetRelease).Methods("GET")
	api.HandleFunc("/releases/{digest}/verify", s.handleVerifyRelease).Methods("POST")
	if svc, err := s.releaseService(); err == nil {
		path, handler := svc.Handler()
		s.router.PathPrefix(path).Handler(handler)
	}
}

func (s *Server) handleBuildRelease(w http.ResponseWriter, r *http.Request) {
	svc, err := s.releaseService()
	if err != nil {
		apierrors.Write(w, apierrors.Internal("release service unavailable", err))
		return
	}
	var req releasesvc.BuildRequest
	if !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	rel, err := svc.Build(r.Context(), req)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": releasesvc.SchemaVersion, "release": rel})
}

func (s *Server) handleGetRelease(w http.ResponseWriter, r *http.Request) {
	svc, err := s.releaseService()
	if err != nil {
		apierrors.Write(w, apierrors.Internal("release service unavailable", err))
		return
	}
	rel, err := svc.Get(mux.Vars(r)["digest"])
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": releasesvc.SchemaVersion, "release": rel})
}

func (s *Server) handleVerifyRelease(w http.ResponseWriter, r *http.Request) {
	svc, err := s.releaseService()
	if err != nil {
		apierrors.Write(w, apierrors.Internal("release service unavailable", err))
		return
	}
	var req releasesvc.VerifyRequest
	if r.ContentLength != 0 && !httputil.DecodeRequestBody(w, r, &req) {
		return
	}
	report, err := svc.Verify(r.Context(), mux.Vars(r)["digest"], req)
	if err != nil {
		if typed := apierrors.As(err); typed != nil {
			err = typed.WithDetail("report", report)
		}
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": releasesvc.SchemaVersion, "report": report})
}
