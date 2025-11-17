package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// PublishRequest represents a publish request
type PublishRequest struct {
	CreateBackup bool                    `json:"create_backup"`
	DeleteDraft  bool                    `json:"delete_draft"`
	Template     *PublishTemplateRequest `json:"template,omitempty"`
}

// PublishTemplateRequest describes template/scenario creation metadata
type PublishTemplateRequest struct {
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	Force     bool              `json:"force"`
	RunHooks  bool              `json:"run_hooks"`
}

// PublishResponse represents the result of a publish operation
type PublishResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	PublishedTo     string `json:"published_to"`
	BackupPath      string `json:"backup_path,omitempty"`
	PublishedAt     string `json:"published_at"`
	DraftRemoved    bool   `json:"draft_removed,omitempty"`
	CreatedScenario bool   `json:"created_scenario,omitempty"`
	ScenarioID      string `json:"scenario_id,omitempty"`
	ScenarioType    string `json:"scenario_type,omitempty"`
	ScenarioPath    string `json:"scenario_path,omitempty"`
}

type ScenarioGenerationResult struct {
	ScenarioID   string
	ScenarioPath string
	TemplateName string
}

func handlePublishDraft(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	draftID := vars["id"]

	// Parse request body (optional)
	var req PublishRequest
	req.CreateBackup = true // Default to creating backup
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Non-fatal: just use defaults if decode fails
			slog.Warn("Failed to decode publish request, using defaults", "error", err)
		}
	}

	// Get draft from database
	draft, err := getDraftByID(draftID)
	if handleDraftError(w, err, "Failed to get draft") {
		return
	}

	publishEntityName := draft.EntityName
	if req.Template != nil {
		if req.Template.Variables == nil {
			req.Template.Variables = make(map[string]string)
		}
		if candidate, ok := req.Template.Variables["SCENARIO_ID"]; ok {
			candidate = strings.TrimSpace(candidate)
			if candidate != "" {
				publishEntityName = candidate
			}
		}
		if publishEntityName == "" {
			publishEntityName = draft.EntityName
		}
		req.Template.Variables["SCENARIO_ID"] = publishEntityName
	}

	// Validate operational targets linkage before publishing
	if err := validateOperationalTargetsLinkage(draft.EntityType, publishEntityName, draft.Content); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
			"error":   "ORPHANED_CRITICAL_TARGETS",
		})
		return
	}

	// Construct target PRD path
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		respondInternalError(w, "Failed to get Vrooli root", err)
		return
	}

	baseDir := filepath.Join(vrooliRoot, draft.EntityType+"s")
	targetPath := filepath.Join(baseDir, publishEntityName, "PRD.md")

	var generationResult *ScenarioGenerationResult
	if req.Template != nil {
		generationResult, err = ensureScenarioFromTemplate(draft, publishEntityName, req.Template)
		if err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}

	// Create backup if requested and file exists
	var backupPath string
	if req.CreateBackup {
		if _, err := os.Stat(targetPath); err == nil {
			backupPath = targetPath + ".backup." + time.Now().Format("20060102-150405")
			if err := copyFile(targetPath, backupPath); err != nil {
				respondInternalError(w, "Failed to create backup", err)
				return
			}
		}
	}

	// Write PRD.md file
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		respondInternalError(w, "Failed to create target directory", err)
		return
	}

	if err := os.WriteFile(targetPath, []byte(draft.Content), 0644); err != nil {
		respondInternalError(w, "Failed to write PRD file", err)
		return
	}

	deleteDraft := req.DeleteDraft
	if req.Template != nil {
		deleteDraft = true
	}

	if deleteDraft {
		if _, err := db.Exec(`DELETE FROM drafts WHERE id = $1`, draftID); err != nil {
			slog.Warn("Failed to delete draft after publish", "error", err, "draft_id", draftID)
		}
	} else {
		_, err = db.Exec(`
			UPDATE drafts
			SET status = $1, updated_at = $2
			WHERE id = $3
		`, DraftStatusPublished, time.Now(), draftID)
		if err != nil {
			// Non-fatal, just log
			slog.Warn("Failed to update draft status", "error", err, "draft_id", draftID)
		}
	}

	// Delete draft file
	draftPath := getDraftPath(draft.EntityType, draft.EntityName)
	if err := os.Remove(draftPath); err != nil && !os.IsNotExist(err) {
		// Non-fatal, just log
		slog.Warn("Failed to delete draft file", "error", err, "path", draftPath)
	}

	response := PublishResponse{
		Success:         true,
		Message:         "Draft published successfully",
		PublishedTo:     targetPath,
		BackupPath:      backupPath,
		PublishedAt:     time.Now().Format(time.RFC3339),
		DraftRemoved:    deleteDraft,
		CreatedScenario: generationResult != nil,
		ScenarioID:      publishEntityName,
		ScenarioType:    draft.EntityType,
	}
	if generationResult != nil {
		response.ScenarioPath = generationResult.ScenarioPath
	}

	respondJSON(w, http.StatusOK, response)
}

func ensureScenarioFromTemplate(draft Draft, publishEntityName string, tpl *PublishTemplateRequest) (*ScenarioGenerationResult, error) {
	if tpl == nil {
		return nil, nil
	}
	if strings.TrimSpace(tpl.Name) == "" {
		return nil, errors.New("template name is required for publishing")
	}

	manifest, err := loadScenarioTemplateManifest(tpl.Name)
	if err != nil {
		return nil, err
	}

	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		return nil, err
	}

	scenarioID := strings.TrimSpace(publishEntityName)
	if scenarioID == "" {
		scenarioID = strings.TrimSpace(draft.EntityName)
	}
	if scenarioID == "" {
		return nil, errors.New("scenario ID could not be determined")
	}

	scenarioPath := filepath.Join(vrooliRoot, "scenarios", scenarioID)
	if !tpl.Force {
		if _, err := os.Stat(scenarioPath); err == nil {
			return nil, fmt.Errorf("scenario %s already exists", scenarioID)
		}
	}

	args, err := buildTemplateArgs(manifest, tpl.Variables)
	if err != nil {
		return nil, err
	}

	cmdArgs := append([]string{"scenario", "generate", tpl.Name}, args...)
	if tpl.RunHooks {
		cmdArgs = append(cmdArgs, "--run-hooks")
	}
	if tpl.Force {
		cmdArgs = append(cmdArgs, "--force")
	}

	cmd := exec.Command("vrooli", cmdArgs...)
	cmd.Dir = vrooliRoot
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to generate scenario: %v\n%s", err, string(output))
	}

	return &ScenarioGenerationResult{
		ScenarioID:   scenarioID,
		ScenarioPath: scenarioPath,
		TemplateName: tpl.Name,
	}, nil
}

func buildTemplateArgs(manifest *TemplateManifest, provided map[string]string) ([]string, error) {
	args := []string{}
	valueFor := func(name string) string {
		if provided == nil {
			return ""
		}
		return strings.TrimSpace(provided[name])
	}

	for _, name := range sortedTemplateKeys(manifest.RequiredVars) {
		def := manifest.RequiredVars[name]
		value := valueFor(name)
		if value == "" {
			return nil, fmt.Errorf("%s is required", name)
		}
		args = append(args, formatTemplateFlag(def, value)...)
	}

	for _, name := range sortedTemplateKeys(manifest.OptionalVars) {
		def := manifest.OptionalVars[name]
		value := valueFor(name)
		if value == "" {
			continue
		}
		args = append(args, formatTemplateFlag(def, value)...)
	}

	for name, value := range provided {
		if _, exists := manifest.RequiredVars[name]; exists {
			continue
		}
		if _, exists := manifest.OptionalVars[name]; exists {
			continue
		}
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		args = append(args, "--var", fmt.Sprintf("%s=%s", name, trimmed))
	}

	return args, nil
}

func formatTemplateFlag(def templateVarDefinition, value string) []string {
	flagName := def.Flag
	if flagName == "" {
		flagName = strings.ToLower(def.Name)
	}
	return []string{"--" + flagName, value}
}

func sortedTemplateKeys(vars map[string]templateVarDefinition) []string {
	keys := make([]string, 0, len(vars))
	for key := range vars {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// validateOperationalTargetsLinkage checks if P0/P1 operational targets have linkages to requirements
func validateOperationalTargetsLinkage(entityType, entityName, content string) error {
	// Get operational targets from draft content
	targets := parseOperationalTargetsV2(content, entityType, entityName)
	if len(targets) == 0 {
		// No targets to validate
		return nil
	}

	// Load requirements to check linkages
	vrooliRoot, err := getVrooliRoot()
	if err != nil {
		slog.Warn("Failed to get Vrooli root for validation", "error", err)
		return nil
	}

	reqsPath := filepath.Join(vrooliRoot, entityType+"s", entityName, "requirements", "index.json")
	if _, err := os.Stat(reqsPath); os.IsNotExist(err) {
		// No requirements file - skip validation
		return nil
	}

	// Find orphaned critical targets (P0/P1 without linked requirements)
	orphanedCritical := []OperationalTarget{}
	for _, target := range targets {
		if (target.Criticality == "P0" || target.Criticality == "P1") && len(target.LinkedRequirements) == 0 {
			orphanedCritical = append(orphanedCritical, target)
		}
	}

	if len(orphanedCritical) > 0 {
		msg := fmt.Sprintf("Cannot publish: %d critical operational target(s) (P0/P1) are not linked to requirements. "+
			"P0 and P1 targets MUST have linked requirements before publishing. "+
			"Orphaned targets: ", len(orphanedCritical))
		for i, target := range orphanedCritical {
			if i > 0 {
				msg += ", "
			}
			msg += fmt.Sprintf("%s (%s)", target.Title, target.Criticality)
		}
		return fmt.Errorf("%s", msg)
	}

	return nil
}

// Helper function to copy a file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return destFile.Sync()
}
