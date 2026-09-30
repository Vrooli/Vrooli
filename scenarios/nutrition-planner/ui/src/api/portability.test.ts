import { beforeEach, describe, expect, it, vi } from "vitest";

const exportGroceriesCSV = vi.hoisted(() => vi.fn());
const exportRecipePDF = vi.hoisted(() => vi.fn());
const exportWeeklyPDF = vi.hoisted(() => vi.fn());
const exportWorkspace = vi.hoisted(() => vi.fn());
const exportRecipes = vi.hoisted(() => vi.fn());
const previewWorkspaceImport = vi.hoisted(() => vi.fn());
const applyWorkspaceImport = vi.hoisted(() => vi.fn());
const previewRecipesImport = vi.hoisted(() => vi.fn());
const applyRecipesImport = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ exportGroceriesCSV, exportRecipePDF, exportWeeklyPDF, exportWorkspace, exportRecipes, previewWorkspaceImport, applyWorkspaceImport, previewRecipesImport, applyRecipesImport }) }));
vi.mock("./client", () => ({ transport: {} }));

import { applyWorkspaceImport as applyBackup, exportGroceriesCSV as exportCSV, exportRecipePDF as exportRecipe, exportWeeklyPDF as exportWeek, exportWorkspace as exportBackup, exportRecipes as exportRecipeCollection, previewWorkspaceImport as previewBackup } from "./portability";
import { applyRecipesImport as applyRecipeImport, previewRecipesImport as previewRecipeImport } from "./portability";

describe("portability API", () => {
  beforeEach(() => vi.clearAllMocks());
  it("exports a revision-pinned grocery CSV", async () => {
    exportGroceriesCSV.mockResolvedValue({ filename: "daily-groceries.csv", contentCsv: "key,label\n", revision: 3n });
    await expect(exportCSV({ workspaceId: "w1", expectedRevision: 3n })).resolves.toEqual({ filename: "daily-groceries.csv", content: "key,label\n", revision: 3n });
    expect(exportGroceriesCSV).toHaveBeenCalledWith({ workspaceId: "w1", expectedRevision: 3n });
  });
  it("exports a recipe PDF artifact", async () => {
    exportRecipePDF.mockResolvedValue({ filename: "recipe-r1.pdf", content: new Uint8Array([37, 80, 68, 70]) });
    await expect(exportRecipe({ workspaceId: "w1", recipeId: "r1" })).resolves.toMatchObject({ filename: "recipe-r1.pdf" });
  });
  it("exports a revision-pinned weekly PDF artifact", async () => {
    exportWeeklyPDF.mockResolvedValue({ filename: "weekly-nutrition-plan.pdf", content: new Uint8Array([37, 80, 68, 70]), revision: 4n });
    await expect(exportWeek({ workspaceId: "w1", expectedRevision: 4n })).resolves.toMatchObject({ filename: "weekly-nutrition-plan.pdf", revision: 4n });
  });
  it("exports and previews native workspace backups", async () => {
    exportWorkspace.mockResolvedValue({ contentJson: "{}", omissions: ["inventory"] });
    previewWorkspaceImport.mockResolvedValue({ valid: true, format: "daily.workspace", schemaVersion: 2, recordCount: 3, recordKinds: ["workspace"], omissions: [], errors: [] });
    await expect(exportBackup("w1")).resolves.toMatchObject({ filename: "daily-workspace.json", omissions: ["inventory"] });
    await expect(previewBackup({ workspaceId: "w1", contentJson: "{}" })).resolves.toMatchObject({ valid: true, recordCount: 3 });
    applyWorkspaceImport.mockResolvedValue({ workspaceRevision: 4n, recipesApplied: 1, checkpointId: "cp1" });
    await expect(applyBackup({ workspaceId: "w1", expectedWorkspaceRevision: 3n, contentJson: "{}", idempotencyKey: "restore-1" })).resolves.toMatchObject({ workspaceRevision: 4n, checkpointId: "cp1" });
  });
  it("exports a native recipe collection", async () => {
    exportRecipes.mockResolvedValue({ contentJson: "{}", omissions: ["nutrition"] });
    await expect(exportRecipeCollection("w1")).resolves.toEqual({ filename: "daily-recipes.json", content: "{}", omissions: ["nutrition"] });
    expect(exportRecipes).toHaveBeenCalledWith({ workspaceId: "w1" });
  });
  it("previews and applies a native recipe collection", async () => {
    previewRecipesImport.mockResolvedValue({ valid: true, format: "daily.recipes", schemaVersion: 2, recipeCount: 2, duplicateCount: 1, conflictCount: 1, errors: [] });
    applyRecipesImport.mockResolvedValue({ workspaceRevision: 5n, recipesApplied: 1, recipesSkipped: 1, remappedIds: ["r1=r2"] });
    await expect(previewRecipeImport({ workspaceId: "w1", contentJson: "{}" })).resolves.toMatchObject({ recipeCount: 2, conflictCount: 1 });
    await expect(applyRecipeImport({ workspaceId: "w1", expectedWorkspaceRevision: 4n, contentJson: "{}", idempotencyKey: "import-1" })).resolves.toMatchObject({ workspaceRevision: 5n, recipesSkipped: 1 });
  });
});
