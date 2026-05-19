package dochealth

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/vrooli/api-core/markedrefs"
)

// DOC: docs/internal/SEAMS.md#dochealth
// inspectMarkdownFile parses a single markdown file. It returns content
// findings (markdown/mermaid/absolute paths), the link targets discovered in
// the file, and per-file metric counts. Filesystem access happens through
// os.Open here; the file path is resolved by the caller relative to the
// scenario root.

var (
	markdownLinkPattern   = regexp.MustCompile(`!?\[[^\]]*\]\(([^)]+)\)`)
	codeFencePattern      = regexp.MustCompile("^(```|~~~)([a-zA-Z0-9_-]+)?")
	absUnixPathPattern    = regexp.MustCompile(`/(Users|home|var|etc|opt|srv|private|Volumes)/`)
	absWindowsPathPattern = regexp.MustCompile(`^[A-Za-z]:\\`)
)

type fileMetrics struct {
	MermaidValidated int
	MermaidFailures  int
	MarkdownWarnings int
	MarkdownFailures int
	AbsoluteFailures int
	AbsoluteHits     int
}

type linkTarget struct {
	File     string
	Line     int
	Dest     string
	isImage  bool
	location string
}

func inspectMarkdownFile(path string, cfg effective) ([]Finding, fileMetrics, []linkTarget, []string) {
	var (
		findings []Finding
		summary  fileMetrics
		links    []linkTarget
		ioErrors []string
	)

	f, err := os.Open(path)
	if err != nil {
		ioErrors = append(ioErrors, fmt.Sprintf("cannot read %s: %v", path, err))
		return findings, summary, links, ioErrors
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	inFence := false
	fenceLang := ""
	fenceMarker := ""
	var mermaidBuf []string
	var mermaidStart int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trim := strings.TrimSpace(line)

		if matches := codeFencePattern.FindStringSubmatch(trim); len(matches) > 0 {
			marker := matches[1]
			lang := strings.TrimSpace(matches[2])
			if !inFence {
				inFence = true
				fenceMarker = marker
				fenceLang = lang
				mermaidBuf = mermaidBuf[:0]
				mermaidStart = lineNum
			} else if marker == fenceMarker {
				if fenceLang == "mermaid" || fenceLang == "mermaidjs" {
					validateMermaidBlock(path, mermaidStart, strings.Join(mermaidBuf, "\n"), cfg.mermaidStrict, &findings, &summary)
				}
				inFence = false
				fenceLang = ""
				fenceMarker = ""
				mermaidBuf = mermaidBuf[:0]
			}
			continue
		}

		if inFence {
			if fenceLang == "mermaid" || fenceLang == "mermaidjs" {
				mermaidBuf = append(mermaidBuf, line)
			}
			continue
		}

		for _, match := range markdownLinkPattern.FindAllStringSubmatchIndex(line, -1) {
			if len(match) < 4 {
				continue
			}
			start, end := match[2], match[3]
			dest := strings.TrimSpace(line[start:end])
			if dest == "" {
				continue
			}
			dest = strings.Trim(dest, "<>")
			isImage := line[match[0]] == '!'
			links = append(links, linkTarget{
				File:     path,
				Line:     lineNum,
				Dest:     dest,
				isImage:  isImage,
				location: fmt.Sprintf("%s:%d", path, lineNum),
			})
		}

		scanAbsolutePath(path, line, trim, lineNum, cfg, &findings, &summary)
	}

	if err := scanner.Err(); err != nil {
		ioErrors = append(ioErrors, fmt.Sprintf("failed to read %s: %v", path, err))
	}

	if inFence {
		summary.MarkdownFailures++
		findings = append(findings, Finding{
			Code:     "markdown_unclosed_fence",
			Severity: SeverityFailure,
			Message:  fmt.Sprintf("%s:%d code fence not closed", path, mermaidStart),
			Path:     path,
			Line:     mermaidStart,
		})
	}

	return findings, summary, links, ioErrors
}

func lineForAbsolutePathScan(line string, lineNum int) string {
	scanLine := line
	for _, ref := range markedrefs.ParseInlineCode(line, lineNum) {
		if markedrefs.UnknownMarker(ref) {
			continue
		}
		if markedrefs.RequiresExistence(ref) {
			continue
		}
		scanLine = strings.Replace(scanLine, ref.Raw, "", 1)
	}
	return scanLine
}
