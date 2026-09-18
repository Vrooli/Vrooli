import { beforeEach, describe, expect, it, vi } from "vitest";

const exportGroceriesCSV = vi.hoisted(() => vi.fn());
const exportRecipePDF = vi.hoisted(() => vi.fn());
const exportWeeklyPDF = vi.hoisted(() => vi.fn());
const exportWorkspace = vi.hoisted(() => vi.fn());
const previewWorkspaceImport = vi.hoisted(() => vi.fn());
const applyWorkspaceImport = vi.hoisted(() => vi.fn());
vi.mock("@connectrpc/connect", () => ({ createClient: () => ({ exportGroceriesCSV, exportRecipePDF, exportWeeklyPDF, exportWorkspace, previewWorkspaceImport, applyWorkspaceImport }) }));
vi.mock("./client", () => ({ transport: {} }));

import { applyWorkspaceImport as applyBackup, exportGroceriesCSV as exportCSV, exportRecipePDF as exportRecipe, exportWeeklyPDF as exportWeek, exportWorkspace as exportBackup, previewWorkspaceImport as previewBackup } from "./portability";

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
});
