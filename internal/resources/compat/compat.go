package compat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/vroolierr"
)

const ErrorCodeCommandUnavailable = "command_unavailable"

type Service struct {
	Root       string
	Home       string
	LookPathFn func(string) (string, error)
}

func New(root, home string, lookPathFn func(string) (string, error)) *Service {
	return &Service{
		Root:       filepath.Clean(root),
		Home:       filepath.Clean(home),
		LookPathFn: lookPathFn,
	}
}

func (s *Service) ResolveCLIPath(name string) (string, bool) {
	if s.LookPathFn == nil {
		return "", false
	}
	if path, err := s.LookPathFn("resource-" + name); err == nil {
		return path, true
	}
	return "", false
}

func (s *Service) CommandForResource(name string, args ...string) (*exec.Cmd, error) {
	if path, ok := s.ResolveCLIPath(name); ok {
		return shell.Command(shell.Spec{
			Name: path,
			Args: args,
			Dir:  s.Root,
			Env:  ResourceEnv(s.Root, s.Home),
		}), nil
	}

	scriptPath := filepath.Join(s.Root, "resources", name, "cli.sh")
	if _, err := os.Stat(scriptPath); err == nil {
		return shell.BashScript(scriptPath, args, shell.Spec{
			Dir: s.Root,
			Env: ResourceEnv(s.Root, s.Home),
		}), nil
	}

	return nil, &vroolierr.Error{
		Code:      ErrorCodeCommandUnavailable,
		Resource:  name,
		Operation: "invoke",
		Category:  "Environment",
		Err:       fmt.Errorf("no installed CLI or cli.sh"),
	}
}

func ResourceEnv(root, home string) []string {
	env := os.Environ()
	env = SetEnvValue(env, "VROOLI_ROOT", root)
	env = SetEnvValue(env, "APP_ROOT", root)
	if strings.TrimSpace(home) != "" {
		env = SetEnvValue(env, "HOME", home)
	}
	return env
}

func SetEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}

func ExtractJSONPayload(output []byte) (json.RawMessage, bool) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 {
		return nil, false
	}
	start := bytes.IndexByte(trimmed, '{')
	end := bytes.LastIndexByte(trimmed, '}')
	if start < 0 || end < start {
		return nil, false
	}
	candidate := bytes.TrimSpace(trimmed[start : end+1])
	if !json.Valid(candidate) {
		return nil, false
	}
	return append(json.RawMessage(nil), candidate...), true
}

func BoolValue(value any, fallback bool) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "true", "yes", "healthy", "running", "installed":
			return true
		case "false", "no", "stopped", "missing":
			return false
		}
	}
	return fallback
}

func StringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	default:
		data, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return strings.Trim(strings.TrimSpace(string(data)), `"`)
	}
}
