package workflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
)

const projectMetadataDirName = ".bas"
const projectMetadataFileName = "project.json"

func projectMetadataPath(projectFolderPath string) string {
	return filepath.Join(projectFolderPath, projectMetadataDirName, projectMetadataFileName)
}

func ensureProjectMetadataDir(projectFolderPath string) error {
	dir := filepath.Join(projectFolderPath, projectMetadataDirName)
	return os.MkdirAll(dir, 0o755)
}

func hydrateProjectProto(ctx context.Context, project *database.ProjectIndex) (*basprojects.Project, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	_ = ctx

	pb := &basprojects.Project{
		Id:         project.ID.String(),
		Name:       project.Name,
		FolderPath: project.FolderPath,
		CreatedAt:  autocontracts.TimeToTimestamp(project.CreatedAt),
		UpdatedAt:  autocontracts.TimeToTimestamp(project.UpdatedAt),
	}

	metaPath := projectMetadataPath(project.FolderPath)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return pb, nil
		}
		return nil, fmt.Errorf("read project metadata: %w", err)
	}

	var meta basprojects.Project
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &meta); err != nil {
		// If the file is corrupt, prefer the DB index but surface a useful error.
		return nil, fmt.Errorf("parse project metadata protojson: %w", err)
	}

	if strings.TrimSpace(meta.Description) != "" {
		pb.Description = meta.Description
	}

	return pb, nil
}

func persistProjectProto(project *database.ProjectIndex, description string) error {
	if project == nil {
		return errors.New("project is nil")
	}
	if strings.TrimSpace(project.FolderPath) == "" {
		return errors.New("project folder path missing")
	}
	if err := ensureProjectMetadataDir(project.FolderPath); err != nil {
		return fmt.Errorf("ensure project metadata dir: %w", err)
	}

	now := time.Now().UTC()
	pb := &basprojects.Project{
		Id:          project.ID.String(),
		Name:        project.Name,
		Description: strings.TrimSpace(description),
		FolderPath:  project.FolderPath,
		CreatedAt:   autocontracts.TimeToTimestamp(project.CreatedAt),
		UpdatedAt:   autocontracts.TimeToTimestamp(now),
	}

	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(pb)
	if err != nil {
		return fmt.Errorf("marshal project metadata: %w", err)
	}
	indented := raw
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err == nil {
		indented = buf.Bytes()
	}

	target := projectMetadataPath(project.FolderPath)
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, indented, 0o644); err != nil {
		return fmt.Errorf("write project metadata tmp: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("finalize project metadata: %w", err)
	}
	return nil
}
