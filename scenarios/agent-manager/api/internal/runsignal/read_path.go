package runsignal

import (
	"strings"

	"agent-manager/internal/domain"
)

// ReadPath conservatively recognizes file-read tool calls. It is shared by
// report presentation and durable projection so reread semantics are stable.
func ReadPath(call *domain.ToolCallEventData) string {
	if call == nil {
		return ""
	}
	name := strings.ToLower(call.ToolName)
	if !strings.Contains(name, "read") && !strings.Contains(name, "cat") && !strings.Contains(name, "view") {
		return ""
	}
	for _, key := range []string{"path", "file", "filename", "file_path", "filepath", "filePath"} {
		if path, ok := call.Input[key].(string); ok && strings.TrimSpace(path) != "" {
			return strings.TrimSpace(path)
		}
	}
	return ""
}
