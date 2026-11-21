package conversations

// Decision-packet attachments: turning --attach / --explainer arguments into
// AskAttachment messages. Reading, hashing and naming happen here; the network
// upload lives in upload.go. See scenarios/notification-hub/docs/concepts/DECISION_PACKETS.md.

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/vrooli/vrooli/packages/proto/gen/go/notification-hub/v1/conversations"
)

// Limits the Hub documents. The CLI checks the byte limits before sending so a
// refused upload never surprises the sender; counts are left to the server.
const (
	maxAttachmentFileBytes  int64 = 64 << 20
	maxAttachmentTotalBytes int64 = 256 << 20
	maxFolderFiles                = 500
	maxAttachmentNameLength       = 96
)

// ignoredFolderDirs and ignoredFolderFiles are skipped when a folder is
// attached: version-control metadata, dependency trees, caches and OS litter
// that are never what an operator wants to judge. Only folder contents are
// filtered; naming a path explicitly always attaches it.
var (
	ignoredFolderDirs = map[string]bool{
		".git": true, ".hg": true, ".svn": true, "node_modules": true,
		"__pycache__": true, ".venv": true, ".mypy_cache": true,
		".pytest_cache": true, ".cache": true, ".idea": true,
	}
	ignoredFolderFiles = map[string]bool{".DS_Store": true, "Thumbs.db": true}
)

// localFile is one regular file the packet will carry.
type localFile struct {
	// Path is where the CLI reads the bytes from.
	Path string
	// Rel is the slash-separated path inside an attached folder; empty for a
	// stand-alone file.
	Rel       string
	Size      int64
	SHA256    string
	MediaType string
}

// packetAttachment is one attachment before it is sent.
type packetAttachment struct {
	Name       string
	Label      string
	Role       v1.AskAttachment_Role
	Kind       v1.AskAttachment_Kind
	SourcePath string
	File       *localFile  // KIND_FILE
	Entries    []localFile // KIND_FOLDER
	Truncated  bool
}

// files lists every file whose bytes the Hub must hold for this attachment.
func (a packetAttachment) files() []localFile {
	if a.Kind == v1.AskAttachment_KIND_FOLDER {
		return a.Entries
	}
	if a.File == nil {
		return nil
	}
	return []localFile{*a.File}
}

func (a packetAttachment) proto(sourceHost string) *v1.AskAttachment {
	out := &v1.AskAttachment{
		Name:       a.Name,
		Label:      a.Label,
		Kind:       a.Kind,
		Role:       a.Role,
		SourcePath: a.SourcePath,
		SourceHost: sourceHost,
		Truncated:  a.Truncated,
	}
	if a.File != nil {
		out.Sha256, out.SizeBytes, out.MediaType = a.File.SHA256, a.File.Size, a.File.MediaType
	}
	for _, entry := range a.Entries {
		out.Entries = append(out.Entries, &v1.AskAttachmentEntry{
			Path: entry.Rel, Sha256: entry.SHA256, SizeBytes: entry.Size, MediaType: entry.MediaType,
		})
	}
	return out
}

// attachSpec is one parsed --attach value.
type attachSpec struct {
	Path  string
	Label string
}

// parseAttachSpec splits PATH[=LABEL]. A value that names an existing path as a
// whole is that path (so a path containing "=" works); otherwise the text after
// the last "=" is the label.
func parseAttachSpec(value string) (attachSpec, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return attachSpec{}, errors.New("--attach needs a path (PATH or PATH=LABEL)")
	}
	if _, err := os.Stat(value); err == nil {
		return attachSpec{Path: value}, nil
	}
	path, label := value, ""
	if i := strings.LastIndex(value, "="); i >= 0 {
		path, label = strings.TrimSpace(value[:i]), strings.TrimSpace(value[i+1:])
	}
	if path == "" {
		return attachSpec{}, fmt.Errorf("--attach %q needs a path before the label", value)
	}
	return attachSpec{Path: path, Label: label}, nil
}

// packet collects attachments, names them uniquely, and remembers what to tell
// the sender about parts the CLI had to trim.
type packet struct {
	attachments []packetAttachment
	names       map[string]bool
	// total is the bytes of every attachment kept so far; later attachments
	// must fit what is left of maxAttachmentTotalBytes.
	total int64
	// Warnings are local notices (printed before the ask is sent).
	Warnings []string
}

func newPacket() *packet { return &packet{names: map[string]bool{}} }

// flagName is how errors name the command-line flag an attachment came from.
func flagName(role v1.AskAttachment_Role) string {
	if role == v1.AskAttachment_ROLE_EXPLAINER {
		return "--explainer"
	}
	return "--attach"
}

// addAttachment reads one file or folder and appends it. Explainers must be a
// single file. An error means this attachment cannot join the packet; the
// caller reports it and carries on without it.
func (p *packet) addAttachment(spec attachSpec, role v1.AskAttachment_Role) error {
	flag := flagName(role)
	abs, err := filepath.Abs(spec.Path)
	if err != nil {
		return fmt.Errorf("%s %q: %w", flag, spec.Path, err)
	}
	info, err := os.Stat(abs) // follows a top-level symlink: the sender named it
	if err != nil {
		return fmt.Errorf("%s %q: %w", flag, spec.Path, unwrapPathError(err))
	}
	att := packetAttachment{Role: role, SourcePath: abs, Label: spec.Label}
	switch {
	case info.IsDir():
		if role == v1.AskAttachment_ROLE_EXPLAINER {
			return fmt.Errorf("%s %q is a folder; give one HTML file", flag, spec.Path)
		}
		att.Kind = v1.AskAttachment_KIND_FOLDER
		if err := p.readFolder(&att, abs); err != nil {
			return fmt.Errorf("%s %q: %w", flag, spec.Path, err)
		}
	case info.Mode().IsRegular():
		att.Kind = v1.AskAttachment_KIND_FILE
		file, err := describeFile(abs, p.budget())
		if err != nil {
			return fmt.Errorf("%s %q: %w", flag, spec.Path, unwrapPathError(err))
		}
		att.File = &file
	default:
		return fmt.Errorf("%s %q is not a regular file or folder", flag, spec.Path)
	}
	for _, file := range att.files() {
		p.total += file.Size
	}
	att.Name = p.uniqueName(filepath.Base(abs), att.Kind == v1.AskAttachment_KIND_FOLDER)
	p.attachments = append(p.attachments, att)
	return nil
}

// budget is how many bytes one more attachment may add to the packet.
func (p *packet) budget() int64 { return maxAttachmentTotalBytes - p.total }

// readFolder lists the files of a folder in path order, hashing each. Symlinks
// are never followed, junk (see ignoredFolderDirs) is skipped, unreadable files
// are skipped with a warning, and at most maxFolderFiles are kept.
func (p *packet) readFolder(att *packetAttachment, abs string) error {
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return unwrapPathError(err)
	}
	var added int64
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root {
				return err
			}
			p.warnf("Left out %s inside %s: %v.", relSlash(root, path), abs, unwrapPathError(err))
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root {
			return nil
		}
		switch {
		case d.IsDir():
			if ignoredFolderDirs[d.Name()] {
				return fs.SkipDir
			}
		case d.Type()&fs.ModeSymlink != 0:
			// Never followed: a link could lead outside the folder.
		case !d.Type().IsRegular() || ignoredFolderFiles[d.Name()]:
		default:
			if len(att.Entries) >= maxFolderFiles {
				att.Truncated = true
				return fs.SkipAll
			}
			file, err := describeFile(path, p.budget()-added)
			var tooLarge *fileTooLargeError
			switch {
			case errors.As(err, &tooLarge) && tooLarge.overTotal:
				return err // the folder as a whole does not fit; the Hub would drop it whole
			case err != nil:
				p.warnf("Left out %s inside %s: %v.", relSlash(root, path), abs, unwrapPathError(err))
				return nil
			}
			added += file.Size
			file.Rel = relSlash(root, path)
			att.Entries = append(att.Entries, file)
		}
		return nil
	})
	if walkErr != nil {
		return unwrapPathError(walkErr)
	}
	if len(att.Entries) == 0 {
		return errors.New("the folder has no files to attach")
	}
	if att.Truncated {
		p.warnf("Folder %s has more than %d files; only the first %d (in path order) are attached.", abs, maxFolderFiles, maxFolderFiles)
	}
	return nil
}

func (p *packet) warnf(format string, args ...any) {
	p.Warnings = append(p.Warnings, fmt.Sprintf(format, args...))
}

// uniqueName derives a Hub-valid attachment name from a base name and makes it
// unique within the packet: a.png, a-2.png, a-3.png.
func (p *packet) uniqueName(base string, folder bool) string {
	stem, ext := sanitizeAttachmentName(base, folder)
	name := stem + ext
	for n := 2; p.names[name]; n++ {
		suffix := "-" + strconv.Itoa(n)
		name = truncateName(stem, len(suffix)+len(ext)) + suffix + ext
	}
	p.names[name] = true
	return name
}

// sanitizeAttachmentName maps a base name onto the Hub's name rule (1-96 of
// A-Z a-z 0-9 . _ -, first a letter or digit) and returns the stem and the
// extension separately so de-duplication can number the stem.
func sanitizeAttachmentName(base string, folder bool) (stem, ext string) {
	var b strings.Builder
	lastDash := false
	for _, r := range base {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '_'
		switch {
		case ok:
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	clean := strings.TrimLeft(b.String(), ".-_")
	if !folder {
		ext = filepath.Ext(clean)
		if len(ext) > 16 {
			ext = ""
		}
	}
	stem = strings.TrimRight(strings.TrimSuffix(clean, ext), "-.")
	if stem == "" {
		stem = "attachment"
	}
	return truncateName(stem, len(ext)), ext
}

// truncateName cuts stem so that stem plus reserved extra bytes fits the limit.
func truncateName(stem string, reserved int) string {
	if limit := maxAttachmentNameLength - reserved; len(stem) > limit {
		return strings.TrimRight(stem[:limit], "-.")
	}
	return stem
}

// fileTooLargeError is a file that does not fit. It is detected from the file's
// size before a byte is hashed.
type fileTooLargeError struct {
	size int64
	// overTotal means the file fits the per-file limit but not what remains of
	// the packet's total budget.
	overTotal bool
}

func (e *fileTooLargeError) Error() string {
	if e.overTotal {
		return fmt.Sprintf("it adds %s, over the %s total limit for one ask", formatBytes(e.size), formatBytes(maxAttachmentTotalBytes))
	}
	return fmt.Sprintf("the file is %s, over the %s per-file limit", formatBytes(e.size), formatBytes(maxAttachmentFileBytes))
}

// describeFile hashes a file in one pass and decides its media type. A file
// over the per-file limit, or over the budget left in the packet's total, is
// refused from its size alone.
func describeFile(path string, budget int64) (localFile, error) {
	info, err := os.Stat(path)
	if err != nil {
		return localFile{}, err
	}
	switch size := info.Size(); {
	case size > maxAttachmentFileBytes:
		return localFile{}, &fileTooLargeError{size: size}
	case size > budget:
		return localFile{}, &fileTooLargeError{size: size, overTotal: true}
	}
	f, err := os.Open(path)
	if err != nil {
		return localFile{}, err
	}
	defer f.Close()
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return localFile{}, err
	}
	head = head[:n]
	hash := sha256.New()
	hash.Write(head)
	rest, err := io.Copy(hash, f)
	if err != nil {
		return localFile{}, err
	}
	return localFile{
		Path:      path,
		Size:      int64(n) + rest,
		SHA256:    hex.EncodeToString(hash.Sum(nil)),
		MediaType: mediaTypeFor(path, head),
	}, nil
}

// extensionMediaTypes covers what operators attach most that Go's built-in
// table or a host's mime.types may lack or disagree on. It wins over both so
// the same file gets the same type on every host.
var extensionMediaTypes = map[string]string{
	".md": "text/markdown", ".markdown": "text/markdown", ".txt": "text/plain",
	".log": "text/plain", ".csv": "text/csv", ".yaml": "application/yaml",
	".yml": "application/yaml", ".toml": "application/toml",
	".html": "text/html", ".htm": "text/html", ".json": "application/json",
	".svg": "image/svg+xml", ".png": "image/png", ".jpg": "image/jpeg",
	".jpeg": "image/jpeg", ".gif": "image/gif", ".webp": "image/webp",
	".avif": "image/avif", ".pdf": "application/pdf",
	".mp4": "video/mp4", ".webm": "video/webm", ".mov": "video/quicktime",
	".mp3": "audio/mpeg", ".wav": "audio/wav", ".ogg": "audio/ogg",
	".glb": "model/gltf-binary", ".gltf": "model/gltf+json",
	".obj": "model/obj", ".stl": "model/stl",
}

// mediaTypeFor picks a bare media type from the extension, then the system
// table, then a sniff of the first bytes, else application/octet-stream.
func mediaTypeFor(path string, head []byte) string {
	ext := strings.ToLower(filepath.Ext(path))
	if known, ok := extensionMediaTypes[ext]; ok {
		return known
	}
	if ext != "" {
		if byExt := mime.TypeByExtension(ext); byExt != "" {
			return bareMediaType(byExt)
		}
	}
	return bareMediaType(http.DetectContentType(head))
}

func bareMediaType(value string) string {
	if parsed, _, err := mime.ParseMediaType(value); err == nil {
		return parsed
	}
	return "application/octet-stream"
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// resolveRef maps an attachment reference written in a form file to the name
// the Hub uses. A reference is an attachment name, name/entry-path inside a
// folder, or a local path of an attached file or folder entry.
func (p *packet) resolveRef(ref string) (string, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	for _, att := range p.attachments {
		if ref == att.Name {
			return att.Name, true
		}
	}
	for _, att := range p.attachments {
		if att.Kind != v1.AskAttachment_KIND_FOLDER {
			continue
		}
		if entry, ok := strings.CutPrefix(ref, att.Name+"/"); ok && att.hasEntry(entry) {
			return ref, true
		}
	}
	abs, err := filepath.Abs(ref)
	if err != nil {
		return "", false
	}
	for _, att := range p.attachments {
		if abs == att.SourcePath {
			return att.Name, true
		}
		if entry, ok := strings.CutPrefix(filepath.ToSlash(abs), filepath.ToSlash(att.SourcePath)+"/"); ok && att.Kind == v1.AskAttachment_KIND_FOLDER && att.hasEntry(entry) {
			return att.Name + "/" + entry, true
		}
	}
	return "", false
}

func (a packetAttachment) hasEntry(rel string) bool {
	for _, entry := range a.Entries {
		if entry.Rel == rel {
			return true
		}
	}
	return false
}

// attachmentRef is the name the Hub uses for an attachment a form file refers to.
// The reference is an attachment name, name/path inside a folder, or a local path
// of something this send attaches. A bare name that this send does not attach is
// passed on as written, because a revision's form may point at an attachment the
// open ask already holds; the Hub reports one it does not find. A local path that
// exists but is not attached is the sender's mistake, and the error says so.
func (p *packet) attachmentRef(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if name, ok := p.resolveRef(ref); ok {
		return name, nil
	}
	if _, err := os.Stat(ref); err == nil {
		return "", fmt.Errorf("%q is a local file that is not attached; add --attach %s, or name an attachment the ask already has", ref, ref)
	}
	return ref, nil
}

// sourceFile returns the stand-alone file a reference to an attachment of this
// send names, for the commands that must read it.
func (p *packet) sourceFile(ref string) (*localFile, bool) {
	name, ok := p.resolveRef(ref)
	if !ok {
		return nil, false
	}
	for _, att := range p.attachments {
		if att.Name == name && att.File != nil {
			return att.File, true
		}
		if entry, ok := strings.CutPrefix(name, att.Name+"/"); ok && att.Kind == v1.AskAttachment_KIND_FOLDER {
			for i := range att.Entries {
				if att.Entries[i].Rel == entry {
					return &att.Entries[i], true
				}
			}
		}
	}
	return nil, false
}

func relSlash(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

// unwrapPathError drops the repeated "open /path:" prefix so messages that
// already name the path read cleanly.
func unwrapPathError(err error) error {
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err
	}
	return err
}
