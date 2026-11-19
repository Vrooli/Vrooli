/**
 * Designs: the durable objects the studio edits.
 *
 * This module is the single boundary between the studio and storage. The
 * bodies below call the scenario API (`/api/designs`); the localStorage store
 * remains **only** as a graceful fallback when the API is unreachable (fetch
 * throws) or the collection route is absent (404). A 4xx/5xx from a route that
 * does exist is a real error and is surfaced — a misconfigured API must not
 * look like a working local studio. It is not a second source of truth:
 * whenever the API answers, its document wins and the local draft is never
 * merged in.
 */
import { API_BASE, AUTHORING_TIMEOUT_MS, boundedFetch, createBoundedFetch } from "../api/client";
import type { TemplateNode, TemplateNodeType, TemplatePalette } from "./templates/spec";

export type DesignDevice = "desktop" | "tablet" | "phone";

export type DesignDisplay = "sans" | "serif";

/**
 * Design-owned chrome. Both keys are optional; an empty object means "use the
 * library defaults". The server validates and stores this object, so a design
 * can change its radius and typeface as well as its palette.
 */
export interface DesignOptions {
  radius?: number;
  display?: DesignDisplay;
}

export interface DesignFrame {
  id: string;
  name: string;
  device: DesignDevice;
  spec: TemplateNode;
}

export interface DesignDocument {
  id: string;
  name: string;
  updatedAt: string;
  revision: number;
  frames: DesignFrame[];
  theme: TemplatePalette;
  /** Chrome owned by the design; `{}` when unset. Older fixtures may omit it. */
  options?: DesignOptions;
  /** Optional design ↔ page binding; null/absent means unbound. */
  scenario?: string | null;
  page?: string | null;
}

export interface DesignSummary {
  id: string;
  name: string;
  updatedAt: string;
  frameCount: number;
  scenario?: string | null;
  page?: string | null;
}

export interface CreateDesignInput {
  name: string;
  frames: DesignFrame[];
  theme: TemplatePalette;
  /** Design-owned chrome; absent means `{}` (library defaults). */
  options?: DesignOptions;
  scenario?: string | null;
  page?: string | null;
}

export interface DesignFilter {
  scenario?: string;
  page?: string;
}

export type DesignAssetFidelity = "reference" | "captured";

export interface DesignAsset {
  id: string;
  designId: string;
  name: string;
  mime: string;
  bytes: number;
  width: number;
  height: number;
  fidelity: DesignAssetFidelity;
  addedAt: string;
  url: string;
}

/** Thrown by `saveDesign` when the server reports a stale revision. The
 * caller can adopt `current` (the authoritative document) or prompt. */
export class DesignConflictError extends Error {
  readonly current: DesignDocument;

  constructor(current: DesignDocument) {
    super("design was modified elsewhere");
    this.name = "DesignConflictError";
    this.current = current;
  }
}

const KEY = "rcl.designs.v1";
const API_PATH = "/api/designs";
const ALL_ASSETS_PATH = "/api/design-assets";

/**
 * True when the last operation had to read or write the localStorage
 * fallback. The studio can surface this ("local draft — API unavailable")
 * instead of presenting a browser-local design as durable.
 */
let fallbackActive = false;

export function designsUsingFallback(): boolean {
  return fallbackActive;
}

function apiUrl(path: string): string {
  return `${API_BASE.replace(/\/$/, "")}${path}`;
}

/** Fetch through the shared bounded transport. A thrown error (network
 * failure or the 5s deadline) is reported as `null` so callers can decide
 * whether it is fallback-worthy. */
async function callApi(path: string, init: RequestInit = {}): Promise<Response | null> {
  const headers = new Headers(init.headers);
  headers.set("accept", "application/json");
  if (init.body !== undefined) headers.set("content-type", "application/json");
  try {
    return await boundedFetch(apiUrl(path), { ...init, headers });
  } catch {
    return null;
  }
}

/** Only a network failure or a 404 on the collection route means "the API is
 * not there". Everything else is a real response the caller must handle. */
function shouldFallback(response: Response | null, collection: boolean): boolean {
  return response === null || (collection && response.status === 404);
}

async function readJson<T>(response: Response): Promise<T> {
  return (await response.json()) as T;
}

async function failureMessage(response: Response | null): Promise<string> {
  if (response === null) return "Designs API is unreachable";
  try {
    const payload = await readJson<{ error?: { message?: string } }>(response);
    if (payload.error?.message) return payload.error.message;
  } catch {
    /* fall through to the status message */
  }
  return `Designs API responded ${response.status}`;
}

async function requireJson<T>(response: Response | null): Promise<T> {
  if (!response?.ok) throw new Error(await failureMessage(response));
  return readJson<T>(response);
}

function markApiAnswered(): void {
  fallbackActive = false;
}

function localFallback<T>(operation: () => T): T {
  fallbackActive = true;
  return operation();
}

// --- local fallback store --------------------------------------------------

function readAll(): DesignDocument[] {
  try {
    const raw = window.localStorage.getItem(KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as DesignDocument[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

function writeAll(designs: DesignDocument[]): void {
  try {
    window.localStorage.setItem(KEY, JSON.stringify(designs));
  } catch {
    /* local drafts are best-effort; the API is the durable store */
  }
}

function newId(prefix: string): string {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

function listLocal(): DesignSummary[] {
  return readAll()
    .map(({ id, name, updatedAt, frames, scenario, page }) => ({
      id,
      name,
      updatedAt,
      frameCount: frames.length,
      scenario: scenario ?? null,
      page: page ?? null,
    }))
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
}

function getLocal(id: string): DesignDocument {
  const found = readAll().find((design) => design.id === id);
  if (!found) throw new Error(`Design not found: ${id}`);
  return found;
}

function createLocal(input: CreateDesignInput): DesignDocument {
  const design: DesignDocument = {
    id: newId("design"),
    name: input.name,
    updatedAt: new Date().toISOString(),
    revision: 1,
    frames: input.frames,
    theme: input.theme,
    options: input.options ?? {},
    scenario: input.scenario ?? null,
    page: input.page ?? null,
  };
  writeAll([design, ...readAll()]);
  return design;
}

function saveLocal(doc: DesignDocument): DesignDocument {
  // A browser-only write does not invent a revision: the server still owns
  // that number. Keeping it means the reconnect retry can PUT against the
  // revision the document actually has and succeed instead of conflicting.
  const next: DesignDocument = { ...doc, updatedAt: new Date().toISOString() };
  const all = readAll();
  const index = all.findIndex((design) => design.id === doc.id);
  if (index === -1) all.unshift(next);
  else all[index] = next;
  writeAll(all);
  return next;
}

function deleteLocal(id: string): void {
  writeAll(readAll().filter((design) => design.id !== id));
}

// --- public design surface -------------------------------------------------

export async function listDesigns(filter: DesignFilter = {}): Promise<DesignSummary[]> {
  const query = new URLSearchParams();
  if (filter.scenario) query.set("scenario", filter.scenario);
  if (filter.page) query.set("page", filter.page);
  const suffix = query.toString() ? `?${query.toString()}` : "";
  const response = await callApi(`${API_PATH}${suffix}`);
  if (response?.ok) {
    markApiAnswered();
    const payload = await readJson<{ designs?: DesignSummary[] }>(response);
    return [...(payload.designs ?? [])].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  }
  if (!shouldFallback(response, true)) throw new Error(await failureMessage(response));
  return localFallback(listLocal);
}

export async function getDesign(id: string): Promise<DesignDocument> {
  const response = await callApi(`${API_PATH}/${encodeURIComponent(id)}`);
  if (response?.ok) {
    markApiAnswered();
    const payload = await readJson<{ design?: DesignDocument }>(response);
    if (!payload.design) throw new Error(`Design not found: ${id}`);
    return payload.design;
  }
  if (!shouldFallback(response, false)) throw new Error(await failureMessage(response));
  return localFallback(() => getLocal(id));
}

export async function createDesign(input: CreateDesignInput): Promise<DesignDocument> {
  const response = await callApi(API_PATH, {
    method: "POST",
    // options is optional on the type but always sent as an object so the
    // server never has to guess whether the client meant "unset".
    body: JSON.stringify({ ...input, options: input.options ?? {} }),
  });
  if (response?.ok) {
    markApiAnswered();
    const payload = await readJson<{ design?: DesignDocument }>(response);
    if (!payload.design) throw new Error("Designs API returned no design");
    return payload.design;
  }
  if (!shouldFallback(response, true)) throw new Error(await failureMessage(response));
  return localFallback(() => createLocal(input));
}

export interface DesignWriteResult {
  design: DesignDocument;
  /** True when the write only reached this browser, not the scenario API. */
  local: boolean;
}

/**
 * Save and say **where** it landed.
 *
 * `saveDesign` alone cannot distinguish "the server accepted it" from "the
 * API was unreachable and localStorage took it". The studio needs the
 * difference: a local-only write must not broadcast a revision to other tabs,
 * must not read as clean for cross-tab reconciliation, and must be retried
 * when the API returns.
 */
export async function putDesign(doc: DesignDocument): Promise<DesignWriteResult> {
  const response = await callApi(`${API_PATH}/${encodeURIComponent(doc.id)}`, {
    method: "PUT",
    // Sending the revision turns the save into an optimistic-concurrency
    // check. The server's 409 carries the document another tab wrote.
    body: JSON.stringify({
      name: doc.name,
      frames: doc.frames,
      theme: doc.theme,
      options: doc.options ?? {},
      scenario: doc.scenario ?? null,
      page: doc.page ?? null,
      revision: doc.revision,
    }),
  });
  if (response?.status === 409) {
    const payload = await readJson<{ design?: DesignDocument }>(response);
    if (!payload.design) throw new Error("Designs API reported a conflict without the current design");
    markApiAnswered();
    throw new DesignConflictError(payload.design);
  }
  if (response?.ok) {
    markApiAnswered();
    const payload = await readJson<{ design?: DesignDocument }>(response);
    if (!payload.design) throw new Error("Designs API returned no design");
    return { design: payload.design, local: false };
  }
  if (response?.status === 404) {
    // Either another tab deleted it, or it was created while the API was
    // unreachable and only has a browser id. Both are actionable; "responded
    // 404" is not.
    throw new Error(
      "This design is not on the server. It may have been deleted elsewhere or created offline; reload the studio to recover.",
    );
  }
  if (!shouldFallback(response, false)) throw new Error(await failureMessage(response));
  return { design: localFallback(() => saveLocal(doc)), local: true };
}

export async function saveDesign(doc: DesignDocument): Promise<DesignDocument> {
  return (await putDesign(doc)).design;
}

export async function deleteDesign(id: string): Promise<void> {
  const response = await callApi(`${API_PATH}/${encodeURIComponent(id)}`, { method: "DELETE" });
  if (response?.ok || response?.status === 404) {
    markApiAnswered();
    return;
  }
  if (!shouldFallback(response, false)) throw new Error(await failureMessage(response));
  localFallback(() => deleteLocal(id));
}

// --- design assets ---------------------------------------------------------

export function designAssetUrl(asset: DesignAsset): string {
  if (/^https?:\/\//i.test(asset.url)) return asset.url;
  return apiUrl(asset.url);
}

function assetsPath(designId: string): string {
  return `${API_PATH}/${encodeURIComponent(designId)}/assets`;
}

export async function listDesignAssets(designId: string): Promise<DesignAsset[]> {
  const response = await callApi(assetsPath(designId));
  const payload = await requireJson<{ assets?: DesignAsset[] }>(response);
  return payload.assets ?? [];
}

export async function listAllDesignAssets(): Promise<DesignAsset[]> {
  const response = await callApi(ALL_ASSETS_PATH);
  const payload = await requireJson<{ assets?: DesignAsset[] }>(response);
  return payload.assets ?? [];
}

function readFileAsBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result !== "string") {
        reject(new Error("Could not read the selected file"));
        return;
      }
      const comma = reader.result.indexOf(",");
      resolve(comma >= 0 ? reader.result.slice(comma + 1) : reader.result);
    };
    reader.onerror = () => reject(reader.error ?? new Error("Could not read the selected file"));
    reader.readAsDataURL(file);
  });
}

export async function uploadDesignAsset(
  designId: string,
  file: File,
  fidelity: DesignAssetFidelity = "reference",
): Promise<DesignAsset> {
  const dataBase64 = await readFileAsBase64(file);
  const response = await callApi(assetsPath(designId), {
    method: "POST",
    body: JSON.stringify({
      name: file.name,
      mime: file.type || "application/octet-stream",
      dataBase64,
      fidelity,
    }),
  });
  const payload = await requireJson<{ asset?: DesignAsset }>(response);
  if (!payload.asset) throw new Error("Designs API returned no asset");
  return payload.asset;
}

export async function deleteDesignAsset(designId: string, assetId: string): Promise<void> {
  const response = await callApi(`${assetsPath(designId)}/${encodeURIComponent(assetId)}`, {
    method: "DELETE",
  });
  if (response?.ok || response?.status === 404) return;
  throw new Error(await failureMessage(response));
}


/* ── AI proposals ────────────────────────────────────────────────────────── */

/**
 * A model-generated patch. The wire shape is deliberately flat (one `kind`
 * plus optional fields) because that is what models follow reliably; the
 * studio validates every field before it touches the document.
 */
export interface DesignPatchOperation {
  kind: "setText" | "setProp" | "addLayer" | "removeLayer" | "moveLayer" | "setTheme";
  nodeId?: string;
  parentId?: string;
  key?: string;
  value?: string | number | boolean;
  text?: string;
  node?: DesignNewLayer;
  index?: number;
}

export interface DesignNewLayer {
  type: TemplateNodeType;
  props: Record<string, string | number | boolean>;
  children?: DesignNewLayer[];
}

export interface DesignProposal {
  summary: string;
  operations: DesignPatchOperation[];
  /** The design revision this proposal was computed against. */
  revision: number;
  operationId: string;
  state: string;
  provider?: string;
  model?: string;
  inputTokens?: number;
  outputTokens?: number;
}

export type DesignProposalErrorCode =
  | "ai_unavailable"
  | "ai_failed"
  | "invalid_proposal"
  | "invalid_argument"
  | "conflict"
  | "unknown";

export class DesignProposalError extends Error {
  readonly code: DesignProposalErrorCode;
  readonly status: number;

  constructor(code: DesignProposalErrorCode, status: number, message: string) {
    super(message);
    this.name = "DesignProposalError";
    this.code = code;
    this.status = status;
  }
}

/**
 * A proposal takes as long as a local model needs. Five seconds — the bound
 * every read uses — would report a healthy gateway as broken, so assistance
 * gets the authoring deadline and an explicit timeout error.
 */
const assistFetch = createBoundedFetch(AUTHORING_TIMEOUT_MS);

const PROPOSAL_CODES: readonly DesignProposalErrorCode[] = [
  "ai_unavailable",
  "ai_failed",
  "invalid_proposal",
  "invalid_argument",
  "conflict",
];

export interface ProposeInput {
  prompt: string;
  scope: { kind: "design" | "frame" | "layer"; frameId?: string; nodeId?: string };
  revision: number;
  idempotencyKey: string;
}

export async function proposeDesignPatch(
  designId: string,
  input: ProposeInput,
  signal?: AbortSignal,
): Promise<DesignProposal> {
  let response: Response;
  try {
    response = await assistFetch(apiUrl(`${API_PATH}/${encodeURIComponent(designId)}/assist`), {
      method: "POST",
      headers: { accept: "application/json", "content-type": "application/json" },
      body: JSON.stringify(input),
      signal,
    });
  } catch (error) {
    if (signal?.aborted) {
      throw new DesignProposalError("ai_unavailable", 0, "The request was canceled.");
    }
    const detail = error instanceof Error ? error.message : "The AI gateway did not answer.";
    throw new DesignProposalError("ai_unavailable", 0, detail);
  }

  if (!response.ok) {
    let code: DesignProposalErrorCode = "unknown";
    let message = `The AI gateway responded ${response.status}.`;
    try {
      const payload = (await response.json()) as { error?: { code?: string; message?: string } };
      if (payload.error?.code && PROPOSAL_CODES.includes(payload.error.code as DesignProposalErrorCode)) {
        code = payload.error.code as DesignProposalErrorCode;
      }
      if (payload.error?.message) message = payload.error.message;
    } catch {
      /* keep the status message */
    }
    throw new DesignProposalError(code, response.status, message);
  }

  const payload = (await response.json()) as {
    operation?: { id?: string; state?: string; provider?: string; model?: string; inputTokens?: number; outputTokens?: number };
    revision?: number;
    proposal?: { summary?: string; operations?: DesignPatchOperation[] };
  };
  if (!payload.proposal || !Array.isArray(payload.proposal.operations) || payload.proposal.operations.length === 0) {
    throw new DesignProposalError("invalid_proposal", response.status, "The gateway returned no patch.");
  }
  return {
    summary: payload.proposal.summary ?? "",
    operations: payload.proposal.operations,
    revision: payload.revision ?? input.revision,
    operationId: payload.operation?.id ?? "",
    state: payload.operation?.state ?? "completed",
    provider: payload.operation?.provider,
    model: payload.operation?.model,
    inputTokens: payload.operation?.inputTokens,
    outputTokens: payload.operation?.outputTokens,
  };
}
