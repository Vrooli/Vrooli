import { createClient } from "@connectrpc/connect";
import { PortabilityService } from "@vrooli/proto-types/nutrition-planner/v1/portability/portability_pb";
import { transport } from "./client";

const client = createClient(PortabilityService, transport);

export async function exportGroceriesCSV(input: { workspaceId: string; expectedRevision: bigint }): Promise<{ filename: string; content: string; revision: bigint }> {
  const response = await client.exportGroceriesCSV(input);
  return { filename: response.filename, content: response.contentCsv, revision: response.revision };
}

export async function exportRecipePDF(input: { workspaceId: string; recipeId: string }): Promise<{ filename: string; content: Uint8Array }> {
  const response = await client.exportRecipePDF(input);
  return { filename: response.filename, content: response.content };
}

export async function exportWeeklyPDF(input: { workspaceId: string; expectedRevision: bigint }): Promise<{ filename: string; content: Uint8Array; revision: bigint }> {
  const response = await client.exportWeeklyPDF(input);
  return { filename: response.filename, content: response.content, revision: response.revision };
}

export async function exportWorkspace(workspaceId: string): Promise<{ filename: string; content: string; omissions: string[] }> {
  const response = await client.exportWorkspace({ workspaceId });
  return { filename: "daily-workspace.json", content: response.contentJson, omissions: response.omissions };
}

export async function previewWorkspaceImport(input: { workspaceId: string; contentJson: string }): Promise<{ valid: boolean; format: string; schemaVersion: number; recordCount: number; recordKinds: string[]; omissions: string[]; errors: string[] }> {
  const response = await client.previewWorkspaceImport(input);
  return { valid: response.valid, format: response.format, schemaVersion: response.schemaVersion, recordCount: response.recordCount, recordKinds: response.recordKinds, omissions: response.omissions, errors: response.errors };
}

export async function applyWorkspaceImport(input: { workspaceId: string; expectedWorkspaceRevision: bigint; contentJson: string; idempotencyKey: string }): Promise<{ workspaceRevision: bigint; recipesApplied: number; checkpointId: string }> {
  const response = await client.applyWorkspaceImport(input);
  return { workspaceRevision: response.workspaceRevision, recipesApplied: response.recipesApplied, checkpointId: response.checkpointId };
}
