package recipe

import (
	connect "github.com/vrooli/vrooli/packages/proto/gen/go/nutrition-planner/v1/recipe/recipe_v1connect"
	"nutrition-planner/internal/module"
)

var Endpoints = []module.EndpointDescriptor{{ID: "recipe_list", Path: connect.RecipeServiceListRecipesProcedure, Method: "POST", Summary: "List recipes", Category: "recipe"}, {ID: "recipe_create", Path: connect.RecipeServiceCreateRecipeProcedure, Method: "POST", Summary: "Create a recipe draft", Category: "recipe"}, {ID: "recipe_get", Path: connect.RecipeServiceGetRecipeProcedure, Method: "POST", Summary: "Get a recipe", Category: "recipe"}, {ID: "recipe_update", Path: connect.RecipeServiceUpdateRecipeProcedure, Method: "POST", Summary: "Create a new recipe revision", Category: "recipe"}}
