package recipe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	recipev1 "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe"
	recipeconnect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe/recipe_v1connect"
)

type handlers struct {
	client recipeconnect.RecipeServiceClient
}

func newHandlers(core *cliapp.ScenarioApp) *handlers {
	httpClient, baseURL := cliapp.NewConnectHTTPClient(core)
	return &handlers{client: recipeconnect.NewRecipeServiceClient(httpClient, baseURL)}
}

func (h *handlers) listCall(ctx cliapp.OperationContext) (*recipev1.ListRecipesResponse, error) {
	resp, err := h.client.ListRecipes(context.Background(), connect.NewRequest(&recipev1.ListRecipesRequest{WorkspaceId: ctx.Flag("workspace-id")}))
	if err != nil {
		return nil, cliapp.WrapAPIError("list recipes", err, nil)
	}
	if resp == nil || resp.Msg == nil {
		return nil, fmt.Errorf("server returned no recipe response")
	}
	return resp.Msg, nil
}

func (h *handlers) listReport(_ cliapp.OperationContext, msg *recipev1.ListRecipesResponse) cliapp.ListReport {
	results := make([]string, 0, len(msg.Recipes))
	for _, item := range msg.Recipes {
		if item != nil {
			results = append(results, fmt.Sprintf("%s — %s [revision=%d status=%s]", item.Id, item.Name, item.Revision, item.Status))
		}
	}
	return cliapp.ListReport{Summary: []string{fmt.Sprintf("Found %d recipe(s).", len(msg.Recipes))}, ResultsHeading: "Recipes", Results: results}
}

func (h *handlers) createCall(ctx cliapp.OperationContext) (*recipev1.CreateRecipeResponse, error) {
	key, err := idempotencyKey()
	if err != nil {
		return nil, err
	}
	resp, err := h.client.CreateRecipe(context.Background(), connect.NewRequest(&recipev1.CreateRecipeRequest{Name: ctx.Flag("name"), Notes: ctx.Flag("notes"), SourceUrl: ctx.Flag("source-url"), SourceType: "manual", WorkspaceId: ctx.Flag("workspace-id"), IdempotencyKey: key}))
	if err != nil {
		return nil, cliapp.WrapAPIError("create recipe", err, nil)
	}
	if resp == nil || resp.Msg == nil || resp.Msg.Recipe == nil {
		return nil, fmt.Errorf("server returned no recipe")
	}
	return resp.Msg, nil
}

func (h *handlers) createReport(_ cliapp.OperationContext, msg *recipev1.CreateRecipeResponse) cliapp.MutationReport {
	return cliapp.MutationReport{Result: []string{fmt.Sprintf("Created recipe %s.", msg.Recipe.Id)}, Changes: []string{fmt.Sprintf("%s — %s [revision=%d]", msg.Recipe.Id, msg.Recipe.Name, msg.Recipe.Revision)}}
}

func idempotencyKey() (string, error) {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate idempotency key: %w", err)
	}
	return hex.EncodeToString(bytes[:]), nil
}
