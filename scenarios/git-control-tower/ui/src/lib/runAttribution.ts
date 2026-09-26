import type { ApprovedChangeFile } from "./api-types-operations";

export interface RunAttribution {
  runId: string;
  owner?: string;
  changeType?: string;
  appliedAt?: string;
}

const palette = ["#38bdf8", "#a78bfa", "#f472b6", "#22d3ee", "#c084fc", "#818cf8"];

export function buildRunIndex(files: ApprovedChangeFile[], provenance: Array<{ runId: string; sandboxOwner?: string; latestAppliedAt?: string; files: Array<{ relativePath: string; appliedAt?: string; changeType?: string }> }> = []): Map<string, RunAttribution> {
  const result = new Map<string, RunAttribution>();
  for (const file of files) {
    const runId = file.agentManagerRunId || file.sandboxId;
    if (!runId) continue;
    result.set(file.relativePath, { runId, owner: file.sandboxOwner, changeType: file.changeType });
  }
  for (const run of provenance) {
    for (const file of run.files) {
      const existing = result.get(file.relativePath);
      const runId = run.runId || existing?.runId;
      if (!runId) continue;
      result.set(file.relativePath, {
        runId,
        owner: existing?.owner || run.sandboxOwner,
        changeType: existing?.changeType || file.changeType,
        appliedAt: file.appliedAt || run.latestAppliedAt,
      });
    }
  }
  return result;
}

export function listRuns(index: Map<string, RunAttribution>) {
  const result: Array<RunAttribution & { fileCount: number }> = [];
  const byID = new Map<string, (typeof result)[number]>();
  for (const attribution of index.values()) {
    let run = byID.get(attribution.runId);
    if (!run) {
      run = { ...attribution, fileCount: 0 };
      byID.set(attribution.runId, run);
      result.push(run);
    }
    run.fileCount += 1;
  }
  return result;
}

export function runHue(runId: string): string {
  let hash = 0;
  for (let index = 0; index < runId.length; index += 1) hash = (hash * 31 + runId.charCodeAt(index)) >>> 0;
  return palette[hash % palette.length] || "#38bdf8";
}

export function shortRunId(runId: string): string { return runId.slice(0, 8); }
