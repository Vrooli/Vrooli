import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { fetchCatalogs, saveCatalogEntry, type CatalogEntry } from "../lib/api";
import { validateRoomBinding } from "../lib/catalogs";
import { AdvancedJson, CatalogForm } from "../components/settings/CatalogForms";
import { SettingsPreview } from "../components/settings/SettingsPreview";

const CATALOG_ORDER = ["rooms", "themes", "compositions", "signals", "connectors", "readouts"] as const;
const CATALOG_BLURB: Record<string, string> = {
  rooms: "A room binds a composition and theme, then wires each slot to a signal.",
  themes: "A theme is the paint: the token colours the board and every scene read.",
  compositions: "A composition is a background scene and the slots it draws.",
  signals: "A signal is one measurement: its shape, unit, coverage, and sample.",
  connectors: "A connector is a data source; disable one to drop it from the board.",
  readouts: "A readout renders one signal kind into a figure.",
};

const str = (value: unknown): string => (typeof value === "string" ? value : "");
const labelOf = (entry: CatalogEntry): string => str(entry.label) || str(entry.title) || entry.id;

type SaveState = "idle" | "saving" | "saved" | "error";

/** The operator surface for editing the instance definition: form-driven catalogs
 *  on the left, a live sample-driven board preview on the right. */
export default function SettingsPage() {
  const navigate = useNavigate();
  const query = useQuery({ queryKey: ["catalogs"], queryFn: fetchCatalogs });
  const catalogs = query.data ?? {};
  const [catalog, setCatalog] = useState<string>("rooms");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [drafts, setDrafts] = useState<Record<string, CatalogEntry>>({});
  const [previewRoomId, setPreviewRoomId] = useState<string | null>(null);
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [saveError, setSaveError] = useState<string | null>(null);

  const entries = (catalogs[catalog] ?? []).filter((entry) => typeof entry.id === "string" && entry.id.length > 0);
  const effectiveId = selectedId && entries.some((entry) => entry.id === selectedId) ? selectedId : entries[0]?.id ?? null;
  const original = entries.find((entry) => entry.id === effectiveId);
  const draftKey = `${catalog}:${effectiveId ?? ""}`;
  const draft = drafts[draftKey] ?? original;
  const dirty = Boolean(draft && original && JSON.stringify(draft) !== JSON.stringify(original));

  const patch = (changes: Record<string, unknown>) => {
    if (!draft) return;
    setDrafts((current) => ({ ...current, [draftKey]: { ...draft, ...changes } }));
    setSaveState("idle");
  };
  const replaceDraft = (next: CatalogEntry) => {
    setDrafts((current) => ({ ...current, [draftKey]: next }));
    setSaveState("idle");
  };
  const resetDraft = () => setDrafts((current) => Object.fromEntries(Object.entries(current).filter(([key]) => key !== draftKey)));

  const blockingError = draft && catalog === "rooms" ? validateRoomBinding(draft, catalogs) : null;

  const save = async () => {
    if (!draft || blockingError) return;
    setSaveState("saving");
    setSaveError(null);
    try {
      await saveCatalogEntry(catalog, draft);
      await query.refetch();
      resetDraft();
      setSaveState("saved");
    } catch (error) {
      setSaveState("error");
      setSaveError(error instanceof Error ? error.message : "Save failed.");
    }
  };

  const selectCatalog = (name: string) => { setCatalog(name); setSelectedId(null); setSaveState("idle"); };
  const selectEntry = (id: string) => { setSelectedId(id); setSaveState("idle"); };

  const rooms = catalogs.rooms ?? [];
  const editingRoom = catalog === "rooms";
  const previewRoom = editingRoom ? draft : rooms.find((room) => room.id === (previewRoomId ?? rooms[0]?.id));
  const previewTheme = catalog === "themes" ? draft : (catalogs.themes ?? []).find((theme) => theme.id === str(previewRoom?.theme));

  return (
    <main className="cc-settings" data-testid="settings-page">
      <header className="cc-settings-header">
        <div>
          <p className="cc-eyebrow">COMMAND CENTER / INSTANCE DEFINITION</p>
          <h1>Settings</h1>
          <p className="cc-settings-lede">Shape this instrument by editing its catalogs. Every change is validated, previewed against the live engine, and written as inspectable config.</p>
        </div>
        <button type="button" className="cc-settings-back" onClick={() => navigate("/mission-control")}>Return to board</button>
      </header>

      {query.isLoading ? <p role="status" className="cc-settings-status">Reading catalogs…</p> : query.error ? <p role="alert" className="cc-settings-error">Catalogs are unavailable: {query.error instanceof Error ? query.error.message : "unknown error"}.</p> : (
        <div className="cc-settings-grid">
          <nav className="cc-settings-nav" aria-label="Catalogs">
            {CATALOG_ORDER.map((name) => (
              <button key={name} type="button" className={name === catalog ? "active" : ""} onClick={() => selectCatalog(name)}>
                <span>{name}</span><span className="cc-settings-count">{catalogs[name]?.length ?? 0}</span>
              </button>
            ))}
          </nav>

          <section className="cc-settings-editor" aria-label={`${catalog} editor`}>
            <p className="cc-settings-blurb">{CATALOG_BLURB[catalog]}</p>
            <div className="cc-settings-list" role="list">
              {entries.map((entry) => (
                <button key={entry.id} type="button" role="listitem" className={entry.id === effectiveId ? "active" : ""} onClick={() => selectEntry(entry.id)}>{labelOf(entry)}</button>
              ))}
            </div>
            {draft ? (
              <>
                <CatalogForm catalog={catalog} draft={draft} patch={patch} catalogs={catalogs} blockingError={blockingError} />
                <AdvancedJson key={draftKey} draft={draft} onReplace={replaceDraft} />
                <div className="cc-settings-actions">
                  <button type="button" className="cc-settings-save" disabled={!dirty || saveState === "saving" || Boolean(blockingError)} onClick={() => void save()}>
                    {saveState === "saving" ? "Saving…" : `Save ${draft.id}`}
                  </button>
                  <button type="button" className="cc-settings-reset" disabled={!dirty || saveState === "saving"} onClick={resetDraft}>Revert</button>
                  <span className="cc-settings-savestate" role="status" aria-live="polite">
                    {saveState === "error" ? <span className="cc-settings-error">{saveError}</span> : saveState === "saved" ? "Saved." : dirty ? "Unsaved changes" : "No changes"}
                  </span>
                </div>
              </>
            ) : <p className="cc-settings-hint">No entries in this catalog.</p>}
          </section>

          <aside className="cc-settings-preview" aria-label="Live preview">
            <div className="cc-settings-preview-head">
              <p className="cc-eyebrow">LIVE PREVIEW</p>
              {!editingRoom && rooms.length ? (
                <label className="cc-settings-preview-room">
                  <span>Preview room</span>
                  <select value={previewRoomId ?? rooms[0]?.id ?? ""} onChange={(event) => setPreviewRoomId(event.target.value)}>
                    {rooms.map((room) => <option key={room.id} value={room.id}>{labelOf(room)}</option>)}
                  </select>
                </label>
              ) : null}
            </div>
            <SettingsPreview catalogs={catalogs} room={previewRoom} theme={previewTheme} />
            <p className="cc-settings-preview-note">The preview runs the same scene engine as the board, driven by authored signal samples.</p>
          </aside>
        </div>
      )}
    </main>
  );
}
