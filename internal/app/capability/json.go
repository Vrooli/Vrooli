package capabilityapp

import (
	"encoding/json"
	"io"

	"github.com/vrooli/vrooli/internal/cliout"
	"google.golang.org/protobuf/types/known/structpb"
)

func writeCapabilityValue(w io.Writer, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return err
	}
	message, err := structpb.NewValue(decoded)
	if err != nil {
		return err
	}
	return cliout.WriteProtoJSON(w, message)
}
