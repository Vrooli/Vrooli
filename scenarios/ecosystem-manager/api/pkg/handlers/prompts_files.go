package handlers

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/prompts"
	"github.com/gorilla/mux"
)

// PromptsHandlers manages prompt file read/write operations.
type PromptsHandlers struct {
	assembler *prompts.Assembler
}

// PromptFileInfo is a lightweight listing item.
type PromptFileInfo struct {
	ID          string `json:"id"`
	Path        string `json:"path"`
	DisplayName string `json:"display_name,omitempty"`
	Type        string `json:"type,omitempty"`
	Size        int64  `json:"size,omitempty"`
	ModifiedAt  string `json:"modified_at,omitempty"`
}

// PromptFile represents file content and metadata.
type PromptFile struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at,omitempty"`
}

// PhaseInfo represents a phase name for the UI.
type PhaseInfo struct {
	Name string `json:"name"`
}

// NewPromptsHandlers creates a new prompts handler set.
func NewPromptsHandlers(assembler *prompts.Assembler) *PromptsHandlers {
	return &PromptsHandlers{assembler: assembler}
}

// ListPromptFilesHandler returns all prompt files under the prompts directory.
func (h *PromptsHandlers) ListPromptFilesHandler(w http.ResponseWriter, r *http.Request) {
	files, err := h.listPromptFiles()
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to list prompt files: %v", err), http.StatusInternalServerError)
		return
	}
	writeJSON(w, files, http.StatusOK)
}

// ListPhaseNamesHandler returns available phase names from prompts/phases/*.md files.
func (h *PromptsHandlers) ListPhaseNamesHandler(w http.ResponseWriter, r *http.Request) {
	phasesDir := filepath.Join(h.assembler.PromptsDir, "phases")
	var phases []PhaseInfo

	err := filepath.WalkDir(phasesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		// Extract phase name from filename (remove .md extension)
		name := strings.TrimSuffix(d.Name(), ".md")
		phases = append(phases, PhaseInfo{Name: name})
		return nil
	})

	if err != nil {
		writeError(w, fmt.Sprintf("Failed to list phase names: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, phases, http.StatusOK)
}

// GetPromptFileHandler returns the content of a prompt file.
func (h *PromptsHandlers) GetPromptFileHandler(w http.ResponseWriter, r *http.Request) {
	relPath, fullPath, ok := h.resolvePromptPath(w, r)
	if !ok {
		return
	}

	content, info, err := readFileWithInfo(fullPath)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to read prompt file: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, PromptFile{
		ID:         relPath,
		Path:       relPath,
		Content:    content,
		Size:       info.Size(),
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
	}, http.StatusOK)
}

// CreatePromptFileHandler creates a new prompt file.
func (h *PromptsHandlers) CreatePromptFileHandler(w http.ResponseWriter, r *http.Request) {
	payload, ok := decodeJSONBody[struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}](w, r)
	if !ok {
		return
	}

	if strings.TrimSpace(payload.Path) == "" {
		writeError(w, "Path is required", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(payload.Content) == "" {
		writeError(w, "Content must not be empty", http.StatusBadRequest)
		return
	}

	clean := filepath.Clean(payload.Path)
	if !strings.HasSuffix(clean, ".md") {
		writeError(w, "Only markdown prompt files can be created", http.StatusBadRequest)
		return
	}

	fullPath := filepath.Join(h.assembler.PromptsDir, clean)
	if !strings.HasPrefix(fullPath, h.assembler.PromptsDir) {
		writeError(w, "Invalid prompt path", http.StatusBadRequest)
		return
	}

	// Check if file already exists
	if _, err := os.Stat(fullPath); err == nil {
		writeError(w, "Prompt file already exists", http.StatusConflict)
		return
	}

	// Ensure parent directory exists
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		writeError(w, fmt.Sprintf("Failed to create directory: %v", err), http.StatusInternalServerError)
		return
	}

	if err := writeFile(fullPath, payload.Content); err != nil {
		writeError(w, fmt.Sprintf("Failed to create prompt file: %v", err), http.StatusInternalServerError)
		return
	}

	content, info, err := readFileWithInfo(fullPath)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to read created prompt: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, PromptFile{
		ID:         filepath.ToSlash(clean),
		Path:       filepath.ToSlash(clean),
		Content:    content,
		Size:       info.Size(),
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
	}, http.StatusCreated)
}

// UpdatePromptFileHandler overwrites a prompt file with new content.
func (h *PromptsHandlers) UpdatePromptFileHandler(w http.ResponseWriter, r *http.Request) {
	relPath, fullPath, ok := h.resolvePromptPath(w, r)
	if !ok {
		return
	}

	payload, ok := decodeJSONBody[struct {
		Content string `json:"content"`
	}](w, r)
	if !ok {
		return
	}

	if strings.TrimSpace(payload.Content) == "" {
		writeError(w, "Content must not be empty", http.StatusBadRequest)
		return
	}

	if err := writeFile(fullPath, payload.Content); err != nil {
		writeError(w, fmt.Sprintf("Failed to save prompt file: %v", err), http.StatusInternalServerError)
		return
	}

	content, info, err := readFileWithInfo(fullPath)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to read saved prompt: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, PromptFile{
		ID:         relPath,
		Path:       relPath,
		Content:    content,
		Size:       info.Size(),
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339),
	}, http.StatusOK)
}

func (h *PromptsHandlers) listPromptFiles() ([]PromptFileInfo, error) {
	promptRoot := h.assembler.PromptsDir
	var files []PromptFileInfo

	err := filepath.WalkDir(promptRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		rel, err := filepath.Rel(promptRoot, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		info, err := d.Info()
		if err != nil {
			return err
		}

		files = append(files, PromptFileInfo{
			ID:          rel,
			Path:        rel,
			DisplayName: filepath.Base(rel),
			Type:        classifyPromptFile(rel),
			Size:        info.Size(),
			ModifiedAt:  info.ModTime().UTC().Format(time.RFC3339),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func (h *PromptsHandlers) resolvePromptPath(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	vars := mux.Vars(r)
	rawPath := vars["path"]
	if strings.TrimSpace(rawPath) == "" {
		writeError(w, "Prompt path is required", http.StatusBadRequest)
		return "", "", false
	}

	clean := filepath.Clean(rawPath)
	if !strings.HasSuffix(clean, ".md") {
		writeError(w, "Only markdown prompt files can be accessed", http.StatusBadRequest)
		return "", "", false
	}
	fullPath := filepath.Join(h.assembler.PromptsDir, clean)
	if !strings.HasPrefix(fullPath, h.assembler.PromptsDir) {
		writeError(w, "Invalid prompt path", http.StatusBadRequest)
		return "", "", false
	}

	if _, err := os.Stat(fullPath); err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "no such file or directory") {
			status = http.StatusNotFound
		}
		writeError(w, fmt.Sprintf("Prompt file not found: %v", err), status)
		return "", "", false
	}

	return filepath.ToSlash(clean), fullPath, true
}

func classifyPromptFile(relPath string) string {
	if strings.HasPrefix(relPath, "templates/") {
		return "template"
	}
	if strings.HasPrefix(relPath, "phases/") {
		return "phase"
	}
	return "other"
}

func readFileWithInfo(path string) (string, fs.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	return string(data), info, nil
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
