package sketch

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"syscall"
	"unicode/utf8"
)

const maxDesignSourceBytes = 128 * 1024

type DesignSourceSnapshot struct {
	Path        string
	ContentHash string
	Content     string
}

func normalizeDesignSource(source string) (string, error) {
	source = strings.TrimPrefix(strings.TrimSpace(source), "path:")
	if source == "" {
		return "", nil
	}
	if len(source) > 1000 || strings.ContainsAny(source, "\\\r\n\x00:") || path.IsAbs(source) {
		return "", fmt.Errorf("design source must be a bounded relative file reference")
	}
	for _, part := range strings.Split(source, "/") {
		if part == ".." {
			return "", fmt.Errorf("design source must remain within the scenario")
		}
	}
	source = path.Clean(source)
	if source == "." {
		return "", fmt.Errorf("design source must name a file")
	}
	return source, nil
}

// ReadDesignSource returns exact bounded text. It does not interpret instructions
// in the document or certify that a candidate conforms to them.
func (s *Store) ReadDesignSource(scenario, source string) (*DesignSourceSnapshot, error) {
	source, err := normalizeDesignSource(source)
	if err != nil {
		return nil, err
	}
	if source == "" {
		return nil, nil
	}
	if !safeSegment.MatchString(scenario) {
		return nil, fmt.Errorf("invalid scenario identity")
	}
	root, err := os.OpenRoot(s.repoRoot)
	if err != nil {
		return nil, err
	}
	for _, part := range []string{"scenarios", scenario} {
		info, err := root.Lstat(part)
		if err != nil {
			root.Close()
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			root.Close()
			return nil, fmt.Errorf("rejected design source: symlink %s", part)
		}
		child, err := root.OpenRoot(part)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = child
	}
	defer root.Close()
	// Nonblocking open prevents a named pipe substituted for a file from
	// hanging the request. Root confines symlinks to this scenario.
	f, err := root.OpenFile(source, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, fmt.Errorf("read design source %s: %w", source, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("design source must be a regular text file")
	}
	data, err := io.ReadAll(io.LimitReader(f, maxDesignSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDesignSourceBytes {
		return nil, fmt.Errorf("design source exceeds %d bytes", maxDesignSourceBytes)
	}
	if !utf8.Valid(data) || strings.ContainsRune(string(data), 0) || strings.TrimSpace(string(data)) == "" {
		return nil, fmt.Errorf("design source must contain nonempty UTF-8 text")
	}
	hash := sha256.Sum256(data)
	return &DesignSourceSnapshot{Path: source, ContentHash: hex.EncodeToString(hash[:]), Content: string(data)}, nil
}
