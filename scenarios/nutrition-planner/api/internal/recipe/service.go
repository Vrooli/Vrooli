package recipe

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

type Service interface {
	Create(context.Context, CreateInput) (Recipe, error)
	List(context.Context, string) ([]Recipe, error)
	Get(context.Context, string, string) (Recipe, error)
	Update(context.Context, UpdateInput) (Recipe, error)
}
type service struct{ repo Repository }

func NewService(r Repository) Service { return &service{r} }
func (s *service) Create(ctx context.Context, in CreateInput) (Recipe, error) {
	in.Name = strings.TrimSpace(in.Name)
	if err := validateName(in.Name); err != nil {
		return Recipe{}, err
	}
	if in.WorkspaceID == "" {
		return Recipe{}, fmt.Errorf("workspace_id: must not be empty")
	}
	for _, method := range in.Methods {
		if err := ValidateMethod(method, map[string]bool{}); err != nil {
			return Recipe{}, err
		}
	}
	return s.repo.Create(ctx, Recipe{WorkspaceID: in.WorkspaceID, Name: in.Name, Notes: in.Notes, SourceURL: in.SourceURL, SourceType: in.SourceType, OriginalText: in.OriginalText, IdempotencyKey: strings.TrimSpace(in.IdempotencyKey), RequestHash: RequestHash(in), Methods: in.Methods, Groups: in.Groups, RequiredAppliances: in.RequiredAppliances, AllergenEvidence: in.AllergenEvidence})
}
func (s *service) List(ctx context.Context, w string) ([]Recipe, error) { return s.repo.List(ctx, w) }
func (s *service) Get(ctx context.Context, id, w string) (Recipe, error) {
	return s.repo.Get(ctx, id, w)
}

func RequestHash(in CreateInput) string {
	methods, _ := json.Marshal(in.Methods)
	groups, _ := json.Marshal(in.Groups)
	appliances, _ := json.Marshal(in.RequiredAppliances)
	evidence, _ := json.Marshal(in.AllergenEvidence)
	h := sha256.Sum256([]byte(strings.Join([]string{strings.TrimSpace(in.WorkspaceID), strings.TrimSpace(in.Name), in.Notes, in.SourceURL, in.SourceType, in.OriginalText, string(methods), string(groups), string(appliances), string(evidence)}, "\x00")))
	return hex.EncodeToString(h[:])
}

func (s *service) Update(ctx context.Context, in UpdateInput) (Recipe, error) {
	in.Name = strings.TrimSpace(in.Name)
	if err := validateName(in.Name); err != nil {
		return Recipe{}, err
	}
	for _, method := range in.Methods {
		if err := ValidateMethod(method, map[string]bool{}); err != nil {
			return Recipe{}, err
		}
	}
	return s.repo.Update(ctx, in)
}
