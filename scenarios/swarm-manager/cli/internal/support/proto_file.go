package support

import (
	"fmt"
	"io"
	"os"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ReadProtoFile is the bounded, strict input seam for review and decision files.
// Do not open a FIFO/device while waiting for an operator's decision document.
func ReadProtoFile(filename string, target proto.Message) error {
	const limit = 128 * 1024
	before, err := os.Stat(filename)
	if err != nil {
		return err
	}
	if !before.Mode().IsRegular() || before.Size() > limit {
		return fmt.Errorf("input must be a regular JSON file of at most 128 KiB")
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return fmt.Errorf("input must be a regular JSON file of at most 128 KiB")
	}
	data, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return err
	}
	if len(data) > limit {
		return fmt.Errorf("input exceeds 128 KiB")
	}
	if err := protojson.Unmarshal(data, target); err != nil {
		return fmt.Errorf("invalid input: %w", err)
	}
	return nil
}
