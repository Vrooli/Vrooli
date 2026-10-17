package census

import (
	"fmt"
	"sync"
)

// IncrementalScanner reuses an immutable report when the scan root's
// filesystem signature is unchanged. Callers can call Invalidate after an
// out-of-band change; a changed root directory metadata value also forces a
// fresh walk.
type IncrementalScanner struct {
	mu     sync.Mutex
	fs     FileSystem
	probe  DeviceProbe
	sign   string
	report Report
	valid  bool
}

func NewIncrementalScanner(fs FileSystem, probe DeviceProbe) *IncrementalScanner {
	if fs == nil {
		fs = hostFileSystem{}
	}
	return &IncrementalScanner{fs: fs, probe: probe}
}

func (s *IncrementalScanner) Scan(root string, manifests map[string][]Declaration) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sign, err := rootSignature(s.fs, root)
	if err != nil {
		return Report{}, err
	}
	if s.valid && s.sign == sign {
		return s.report, nil
	}
	report, err := ScanWithFileSystem(root, manifests, s.fs, s.probe)
	if err != nil {
		return Report{}, err
	}
	s.sign, s.report, s.valid = sign, report, true
	return report, nil
}

func (s *IncrementalScanner) Invalidate() {
	s.mu.Lock()
	s.valid = false
	s.mu.Unlock()
}

func rootSignature(fs FileSystem, root string) (string, error) {
	info, err := fs.Stat(root)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%t:%s", info.ModTime().UnixNano(), info.Size(), info.IsDir(), info.Mode()), nil
}
