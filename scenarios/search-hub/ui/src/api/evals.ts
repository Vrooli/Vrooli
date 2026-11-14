/**
 * UI ↔ search-hub API boundary for the eval (search-quality baseline) surface.
 *
 * Backed by one Connect service, EvalService:
 *   - ListSuites   — the provider-owned golden suites.
 *   - ListRuns     — a suite's run history (newest first), the trend source.
 *   - GetRun       — one immutable run's per-case detail.
 *   - CompareRuns  — A-vs-B per-case delta (the experimentation surface).
 *
 * Each call is a thin async function (not the raw client) so tests mock this
 * module the same way they mock ./api/search.
 */
import { createClient } from "@connectrpc/connect";
import { EvalService } from "@vrooli/proto-types/search-hub/v1/eval/eval_pb";
import type {
  EvalSuite,
  EvalRun,
  CompareRunsResponse,
} from "@vrooli/proto-types/search-hub/v1/eval/eval_pb";

import { transport } from "./client";

const evalClient = createClient(EvalService, transport);

/** List registered eval suites, optionally filtered to one provider. */
export async function listSuites(providerId = ""): Promise<EvalSuite[]> {
  const res = await evalClient.listSuites({ providerId });
  return res.suites;
}

/** List a suite's run history (newest first), optionally filtered by tag. */
export async function listRuns(suiteId: string, tag = ""): Promise<EvalRun[]> {
  const res = await evalClient.listRuns({ suiteId, tag, limit: 0 });
  return res.runs;
}

/** Fetch one immutable run by id (per-case detail). */
export async function getRun(runId: string): Promise<EvalRun> {
  const res = await evalClient.getRun({ runId });
  if (!res.run) throw new Error(`run ${runId} not found`);
  return res.run;
}

/** Compare two runs; returns both runs plus a per-case delta. */
export async function compareRuns(runA: string, runB: string): Promise<CompareRunsResponse> {
  return evalClient.compareRuns({ runA, runB });
}

export type { EvalSuite, EvalRun, CompareRunsResponse };
