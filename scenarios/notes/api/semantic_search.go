package main

import (
	"context"
	"fmt"

	schema "notes-api/internal/notes"
)

// semanticServing is injected by the Notes composition root after the shared
// owner supplies an active generation, exact recipe policy, runtime and vector
// adapter. A nil value means semantic serving is unavailable; no environment
// default or fixed collection is used.
var semanticServing *schema.SemanticServing

func setSemanticServing(service *schema.SemanticServing) {
	semanticServing = service
}

func fleetDocument(note Note) schema.FleetDocument {
	return schema.FleetDocument{
		NoteID: note.ID, UserID: note.UserID, Title: note.Title, Content: note.Content,
		Summary: note.Summary, FolderID: note.FolderID, IsPinned: note.IsPinned,
		IsArchived: note.IsArchived, IsFavorite: note.IsFavorite,
	}
}

func indexNoteInSemanticOwner(ctx context.Context, note Note, operationID string, sourceSequence int64) (schema.SemanticFleetOperationResult, error) {
	if semanticServing == nil {
		return schema.SemanticFleetOperationResult{}, fmt.Errorf("Notes semantic serving owner is not configured")
	}
	return semanticServing.UpsertNote(ctx, fleetDocument(note), operationID, sourceSequence)
}

func deleteNoteFromSemanticOwner(ctx context.Context, tenantID, noteID, operationID string, sourceSequence int64) (schema.SemanticFleetOperationResult, error) {
	if semanticServing == nil {
		return schema.SemanticFleetOperationResult{}, fmt.Errorf("Notes semantic serving owner is not configured")
	}
	return semanticServing.DeleteNote(ctx, tenantID, noteID, operationID, sourceSequence)
}
