package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) iconPreviewHandler(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimSpace(r.URL.Query().Get("path"))
	if rawPath == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}

	absPath := rawPath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(detectVrooliRoot(), absPath)
	}
	absPath = filepath.Clean(absPath)

	if !strings.HasSuffix(strings.ToLower(absPath), ".png") {
		http.Error(w, "icon must be a .png file", http.StatusBadRequest)
		return
	}

	root := detectVrooliRoot()
	absRoot, _ := filepath.Abs(root)
	absPath, _ = filepath.Abs(absPath)
	if absPath != absRoot && !strings.HasPrefix(absPath, absRoot+string(filepath.Separator)) {
		http.Error(w, "icon path must be inside the Vrooli workspace", http.StatusForbidden)
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.Mode().IsRegular() {
		http.Error(w, "icon not found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, absPath)
}
