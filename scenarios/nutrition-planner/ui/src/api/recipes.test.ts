import { beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { RecipeSchema } from "@vrooli/proto-types/nutrition-planner/v1/recipe/recipe_pb";

const list = vi.hoisted(() => vi.fn());
const createRecipe = vi.hoisted(() => vi.fn());
const get = vi.hoisted(() => vi.fn());
const getRevision = vi.hoisted(() => vi.fn());
const updateRecipe = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ listRecipes: list, createRecipe, getRecipe: get, getRecipeRevision: getRevision, updateRecipe }) }));
vi.mock("./client", () => ({ transport: {} }));

import { createRecipe as saveRecipe, getRecipe, getRecipeRevision, listRecipes, updateRecipe as editRecipe } from "./recipes";

describe("recipe API", () => {
  beforeEach(() => { vi.clearAllMocks(); });
  it("lists recipes through the generated client", async () => { const recipe = create(RecipeSchema, { name: "Soup" }); list.mockResolvedValue({ recipes: [recipe] }); await expect(listRecipes("w1")).resolves.toEqual([recipe]); });
  it("sends a durable capture key and returns the saved recipe", async () => { const recipe = create(RecipeSchema, { name: "Soup" }); createRecipe.mockResolvedValue({ recipe }); const result = await saveRecipe({ workspaceId: "w1", name: "Soup", notes: "rough" }); expect(createRecipe).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", name: "Soup", notes: "rough", idempotencyKey: expect.any(String) })); expect(result).toBe(recipe); });
  it("rejects a hollow create response", async () => { createRecipe.mockResolvedValue({}); await expect(saveRecipe({ workspaceId: "w1", name: "Soup" })).rejects.toThrow("no saved meal"); });
  it("gets a pinned recipe revision", async () => { const recipe = create(RecipeSchema, { id: "r1", revision: 2n, name: "Soup" }); get.mockResolvedValue({ recipe }); await expect(getRecipe("w1", "r1")).resolves.toBe(recipe); expect(get).toHaveBeenCalledWith({ workspaceId: "w1", id: "r1" }); });
  it("retrieves an explicitly requested immutable revision", async () => { const recipe = create(RecipeSchema, { id: "r1", revision: 1n, name: "Original soup" }); getRevision.mockResolvedValue({ recipe }); await expect(getRecipeRevision("w1", "r1", 1n)).resolves.toBe(recipe); expect(getRevision).toHaveBeenCalledWith({ workspaceId: "w1", id: "r1", revision: 1n }); });
  it("rejects a hollow revision response", async () => { getRevision.mockResolvedValue({}); await expect(getRecipeRevision("w1", "r1", 1n)).rejects.toThrow("no recipe revision"); });
  it("rejects a missing recipe response", async () => { get.mockResolvedValue({}); await expect(getRecipe("w1", "missing")).rejects.toThrow("no recipe"); });
  it("sends the expected revision for an edit", async () => { const recipe = create(RecipeSchema, { id: "r1", revision: 2n, name: "Soup revised" }); updateRecipe.mockResolvedValue({ recipe }); await expect(editRecipe({ workspaceId: "w1", id: "r1", expectedRevision: 1n, name: "Soup revised", notes: "updated" })).resolves.toBe(recipe); expect(updateRecipe).toHaveBeenCalledWith(expect.objectContaining({ workspaceId: "w1", id: "r1", expectedRevision: 1n, name: "Soup revised", notes: "updated", idempotencyKey: expect.any(String) })); });
});
