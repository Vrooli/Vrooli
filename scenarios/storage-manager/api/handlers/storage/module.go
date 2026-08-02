package storage

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorilla/mux"
	corestorage "github.com/vrooli/api-core/storage"
	"storage-manager/internal/census"
	"storage-manager/internal/module"
)

type ModuleDeps struct{ RepoRoot string }

func Module(d ModuleDeps) module.Module {
	return module.Module{Name: "storage", Mount: func(r *mux.Router) {
		r.HandleFunc("/api/v1/census", func(w http.ResponseWriter, req *http.Request) {
			root := d.RepoRoot
			if requested := strings.TrimSpace(req.URL.Query().Get("root")); requested != "" {
				root = requested
			}
			report, err := census.Scan(root, declarations(d.RepoRoot))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(report)
		}).Methods(http.MethodGet)
	}, Endpoints: Endpoints}
}

func declarations(repoRoot string) map[string][]census.Declaration {
	result := map[string][]census.Declaration{}
	entries, _ := os.ReadDir(filepath.Join(repoRoot, "scenarios"))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(repoRoot, "scenarios", entry.Name(), ".vrooli", "service.json")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw struct {
			Storage struct {
				Entries map[string]struct {
					Path   json.RawMessage `json:"path"`
					Budget json.RawMessage `json:"budget"`
				} `json:"entries"`
			} `json:"storage"`
		}
		if json.Unmarshal(data, &raw) != nil {
			continue
		}
		for name, declaration := range raw.Storage.Entries {
			var portable corestorage.PortablePath
			if json.Unmarshal(declaration.Path, &portable) != nil {
				continue
			}
			path, err := corestorage.ResolvePortablePath(name, portable, corestorage.Platform(runtime.GOOS), corestorage.PlatformSeams{})
			if err != nil {
				continue
			}
			if !filepath.IsAbs(path) {
				path = filepath.Join(repoRoot, "scenarios", entry.Name(), path)
			}
			result[entry.Name()] = append(result[entry.Name()], census.Declaration{Name: name, Path: path, Budgeted: len(declaration.Budget) > 0})
		}
	}
	return result
}

var Endpoints = []module.EndpointDescriptor{{ID: "storage_census", Path: "/api/v1/census", Method: http.MethodGet, Summary: "Measure declared and unattributed storage", Description: "Read-only closed accounting over the selected root."}}
