package support

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"strings"
)

// ReadProtoData decodes the offered typed JSON document without altering its
// reference, optional-field presence, actor or authority claims. The owner
// still qualifies every retained reference and decision through caller proof.
func ReadProtoData(data string, target proto.Message) error {
	if len(data) > 128*1024 {
		return fmt.Errorf("--data exceeds 128 KiB")
	}
	if strings.TrimSpace(data) == "" {
		return fmt.Errorf("--data requires a typed JSON object")
	}
	if err := protojson.Unmarshal([]byte(data), target); err != nil {
		return fmt.Errorf("invalid --data: %w", err)
	}
	return nil
}
