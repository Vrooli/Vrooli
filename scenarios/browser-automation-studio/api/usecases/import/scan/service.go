package scan

import (
	"context"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/usecases/import/routines"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	"google.golang.org/protobuf/encoding/protojson"
)

var assetExtensions = []string{
	".png",
	".jpg",
	".jpeg",
	".webp",
	".gif",
	".json",
	".csv",
	".txt",
	".md",
}

// Service handles unified scan logic for import workflows.
type Service struct {
	scanner           shared.DirectoryScanner
	projecter         shared.ProjectIndexer
	workflowIndexer   shared.WorkflowIndexer
	workflowValidator *routines.Validator
	log               *logrus.Logger
}

// NewService creates a new Service.
func NewService(
	scanner shared.DirectoryScanner,
	projecter shared.ProjectIndexer,
	workflowIndexer shared.WorkflowIndexer,
	log *logrus.Logger,
) *Service {
	return &Service{
		scanner:           scanner,
		projecter:         projecter,
		workflowIndexer:   workflowIndexer,
		workflowValidator: routines.NewValidator(),
		log:               log,
	}
}

// Scan performs a mode-based directory scan for BAS objects.
func (s *Service) Scan(ctx context.Context, req *ScanRequest) (*ScanResponse, error) {
	mode := ScanMode(strings.ToLower(strings.TrimSpace(req.Mode)))
	if mode == "" {
		return nil, fmt.Errorf("mode is required")
	}

	var scanPath string
	var defaultRoot string
	var project *shared.ProjectIndexData

	switch mode {
	case ScanModeProjects:
		defaultRoot = shared.DefaultProjectsRoot()
		scanPath = strings.TrimSpace(req.Path)
		if scanPath == "" {
			scanPath = defaultRoot
		}
	case ScanModeWorkflows:
		projectID, err := parseProjectID(req.ProjectID)
		if err != nil {
			return nil, err
		}
		project, err = s.projecter.GetProjectByID(ctx, projectID)
		if err != nil || project == nil {
			return nil, fmt.Errorf("project not found")
		}
		defaultRoot = filepath.Join(project.FolderPath, shared.WorkflowsDir)
		scanPath = strings.TrimSpace(req.Path)
		if scanPath == "" {
			scanPath = defaultRoot
		}
	case ScanModeAssets, ScanModeFiles:
		defaultRoot = resolveHomeDir()
		scanPath = strings.TrimSpace(req.Path)
		if scanPath == "" {
			scanPath = defaultRoot
		}
	default:
		return nil, fmt.Errorf("unsupported mode: %s", req.Mode)
	}

	absScanPath, err := shared.NormalizePath(scanPath)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	if req.Path != "" {
		exists, err := s.scanner.Exists(ctx, absScanPath)
		if err != nil {
			return nil, fmt.Errorf("failed to check path: %w", err)
		}
		if !exists {
			return nil, fmt.Errorf("path does not exist")
		}
	}

	exists, err := s.scanner.Exists(ctx, absScanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check path: %w", err)
	}
	if !exists {
		return &ScanResponse{
			Path:        absScanPath,
			DefaultRoot: defaultRoot,
			Entries:     []ScanEntry{},
		}, nil
	}

	isDir, err := s.scanner.IsDir(ctx, absScanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check directory: %w", err)
	}
	if !isDir {
		return nil, fmt.Errorf("path is not a directory")
	}

	entries, err := s.scanner.ScanDirectory(ctx, absScanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %w", err)
	}

	parent := filepath.Dir(absScanPath)
	if parent == absScanPath {
		parent = ""
	}

	response := &ScanResponse{
		Path:        absScanPath,
		DefaultRoot: defaultRoot,
		Entries:     []ScanEntry{},
	}
	if parent != "" {
		response.Parent = &parent
	}

	switch mode {
	case ScanModeProjects:
		indexed, _ := s.projecter.ListProjects(ctx, 1000, 0)
		indexedByPath := make(map[string]*shared.ProjectIndexData)
		for _, p := range indexed {
			indexedByPath[p.FolderPath] = p
		}

		for _, entry := range entries {
			if !entry.IsDir {
				continue
			}

			scanEntry := ScanEntry{
				Name:     entry.Name,
				Path:     entry.Path,
				IsDir:    true,
				IsTarget: true,
			}

			if metaName := readProjectName(entry.Path); metaName != "" {
				scanEntry.SuggestedName = metaName
			}

			if existing, ok := indexedByPath[entry.Path]; ok {
				scanEntry.IsRegistered = true
				scanEntry.RegisteredID = existing.ID.String()
				if scanEntry.SuggestedName == "" {
					scanEntry.SuggestedName = existing.Name
				}
			}

			response.Entries = append(response.Entries, scanEntry)
		}
	case ScanModeWorkflows:
		indexedByPath := map[string]*shared.WorkflowIndexData{}
		if project != nil {
			indexed, _ := s.workflowIndexer.ListWorkflowsByProject(ctx, project.ID)
			for _, w := range indexed {
				indexedByPath[w.FilePath] = w
			}
		}

		for _, entry := range entries {
			if entry.IsDir {
				response.Entries = append(response.Entries, ScanEntry{
					Name:     entry.Name,
					Path:     entry.Path,
					IsDir:    true,
					IsTarget: false,
				})
				continue
			}

			if !shared.IsWorkflowFile(entry.Name) {
				continue
			}

			scanEntry := ScanEntry{
				Name:      entry.Name,
				Path:      entry.Path,
				IsDir:     false,
				IsTarget:  false,
				MimeType:  mime.TypeByExtension(filepath.Ext(entry.Name)),
				SizeBytes: entry.Size,
			}

			content, readErr := s.scanner.ReadFile(ctx, entry.Path)
			if readErr == nil {
				if validation, valErr := s.workflowValidator.ValidateWorkflowJSON(content); valErr == nil && validation != nil {
					scanEntry.IsTarget = validation.IsValid
				}

				preview, previewErr := s.workflowValidator.ExtractWorkflowPreview(content)
				if previewErr == nil && preview != nil && preview.Name != "" {
					scanEntry.SuggestedName = preview.Name
				}
			}

			if project != nil {
				if relPath, relErr := shared.RelativePath(project.FolderPath, entry.Path); relErr == nil {
					if existing, ok := indexedByPath[relPath]; ok {
						scanEntry.IsRegistered = true
						scanEntry.RegisteredID = existing.ID.String()
						if scanEntry.SuggestedName == "" {
							scanEntry.SuggestedName = existing.Name
						}
					}
				}
			}

			response.Entries = append(response.Entries, scanEntry)
		}
	case ScanModeAssets:
		for _, entry := range entries {
			if entry.IsDir {
				response.Entries = append(response.Entries, ScanEntry{
					Name:     entry.Name,
					Path:     entry.Path,
					IsDir:    true,
					IsTarget: false,
				})
				continue
			}

			if !shared.HasExtension(entry.Name, assetExtensions...) {
				continue
			}

			response.Entries = append(response.Entries, ScanEntry{
				Name:      entry.Name,
				Path:      entry.Path,
				IsDir:     false,
				IsTarget:  true,
				MimeType:  mime.TypeByExtension(filepath.Ext(entry.Name)),
				SizeBytes: entry.Size,
			})
		}
	case ScanModeFiles:
		for _, entry := range entries {
			if entry.IsDir {
				response.Entries = append(response.Entries, ScanEntry{
					Name:     entry.Name,
					Path:     entry.Path,
					IsDir:    true,
					IsTarget: false,
				})
				continue
			}

			response.Entries = append(response.Entries, ScanEntry{
				Name:      entry.Name,
				Path:      entry.Path,
				IsDir:     false,
				IsTarget:  true,
				MimeType:  mime.TypeByExtension(filepath.Ext(entry.Name)),
				SizeBytes: entry.Size,
			})
		}
	}

	return response, nil
}

func parseProjectID(raw string) (uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return uuid.UUID{}, fmt.Errorf("project_id is required")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid project_id")
	}
	return id, nil
}

func resolveHomeDir() string {
	if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
		return home
	}
	return string(filepath.Separator)
}

func readProjectName(folderPath string) string {
	metaPath := filepath.Join(folderPath, shared.ProjectMetadataPath)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return ""
	}
	var meta basprojects.Project
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(data, &meta); err != nil {
		return ""
	}
	return strings.TrimSpace(meta.Name)
}
