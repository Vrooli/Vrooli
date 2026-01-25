package main

import (
	"context"
	"strings"

	"git-control-tower/filerelations"
)

// fileRelationsService is a singleton instance of the file relations service
var fileRelationsService *filerelations.Service

func init() {
	fileRelationsService = filerelations.NewService()
}

// GetRelatedFiles finds files related to the given path
func GetRelatedFiles(ctx context.Context, deps FileDeps, filePath string) ([]RelatedFile, error) {
	cleanPath := cleanFilePath(filePath)
	if cleanPath == "" || strings.HasPrefix(cleanPath, "..") {
		return []RelatedFile{}, nil
	}

	// Use the filerelations service to find related files
	serviceRelated, err := fileRelationsService.GetRelatedFiles(ctx, cleanPath, deps.RepoDir)
	if err != nil {
		// Log the error but continue with convention-based discovery
		serviceRelated = nil
	}

	// Convert filerelations.RelatedFile to main.RelatedFile
	var related []RelatedFile
	for _, r := range serviceRelated {
		var relType RelationType
		switch r.RelationType {
		case filerelations.RelationImports:
			relType = RelationTypeImports
		case filerelations.RelationImportedBy:
			relType = RelationTypeImportedBy
		case filerelations.RelationTest:
			relType = RelationTypeTest
		case filerelations.RelationIndex:
			relType = RelationTypeIndex
		case filerelations.RelationTypes:
			relType = RelationTypeTypes
		default:
			continue
		}
		related = append(related, RelatedFile{
			Path:         r.Path,
			RelationType: relType,
		})
	}

	return related, nil
}
