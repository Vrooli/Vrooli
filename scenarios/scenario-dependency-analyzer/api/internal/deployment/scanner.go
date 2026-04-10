package deployment

import (
	"os"
	"path/filepath"
	"strings"
)

// LanguageType represents a detected programming language.
type LanguageType string

const (
	LangGo         LanguageType = "go"
	LangTypeScript LanguageType = "typescript"
	LangJavaScript LanguageType = "javascript"
	LangRust       LanguageType = "rust"
	LangPython     LanguageType = "python"
	LangUnknown    LanguageType = "unknown"
)

// FolderRole represents the purpose of a folder.
type FolderRole string

const (
	RoleAPI     FolderRole = "api"
	RoleCLI     FolderRole = "cli"
	RoleUI      FolderRole = "ui"
	RoleLibrary FolderRole = "library"
	RoleWorker  FolderRole = "worker"
	RoleUnknown FolderRole = "unknown"
)

// DetectedFolder represents a buildable folder in a scenario.
type DetectedFolder struct {
	Name     string       // Folder name (e.g., "api", "cli", "ui", "bas")
	Path     string       // Relative path from scenario root
	Language LanguageType // Detected language
	Role     FolderRole   // Inferred role (api, cli, ui, library, etc.)
}

// markerPriorities lists marker files in detection priority order.
// More specific markers (go.mod, Cargo.toml) take precedence over generic ones (package.json).
var markerPriorities = []struct {
	marker   string
	language LanguageType
}{
	{"go.mod", LangGo},
	{"Cargo.toml", LangRust},
	{"tsconfig.json", LangTypeScript},
	{"pyproject.toml", LangPython},
	{"setup.py", LangPython},
	{"requirements.txt", LangPython},
	{"package.json", LangTypeScript}, // Default to TS for package.json; tsconfig check refines this
}

// ScanScenarioFolders scans top-level directories and detects buildable components.
// It returns only folders with recognized language markers, skipping config/data folders.
func ScanScenarioFolders(scenarioPath string) ([]DetectedFolder, error) {
	entries, err := os.ReadDir(scenarioPath)
	if err != nil {
		return nil, err
	}

	var folders []DetectedFolder
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		folderPath := filepath.Join(scenarioPath, entry.Name())
		lang := detectLanguage(folderPath)
		if lang == LangUnknown {
			continue // Skip non-buildable folders (docs, config, etc.)
		}

		role := inferRole(entry.Name(), lang, folderPath)
		folders = append(folders, DetectedFolder{
			Name:     entry.Name(),
			Path:     entry.Name(),
			Language: lang,
			Role:     role,
		})
	}
	return folders, nil
}

// detectLanguage checks for marker files to determine the language of a folder.
func detectLanguage(folderPath string) LanguageType {
	for _, mp := range markerPriorities {
		markerPath := filepath.Join(folderPath, mp.marker)
		if _, err := os.Stat(markerPath); err == nil {
			// Refine package.json detection: if tsconfig exists, it's TypeScript; otherwise JavaScript
			if mp.marker == "package.json" {
				if hasFile(folderPath, "tsconfig.json") {
					return LangTypeScript
				}
				return LangJavaScript
			}
			return mp.language
		}
	}
	return LangUnknown
}

// inferRole guesses the folder's purpose from its name and contents.
func inferRole(name string, lang LanguageType, folderPath string) FolderRole {
	nameLower := strings.ToLower(name)

	// Direct name matches - commonly used folder names
	switch nameLower {
	case "api", "server", "backend":
		return RoleAPI
	case "cli", "cmd":
		return RoleCLI
	case "ui", "frontend", "web", "client":
		return RoleUI
	case "lib", "pkg", "internal", "shared":
		return RoleLibrary
	case "worker", "workers", "jobs", "queue":
		return RoleWorker
	}

	// Infer from language and folder structure
	switch lang {
	case LangTypeScript, LangJavaScript:
		// Check for UI indicators (React, Vue, Angular, etc.)
		if hasFile(folderPath, "index.html") ||
			hasDir(folderPath, "src/components") ||
			hasDir(folderPath, "src/pages") ||
			hasFile(folderPath, "vite.config.ts") ||
			hasFile(folderPath, "vite.config.js") ||
			hasFile(folderPath, "next.config.js") ||
			hasFile(folderPath, "next.config.ts") {
			return RoleUI
		}
		// Node.js backends often have server.js/ts or app.js/ts
		if hasFile(folderPath, "server.ts") ||
			hasFile(folderPath, "server.js") ||
			hasFile(folderPath, "src/server.ts") ||
			hasFile(folderPath, "src/server.js") ||
			hasFile(folderPath, "app.ts") ||
			hasFile(folderPath, "app.js") ||
			hasDir(folderPath, "routes") ||
			hasDir(folderPath, "src/routes") ||
			hasDir(folderPath, "handlers") ||
			hasDir(folderPath, "src/handlers") {
			return RoleAPI
		}
	case LangGo:
		// Go folders with main.go are executables
		if hasFile(folderPath, "main.go") {
			// Check for API indicators
			if strings.Contains(nameLower, "api") ||
				strings.Contains(nameLower, "server") ||
				hasDir(folderPath, "handlers") ||
				hasDir(folderPath, "routes") ||
				hasDir(folderPath, "internal/server") {
				return RoleAPI
			}
			// Default Go executable to CLI
			return RoleCLI
		}
		// Go libraries don't have main.go
		return RoleLibrary
	case LangRust:
		// Rust: check for binary vs library in Cargo.toml
		if hasFile(folderPath, "src/main.rs") {
			if strings.Contains(nameLower, "api") || strings.Contains(nameLower, "server") {
				return RoleAPI
			}
			return RoleCLI
		}
		return RoleLibrary
	case LangPython:
		// Python: check for common patterns
		if hasFile(folderPath, "manage.py") || hasDir(folderPath, "views") || hasDir(folderPath, "routes") {
			return RoleAPI
		}
		if hasFile(folderPath, "__main__.py") || hasFile(folderPath, "cli.py") {
			return RoleCLI
		}
	}

	return RoleUnknown
}

// hasFile checks if a file exists in the given directory.
func hasFile(dir, filename string) bool {
	info, err := os.Stat(filepath.Join(dir, filename))
	return err == nil && !info.IsDir()
}

// hasDir checks if a subdirectory exists in the given directory.
func hasDir(dir, subdir string) bool {
	info, err := os.Stat(filepath.Join(dir, subdir))
	return err == nil && info.IsDir()
}

// FilterByRole returns folders that match the specified role.
func FilterByRole(folders []DetectedFolder, role FolderRole) []DetectedFolder {
	var filtered []DetectedFolder
	for _, f := range folders {
		if f.Role == role {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// HasRole checks if any folder has the specified role.
func HasRole(folders []DetectedFolder, role FolderRole) bool {
	for _, f := range folders {
		if f.Role == role {
			return true
		}
	}
	return false
}

// FilterByLanguage returns folders that match the specified language.
func FilterByLanguage(folders []DetectedFolder, lang LanguageType) []DetectedFolder {
	var filtered []DetectedFolder
	for _, f := range folders {
		if f.Language == lang {
			filtered = append(filtered, f)
		}
	}
	return filtered
}
