package dochealing

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// SkillProvider resolves prompt-manager skills for prompt composition.
type SkillProvider interface {
	GetSkill(ctx context.Context, id string) (string, error)
}

// CompositeSkillProvider tries the primary provider, then the fallback.
type CompositeSkillProvider struct {
	Primary  SkillProvider
	Fallback SkillProvider
}

func (c CompositeSkillProvider) GetSkill(ctx context.Context, id string) (string, error) {
	if c.Primary != nil {
		if content, err := c.Primary.GetSkill(ctx, id); err == nil && strings.TrimSpace(content) != "" {
			return content, nil
		}
	}
	if c.Fallback != nil {
		return c.Fallback.GetSkill(ctx, id)
	}
	return "", fmt.Errorf("skill provider not configured")
}

// FileSkillProvider loads a skill from disk.
type FileSkillProvider struct {
	Path string
}

func (f FileSkillProvider) GetSkill(_ context.Context, _ string) (string, error) {
	path := strings.TrimSpace(f.Path)
	if path == "" {
		return "", fmt.Errorf("skill path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
