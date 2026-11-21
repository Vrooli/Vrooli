// The rows of the `usage.annotation_gate` measure, read into one shape. The API
// owns every number; this file only parses cells and words a gate in plain
// language. A cell the API left out stays undefined, so a missing measurement
// never reads as zero.

/** One row of the measure: every cell is a string on the wire. */
export type GateCells = Record<string, string>;

export type GateState = "proven" | "partial" | "unproven" | "not_observed" | "unavailable";

/** How a proof's probabilities are read: fitted to the hold-out, as the model gave them, or no model at all. */
export type Calibration = "calibrated" | "raw" | "rule";

/** What one (model, resource role) answered for a label and what its own gate says. */
export interface GateModel {
  model: string;
  role: string;
  observed: number;
  proven: boolean;
  gateState: string;
}

export interface GateLabel {
  id: string;
  group: string;
  /** "message" for a label asked of each human message; "run" otherwise. */
  scope: string;
  state: string;
  friction: boolean;
  observed: number;
  eligible: number;
  /** Observations whose own model's hold-out clears the label's targets; a rate counts only these. */
  provenObserved: number;
  rate?: number;
  populationUnit?: string;
  gateState: string;
  gateModel?: string;
  gateRole?: string;
  calibration?: Calibration;
  temperature?: number;
  precision?: number;
  recall?: number;
  ece?: number;
  gold?: number;
  goldPositives?: number;
  targetPrecision: number;
  targetRecall: number;
  minGold: number;
  reason?: string;
  models: GateModel[];
}

export interface AnnotationGate {
  pack: string;
  packVersion: string;
  eligibleRuns: number;
  observedRuns: number;
  coverage: number;
  /** Runs and messages a person wrote in the window; undefined until something was annotated. */
  humanTextRuns?: number;
  humanMessages?: number;
  /** Runs whose authorship was computed under an older policy; counted in no population yet. */
  staleRuns?: number;
  note?: string;
  labels: GateLabel[];
}

function num(cells: GateCells, key: string): number | undefined {
  const raw = cells[key];
  if (raw === undefined || raw === "") return undefined;
  const value = Number(raw);
  return Number.isFinite(value) ? value : undefined;
}

function calibrationOf(value: string | undefined): Calibration | undefined {
  return value === "calibrated" || value === "raw" || value === "rule" ? value : undefined;
}

/** Group the measure's rows into the coverage and the labels, each label with its models. */
export function parseGate(rows: readonly GateCells[]): AnnotationGate {
  const gate: AnnotationGate = { pack: "", packVersion: "", eligibleRuns: 0, observedRuns: 0, coverage: 0, labels: [] };
  const byId = new Map<string, GateLabel>();
  for (const row of rows) {
    if (row.kind === "coverage") {
      gate.pack = row.pack ?? "";
      gate.packVersion = row.pack_version ?? "";
      gate.eligibleRuns = num(row, "eligible_runs") ?? 0;
      gate.observedRuns = num(row, "observed_runs") ?? 0;
      gate.coverage = num(row, "coverage") ?? 0;
      gate.humanTextRuns = num(row, "human_text_runs");
      gate.humanMessages = num(row, "human_messages");
      gate.staleRuns = num(row, "stale_runs");
      gate.note = row.note || undefined;
    } else if (row.kind === "label") {
      const label: GateLabel = {
        id: row.label ?? "", group: row.group ?? "", scope: row.scope ?? "", state: row.state ?? "",
        friction: row.friction === "true",
        observed: num(row, "observed") ?? 0, eligible: num(row, "eligible") ?? 0, provenObserved: num(row, "proven_observed") ?? 0,
        rate: num(row, "rate"), populationUnit: row.population_unit || undefined,
        gateState: row.gate_state ?? "", gateModel: row.gate_model || undefined, gateRole: row.gate_role || undefined,
        calibration: calibrationOf(row.calibration), temperature: num(row, "temperature"),
        precision: num(row, "precision"), recall: num(row, "recall"), ece: num(row, "ece"),
        gold: num(row, "gold"), goldPositives: num(row, "gold_positives"),
        targetPrecision: num(row, "target_precision") ?? 0, targetRecall: num(row, "target_recall") ?? 0, minGold: num(row, "min_gold") ?? 0,
        reason: row.gate_reason || undefined, models: [],
      };
      byId.set(label.id, label);
      gate.labels.push(label);
    } else if (row.kind === "model") {
      byId.get(row.label ?? "")?.models.push({
        model: row.model ?? "", role: row.resource_role ?? "", observed: num(row, "observed") ?? 0,
        proven: row.proven === "true", gateState: row.gate_state ?? "",
      });
    }
  }
  return gate;
}

/** A label counts as seen when it has an observation or is proven; the rest are silent, not zero. */
export function isSilent(label: GateLabel): boolean {
  return label.observed === 0 && label.state !== "proven";
}

/** The model a proof is for, "model · role". Empty when nothing was measured. */
export function proofModel(label: Pick<GateLabel, "gateModel" | "gateRole">): string {
  if (!label.gateModel) return "";
  return label.gateRole ? `${label.gateModel} · ${label.gateRole}` : label.gateModel;
}

/** How the proof reads its probabilities: "calibrated (T 1.4)", "raw" or "rule". */
export function calibrationText(label: Pick<GateLabel, "calibration" | "temperature">): string {
  if (label.calibration === "calibrated") return `calibrated (T ${label.temperature ?? 1})`;
  return label.calibration ?? "";
}

/** The hold-out numbers behind a proof, or nothing when no measurement exists. */
export function measurementText(label: Pick<GateLabel, "precision" | "recall" | "ece" | "gold" | "goldPositives" | "calibration">): string {
  if (label.precision === undefined || label.recall === undefined) return "";
  const parts = [`hold-out precision ${label.precision.toFixed(2)} · recall ${label.recall.toFixed(2)}`];
  if (label.ece !== undefined && label.calibration !== "rule") parts.push(`ECE ${label.ece.toFixed(3)}`);
  if (label.gold !== undefined) parts.push(`gold ${label.gold}${label.goldPositives !== undefined ? ` (${label.goldPositives} positive)` : ""}`);
  return parts.join(" · ");
}

/** Why a label is not proven, in words: what is missing or how far it falls short. */
export function whyUnproven(label: Pick<GateLabel, "gateState" | "gateModel" | "gateRole" | "gold" | "minGold" | "precision" | "recall" | "targetPrecision" | "targetRecall">): string {
  switch (label.gateState) {
    case "tuning_only":
      return `${label.gateModel ?? "the model"} was measured on its tuning rows only: no hold-out yet`;
    case "insufficient_gold":
      return `hold-out has ${label.gold ?? 0} of ${label.minGold} gold rows`;
    case "unproven":
      return `hold-out precision ${(label.precision ?? 0).toFixed(2)} / recall ${(label.recall ?? 0).toFixed(2)} is below target ${label.targetPrecision.toFixed(2)} / ${label.targetRecall.toFixed(2)}`;
    case "proven":
      return `proven for ${proofModel(label)}, but none of the observed answers came from it`;
    case "derived":
      return "derived from other labels: it is proven only where every input is";
    case "legacy_unattributed":
      return "measured before gates were per model, so the model is unknown";
    default:
      return `no hold-out measurement (target ${label.targetPrecision.toFixed(2)} / ${label.targetRecall.toFixed(2)})`;
  }
}

/** The label's gate as one plain word for a badge: proven, partial or unproven. */
export function gateWord(label: Pick<GateLabel, "state">): "proven" | "partial" | "unproven" {
  return label.state === "proven" || label.state === "partial" ? label.state : "unproven";
}
