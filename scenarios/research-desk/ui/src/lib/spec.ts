import type { BlockerChoice, PacketLimits } from "../api/launch";

/**
 * The launch form as the packet builder's spec (packet.py SPEC_KEYS). Pure, so
 * the exact JSON the desk sends is covered by a unit test. Optional fields are
 * left out rather than sent empty; packet.py fills its defaults.
 */
export interface LaunchForm {
  topics: string[];
  order: Record<string, string[]>;
  notes: Record<string, string>;
  emphasis: string;
  quotes: string[];
  /** The file the quotes were filled from while the operator has not edited them; "" once typed. */
  quotesFrom: string;
  allowanceHours: number;
  supervisorModel: string;
  supervisorEffort: string;
  ownerModelTier: string;
  readFirst: Record<string, string>;
  /** Topics whose offered read-first the operator unticked; never offered again. */
  readFirstDeclined: string[];
  /**
   * Per earlier commission: what this launch does about it ("" = undecided).
   * Carried by the preflight and launch requests, not the spec.
   */
  choices: Record<string, BlockerChoice | "">;
  slug: string;
}

export const LIVE_WORKSTREAM = new Set(["open", "active", "approved"]);

/** The form's starting values: the models and the allowance the launch options name. */
export function emptyForm(defaults: { supervisorModel: string; supervisorEffort: string; ownerModelTier: string; allowanceHours: number }): LaunchForm {
  return { topics: [], order: {}, notes: {}, emphasis: "", quotes: [""], quotesFrom: "", readFirst: {}, readFirstDeclined: [], choices: {}, slug: "", ...defaults };
}

/** What still has to be filled in before a preview makes sense. */
export function missing(form: LaunchForm, limits: Pick<PacketLimits, "allowanceHoursMin" | "allowanceHoursMax">): ("topics" | "quotes" | "allowance")[] {
  const out: ("topics" | "quotes" | "allowance")[] = [];
  if (form.topics.length === 0) out.push("topics");
  if (!form.quotes.some((q) => q.trim())) out.push("quotes");
  if (!Number.isInteger(form.allowanceHours) || form.allowanceHours < limits.allowanceHoursMin || form.allowanceHours > limits.allowanceHoursMax) out.push("allowance");
  return out;
}

export function toSpec(form: LaunchForm): Record<string, unknown> {
  const picked = <T,>(map: Record<string, T>, keep: (value: T) => boolean) =>
    Object.fromEntries(form.topics.filter((t) => t in map && keep(map[t] as T)).map((t) => [t, map[t] as T]));
  const spec: Record<string, unknown> = {
    topics: form.topics,
    allowance_hours: form.allowanceHours,
    emphasis: form.emphasis.trim(),
    stance_quotes: form.quotes.map((q) => q.trim()).filter(Boolean),
    supervisor_model: form.supervisorModel,
    supervisor_effort: form.supervisorEffort,
    owner_model_tier: form.ownerModelTier,
  };
  const order = picked(form.order, (ids) => ids.length > 0);
  const notes = Object.fromEntries(Object.entries(picked(form.notes, (n) => n.trim() !== "")).map(([k, v]) => [k, v.trim()]));
  const readFirst = picked(form.readFirst, (p) => p !== "");
  if (Object.keys(order).length > 0) spec.order = order;
  if (Object.keys(notes).length > 0) spec.card_notes = notes;
  if (Object.keys(readFirst).length > 0) spec.read_first = readFirst;
  if (form.slug.trim()) spec.slug = form.slug.trim();
  return spec;
}

/** Move one item up (-1) or down (+1) in a list, for keyboard- and thumb-friendly ordering. */
export function move<T>(list: T[], index: number, by: -1 | 1): T[] {
  const to = index + by;
  if (to < 0 || to >= list.length) return list;
  const out = [...list];
  [out[index], out[to]] = [out[to] as T, out[index] as T];
  return out;
}

/** Where the form waits while the operator looks elsewhere: this tab only. */
const STORE_KEY = "research-desk.launch-form";

export function saveForm(form: LaunchForm): void {
  try {
    sessionStorage.setItem(STORE_KEY, JSON.stringify(form));
  } catch {
    // Storage full or blocked: the form still works, it just does not survive leaving.
  }
}

export function clearSavedForm(): void {
  try {
    sessionStorage.removeItem(STORE_KEY);
  } catch {
    // Nothing to clear.
  }
}

/**
 * The saved form over the defaults, field by field, keeping only values of
 * the right shape; a model the options no longer offer falls back to the
 * default.
 */
export function loadForm(defaults: Parameters<typeof emptyForm>[0], offered: { supervisorModels: string[]; supervisorEfforts: string[]; ownerModelTiers: string[] }): LaunchForm {
  const base = emptyForm(defaults);
  let saved: Partial<Record<keyof LaunchForm, unknown>> = {};
  try {
    const raw = sessionStorage.getItem(STORE_KEY);
    const parsed: unknown = raw ? JSON.parse(raw) : null;
    if (parsed && typeof parsed === "object") saved = parsed;
  } catch {
    return base;
  }
  const strings = (v: unknown) => Array.isArray(v) && v.every((x) => typeof x === "string");
  const record = (v: unknown) => v !== null && typeof v === "object" && !Array.isArray(v);
  const pick = <K extends keyof LaunchForm>(key: K, ok: (v: unknown) => boolean): LaunchForm[K] => (ok(saved[key]) ? (saved[key] as LaunchForm[K]) : base[key]);
  const oneOf = (list: string[]) => (v: unknown) => typeof v === "string" && list.includes(v);
  return {
    topics: pick("topics", strings),
    order: pick("order", record),
    notes: pick("notes", record),
    emphasis: pick("emphasis", (v) => typeof v === "string"),
    quotes: pick("quotes", (v) => strings(v) && Array.isArray(v) && v.length > 0),
    quotesFrom: pick("quotesFrom", (v) => typeof v === "string"),
    allowanceHours: pick("allowanceHours", (v) => typeof v === "number"),
    supervisorModel: pick("supervisorModel", oneOf(offered.supervisorModels)),
    supervisorEffort: pick("supervisorEffort", oneOf(offered.supervisorEfforts)),
    ownerModelTier: pick("ownerModelTier", oneOf(offered.ownerModelTiers)),
    readFirst: pick("readFirst", record),
    readFirstDeclined: pick("readFirstDeclined", strings),
    choices: pick("choices", record),
    slug: pick("slug", (v) => typeof v === "string"),
  };
}
