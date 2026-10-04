import { createClient } from "@connectrpc/connect";
import { RecipeService, type Recipe } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";
import { transport } from "./client";

const client = createClient(RecipeService, transport);

export async function listRecipes(workspaceId: string): Promise<Recipe[]> {
  const response = await client.listRecipes({ workspaceId });
  return response.recipes;
}

export async function getRecipe(workspaceId: string, id: string): Promise<Recipe> {
  const response = await client.getRecipe({ workspaceId, id });
  if (!response.recipe) throw new Error("The API returned no recipe.");
  return response.recipe;
}

export async function getRecipeRevision(workspaceId: string, id: string, revision: bigint): Promise<Recipe> {
  const response = await client.getRecipeRevision({ workspaceId, id, revision });
  if (!response.recipe) throw new Error("The API returned no recipe revision.");
  return response.recipe;
}

export async function createRecipe(input: { workspaceId: string; name: string; notes?: string; originalText?: string; sourceUrl?: string; sourceType?: string }): Promise<Recipe> {
  const response = await client.createRecipe({
    name: input.name,
    notes: input.notes ?? "",
    originalText: input.originalText ?? "",
    sourceUrl: input.sourceUrl ?? "",
    sourceType: input.sourceType ?? "manual",
    workspaceId: input.workspaceId,
    idempotencyKey: crypto.randomUUID(),
  });
  if (!response.recipe) throw new Error("The API returned no saved meal.");
  return response.recipe;
}

export async function updateRecipe(input: { workspaceId: string; id: string; expectedRevision: bigint; name: string; notes?: string; originalText?: string; sourceUrl?: string; sourceType?: string }): Promise<Recipe> {
  const response = await client.updateRecipe({
    workspaceId: input.workspaceId,
    id: input.id,
    expectedRevision: input.expectedRevision,
    name: input.name,
    notes: input.notes ?? "",
    originalText: input.originalText ?? "",
    sourceUrl: input.sourceUrl ?? "",
    sourceType: input.sourceType ?? "manual",
    idempotencyKey: crypto.randomUUID(),
  });
  if (!response.recipe) throw new Error("The API returned no updated meal.");
  return response.recipe;
}
