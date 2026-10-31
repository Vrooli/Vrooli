package ai

import (
	"context"
	"errors"
)

type rolePromptCall struct {
	Role   string
	Prompt string
}

type mockRolePromptClient struct {
	Response string
	Err      error
	Calls    []rolePromptCall
}

func newMockRolePromptClient(response string) *mockRolePromptClient {
	return &mockRolePromptClient{Response: response}
}

func (m *mockRolePromptClient) ExecutePromptWithRole(_ context.Context, role, prompt string) (string, error) {
	m.Calls = append(m.Calls, rolePromptCall{Role: role, Prompt: prompt})
	if m.Err != nil {
		return "", m.Err
	}
	if m.Response == "" {
		return "", errors.New("empty mock response")
	}
	return m.Response, nil
}
