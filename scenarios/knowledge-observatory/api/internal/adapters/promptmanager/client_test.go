package promptmanager

import (
	"context"
	"testing"
)

func TestGetSkillRequiresID(t *testing.T) {
	t.Parallel()

	client := NewClient(0)
	_, err := client.GetSkill(context.Background(), "")
	if err == nil {
		t.Fatalf("expected error for empty skill id")
	}
}
