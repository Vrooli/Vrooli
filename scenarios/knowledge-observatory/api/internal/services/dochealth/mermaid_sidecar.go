package dochealth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

const defaultMermaidSidecarTimeout = 10 * time.Second

type mermaidSidecarValidator struct {
	root    string
	timeout time.Duration
}

func NewMermaidSidecarValidator() DiagramValidator { return mermaidSidecarValidator{} }

func (v mermaidSidecarValidator) ValidateDiagrams(ctx context.Context, blocks []DiagramBlock) (DiagramValidation, error) {
	root := v.root
	if root == "" {
		root = filepath.Join("..", "tools", "mermaid-lint")
	}
	payload, err := json.Marshal(struct {
		Blocks []DiagramBlock `json:"blocks"`
	}{blocks})
	if err != nil {
		return DiagramValidation{}, err
	}
	timeout := v.timeout
	if timeout <= 0 {
		timeout = defaultMermaidSidecarTimeout
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(callCtx, "node", "lint.mjs")
	cmd.Dir = root
	cmd.Stdin = bytes.NewReader(payload)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return DiagramValidation{}, fmt.Errorf("run Mermaid parser: %w: %s", err, bytes.TrimSpace(out))
	}
	var response struct {
		Engine  string `json:"engine"`
		Results []struct {
			ID    string `json:"id"`
			Valid bool   `json:"valid"`
			Error string `json:"error"`
			Line  *int   `json:"line"`
		} `json:"results"`
	}
	if err := json.Unmarshal(out, &response); err != nil {
		return DiagramValidation{}, fmt.Errorf("decode Mermaid parser output: %w", err)
	}
	result := DiagramValidation{Engine: response.Engine}
	for _, item := range response.Results {
		line := 0
		if item.Line != nil {
			line = *item.Line
		}
		result.Verdicts = append(result.Verdicts, DiagramVerdict{ID: item.ID, Valid: item.Valid, Error: item.Error, Line: line})
	}
	return result, nil
}
