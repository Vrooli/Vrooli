package handlers

import (
	"agent-inbox/domain"
	"agent-inbox/integrations"
	"agent-inbox/services"
	"bufio"
	"encoding/json"
	"log"
	"strings"
)

// parseStreamingChunks reads and accumulates SSE chunks into a CompletionResult.
//
// # SSE Parsing
//
// Reads SSE-formatted data from the body:
//   - Lines starting with "data: " contain JSON payloads
//   - "data: [DONE]" signals end of stream
//   - Other lines (comments, empty) are ignored
//
// # Accumulation
//
// Uses StreamingAccumulator to build the complete response:
//   - Content is concatenated from all chunks
//   - Tool calls are assembled from partial deltas
//   - Images are collected from multimodal responses
//   - Usage data is captured from the final chunk
//
// # Multimodal Support
//
// Content can be either:
//   - string: Regular text content
//   - []interface{}: Array of content parts (text + images)
//
// Images can appear in:
//   - delta.Content as image_url parts
//   - delta.Images array (legacy format)
//
// # Buffer Sizing
//
// Uses MaxSSEScanTokenSize for buffer to handle large payloads
// like base64-encoded generated images.
func parseStreamingChunks(body interface{ Read([]byte) (int, error) }, sw *StreamWriter) *domain.CompletionResult {
	acc := domain.NewStreamingAccumulator()
	scanner := bufio.NewScanner(body)
	// Increase buffer size to handle large SSE chunks (e.g., generated images as base64)
	buf := make([]byte, services.MaxSSEScanTokenSize)
	scanner.Buffer(buf, services.MaxSSEScanTokenSize)

	lineCount := 0
	dataLineCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineCount++
		if lineCount <= 5 {
			log.Printf("[DEBUG] SSE line %d: %s", lineCount, line[:min(len(line), 100)])
		}
		if strings.Contains(strings.ToLower(line), "data") {
			dataLineCount++
			log.Printf("[DEBUG] SSE data line %d: %s", dataLineCount, line[:min(len(line), 200)])
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		if dataLineCount <= 3 {
			log.Printf("[DEBUG] SSE raw chunk %d: %s", dataLineCount, data[:min(500, len(data))])
		}

		var chunk integrations.OpenRouterResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Printf("[DEBUG] SSE parse error: %v, data: %s", err, data[:min(100, len(data))])
			continue
		}

		log.Printf("[DEBUG] SSE chunk: id=%s, choices=%d", chunk.ID, len(chunk.Choices))
		if len(chunk.Choices) > 0 {
			choice := chunk.Choices[0]
			contentBytes, _ := json.Marshal(choice.Delta.Content)
			log.Printf("[DEBUG] SSE choice[0]: finish_reason=%s, delta.content=%s, delta.images=%d",
				choice.FinishReason, string(contentBytes)[:min(200, len(contentBytes))], len(choice.Delta.Images))
		}
		acc.SetResponseID(chunk.ID)

		// Capture usage data if present (typically in final chunk)
		if chunk.Usage.PromptTokens > 0 || chunk.Usage.CompletionTokens > 0 {
			acc.SetUsage(chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens, chunk.Usage.TotalTokens)
		}

		if len(chunk.Choices) > 0 {
			processStreamingChoice(chunk.Choices[0], acc, sw)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[DEBUG] SSE scanner error: %v", err)
	}

	result := acc.ToResult()
	log.Printf("[DEBUG] SSE parsing complete: %d total lines, %d data lines, content length=%d, tool_calls=%d, images=%d",
		lineCount, dataLineCount, len(result.Content), len(result.ToolCalls), len(result.Images))
	return result
}

// processStreamingChoice processes a single choice from a streaming chunk.
func processStreamingChoice(choice integrations.OpenRouterChoice, acc *domain.StreamingAccumulator, sw *StreamWriter) {
	log.Printf("[DEBUG] processStreamingChoice: delta.Content type=%T", choice.Delta.Content)

	switch c := choice.Delta.Content.(type) {
	case string:
		if c != "" {
			sw.WriteContentChunk(c)
			acc.AppendContent(c)
		}
	case []interface{}:
		// Multimodal streaming - extract text and images from content parts
		log.Printf("[DEBUG] Streaming multimodal content with %d parts", len(c))
		processMultimodalStreamingParts(c, acc, sw)
	}

	// Handle generated images in legacy Images field
	for _, img := range choice.Delta.Images {
		if img.ImageURL != nil && img.ImageURL.URL != "" {
			log.Printf("[DEBUG] Received generated image in delta.Images")
			acc.AppendImage(img.ImageURL.URL)
			sw.WriteImageGenerated(img.ImageURL.URL)
		}
	}

	// Accumulate tool calls
	for _, tc := range choice.Delta.ToolCalls {
		acc.AppendToolCallDelta(tc)
	}

	acc.SetFinishReason(choice.FinishReason)
}

// processMultimodalStreamingParts extracts text and images from multimodal content parts
// in a streaming response.
func processMultimodalStreamingParts(parts []interface{}, acc *domain.StreamingAccumulator, sw *StreamWriter) {
	for _, part := range parts {
		partMap, ok := part.(map[string]interface{})
		if !ok {
			continue
		}
		partType, _ := partMap["type"].(string)

		switch partType {
		case "text":
			if text, ok := partMap["text"].(string); ok && text != "" {
				sw.WriteContentChunk(text)
				acc.AppendContent(text)
			}
		case "image_url":
			if imgURL, ok := partMap["image_url"].(map[string]interface{}); ok {
				if url, ok := imgURL["url"].(string); ok && url != "" {
					log.Printf("[DEBUG] Streaming: found image in content part")
					acc.AppendImage(url)
					sw.WriteImageGenerated(url)
				}
			}
		}
	}
}
