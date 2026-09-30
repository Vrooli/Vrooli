package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	"github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain/domainconnect"
)

type documentationConnectService struct {
	domainconnect.UnimplementedDocumentationServiceHandler
	server *Server
}

var _ domainconnect.DocumentationServiceHandler = (*documentationConnectService)(nil)

func (s documentationConnectService) GetDocumentationManifest(_ context.Context, _ *connect.Request[domainv1.DocumentationManifestRequest]) (*connect.Response[domainv1.DocumentationManifestResponse], error) {
	data, err := os.ReadFile(filepath.Join(s.server.resolveDocsDir(), "manifest.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("docs manifest not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	var manifest DocsManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("invalid docs manifest format: %w", err))
	}
	result := &domainv1.DocumentationManifestResponse{Version: manifest.Version, Title: manifest.Title, DefaultDocument: manifest.DefaultDocument, PrimaryNavigation: manifest.Navigation.Primary, SecondaryNavigation: manifest.Navigation.Secondary}
	if manifest.Description != "" {
		result.Description = &manifest.Description
	}
	for _, section := range manifest.Sections {
		out := &domainv1.DocumentationSection{Id: section.ID, Title: section.Title}
		if section.Icon != "" {
			out.Icon = &section.Icon
		}
		if section.Description != "" {
			out.Description = &section.Description
		}
		for _, document := range section.Documents {
			value := &domainv1.DocumentationDocument{Path: document.Path, Title: document.Title}
			if document.Description != "" {
				value.Description = &document.Description
			}
			out.Documents = append(out.Documents, value)
		}
		result.Sections = append(result.Sections, out)
	}
	return connect.NewResponse(result), nil
}

func (s documentationConnectService) GetDocumentationContent(_ context.Context, request *connect.Request[domainv1.DocumentationContentRequest]) (*connect.Response[domainv1.DocumentationContentResponse], error) {
	path := request.Msg.GetPath()
	clean := filepath.Clean(path)
	if path == "" || filepath.IsAbs(clean) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid path"))
	}
	docsDir := s.server.resolveDocsDir()
	fullPath := filepath.Join(docsDir, clean)
	absDocs, _ := filepath.Abs(docsDir)
	absFull, _ := filepath.Abs(fullPath)
	absRoot, _ := filepath.Abs(detectVrooliRoot())
	if !strings.HasPrefix(absFull, absDocs) && !strings.HasPrefix(absFull, absRoot) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid path"))
	}
	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("document not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&domainv1.DocumentationContentResponse{Path: path, Content: string(data)}), nil
}
