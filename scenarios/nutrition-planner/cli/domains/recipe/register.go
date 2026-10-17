package recipe

import (
	"fmt"

	"github.com/vrooli/cli-core/cliapp"
)

const GroupName = "recipe"

func Register(core *cliapp.ScenarioApp, manifest []byte) (cliapp.SubcommandGroup, error) {
	h := newHandlers(core)
	group, err := cliapp.LoadFromManifestPrimitives(manifest, GroupName, map[string]cliapp.PrimitiveHandler{
		"RecipeService.ListRecipes":  cliapp.ProtoList(h.listCall, h.listReport),
		"RecipeService.CreateRecipe": cliapp.ProtoMutation(h.createCall, h.createReport),
	})
	if err != nil {
		return cliapp.SubcommandGroup{}, fmt.Errorf("recipe: load from manifest: %w", err)
	}
	return group, nil
}
