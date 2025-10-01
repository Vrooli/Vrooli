package main

import (
	"encoding/base64"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

var whitespaceSequence = regexp.MustCompile("-+")

// decodeBase64Payload decodes a base64-encoded payload
func decodeBase64Payload(data string) ([]byte, error) {
	trimmed := strings.TrimSpace(data)
	if trimmed == "" {
		return nil, fmt.Errorf("empty payload")
	}
	if idx := strings.Index(trimmed, ","); idx != -1 && strings.Contains(trimmed[:idx], "base64") {
		trimmed = trimmed[idx+1:]
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return base64.StdEncoding.DecodeString(trimmed)
}

// extensionFromContentType returns file extension for a content type
func extensionFromContentType(contentType string) string {
	switch strings.ToLower(strings.TrimSpace(contentType)) {
	case "image/png":
		return "png"
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "application/json":
		return "json"
	case "text/plain":
		return "txt"
	default:
		return ""
	}
}

// looksLikeJSON checks if a string looks like JSON
func looksLikeJSON(payload string) bool {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return false
	}
	first := trimmed[0]
	last := trimmed[len(trimmed)-1]
	return (first == '{' && last == '}') || (first == '[' && last == ']')
}

// sanitizeFileComponent creates a safe filename component
func sanitizeFileComponent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	replaced := strings.ReplaceAll(trimmed, "\\", "-")
	replaced = strings.ReplaceAll(replaced, "/", "-")
	replaced = strings.ReplaceAll(replaced, " ", "-")
	var builder strings.Builder
	for _, r := range replaced {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case r == '-' || r == '_' || r == '.':
			builder.WriteRune(r)
		default:
			builder.WriteRune('-')
		}
	}
	sanitized := builder.String()
	sanitized = whitespaceSequence.ReplaceAllString(sanitized, "-")
	sanitized = strings.Trim(sanitized, "-_")
	sanitized = strings.TrimLeft(sanitized, ".")
	return sanitized
}

// ensureUniqueFilename ensures a filename is unique by appending a counter if needed
func ensureUniqueFilename(filename string, used map[string]int) string {
	if used == nil {
		used = make(map[string]int)
	}
	if filename == "" {
		filename = "artifact"
	}
	if used[filename] == 0 {
		used[filename] = 1
		return filename
	}
	base := filename
	ext := ""
	if strings.Contains(filename, ".") {
		ext = filepath.Ext(filename)
		base = strings.TrimSuffix(filename, ext)
	}
	candidate := filename
	counter := used[filename]
	for {
		candidate = fmt.Sprintf("%s-%d%s", base, counter, ext)
		if used[candidate] == 0 {
			used[filename] = counter + 1
			used[candidate] = 1
			return candidate
		}
		counter++
	}
}

// normalizeAttachmentPath normalizes an attachment path
func normalizeAttachmentPath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	replaced := strings.ReplaceAll(trimmed, "\\", "/")
	cleaned := path.Clean("/" + replaced)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return ""
	}
	return cleaned
}

// resolveAttachmentPath resolves and validates an attachment path
func resolveAttachmentPath(issueDir, relativeRef string) (string, error) {
	normalized := normalizeAttachmentPath(relativeRef)
	if normalized == "" {
		return "", fmt.Errorf("invalid attachment path")
	}
	fsRelative := filepath.FromSlash(normalized)
	full := filepath.Join(issueDir, fsRelative)
	relative, err := filepath.Rel(issueDir, full)
	if err != nil {
		return "", err
	}
	if relative == "." || strings.HasPrefix(relative, "..") || strings.HasPrefix(filepath.ToSlash(relative), "../") {
		return "", fmt.Errorf("invalid attachment path")
	}
	return full, nil
}

// decodeArtifactContent decodes artifact content based on encoding
func decodeArtifactContent(payload ArtifactPayload) ([]byte, error) {
	encoding := strings.ToLower(strings.TrimSpace(payload.Encoding))
	switch encoding {
	case "", "plain", "text", "markdown", "json":
		return []byte(payload.Content), nil
	case "base64":
		return decodeBase64Payload(payload.Content)
	default:
		return nil, fmt.Errorf("unsupported artifact encoding: %s", payload.Encoding)
	}
}

// determineContentType determines content type for an artifact
func determineContentType(payload ArtifactPayload, defaultForPlain string) string {
	contentType := strings.TrimSpace(payload.ContentType)
	if contentType != "" {
		return contentType
	}
	encoding := strings.ToLower(strings.TrimSpace(payload.Encoding))
	switch encoding {
	case "json":
		return "application/json"
	case "", "plain", "text", "markdown":
		if defaultForPlain != "" {
			return defaultForPlain
		}
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

// persistArtifacts saves artifacts to disk and returns attachment metadata
func (s *Server) persistArtifacts(issueDir string, payloads []ArtifactPayload) ([]Attachment, error) {
	if len(payloads) == 0 {
		return nil, nil
	}
	trimmed := make([]ArtifactPayload, 0, len(payloads))
	for _, payload := range payloads {
		if strings.TrimSpace(payload.Content) == "" {
			continue
		}
		trimmed = append(trimmed, payload)
	}
	if len(trimmed) == 0 {
		return nil, nil
	}
	destination := filepath.Join(issueDir, artifactsDirName)
	if err := os.MkdirAll(destination, 0755); err != nil {
		return nil, err
	}
	usedFilenames := make(map[string]int)
	var attachments []Attachment
	for idx, artifact := range trimmed {
		data, err := decodeArtifactContent(artifact)
		if err != nil {
			return nil, fmt.Errorf("artifact %d: %w", idx, err)
		}
		displayName := strings.TrimSpace(artifact.Name)
		if displayName == "" {
			displayName = fmt.Sprintf("Artifact %d", idx+1)
		}
		baseComponent := sanitizeFileComponent(displayName)
		if baseComponent == "" {
			if artifact.Category != "" {
				baseComponent = sanitizeFileComponent(artifact.Category)
			}
			if baseComponent == "" {
				baseComponent = fmt.Sprintf("artifact-%d", idx+1)
			}
		}
		ext := ""
		if dot := strings.LastIndex(baseComponent, "."); dot != -1 {
			ext = baseComponent[dot+1:]
		}
		filename := baseComponent
		if ext == "" {
			guessed := extensionFromContentType(artifact.ContentType)
			if guessed == "" {
				encoding := strings.ToLower(strings.TrimSpace(artifact.Encoding))
				if encoding == "json" && !strings.HasSuffix(filename, ".json") {
					guessed = "json"
				}
			}
			if guessed != "" && !strings.HasSuffix(strings.ToLower(filename), "."+guessed) {
				filename = fmt.Sprintf("%s.%s", filename, guessed)
			}
		}
		filename = ensureUniqueFilename(filename, usedFilenames)
		contentType := determineContentType(artifact, "")
		if contentType == "" {
			switch strings.ToLower(filepath.Ext(filename)) {
			case ".txt":
				contentType = "text/plain"
			case ".json":
				contentType = "application/json"
			case ".png":
				contentType = "image/png"
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".webp":
				contentType = "image/webp"
			default:
				contentType = "application/octet-stream"
			}
		}
		relativePath := path.Join(artifactsDirName, filename)
		filePath := filepath.Join(destination, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return nil, fmt.Errorf("artifact %q: %w", filename, err)
		}
		attachments = append(attachments, Attachment{
			Name:     displayName,
			Type:     contentType,
			Path:     relativePath,
			Size:     int64(len(data)),
			Category: strings.TrimSpace(artifact.Category),
		})
	}
	return attachments, nil
}

// mergeCreateArtifacts merges all artifact sources from a create request
func mergeCreateArtifacts(req *CreateIssueRequest) []ArtifactPayload {
	var artifacts []ArtifactPayload
	artifacts = append(artifacts, req.Artifacts...)
	if strings.TrimSpace(req.AppLogs) != "" {
		artifacts = append(artifacts, ArtifactPayload{
			Name:        "Application Logs",
			Category:    "logs",
			Content:     req.AppLogs,
			Encoding:    "plain",
			ContentType: "text/plain",
		})
	}
	if strings.TrimSpace(req.ConsoleLogs) != "" {
		artifacts = append(artifacts, ArtifactPayload{
			Name:        "Console Logs",
			Category:    "console",
			Content:     req.ConsoleLogs,
			Encoding:    "plain",
			ContentType: "text/plain",
		})
	}
	if strings.TrimSpace(req.NetworkLogs) != "" {
		contentType := "application/json"
		if !looksLikeJSON(req.NetworkLogs) {
			contentType = "text/plain"
		}
		artifacts = append(artifacts, ArtifactPayload{
			Name:        "Network Requests",
			Category:    "network",
			Content:     req.NetworkLogs,
			Encoding:    "plain",
			ContentType: contentType,
		})
	}
	if strings.TrimSpace(req.ScreenshotData) != "" {
		contentType := strings.TrimSpace(req.ScreenshotContentType)
		if contentType == "" {
			contentType = "image/png"
		}
		artifacts = append(artifacts, ArtifactPayload{
			Name:        fallbackScreenshotName(req.ScreenshotFilename),
			Category:    "screenshot",
			Content:     req.ScreenshotData,
			Encoding:    "base64",
			ContentType: contentType,
		})
	}
	for _, attachment := range req.Attachments {
		content := strings.TrimSpace(attachment.Content)
		name := strings.TrimSpace(attachment.Name)
		if content == "" || name == "" {
			continue
		}
		encoding := strings.TrimSpace(attachment.Encoding)
		if encoding == "" {
			encoding = "plain"
		}
		artifacts = append(artifacts, ArtifactPayload{
			Name:        name,
			Category:    "attachment",
			Content:     content,
			Encoding:    encoding,
			ContentType: strings.TrimSpace(attachment.ContentType),
		})
	}
	return artifacts
}

// fallbackScreenshotName provides a default name for screenshots
func fallbackScreenshotName(filename string) string {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return "Screenshot"
	}
	return trimmed
}
