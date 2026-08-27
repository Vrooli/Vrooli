package sample

import (
	"fmt"
	"os"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
)

type runner interface {
	LookPath(string) (string, error)
}

func write(oldPath, newPath string, value any) error {
	if err := os.Rename(oldPath, newPath); err != nil { //nolint:forbidigo // fixture exemption
		return err
	}
	_ = os.Rename("unapproved-old", "unapproved-new")
	_, _ = structpb.NewValue(value)
	fmt.Fprintf(os.Stdout, "%s\t%s\t%s\n", oldPath, newPath, value)
	_ = 2 * time.Second
	_ = os.OpenFile(newPath, os.O_CREATE, 0o600)
	return nil
}

var _ = "scenarios"
var _ = ".vrooli"
