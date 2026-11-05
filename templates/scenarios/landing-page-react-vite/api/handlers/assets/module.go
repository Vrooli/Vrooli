// Package assets is the assets domain's API contribution: the generated
// AssetsService Connect handler (list/get/delete) plus two REST exceptions that
// cannot be Connect RPCs — the multipart POST /api/v1/admin/assets/upload
// endpoint and the public static /api/v1/uploads/{path} file server. Business
// logic and image processing live in internal/assets.
package assets

import (
	"landing-page-react-vite-api/internal/module"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"google.golang.org/protobuf/encoding/protojson"

	landingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/landing-page-react-vite/v1/landing_page_react_vite_v1connect"

	internalassets "landing-page-react-vite-api/internal/assets"
)

const (
	uploadPath = "/api/v1/admin/assets/upload"
	servePath  = "/api/v1/uploads/{path:.*}"
)

// Module returns the assets domain's contribution: the AssetsService Connect
// handler plus the raw multipart upload and static file-serving routes.
func Module(svc *internalassets.Service, logger *log.Logger) module.Module {
	if logger == nil {
		logger = log.Default()
	}
	path, handler := landingconnect.NewAssetsServiceHandler(NewConnectHandler(Deps{Service: svc, Logger: logger}))
	return module.Module{
		Name: "assets",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: path, Handler: handler})
			r.HandleFunc(uploadPath, uploadHandler(svc, logger)).Methods(http.MethodPost)
			r.HandleFunc(servePath, serveHandler(svc)).Methods(http.MethodGet)
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports the assets domain's SQL for the modules registry.
func Schema() string { return internalassets.Schema() }

func uploadHandler(svc *internalassets.Service, logger *log.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, internalassets.MaxUploadSize)
		if err := r.ParseMultipartForm(internalassets.MaxUploadSize); err != nil {
			http.Error(w, "File too large or invalid form data", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided in 'file' field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		asset, err := svc.Upload(&internalassets.UploadRequest{
			File:       file,
			Header:     header,
			Category:   r.FormValue("category"),
			AltText:    r.FormValue("alt_text"),
			UploadedBy: r.FormValue("uploaded_by"),
		})
		if err != nil {
			status := http.StatusInternalServerError
			switch {
			case strings.Contains(err.Error(), "invalid file type"):
				status = http.StatusBadRequest
			case strings.Contains(err.Error(), "file exceeds"):
				status = http.StatusRequestEntityTooLarge
			}
			logger.Printf("assets.upload: %v", err)
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		payload, _ := protojson.MarshalOptions{UseProtoNames: true}.Marshal(assetToProto(asset))
		_, _ = w.Write(payload)
	}
}

func serveHandler(svc *internalassets.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		storagePath := mux.Vars(r)["path"]
		if storagePath == "" {
			http.Error(w, "File path required", http.StatusBadRequest)
			return
		}
		cleanPath := filepath.Clean(storagePath)
		if strings.Contains(cleanPath, "..") {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}
		fullPath := svc.GetFilePath(cleanPath)
		stat, err := os.Stat(fullPath)
		if err != nil {
			http.Error(w, "File not found", http.StatusNotFound)
			return
		}
		if stat.IsDir() {
			http.Error(w, "Cannot serve directory", http.StatusBadRequest)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeFile(w, r, fullPath)
	}
}
