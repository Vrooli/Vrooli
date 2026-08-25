//go:build windows

package hostpresentation

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

func effectiveUID() int { return 1 }

func nativeWindowsSession(_ context.Context) ([]byte, error) {
	var processSession uint32
	if err := windows.ProcessIdToSessionId(uint32(os.Getpid()), &processSession); err != nil {
		return nil, fmt.Errorf("resolve process session: %w", err)
	}
	return []byte(fmt.Sprintf("process=%d console=%d", processSession, windows.WTSGetActiveConsoleSessionId())), nil
}
