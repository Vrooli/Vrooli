package pathredact

import (
	"bytes"
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	repocontract "github.com/vrooli/repo-contract-go"
)

// Redactor rewrites generated artifact content so persisted state uses
// portable references instead of operator-specific absolute paths.
type Redactor struct {
	RepoRoots     []string
	HomeDirs      []string
	Usernames     []string
	IdentityTerms []string
}

// NewFromEnvironment builds a redactor for the current repo and operator.
func NewFromEnvironment(repoRoot string) Redactor {
	r := Redactor{}
	if repoRoot = cleanAbs(repoRoot); repoRoot != "" {
		r.RepoRoots = append(r.RepoRoots, repoRoot)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if home = cleanAbs(home); home != "" {
			r.HomeDirs = append(r.HomeDirs, home)
			r.Usernames = append(r.Usernames, filepath.Base(home))
		}
	}
	if current, err := user.Current(); err == nil && current != nil {
		r.Usernames = append(r.Usernames, current.Username)
		r.IdentityTerms = append(r.IdentityTerms, current.Name)
	}
	r.Usernames = append(r.Usernames, os.Getenv("USER"), os.Getenv("USERNAME"), os.Getenv("LOGNAME"))
	r.normalize()
	return r
}

// NewForArtifactPath builds a redactor using the repo containing path when it
// can be discovered, falling back to process-level repo discovery.
func NewForArtifactPath(path string) Redactor {
	repoRoot := ""
	if path = strings.TrimSpace(path); path != "" {
		start := path
		if filepath.Ext(start) != "" {
			start = filepath.Dir(start)
		}
		if root, err := repocontract.FindRepoRootFromPath(start); err == nil {
			repoRoot = root
		}
	}
	if repoRoot == "" {
		if root, err := repocontract.FindRepoRootFromEnvOrCWD(); err == nil {
			repoRoot = root
		}
	}
	return NewFromEnvironment(repoRoot)
}

// RedactString rewrites personal filesystem identity in generated artifact text.
func (r Redactor) RedactString(value string) string {
	if value == "" {
		return value
	}
	r.normalize()
	out := filepath.ToSlash(value)
	for _, root := range r.RepoRoots {
		out = redactRepoRoot(out, filepath.ToSlash(root))
	}
	for _, home := range r.HomeDirs {
		out = redactHomeDir(out, filepath.ToSlash(home))
	}
	// Artifacts can contain paths captured by another operator or CI fixture.
	// Redacting only this process's home leaks those identities when the
	// artifact crosses machines, so apply the portable home-dir shape too.
	out = redactGenericHomeDirs(out)
	for _, term := range r.IdentityTerms {
		out = replaceStandalone(out, term, "<user>", false)
	}
	for _, username := range r.Usernames {
		out = replaceStandalone(out, username, "<user>", true)
	}
	return out
}

// RedactBytes rewrites text artifacts. Binary artifacts are returned unchanged.
func (r Redactor) RedactBytes(path string, data []byte) ([]byte, bool) {
	if !IsTextArtifact(path, data) {
		return data, false
	}
	redacted := []byte(r.RedactString(string(data)))
	return redacted, !bytes.Equal(data, redacted)
}

// RedactJSONValue walks a JSON-like value and redacts strings in place by
// returning a deep redacted copy. Structs are converted through JSON.
func (r Redactor) RedactJSONValue(value any) (any, bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, false, err
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, false, err
	}
	redacted, changed := r.redactValue(decoded)
	return redacted, changed, nil
}

func (r Redactor) redactValue(value any) (any, bool) {
	switch typed := value.(type) {
	case string:
		redacted := r.RedactString(typed)
		return redacted, redacted != typed
	case []any:
		changed := false
		out := make([]any, len(typed))
		for i, item := range typed {
			redacted, itemChanged := r.redactValue(item)
			out[i] = redacted
			changed = changed || itemChanged
		}
		return out, changed
	case map[string]any:
		changed := false
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			var itemChanged bool
			out[key], itemChanged = r.redactValue(item)
			changed = changed || itemChanged
		}
		return out, changed
	default:
		return value, false
	}
}

// IsTextArtifact returns true for UTF-8 text-like generated artifacts.
func IsTextArtifact(path string, data []byte) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".mp4", ".mov", ".zip", ".gz", ".tgz", ".tar", ".pdf":
		return false
	}
	if bytes.IndexByte(data, 0) >= 0 {
		return false
	}
	return utf8.Valid(data)
}

func redactRepoRoot(value, root string) string {
	if root == "" {
		return value
	}
	out := strings.ReplaceAll(value, "file://"+root+"/", "file://path:")
	out = strings.ReplaceAll(out, "file://"+root, "file://path:.")
	out = strings.ReplaceAll(out, root+"/", "path:")
	out = strings.ReplaceAll(out, root, "path:.")
	return out
}

func redactHomeDir(value, home string) string {
	if home == "" {
		return value
	}
	out := strings.ReplaceAll(value, home+"/.vrooli/", "<vrooli-home>/")
	out = strings.ReplaceAll(out, home+"/", "<home>/")
	out = strings.ReplaceAll(out, home, "<home>")
	return out
}

var genericHomeDir = regexp.MustCompile(`(?i)(?:file://)?/(?:home|users)/[^/\\]+(?:/\.vrooli)?/`)

func redactGenericHomeDirs(value string) string {
	return genericHomeDir.ReplaceAllStringFunc(value, func(match string) string {
		if strings.HasPrefix(strings.ToLower(match), "file://") {
			return "file://<home>/"
		}
		return "<home>/"
	})
}

func replaceStandalone(value, needle, replacement string, caseInsensitive bool) string {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return value
	}
	prefix := ""
	if caseInsensitive {
		prefix = "(?i)"
	}
	re := regexp.MustCompile(prefix + `(^|[^A-Za-z0-9._-])(` + regexp.QuoteMeta(needle) + `)([^A-Za-z0-9._-]|$)`)
	for {
		next := re.ReplaceAllString(value, `${1}`+replacement+`${3}`)
		if next == value {
			return next
		}
		value = next
	}
}

func (r *Redactor) normalize() {
	r.RepoRoots = uniqueCleanAbs(r.RepoRoots)
	r.HomeDirs = uniqueCleanAbs(r.HomeDirs)
	r.Usernames = uniqueTrimmed(r.Usernames)
	r.IdentityTerms = uniqueTrimmed(r.IdentityTerms)
	sort.Slice(r.RepoRoots, func(i, j int) bool { return len(r.RepoRoots[i]) > len(r.RepoRoots[j]) })
	sort.Slice(r.HomeDirs, func(i, j int) bool { return len(r.HomeDirs[i]) > len(r.HomeDirs[j]) })
	sort.Slice(r.Usernames, func(i, j int) bool { return len(r.Usernames[i]) > len(r.Usernames[j]) })
	sort.Slice(r.IdentityTerms, func(i, j int) bool { return len(r.IdentityTerms[i]) > len(r.IdentityTerms[j]) })
}

func uniqueCleanAbs(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = cleanAbs(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func uniqueTrimmed(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func cleanAbs(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}
