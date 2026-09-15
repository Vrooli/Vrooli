package testsyntax

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type NodeRunner struct{ Script string }

func (r NodeRunner) Identity() string {
	// Configuration, implementation and installed dependency versions all affect
	// reusable evidence. A rebuilt tool or lockfile invalidates the memo key.
	h := sha256.New()
	for _, path := range []string{r.Script, filepath.Join(filepath.Dir(r.Script), "../eslint-rules/vitest-profile.mjs"), filepath.Join(filepath.Dir(r.Script), "../pnpm-lock.yaml")} {
		data, err := os.ReadFile(path)
		if err != nil {
			return "unavailable:" + path
		}
		h.Write(data)
	}
	return hex.EncodeToString(h.Sum(nil))
}

type boundedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("native lint output exceeds limit")
	}
	return b.Buffer.Write(p)
}
func (r NodeRunner) Run(ctx context.Context, inputs []Input) ([]Observation, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	data, err := json.Marshal(inputs)
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "node", r.Script)
	cmd.Stdin = bytes.NewReader(data)
	output := &boundedBuffer{limit: 8 * 1024 * 1024}
	stderr := &boundedBuffer{limit: 16 * 1024}
	cmd.Stdout = output
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("native lint owner: %w: %s", err, stderr.String())
	}
	var rows []Observation
	if err := json.Unmarshal(output.Bytes(), &rows); err != nil {
		return nil, fmt.Errorf("native lint malformed output: %w", err)
	}
	return rows, nil
}
