package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoJSONMarshaler = protojson.MarshalOptions{
	UseProtoNames:   true,
	EmitUnpopulated: false,
}

// respondProto marshals and writes a proto message using protojson with snake_case field names.
func (h *Handler) respondProto(w http.ResponseWriter, statusCode int, msg proto.Message) {
	data, err := protoJSONMarshaler.Marshal(msg)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Error("Failed to marshal proto response")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "failed to encode response",
		}))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(data)
}

// respondProtoList marshals a collection of proto messages under the provided field name.
// This keeps list endpoints aligned to protojson snake_case output without requiring a wrapper proto.
func (h *Handler) respondProtoList(w http.ResponseWriter, statusCode int, field string, msgs []proto.Message) {
	if field == "" {
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "invalid proto list field",
		}))
		return
	}

	items := make([][]byte, 0, len(msgs))
	for idx, msg := range msgs {
		data, err := protoJSONMarshaler.Marshal(msg)
		if err != nil {
			if h.log != nil {
				h.log.WithError(err).Error(fmt.Sprintf("Failed to marshal proto list item %d", idx))
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": fmt.Sprintf("failed to encode %s[%d]", field, idx),
			}))
			return
		}
		items = append(items, data)
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	_, _ = buf.WriteString(fmt.Sprintf(`"%s":[`, field))
	for i, data := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(data)
	}
	buf.WriteString("]}")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
}

// respondExecutionsWithExportability marshals executions with an additional exportability map.
// Response format: {"executions": [...], "exportability": {"exec_id": {...}, ...}}
func (h *Handler) respondExecutionsWithExportability(w http.ResponseWriter, statusCode int, msgs []proto.Message, exportability map[string]ExecutionExportability) {
	// Marshal executions
	items := make([][]byte, 0, len(msgs))
	for idx, msg := range msgs {
		data, err := protoJSONMarshaler.Marshal(msg)
		if err != nil {
			if h.log != nil {
				h.log.WithError(err).Error(fmt.Sprintf("Failed to marshal execution %d", idx))
			}
			h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
				"error": fmt.Sprintf("failed to encode executions[%d]", idx),
			}))
			return
		}
		items = append(items, data)
	}

	// Marshal exportability map
	exportabilityJSON, err := json.Marshal(exportability)
	if err != nil {
		if h.log != nil {
			h.log.WithError(err).Error("Failed to marshal exportability map")
		}
		h.respondError(w, ErrInternalServer.WithDetails(map[string]string{
			"error": "failed to encode exportability",
		}))
		return
	}

	var buf bytes.Buffer
	buf.WriteByte('{')
	buf.WriteString(`"executions":[`)
	for i, data := range items {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(data)
	}
	buf.WriteString(`],"exportability":`)
	buf.Write(exportabilityJSON)
	buf.WriteByte('}')

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write(buf.Bytes())
}
