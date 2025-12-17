package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
	"strings"
)

func HashDocument(namespace, content string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(content, "\r\n", "\n"))
	sum := sha256.Sum256([]byte(strings.TrimSpace(namespace) + "\n" + normalized))
	return hex.EncodeToString(sum[:])
}

func ChunkText(text string, chunkSize, chunkOverlap int, maxChunks int) []string {
	runes := []rune(text)
	if len(runes) == 0 {
		return []string{}
	}
	if chunkSize <= 0 {
		return []string{strings.TrimSpace(text)}
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 4
	}
	step := chunkSize - chunkOverlap
	if step <= 0 {
		step = chunkSize
	}

	out := []string{}
	for start := 0; start < len(runes); start += step {
		end := start + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			out = append(out, chunk)
		}
		if end == len(runes) {
			break
		}
		if maxChunks > 0 && len(out) >= maxChunks {
			break
		}
	}
	return out
}

func RecordIDForChunk(namespace, documentID string, chunkIndex int, chunk string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(namespace) + "\n" + strings.TrimSpace(documentID) + "\n" + strconv.Itoa(chunkIndex) + "\n" + chunk))
	return hex.EncodeToString(sum[:])
}

