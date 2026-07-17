package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/promptmanager"
)

// resolveWorkflowPromptRefs embeds prompt-manager skill content into any run or
// continue node that authored a promptRef, returning the definition bytes the
// catalog then parses and digests. Resolution happens before the digest so the
// resolved content and its provenance are pinned into the revision: a later skill
// change yields different content, a different digest, and therefore a new
// revision on the next reconcile — never a silent behavior change under a fixed
// digest. When no node authors a promptRef the original bytes are returned
// untouched so the source hash and digest are unaffected.
func (o *Orchestrator) resolveWorkflowPromptRefs(ctx context.Context, data []byte) ([]byte, error) {
	if !bytes.Contains(data, []byte(`"promptRef"`)) {
		return data, nil
	}
	var def domain.WorkflowDefinition
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&def); err != nil {
		return nil, fmt.Errorf("decode workflow definition: %w", err)
	}
	source, ok := o.promptClient.(promptmanager.SourceClient)
	if !ok || source == nil {
		return nil, fmt.Errorf("promptRef resolution requires a prompt-manager source client")
	}
	resolved := false
	for i := range def.Nodes {
		n := &def.Nodes[i]
		if n.Run != nil && n.Run.PromptRef != nil {
			if err := resolvePromptRef(ctx, source, &n.Run.PromptTemplate, &n.Run.PromptRef, &n.Run.PromptProvenance); err != nil {
				return nil, fmt.Errorf("nodes[%d].run.promptRef: %w", i, err)
			}
			resolved = true
		}
		if n.Continue != nil && n.Continue.PromptRef != nil {
			if err := resolvePromptRef(ctx, source, &n.Continue.PromptTemplate, &n.Continue.PromptRef, &n.Continue.PromptProvenance); err != nil {
				return nil, fmt.Errorf("nodes[%d].continue.promptRef: %w", i, err)
			}
			resolved = true
		}
	}
	if !resolved {
		return data, nil
	}
	return json.Marshal(def)
}

// resolvePromptRef reads one skill and rewrites the node in place: the resolved
// content replaces the empty promptTemplate, provenance is recorded, and the
// promptRef is cleared so the canonical (digested) form carries only concrete
// content plus its pinned source.
func resolvePromptRef(ctx context.Context, source promptmanager.SourceClient, tmpl *string, refPtr **domain.WorkflowPromptRef, provPtr **domain.WorkflowPromptSource) error {
	ref := *refPtr
	if strings.TrimSpace(ref.SkillID) == "" {
		return fmt.Errorf("promptRef requires a skillId")
	}
	if strings.TrimSpace(*tmpl) != "" {
		return fmt.Errorf("promptRef and promptTemplate are mutually exclusive")
	}
	snap, err := source.ReadSkillSource(ctx, ref.SkillID, ref.Variables, ref.WithScope)
	if err != nil {
		return err
	}
	if strings.TrimSpace(snap.Content) == "" {
		return fmt.Errorf("resolved prompt for skill %q is empty", ref.SkillID)
	}
	*tmpl = snap.Content
	*provPtr = &domain.WorkflowPromptSource{
		SkillID:     snap.SkillID,
		Revision:    snap.Revision,
		VariantID:   snap.VariantID,
		ContentHash: snap.ContentHash,
	}
	*refPtr = nil
	return nil
}
