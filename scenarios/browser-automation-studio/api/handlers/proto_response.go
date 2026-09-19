package handlers

import (
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
