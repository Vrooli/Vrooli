package api

/*
Rule: Streaming Content-Type Headers
Description: Server-Sent Events and streaming responses must set appropriate Content-Type headers
Reason: Ensures browsers handle streaming connections correctly
Category: api
Severity: high
Standard: api-design-v1
Targets: api

IMPLEMENTATION APPROACH:
This rule detects streaming response patterns (SSE, chunked transfers, WebSocket upgrades)
and ensures they set proper Content-Type headers before starting the stream.

Key Detection Patterns:

1. Server-Sent Events (SSE):
   - Content-Type: text/event-stream
   - w.(http.Flusher).Flush() in loop
   - fmt.Fprintf(w, "data: %s\\n\\n", msg)
   - "event:", "data:", "id:", "retry:" patterns

2. Chunked Transfer:
   - Transfer-Encoding: chunked (auto-set by Go)
   - Repeated Flush() calls
   - Streaming large files/data

3. NDJSON (Newline Delimited JSON):
   - Content-Type: application/x-ndjson or application/jsonlines
   - Multiple JSON objects separated by newlines
   - Common for log streaming

4. Long polling:
   - Keep connection open
   - Periodic Flush()
   - May use various content types

Edge Cases to Handle:
- SSE with custom event types
- SSE with reconnection logic
- Multipart responses (file uploads)
- WebSocket upgrades (different protocol)
- gRPC streaming (different handling)
- HTTP/2 server push

Framework Patterns:
- Gin: c.Stream() for SSE
- Echo: c.Stream() for SSE
- Chi: Manual SSE with Flusher

Critical Requirements:
1. SSE MUST have Content-Type: text/event-stream
2. SSE MUST flush after each message
3. SSE MUST keep connection open
4. Headers MUST be set before first Flush()

Detection Strategy:
1. Look for http.Flusher type assertion
2. Detect Flush() calls in loops/goroutines
3. Check for SSE message format (data:, event:)
4. Verify Content-Type set before Flush()
5. Look for text/event-stream header

Lookback Window:
- 50 lines because streaming setup is complex:
  * Connection upgrade checks (5-10 lines)
  * Header setup (5-10 lines)
  * Flusher setup (5-10 lines)
  * Channel/goroutine setup (10-15 lines)
  * Error handling (5-10 lines)

Common False Positives to Avoid:
- Buffer flushing (not HTTP)
- File I/O flush operations
- Database transaction flush
- Comments about streaming
- Test code

Severity Considerations:
- HIGH severity because:
  * SSE completely fails without text/event-stream
  * Browsers won't establish EventSource connection
  * Critical for real-time features
  * No fallback behavior
  * Can't fix after first flush

Best Practices for SSE:
- Set headers in this order:
  1. Content-Type: text/event-stream
  2. Cache-Control: no-cache
  3. Connection: keep-alive
  4. X-Accel-Buffering: no (for nginx)

Real-World SSE Example:
```go
func streamHandler(w http.ResponseWriter, r *http.Request) {
    // MUST set headers first
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", 500)
        return
    }

    for msg := range messageChan {
        fmt.Fprintf(w, "data: %s\\n\\n", msg)
        flusher.Flush() // Send immediately
    }
}
```

NDJSON Example:
```go
func streamLogs(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/x-ndjson")
    flusher, _ := w.(http.Flusher)

    for log := range logs {
        json.NewEncoder(w).Encode(log)
        w.Write([]byte("\\n"))
        flusher.Flush()
    }
}
```

Violation Patterns:
1. Flush before setting Content-Type:
   ```go
   flusher.Flush()
   w.Header().Set("Content-Type", "text/event-stream") // TOO LATE!
   ```

2. Missing Content-Type entirely:
   ```go
   flusher, _ := w.(http.Flusher)
   fmt.Fprintf(w, "data: hello\\n\\n")
   flusher.Flush() // WRONG - no Content-Type set
   ```

3. Wrong Content-Type for SSE:
   ```go
   w.Header().Set("Content-Type", "text/plain") // WRONG - use text/event-stream
   ```

Detection Challenges:
- Flush() calls may be in different functions
- Goroutines complicate control flow
- Need to track if headers set before ANY Flush()
- Multiple Flush() calls - only first matters

WebSocket Handling:
- WebSocket upgrade is special case
- Uses "Connection: Upgrade" header
- No Content-Type needed (protocol switch)
- Different rule or skip WebSocket upgrades

TODO: Implementation required
- Detect http.Flusher usage
- Check for Flush() calls
- Verify Content-Type set before first Flush()
- Detect SSE message patterns
- Handle NDJSON streaming
- Skip WebSocket upgrades
- Check for text/event-stream specifically
- Verify Cache-Control and Connection headers
- Complex AST analysis for goroutines
*/

// CheckStreamingContentTypeHeaders validates streaming response Content-Type headers
func CheckStreamingContentTypeHeaders(content []byte, filePath string) []Violation {
	var violations []Violation

	// TODO: Implement streaming Content-Type checking
	// This is complex due to goroutines and control flow
	// Must detect Flush() and verify headers set first
	// SSE is critical - broken without text/event-stream
	// Consider using data flow analysis

	_ = content
	_ = filePath

	return violations
}
