import { useEffect, useState } from "react";

import { API_BASE, boundedFetch } from "../../api/client";
import { DEFAULT_TEMPLATE, TEMPLATES } from "./library";
import { node, type Template, type TemplateCategory, type TemplateNode } from "./spec";

/**
 * The studio's template shelf.
 *
 * A template is an authoring draft: it must be duplicable and saveable without
 * publishing a library release. The shelf therefore has two layers:
 *
 * - `localStorage` (`rcl.workbench.templates.v1`) is the synchronous cache that
 *   gives the first paint something to render and keeps the studio usable when
 *   the API is unreachable.
 * - the scenario API (`/api/templates`) is the durable store. On mount the
 *   shelf reads the server list, merges it over the cache, and uploads any
 *   cache-only templates **once** (guarded by `rcl.workbench.templates.migrated.v1`).
 *
 * Writes update the cache immediately and then run through a serialized
 * per-template queue. A failed write keeps the cache entry, records it as
 * pending, and emits `rcl:workbench-templates-error`; the next successful list
 * retries it. `local: true` still means "authoring draft", not "published".
 */
export const LOCAL_TEMPLATE_KEY = "rcl.workbench.templates.v1";
const EVENT = "rcl:workbench-templates";
/** Window event emitted when a write could not reach the API. */
export const TEMPLATE_ERROR_EVENT = "rcl:workbench-templates-error";
const MIGRATED_KEY = "rcl.workbench.templates.migrated.v1";
const PENDING_KEY = "rcl.workbench.templates.pending.v1";
const CHANNEL_NAME = "rcl.templates";
const API_PATH = "/api/templates";
const MAX_ERROR_LENGTH = 200;

/** Dispatched on `window` when a write is kept locally instead of saved. */
export interface TemplateSyncErrorDetail {
  message: string;
}

/**
 * True when the shelf is reading/writing the localStorage cache because the
 * API is unreachable or the collection route is absent (404). The studio can
 * surface "saved in this browser only" rather than claiming durability.
 */
let fallbackActive = false;

export function templatesUsingFallback(): boolean {
  return fallbackActive;
}

interface ApiTemplate {
  id?: unknown;
  name?: unknown;
  tagline?: unknown;
  category?: unknown;
  palette?: unknown;
  display?: unknown;
  radius?: unknown;
  spec?: unknown;
  slotHint?: unknown;
  idea?: unknown;
  createdAt?: unknown;
  updatedAt?: unknown;
}

function isTemplate(value: unknown): value is Template {
  if (!value || typeof value !== "object") return false;
  const record = value as Record<string, unknown>;
  return (
    typeof record.id === "string" &&
    typeof record.name === "string" &&
    typeof record.spec === "object" &&
    record.spec !== null &&
    typeof record.palette === "object" &&
    record.palette !== null
  );
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

// --- cache -----------------------------------------------------------------

export function readLocalTemplates(): Template[] {
  try {
    const raw = window.localStorage.getItem(LOCAL_TEMPLATE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(isTemplate).map((template) => ({ ...template, local: true }));
  } catch {
    return [];
  }
}

function writeLocalTemplates(templates: Template[]): void {
  try {
    window.localStorage.setItem(LOCAL_TEMPLATE_KEY, JSON.stringify(templates));
    window.dispatchEvent(new Event(EVENT));
  } catch {
    /* storage can be full or disabled; the shelf simply stays in memory */
  }
}

// --- pending-sync bookkeeping ---------------------------------------------

function readPending(): Set<string> {
  try {
    const raw = window.localStorage.getItem(PENDING_KEY);
    if (!raw) return new Set();
    const parsed = JSON.parse(raw) as unknown;
    if (!Array.isArray(parsed)) return new Set();
    return new Set(parsed.filter((value): value is string => typeof value === "string"));
  } catch {
    return new Set();
  }
}

function writePending(ids: Set<string>): void {
  try {
    window.localStorage.setItem(PENDING_KEY, JSON.stringify([...ids]));
  } catch {
    /* best effort: the next mount will simply try again */
  }
}

function markPending(id: string): void {
  const pending = readPending();
  if (pending.has(id)) return;
  pending.add(id);
  writePending(pending);
}

function clearPending(id: string): void {
  const pending = readPending();
  if (!pending.delete(id)) return;
  writePending(pending);
}

function migrationDone(): boolean {
  try {
    return window.localStorage.getItem(MIGRATED_KEY) === "1";
  } catch {
    return false;
  }
}

function markMigrationDone(): void {
  try {
    window.localStorage.setItem(MIGRATED_KEY, "1");
  } catch {
    /* the guard is an optimization; re-uploading an existing id is harmless */
  }
}

// --- API transport ---------------------------------------------------------

function apiUrl(path: string): string {
  return `${API_BASE.replace(/\/$/, "")}${path}`;
}

/** A thrown error (network failure or the shared deadline) is `null` so
 * callers can distinguish "not reachable" from a real HTTP failure. */
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

async function failureMessage(response: Response | null): Promise<string> {
  if (response === null) return "Templates API is unreachable";
  try {
    const payload = (await response.json()) as { error?: { message?: string } };
    if (payload.error?.message) return payload.error.message;
  } catch {
    /* fall through to the status message */
  }
  return `Templates API responded ${response.status}`;
}

function emitSyncError(message: string): void {
  const bounded =
    message.length > MAX_ERROR_LENGTH ? `${message.slice(0, MAX_ERROR_LENGTH - 1)}…` : message;
  try {
    window.dispatchEvent(
      new CustomEvent<TemplateSyncErrorDetail>(TEMPLATE_ERROR_EVENT, {
        detail: { message: bounded },
      }),
    );
  } catch {
    window.dispatchEvent(new Event(TEMPLATE_ERROR_EVENT));
  }
}

// --- server <-> shelf mapping ---------------------------------------------

function fromApi(raw: ApiTemplate): Template | null {
  if (typeof raw.id !== "string" || typeof raw.name !== "string") return null;
  if (!isPlainObject(raw.spec) || !isPlainObject(raw.palette)) return null;
  return {
    id: raw.id,
    name: raw.name,
    tagline: typeof raw.tagline === "string" ? raw.tagline : "",
    category: (typeof raw.category === "string"
      ? raw.category
      : DEFAULT_TEMPLATE.category) as TemplateCategory,
    palette: raw.palette as unknown as Template["palette"],
    display: raw.display === "serif" ? "serif" : "sans",
    radius: typeof raw.radius === "number" ? raw.radius : DEFAULT_TEMPLATE.radius,
    spec: raw.spec as unknown as TemplateNode,
    slotHint: typeof raw.slotHint === "string" ? raw.slotHint : "",
    idea: typeof raw.idea === "string" ? raw.idea : "",
    local: true,
  };
}

function toApiBody(template: Template): Record<string, unknown> {
  return {
    id: template.id,
    name: template.name,
    tagline: template.tagline,
    category: template.category,
    palette: template.palette,
    display: template.display,
    radius: template.radius,
    spec: template.spec,
    slotHint: template.slotHint,
    idea: template.idea,
  };
}

// --- serialized per-template queue ----------------------------------------

/** Tail of the write chain per template id. Two saves for one id never race. */
const queues = new Map<string, Promise<void>>();

function enqueue(id: string, task: () => Promise<boolean>): Promise<boolean> {
  const previous = queues.get(id) ?? Promise.resolve();
  const run = previous.then(task, task);
  const tail = run.then(
    () => undefined,
    () => undefined,
  );
  queues.set(id, tail);
  void tail.then(() => {
    if (queues.get(id) === tail) queues.delete(id);
  });
  return run;
}

/** Await every queued write. Exposed for tests and for a caller that wants to
 * flush before navigating away. */
export async function flushLocalTemplateWrites(): Promise<void> {
  while (queues.size > 0) {
    await Promise.all([...queues.values()]);
  }
}

/** Ids confirmed to exist on the server, so the queue picks PUT over POST. */
const serverKnown = new Set<string>();

async function syncTemplate(id: string): Promise<boolean> {
  const template = readLocalTemplates().find((entry) => entry.id === id);
  // Removed while queued: nothing to save. The delete path owns it.
  if (!template) return true;

  const body = JSON.stringify(toApiBody(template));
  const known = serverKnown.has(id);
  // Create posts to the collection; only an existing row is addressed by id.
  // Sending POST to /api/templates/{id} is a 405 and was silently keeping
  // every new template in this browser.
  let response = await callApi(known ? `${API_PATH}/${encodeURIComponent(id)}` : API_PATH, {
    method: known ? "PUT" : "POST",
    body,
  });

  if (response?.status === 404 && known) {
    // The row disappeared (another tab or an earlier 404); recreate it.
    response = await callApi(API_PATH, { method: "POST", body });
  } else if (response?.status === 409 && !known) {
    // Another tab created it between our list and this write.
    response = await callApi(`${API_PATH}/${encodeURIComponent(id)}`, { method: "PUT", body });
  }

  if (response?.ok) {
    serverKnown.add(id);
    clearPending(id);
    fallbackActive = false;
    broadcastChange();
    return true;
  }

  if (response === null || response.status === 404) {
    // Network failure, or the collection route does not exist: honest
    // browser-local operation, retried on the next successful list.
    fallbackActive = true;
  }
  markPending(id);
  emitSyncError(`Could not save template "${template.name}": ${await failureMessage(response)}`);
  return false;
}

export function upsertLocalTemplate(template: Template): void {
  const next = [
    ...readLocalTemplates().filter((entry) => entry.id !== template.id),
    { ...template, local: true },
  ];
  writeLocalTemplates(next);
  void enqueue(template.id, () => syncTemplate(template.id));
}

export function removeLocalTemplate(id: string): void {
  const existing = readLocalTemplates().find((entry) => entry.id === id);
  const next = readLocalTemplates().filter((entry) => entry.id !== id);
  writeLocalTemplates(next);
  if (!existing) return;

  void enqueue(id, async () => {
    const response = await callApi(`${API_PATH}/${encodeURIComponent(id)}`, {
      method: "DELETE",
    });
    if (response?.ok || response?.status === 404) {
      serverKnown.delete(id);
      clearPending(id);
      broadcastChange();
      return true;
    }
    if (response === null) fallbackActive = true;

    // The delete did not land. Restoring the entry is the honest outcome: the
    // shelf keeps showing the template, marked pending so the next successful
    // list re-saves it instead of silently forgetting it.
    if (!readLocalTemplates().some((entry) => entry.id === id)) {
      writeLocalTemplates([...readLocalTemplates(), existing]);
    }
    markPending(id);
    emitSyncError(
      `Could not delete template "${existing.name}": ${await failureMessage(response)}`,
    );
    return false;
  });
}

// --- cross-tab ------------------------------------------------------------

let channel: BroadcastChannel | null | undefined;

function getChannel(): BroadcastChannel | null {
  if (channel !== undefined) return channel;
  // Resolve through `window` on purpose: the Node runtime that hosts the test
  // environment also exposes BroadcastChannel, and this shelf must stay
  // dependency-free and browser-only.
  if (typeof window === "undefined" || typeof window.BroadcastChannel === "undefined") {
    channel = null;
    return channel;
  }
  channel = new window.BroadcastChannel(CHANNEL_NAME);
  return channel;
}

function broadcastChange(): void {
  try {
    getChannel()?.postMessage({ type: "templates-changed" });
  } catch {
    /* a tab that cannot broadcast still has the storage event */
  }
}

function subscribeChannel(onChange: () => void): () => void {
  const bus = getChannel();
  if (!bus) return () => {};
  const handler = (event: MessageEvent) => {
    if (isPlainObject(event.data) && event.data.type === "templates-changed") onChange();
  };
  bus.addEventListener("message", handler);
  return () => bus.removeEventListener("message", handler);
}

// --- list + one-time migration -------------------------------------------

/** Runs once per page load, shared by every `useLocalTemplates` mount. */
let initialSync: Promise<void> | null = null;

/**
 * Read the API list, merge it over the cache, upload cache-only templates once,
 * and retry any write that failed earlier. Safe to call repeatedly: the
 * one-time migration guard makes later calls a plain re-read.
 */
export async function refreshLocalTemplates(): Promise<void> {
  const response = await callApi(API_PATH);
  if (response === null || response.status === 404) {
    // No API: keep working from the cache exactly as before.
    fallbackActive = true;
    return;
  }
  if (!response.ok) {
    fallbackActive = false;
    emitSyncError(`Could not load templates: ${await failureMessage(response)}`);
    return;
  }

  let payload: { templates?: unknown };
  try {
    payload = (await response.json()) as { templates?: unknown };
  } catch {
    emitSyncError("Could not load templates: the API returned invalid JSON");
    return;
  }

  fallbackActive = false;
  const serverTemplates = Array.isArray(payload.templates)
    ? payload.templates
        .map((entry) => fromApi(entry as ApiTemplate))
        .filter((entry): entry is Template => entry !== null)
    : [];
  for (const template of serverTemplates) serverKnown.add(template.id);

  const cache = readLocalTemplates();
  const cacheById = new Map(cache.map((template) => [template.id, template]));
  const serverIds = new Set(serverTemplates.map((template) => template.id));
  const pending = readPending();

  // Server wins for an id in both — except when a local write for that id is
  // still pending, in which case the cache holds the newer, unsynced edit.
  const merged = [
    ...serverTemplates.map((template) =>
      pending.has(template.id) ? (cacheById.get(template.id) ?? template) : template,
    ),
    ...cache.filter((template) => !serverIds.has(template.id)),
  ];
  writeLocalTemplates(merged);

  // Forget pending ids whose template no longer exists anywhere.
  for (const id of pending) {
    if (!merged.some((template) => template.id === id)) clearPending(id);
  }

  const cacheOnly = merged.filter((template) => !serverIds.has(template.id));
  const uploadIds = migrationDone() ? [] : cacheOnly.map((template) => template.id);
  const retryIds = [...readPending()].filter(
    (id) => !uploadIds.includes(id) && merged.some((template) => template.id === id),
  );

  let migrationClean = true;
  for (const id of [...uploadIds, ...retryIds]) {
    const saved = await enqueue(id, () => syncTemplate(id));
    if (!saved && uploadIds.includes(id)) migrationClean = false;
  }
  // Mark the one-time migration complete only when every cache-only template
  // landed. A failure stays pending and is retried by the next successful list.
  if (!migrationDone() && migrationClean) markMigrationDone();
}

export function localTemplateId(name: string): string {
  const base = name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 32);
  return `${base || "template"}-${Date.now().toString(36)}`;
}

export function useLocalTemplates(): Template[] {
  const [templates, setTemplates] = useState<Template[]>(() => readLocalTemplates());
  useEffect(() => {
    const sync = () => setTemplates(readLocalTemplates());
    const onStorage = (event: StorageEvent) => {
      if (event.key !== null && event.key !== LOCAL_TEMPLATE_KEY) return;
      sync();
      // Another tab changed the shelf; fetch the authoritative list.
      void refreshLocalTemplates();
    };
    window.addEventListener(EVENT, sync);
    window.addEventListener("storage", onStorage);
    const unsubscribe = subscribeChannel(() => {
      void refreshLocalTemplates();
    });
    if (!initialSync) {
      // One list per page load, shared by every mount. If the API was absent
      // or writes are pending, release the memo so the next mount retries.
      initialSync = refreshLocalTemplates().then(() => {
        if (fallbackActive || readPending().size > 0) initialSync = null;
      });
    }
    initialSync.catch(() => {
      initialSync = null;
    });
    return () => {
      window.removeEventListener(EVENT, sync);
      window.removeEventListener("storage", onStorage);
      unsubscribe();
    };
  }, []);
  return templates;
}

export function templateShelf(locals: Template[] = []): Template[] {
  return [...TEMPLATES, ...locals];
}

export function resolveTemplate(id: string, locals: Template[] = []): Template | undefined {
  return templateShelf(locals).find((template) => template.id === id);
}

export function cloneTemplate(template: Template, name?: string): Template {
  const stamp = Date.now().toString(36);
  const rename = (current: TemplateNode): TemplateNode => ({
    ...current,
    id: `${template.id}.${stamp}.${current.id}`,
    children: current.children?.map(rename),
  });
  return {
    ...template,
    id: `${template.id}-copy-${stamp}`,
    name: name ?? `${template.name} copy`,
    tagline: template.tagline,
    spec: rename(template.spec),
    local: true,
  };
}

/** A deliberately quiet starter: one page, one stack, one title and a slot. */
export function blankTemplate(name = "Untitled page"): Template {
  const stamp = Date.now().toString(36);
  return {
    id: `untitled-${stamp}`,
    name,
    tagline: "Blank composition",
    category: DEFAULT_TEMPLATE.category,
    slotHint: "Place the component where the story needs it",
    idea: "Start from one clear idea, then add evidence around it.",
    palette: { ...DEFAULT_TEMPLATE.palette },
    display: DEFAULT_TEMPLATE.display,
    radius: DEFAULT_TEMPLATE.radius,
    local: true,
    spec: node(
      `untitled.${stamp}.page`,
      "page",
      { pad: 40 },
      [
        node(`untitled.${stamp}.stack`, "stack", { gap: 18 }, [
          node(`untitled.${stamp}.eyebrow`, "text", {
            text: "NEW PAGE",
            variant: "eyebrow",
            tone: "accent",
          }),
          node(`untitled.${stamp}.title`, "text", {
            text: "Name the one idea this page makes inevitable.",
            variant: "display",
          }),
          node(`untitled.${stamp}.body`, "text", {
            text: "Add the supporting evidence, then place a real component in the slot.",
            variant: "body",
            tone: "muted",
          }),
          node(`untitled.${stamp}.slot`, "slot", {
            hint: "Place the component where the story needs it",
            minH: 48,
          }),
        ]),
      ],
    ),
  };
}

export const TEMPLATE_PALETTES: Array<{
  id: string;
  label: string;
  palette: Template["palette"];
}> = [
  {
    id: "nocturne",
    label: "Nocturne",
    palette: DEFAULT_TEMPLATE.palette,
  },
  {
    id: "ivory",
    label: "Ivory",
    palette: {
      bg: "#f7f3ea",
      surface: "#fffdf8",
      surfaceAlt: "#efe8da",
      ink: "#1e1a16",
      muted: "#6d6458",
      line: "#ded4c3",
      accent: "#a8452a",
      accentInk: "#fffaf4",
      accentSoft: "#f2dfd6",
      secondary: "#234c44",
    },
  },
  {
    id: "cobalt",
    label: "Cobalt",
    palette: {
      bg: "#070d1f",
      surface: "#0e1730",
      surfaceAlt: "#152243",
      ink: "#f1f5ff",
      muted: "#9fb0d4",
      line: "#24345c",
      accent: "#7aa2ff",
      accentInk: "#07101f",
      accentSoft: "#1d2f5c",
      secondary: "#6fe3c4",
    },
  },
  {
    id: "clay",
    label: "Clay",
    palette: {
      bg: "#f3e7dc",
      surface: "#fbf3ea",
      surfaceAlt: "#ead9c9",
      ink: "#2a1c14",
      muted: "#7a6151",
      line: "#dcc5ae",
      accent: "#c4562f",
      accentInk: "#fff8f2",
      accentSoft: "#f0d3c1",
      secondary: "#3d5a4c",
    },
  },
  {
    id: "grove",
    label: "Grove",
    palette: {
      bg: "#0b1510",
      surface: "#112018",
      surfaceAlt: "#172c21",
      ink: "#eef7ef",
      muted: "#9db8a5",
      line: "#254433",
      accent: "#8ede6b",
      accentInk: "#0b1510",
      accentSoft: "#22452f",
      secondary: "#e6c37a",
    },
  },
];

export function withPalette(template: Template, palette: Template["palette"]): Template {
  return { ...template, palette: { ...palette } };
}
