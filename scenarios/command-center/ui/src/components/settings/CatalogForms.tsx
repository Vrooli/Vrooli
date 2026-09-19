import { useMemo, useState } from "react";
import type { CatalogEntry, Catalogs } from "../../lib/api";

const str = (value: unknown): string => (typeof value === "string" ? value : "");
const obj = (value: unknown): Record<string, unknown> => (value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : {});
const labelOf = (entry: CatalogEntry): string => str(entry.label) || entry.id;

export type Patch = (changes: Record<string, unknown>) => void;
interface EditorProps { draft: CatalogEntry; patch: Patch; catalogs: Catalogs; }

// ── Field primitives ─────────────────────────────────────────────────────────

interface Option { value: string; label: string; }

function TextField({ label, value, onChange, hint }: { label: string; value: string; onChange: (value: string) => void; hint?: string }) {
  return (
    <label className="cc-field">
      <span className="cc-field-label">{label}{hint ? <em>{hint}</em> : null}</span>
      <input type="text" value={value} onChange={(event) => onChange(event.target.value)} spellCheck={false} />
    </label>
  );
}

function TextAreaField({ label, value, onChange, hint, rows = 3 }: { label: string; value: string; onChange: (value: string) => void; hint?: string; rows?: number }) {
  return (
    <label className="cc-field">
      <span className="cc-field-label">{label}{hint ? <em>{hint}</em> : null}</span>
      <textarea value={value} onChange={(event) => onChange(event.target.value)} rows={rows} spellCheck={false} />
    </label>
  );
}

function SelectField({ label, value, onChange, options, hint }: { label: string; value: string; onChange: (value: string) => void; options: Option[]; hint?: string }) {
  return (
    <label className="cc-field">
      <span className="cc-field-label">{label}{hint ? <em>{hint}</em> : null}</span>
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        {options.some((option) => option.value === value) ? null : <option value={value}>{value || "— select —"}</option>}
        {options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </label>
  );
}

function CheckboxField({ label, checked, onChange, hint }: { label: string; checked: boolean; onChange: (value: boolean) => void; hint?: string }) {
  return (
    <label className="cc-field cc-field-check">
      <input type="checkbox" checked={checked} onChange={(event) => onChange(event.target.checked)} />
      <span className="cc-field-label">{label}{hint ? <em>{hint}</em> : null}</span>
    </label>
  );
}

const HEX = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i;
function ColorField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  const isHex = HEX.test(value.trim());
  return (
    <label className="cc-field cc-field-color">
      <span className="cc-field-label">{label}</span>
      <span className="cc-field-color-row">
        {isHex
          ? <input type="color" value={value} onChange={(event) => onChange(event.target.value)} aria-label={`${label} colour`} />
          : <span className="cc-field-color-swatch" style={{ background: value || "transparent" }} aria-hidden="true" />}
        <input type="text" value={value} onChange={(event) => onChange(event.target.value)} aria-label={label} spellCheck={false} />
      </span>
    </label>
  );
}

// ── Advanced JSON escape hatch (available for every catalog) ──────────────────

export function AdvancedJson({ draft, onReplace }: { draft: CatalogEntry; onReplace: (next: CatalogEntry) => void }) {
  const canonical = useMemo(() => JSON.stringify(draft, null, 2), [draft]);
  const [buffer, setBuffer] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const value = buffer ?? canonical;
  const onChange = (next: string) => {
    setBuffer(next);
    try {
      const parsed: unknown = JSON.parse(next);
      if (!parsed || typeof parsed !== "object" || typeof (parsed as { id?: unknown }).id !== "string") { setError("Entry must be an object with a string id."); return; }
      setError(null);
      onReplace(parsed as CatalogEntry);
      setBuffer(null);
    } catch {
      setError("Invalid JSON — form fields left unchanged.");
    }
  };
  return (
    <details className="cc-settings-advanced">
      <summary>Advanced · raw JSON</summary>
      <textarea aria-label="Advanced JSON editor" value={value} onChange={(event) => onChange(event.target.value)} spellCheck={false} />
      {error ? <p className="cc-settings-error" role="alert">{error}</p> : null}
    </details>
  );
}

// ── Per-catalog editors ───────────────────────────────────────────────────────

function BeatsSummary({ draft }: { draft: CatalogEntry }) {
  const beats = Array.isArray(draft.beats) ? (draft.beats as Array<Record<string, unknown>>) : [];
  if (!beats.length) return null;
  return (
    <div className="cc-settings-beats">
      <span className="cc-field-label">Beats <em>edit via advanced JSON</em></span>
      <ol>{beats.map((beat, index) => <li key={index}>{str(beat.hero) || "—"}{typeof beat.dwellSeconds === "number" ? ` · ${beat.dwellSeconds}s` : ""}</li>)}</ol>
    </div>
  );
}

function RoomEditor({ draft, patch, catalogs }: EditorProps) {
  const themes = catalogs.themes ?? [];
  const compositions = catalogs.compositions ?? [];
  const signals = catalogs.signals ?? [];
  const compositionId = str(draft.composition);
  const composition = compositions.find((entry) => entry.id === compositionId);
  const slots = obj(composition?.slots) as Record<string, { shape?: string; role?: string; whenUnbound?: string }>;
  const bind = obj(draft.bind);
  const setBind = (slot: string, signalId: string) => {
    const next = Object.fromEntries(Object.entries(bind).filter(([key]) => key !== slot));
    if (signalId) next[slot] = signalId;
    patch({ bind: next });
  };
  const slotNames = Object.keys(slots);
  return (
    <>
      <TextField label="Title" value={str(draft.title)} onChange={(value) => patch({ title: value })} />
      <SelectField label="Theme" value={str(draft.theme)} onChange={(value) => patch({ theme: value })} options={themes.map((entry) => ({ value: entry.id, label: labelOf(entry) }))} />
      <SelectField label="Composition" value={compositionId} onChange={(value) => patch({ composition: value })} options={compositions.map((entry) => ({ value: entry.id, label: entry.id }))} />
      <fieldset className="cc-settings-slots">
        <legend>Slot bindings</legend>
        {slotNames.length === 0
          ? <p className="cc-settings-hint">This composition declares no slots.</p>
          : slotNames.map((slot) => {
              const spec = slots[slot] ?? {};
              const compatible = signals.filter((signal) => !spec.shape || signal.shape === spec.shape);
              return (
                <SelectField
                  key={slot}
                  label={`${slot} · ${spec.shape ?? "any"}`}
                  hint={spec.whenUnbound ? `unbound → ${spec.whenUnbound}` : undefined}
                  value={str(bind[slot])}
                  onChange={(value) => setBind(slot, value)}
                  options={[{ value: "", label: "— unbound —" }, ...compatible.map((signal) => ({ value: signal.id, label: labelOf(signal) }))]}
                />
              );
            })}
      </fieldset>
      <BeatsSummary draft={draft} />
    </>
  );
}

const THEME_TOKENS: Array<[string, string]> = [
  ["--color-background", "Background"],
  ["--color-background-deep", "Background deep"],
  ["--color-foreground", "Foreground"],
  ["--color-primary", "Primary"],
  ["--color-accent", "Accent"],
  ["--color-glow", "Glow"],
  ["--color-muted-foreground", "Muted foreground"],
  ["--color-gap", "Gap"],
];

function ThemeEditor({ draft, patch }: EditorProps) {
  const tokens = obj(draft.tokens);
  const setToken = (name: string, value: string) => patch({ tokens: { ...tokens, [name]: value } });
  return (
    <>
      <div className="cc-settings-tokens">
        {THEME_TOKENS.map(([name, label]) => <ColorField key={name} label={label} value={str(tokens[name])} onChange={(value) => setToken(name, value)} />)}
      </div>
      <TextField label="Corner radius" value={str(draft.cornerRadius)} onChange={(value) => patch({ cornerRadius: value })} />
      <TextField label="Gradient" value={str(draft.gradient)} onChange={(value) => patch({ gradient: value })} hint="CSS background" />
    </>
  );
}

function ConnectorEditor({ draft, patch }: EditorProps) {
  const paths = Array.isArray(draft.paths) ? (draft.paths as unknown[]).map(String) : [];
  return (
    <>
      <CheckboxField label="Enabled" checked={draft.enabled !== false} onChange={(value) => patch({ enabled: value })} hint="off keeps this source out of the board" />
      {draft.pack === true ? <CheckboxField label="Connector pack" checked hint="registers bespoke readouts/compositions" onChange={() => undefined} /> : null}
      <SelectField label="Transport" value={str(draft.transport) || "http-json"} onChange={(value) => patch({ transport: value })} options={["http-json", "connect", "graphql"].map((transport) => ({ value: transport, label: transport }))} />
      <TextField label="Base URL env var" value={str(draft.baseUrlEnv)} onChange={(value) => patch({ baseUrlEnv: value })} hint="resolved at runtime; never a token" />
      <TextAreaField label="Paths" value={paths.join("\n")} onChange={(value) => patch({ paths: value.split("\n").map((path) => path.trim()).filter(Boolean) })} hint="one per line" rows={3} />
    </>
  );
}

function SignalEditor({ draft, patch }: EditorProps) {
  return (
    <>
      <TextField label="Label" value={str(draft.label)} onChange={(value) => patch({ label: value })} />
      <TextAreaField label="Description" value={str(draft.description)} onChange={(value) => patch({ description: value })} rows={2} />
      <TextField label="Unit" value={str(draft.unit)} onChange={(value) => patch({ unit: value })} />
      <SelectField label="Coverage" value={str(draft.coverage) || "NOW"} onChange={(value) => patch({ coverage: value })} options={["NOW", "IN-REACH", "MISSING", "UNREGISTERED"].map((coverage) => ({ value: coverage, label: coverage }))} />
      <p className="cc-settings-hint">Shape <code>{str(draft.shape) || "unknown"}</code> and source binding are structural — edit them through advanced JSON.</p>
    </>
  );
}

/** Structured form for the catalogs that have one; the rest lean on advanced JSON. */
export function CatalogForm({ catalog, draft, patch, catalogs, blockingError }: EditorProps & { catalog: string; blockingError: string | null }) {
  return (
    <div className="cc-settings-form">
      {catalog === "rooms" ? <RoomEditor draft={draft} patch={patch} catalogs={catalogs} /> : null}
      {catalog === "themes" ? <ThemeEditor draft={draft} patch={patch} catalogs={catalogs} /> : null}
      {catalog === "connectors" ? <ConnectorEditor draft={draft} patch={patch} catalogs={catalogs} /> : null}
      {catalog === "signals" ? <SignalEditor draft={draft} patch={patch} catalogs={catalogs} /> : null}
      {catalog === "compositions" || catalog === "readouts"
        ? <p className="cc-settings-hint">{catalog === "compositions" ? "Compositions declare slots in their module export" : "Readouts map a signal kind to a component"} — structural, edited as JSON below.</p>
        : null}
      {blockingError ? <p className="cc-settings-error" role="alert">{blockingError}</p> : null}
    </div>
  );
}
