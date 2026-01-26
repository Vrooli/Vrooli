package main

import "time"

// FileStatus represents the git status of a file
type FileStatus string

const (
	FileStatusTracked   FileStatus = "tracked"
	FileStatusUntracked FileStatus = "untracked"
	FileStatusIgnored   FileStatus = "ignored"
)

// FileInfo represents a single file in the repository
type FileInfo struct {
	Path     string     `json:"path"`
	Language string     `json:"language,omitempty"` // Derived from extension
	Status   FileStatus `json:"status,omitempty"`
}

// FileTreeRequest configures file listing
type FileTreeRequest struct {
	Pattern string `json:"pattern,omitempty"` // Glob filter (optional)
	Limit   int    `json:"limit,omitempty"`   // Max results (default 1000, max 5000)
	Deep    bool   `json:"deep,omitempty"`    // If true, search ALL files including .gitignore'd
	Timeout int    `json:"timeout,omitempty"` // Max search duration in ms (default 5000, max 30000)
}

// FileTreeResponse contains the file listing result
type FileTreeResponse struct {
	Files      []FileInfo `json:"files"`
	Truncated  bool       `json:"truncated"`   // True if limit reached
	Cancelled  bool       `json:"cancelled"`   // True if timeout hit
	SearchMode string     `json:"search_mode"` // "default" | "deep"
	Timestamp  time.Time  `json:"timestamp"`
}

// DefaultFileTreeLimit is the default limit for file listing
const DefaultFileTreeLimit = 1000

// MaxFileTreeLimit is the maximum limit for file listing
const MaxFileTreeLimit = 5000

// DefaultFileTreeTimeout is the default timeout in milliseconds
const DefaultFileTreeTimeout = 5000

// MaxFileTreeTimeout is the maximum timeout in milliseconds
const MaxFileTreeTimeout = 30000

// LanguageFromExtension returns a language name based on file extension
func LanguageFromExtension(path string) string {
	extMap := map[string]string{
		".go":    "go",
		".ts":    "typescript",
		".tsx":   "typescript",
		".js":    "javascript",
		".jsx":   "javascript",
		".py":    "python",
		".rb":    "ruby",
		".rs":    "rust",
		".java":  "java",
		".kt":    "kotlin",
		".c":     "c",
		".cpp":   "cpp",
		".h":     "c",
		".hpp":   "cpp",
		".cs":    "csharp",
		".swift": "swift",
		".php":   "php",
		".sh":    "shell",
		".bash":  "shell",
		".zsh":   "shell",
		".json":  "json",
		".yaml":  "yaml",
		".yml":   "yaml",
		".toml":  "toml",
		".xml":   "xml",
		".html":  "html",
		".css":   "css",
		".scss":  "scss",
		".less":  "less",
		".md":    "markdown",
		".sql":   "sql",
		".vue":   "vue",
		".svelte": "svelte",
	}

	// Find extension from path
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			ext := path[i:]
			if lang, ok := extMap[ext]; ok {
				return lang
			}
			return ""
		}
		if path[i] == '/' {
			break
		}
	}
	return ""
}

// RelationType describes the relationship between files
type RelationType string

const (
	RelationTypeImports    RelationType = "imports"     // Files this file imports
	RelationTypeImportedBy RelationType = "imported_by" // Files that import this file
	RelationTypeTest       RelationType = "test"        // Test file for this file
	RelationTypeIndex      RelationType = "index"       // Index file that exports this file
	RelationTypeTypes      RelationType = "types"       // Type definition file
)

// RelatedFile represents a file related to the current file
type RelatedFile struct {
	Path         string       `json:"path"`
	RelationType RelationType `json:"relation_type"`
}

// RelatedFilesResponse contains files related to the requested path
type RelatedFilesResponse struct {
	Path      string        `json:"path"`
	Related   []RelatedFile `json:"related"`
	Timestamp time.Time     `json:"timestamp"`
}

// DirEntry represents a file or folder in a directory listing
type DirEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"is_dir"`
	Language string `json:"language,omitempty"`
	Tracked  bool   `json:"tracked"` // True if tracked by git (files only; folders always true)
}

// DirListResponse contains contents of a directory
type DirListResponse struct {
	Path      string     `json:"path"`
	Entries   []DirEntry `json:"entries"`
	Timestamp time.Time  `json:"timestamp"`
}

// MaxDirDepth is the maximum depth for directory listing (safety limit)
const MaxDirDepth = 20
