// Package backlog provides HTTP handlers for backlog management.
package backlog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PromptCategory represents the category of prompt templates.
type PromptCategory string

const (
	PromptCategoryResearch   PromptCategory = "research"
	PromptCategoryWorkflow   PromptCategory = "workflow"
	PromptCategoryProcessing PromptCategory = "processing"
)

// PromptLoader loads and caches prompt templates from markdown files.
// Templates are loaded from scenarios/swarm-manager/prompts/ and support
// variable substitution using {{VARIABLE}} syntax.
type PromptLoader struct {
	promptDir string
	cache     map[string]*cachedPrompt
	mu        sync.RWMutex
}

type cachedPrompt struct {
	content  string
	loadedAt time.Time
	modTime  time.Time
}

// promptCacheTTL is how long to cache prompts before checking for changes.
const promptCacheTTL = 5 * time.Minute

// NewPromptLoader creates a new prompt loader.
// The rootDir should be the swarm-manager scenario root (e.g., "scenarios/swarm-manager").
func NewPromptLoader(rootDir string) *PromptLoader {
	return &PromptLoader{
		promptDir: filepath.Join(rootDir, "prompts"),
		cache:     make(map[string]*cachedPrompt),
	}
}

// Load loads a prompt template by category and name.
// Returns the raw template content with {{VARIABLE}} placeholders.
// Falls back to embedded defaults if file is not found.
func (pl *PromptLoader) Load(category PromptCategory, name string) (string, error) {
	cacheKey := fmt.Sprintf("%s/%s", category, name)

	// Check cache first
	pl.mu.RLock()
	cached, ok := pl.cache[cacheKey]
	pl.mu.RUnlock()

	if ok && time.Since(cached.loadedAt) < promptCacheTTL {
		// Check if file was modified
		filePath := pl.filePath(category, name)
		info, err := os.Stat(filePath)
		if err == nil && !info.ModTime().After(cached.modTime) {
			return cached.content, nil
		}
	}

	// Load from file
	content, modTime, err := pl.loadFromFile(category, name)
	if err != nil {
		// Fall back to embedded default
		content, ok = pl.getFallback(category, name)
		if !ok {
			return "", fmt.Errorf("prompt not found: %s/%s", category, name)
		}
		modTime = time.Time{} // No mod time for fallbacks
	}

	// Update cache
	pl.mu.Lock()
	pl.cache[cacheKey] = &cachedPrompt{
		content:  content,
		loadedAt: time.Now(),
		modTime:  modTime,
	}
	pl.mu.Unlock()

	return content, nil
}

// Build substitutes template variables with values from the backlog item.
func (pl *PromptLoader) Build(template string, item BacklogItem, itemFolder string) string {
	replacements := map[string]string{
		"{{ITEM_NAME}}":        item.Name,
		"{{ITEM_TITLE}}":       item.Title,
		"{{ITEM_DESCRIPTION}}": item.Description,
		"{{ITEM_KIND}}":        string(item.Kind),
		"{{ITEM_STATUS}}":      string(item.Status),
		"{{ITEM_PRIORITY}}":    fmt.Sprintf("%d", item.Priority),
		"{{ITEM_TAGS}}":        strings.Join(item.Tags, ", "),
		"{{ITEM_FOLDER}}":      itemFolder,
		"{{RESEARCH_TARGET}}":  item.ResearchTarget,
	}

	result := template
	for placeholder, value := range replacements {
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// filePath returns the full path to a prompt file.
func (pl *PromptLoader) filePath(category PromptCategory, name string) string {
	return filepath.Join(pl.promptDir, string(category), name+".md")
}

// loadFromFile loads a prompt from the filesystem.
func (pl *PromptLoader) loadFromFile(category PromptCategory, name string) (string, time.Time, error) {
	filePath := pl.filePath(category, name)

	info, err := os.Stat(filePath)
	if err != nil {
		return "", time.Time{}, err
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", time.Time{}, err
	}

	return string(content), info.ModTime(), nil
}

// getFallback returns embedded fallback content for critical prompts.
// This ensures the system works even if prompt files are missing.
func (pl *PromptLoader) getFallback(category PromptCategory, name string) (string, bool) {
	key := fmt.Sprintf("%s/%s", category, name)
	content, ok := fallbackPrompts[key]
	return content, ok
}

// ResearchPromptName returns the prompt file name for a research mode and item kind.
func ResearchPromptName(mode ResearchMode, kind BacklogKind) string {
	switch mode {
	case ResearchModeClarify:
		return "clarify"
	case ResearchModeSuggest:
		return "suggest"
	case ResearchModeEnhance:
		return "enhance"
	default:
		// Deep research mode - varies by kind
		switch kind {
		case KindIdea:
			return "deep-research-idea"
		case KindFix:
			return "deep-research-fix"
		default:
			return "deep-research-other"
		}
	}
}

// ProcessingPromptName returns the prompt file name for a processing operation.
func ProcessingPromptName(kind BacklogKind) string {
	switch kind {
	case KindIdea:
		return "process-idea"
	case KindFix:
		return "process-fix"
	default:
		return "process-execute"
	}
}

// fallbackPrompts provides minimal defaults when prompt files are unavailable.
var fallbackPrompts = map[string]string{
	"workflow/clarify": `# Clarify

Generate clarifying questions for the backlog item.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Kind:** {{ITEM_KIND}}

Goal: Generate the most important clarifying questions about scope, requirements, constraints, and implementation details.
Write results to clarify/questions.json with schema:
{"questions":[{"id":"q1","question":"...","category":"users|technical|scope|constraints|integration","importance":"critical|important|nice-to-have","options":[],"answer":""}]}
Preserve existing questions and answers; append new questions if needed.
Maximum 10 questions.
`,

	"workflow/suggest": `# Suggest

Propose improvements for the backlog item.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Kind:** {{ITEM_KIND}}

Goal: Propose improvements or alternative approaches for this idea.
Write results to suggest/suggestions.json with schema:
{"suggestions":[{"id":"s1","suggestion":"...","details":"...","category":"architecture|ux|scope|risk|opportunity","impact":"high|medium|low","status":"pending","rejection_reason":""}]}
Preserve existing suggestions and decisions; append new suggestions if needed.
Maximum 7 suggestions.
`,

	"workflow/enhance": `# Enhance

Synthesize clarifications and suggestions into a refined plan.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Kind:** {{ITEM_KIND}}

Goal: Produce a refined plan based on clarifications and accepted suggestions.
Read clarify/questions.json and suggest/suggestions.json if present.
Apply accepted suggestions, ignore rejected ones.
Write the enhancement summary to enhance/summary.md and update spec.json if necessary.
`,

	"research/deep-research-idea": `# Deep Research: Idea

Research the feasibility and implementation approach for this idea.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Folder:** {{ITEM_FOLDER}}

Goal: Provide a concise research summary covering feasibility, dependencies, implementation approaches, and risks.
Write a summary to research/summary.md and add supporting files as needed.
`,

	"research/deep-research-fix": `# Deep Research: Fix

Analyze the root cause and remediation approach for this fix.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Folder:** {{ITEM_FOLDER}}

Goal: Identify the root cause, affected components, and safe remediation approach.
Write a summary to research/summary.md and add supporting files as needed.
`,

	"research/deep-research-other": `# Deep Research: General

Conduct research for this backlog item.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Research Target:** {{RESEARCH_TARGET}}
**Folder:** {{ITEM_FOLDER}}

Goal: Provide thorough research findings with analysis and recommendations.
Write a summary to research/summary.md and add supporting files as needed.
`,

	"processing/process-idea": `# Process: Idea

Create or improve a scenario based on this idea.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Folder:** {{ITEM_FOLDER}}

Use the idea folder as the complete context: spec.json, clarify/questions.json, suggest/suggestions.json, enhance/summary.md, and any user-added files.
Respect answers and accepted suggestions when generating or improving a scenario.
Leave a concise completion summary in notes.md.
`,

	"processing/process-fix": `# Process: Fix

Apply the fix described in this backlog item.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Folder:** {{ITEM_FOLDER}}

Review research/summary.md and any supporting files before acting.
Apply the fix following the research recommendations.
Leave a concise completion summary in notes.md.
`,

	"processing/process-execute": `# Process: Execute

Carry out the execution task described in this backlog item.

**Item:** {{ITEM_TITLE}}
**Description:** {{ITEM_DESCRIPTION}}
**Folder:** {{ITEM_FOLDER}}

Review research/summary.md and any supporting files before acting.
Complete the requested task.
Leave a concise completion summary in summary.md.
`,
}
