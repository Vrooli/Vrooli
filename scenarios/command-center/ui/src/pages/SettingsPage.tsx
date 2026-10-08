import { useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ArrowLeft, Check, ChevronRight, Layers, LayoutGrid, Loader2, Palette, PanelsTopLeft, Plus, Radar, Search } from "lucide-react";
import { Slider } from "@vrooli/react-component-library/Slider/1.2.4";
import { deleteCatalogEntry, fetchBoardSettings, fetchCatalogs, saveBoardSettings, saveCatalogEntry, type BoardSettings, type CatalogEntry } from "../lib/api";
import { validateRoomBinding } from "../lib/catalogs";
import { AdvancedJson, CatalogForm } from "../components/settings/CatalogForms";
import { SettingsPreview } from "../components/settings/SettingsPreview";
import { CATALOG_SINGULAR, describeEntry, seedEntry, slugify, taskById, type ChainNode } from "../lib/settingsTasks";

const CATALOG_ORDER = ["rooms", "themes", "compositions", "signals", "connectors", "readouts"] as const;
const CATALOG_BLURB: Record<string, string> = {
  rooms: "A room binds a composition and theme, then wires each slot to a signal.",
  themes: "A theme is the paint: its tokens colour the board and every scene read.",
  compositions: "A composition is a background scene and the slots it draws.",
  signals: "A signal is one measurement: its shape, unit, coverage, and sample.",
  connectors: "A connector is a data source; disable one to drop it from the board.",
  readouts: "A readout renders one signal kind into a figure.",
};

const str = (value: unknown): string => (typeof value === "string" ? value : "");
const labelOf = (entry: CatalogEntry): string => str(entry.label) || str(entry.title) || entry.id;

interface Pending { catalog: string; entry: CatalogEntry; isNew: boolean; error: string | null; }
type SaveState = "idle" | "saving" | "saved" | "error";

/**
 * The customization surface. A newcomer lands on a task hub ("Recolor the board");
 * each task opens a focused workspace that shows the real model and previews the
 * result live. Operators drop to the full catalog surface with one click. The path
 * stays `/settings` throughout — navigation is by search param — so the board's
 * kiosk-cycle exemption is never disturbed.
 */
export default function SettingsPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const query = useQuery({ queryKey: ["catalogs"], queryFn: fetchCatalogs });
  const boardSettingsQuery = useQuery({ queryKey: ["board-settings"], queryFn: fetchBoardSettings, enabled: params.get("destination") === "board" });
  const catalogs = query.data ?? {};

  const mode = params.get("mode"); // "catalogs" → operator surface
  const task = taskById(params.get("task"));
  const catalogsMode = mode === "catalogs";
  const destination = params.get("destination");

  const [drafts, setDrafts] = useState<Record<string, CatalogEntry>>({});
  const [newDraft, setNewDraft] = useState<{ catalog: string; entry: CatalogEntry } | null>(null);
  const [previewRoomId, setPreviewRoomId] = useState<string | null>(null);
  const [search, setSearch] = useState("");
  const [saveState, setSaveState] = useState<SaveState>("idle");
  const [saveError, setSaveError] = useState<string | null>(null);

  const setParam = (patch: Record<string, string | null>) => {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(patch)) { if (value === null) next.delete(key); else next.set(key, value); }
    setParams(next);
    setSaveState("idle"); setSearch("");
  };

  // ── The active catalog + selection ─────────────────────────────────────────
  const activeCatalog = catalogsMode ? (params.get("cat") ?? "rooms") : task?.catalog ?? "rooms";
  const entries = (catalogs[activeCatalog] ?? []).filter((entry) => typeof entry.id === "string" && entry.id.length > 0);
  const entryParam = params.get("entry");
  const effectiveId = entryParam && entries.some((entry) => entry.id === entryParam) ? entryParam : entries[0]?.id ?? null;
  const draftKey = `${activeCatalog}:${effectiveId ?? ""}`;
  const original = entries.find((entry) => entry.id === effectiveId);

  const showingNew = Boolean(newDraft && newDraft.catalog === activeCatalog && (params.get("new") === "1" || (task?.create && !entryParam)));
  const draft = showingNew ? newDraft?.entry : drafts[draftKey] ?? original;

  // A create-first task ("Add a room" / "Add a data source") opens straight into a
  // blank draft rather than pre-selecting an existing entry to edit by accident.
  useEffect(() => {
    if (task?.create && !catalogsMode && !entryParam && !newDraft) {
      setNewDraft({ catalog: task.catalog, entry: seedEntry(task.catalog, "", catalogs) });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [task?.id, catalogsMode, entryParam, newDraft]);

  const startNew = () => {
    setNewDraft({ catalog: activeCatalog, entry: seedEntry(activeCatalog, "", catalogs) });
    setParam({ new: "1", entry: null });
  };
  const patch = (changes: Record<string, unknown>) => {
    if (showingNew && newDraft) { setNewDraft({ ...newDraft, entry: { ...newDraft.entry, ...changes } }); setSaveState("idle"); return; }
    if (!draft) return;
    setDrafts((current) => ({ ...current, [draftKey]: { ...draft, ...changes } }));
    setSaveState("idle");
  };
  const replaceDraft = (next: CatalogEntry) => {
    if (showingNew && newDraft) { setNewDraft({ ...newDraft, entry: next }); setSaveState("idle"); return; }
    setDrafts((current) => ({ ...current, [draftKey]: next })); setSaveState("idle");
  };

  // ── Pending changes across the whole session (drives the global save bar) ───
  const existingIds = useMemo(() => new Set(entries.map((entry) => entry.id)), [entries]);
  const pending = useMemo<Pending[]>(() => {
    const list: Pending[] = [];
    for (const [key, entry] of Object.entries(drafts)) {
      const catalog = key.slice(0, key.indexOf(":"));
      const id = key.slice(key.indexOf(":") + 1);
      const orig = (catalogs[catalog] ?? []).find((item) => item.id === id);
      if (orig && JSON.stringify(orig) === JSON.stringify(entry)) continue;
      list.push({ catalog, entry, isNew: !orig, error: catalog === "rooms" ? validateRoomBinding(entry, catalogs) : null });
    }
    if (newDraft && str(newDraft.entry.id)) {
      const dup = (catalogs[newDraft.catalog] ?? []).some((item) => item.id === newDraft.entry.id);
      list.push({ catalog: newDraft.catalog, entry: newDraft.entry, isNew: true, error: dup ? `A ${CATALOG_SINGULAR[newDraft.catalog] ?? "entry"} named ${newDraft.entry.id} already exists.` : newDraft.catalog === "rooms" ? validateRoomBinding(newDraft.entry, catalogs) : null });
    }
    return list;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [drafts, newDraft, JSON.stringify(catalogs)]);

  const blockingError = draft && (showingNew ? newDraft?.catalog : activeCatalog) === "rooms" ? validateRoomBinding(draft, catalogs) : null;
  const idError = showingNew && newDraft ? (!str(newDraft.entry.id) ? "Give this a lowercase-kebab identifier." : existingIds.has(newDraft.entry.id) ? "That identifier is already taken." : null) : null;
  const hasBlocking = pending.some((change) => change.error) || Boolean(idError);

  const saveAll = async () => {
    const savable = pending.filter((change) => !change.error);
    if (!savable.length || hasBlocking) return;
    setSaveState("saving"); setSaveError(null);
    try {
      for (const change of savable) await saveCatalogEntry(change.catalog, change.entry);
      await query.refetch();
      const savedNew = newDraft && savable.some((change) => change.entry === newDraft.entry) ? newDraft.entry.id : null;
      setDrafts({}); setNewDraft(null); setSaveState("saved");
      if (savedNew) setParam({ new: null, entry: savedNew });
    } catch (error) {
      setSaveState("error"); setSaveError(error instanceof Error ? error.message : "Save failed.");
    }
  };
  const discardAll = () => { setDrafts({}); setNewDraft(null); setSaveState("idle"); if (params.get("new")) setParam({ new: null }); };

  // ── Preview inputs ──────────────────────────────────────────────────────────
  const rooms = catalogs.rooms ?? [];
  const editingRoom = activeCatalog === "rooms";
  const previewRoom = editingRoom ? draft : rooms.find((room) => room.id === (previewRoomId ?? rooms[0]?.id));
  const previewTheme = activeCatalog === "themes" ? draft : (catalogs.themes ?? []).find((theme) => theme.id === str(previewRoom?.theme));

  // ── Filtered scoped list ─────────────────────────────────────────────────────
  const visibleEntries = entries.filter((entry) => !search || labelOf(entry).toLowerCase().includes(search.toLowerCase()) || entry.id.toLowerCase().includes(search.toLowerCase()));
  const canCreate = catalogsMode ? ["rooms", "connectors"].includes(activeCatalog) : Boolean(task?.create);

  // ── States ────────────────────────────────────────────────────────────────
  if (query.isLoading) return <main className="cc-cx" data-testid="settings-page"><Header title="Customize" onExit={() => navigate("/mission-control")} /><p role="status" className="cc-cx-status">Reading catalogs…</p></main>;
  if (query.error) return <main className="cc-cx" data-testid="settings-page"><Header title="Customize" onExit={() => navigate("/mission-control")} /><p role="alert" className="cc-cx-error-block">Catalogs are unavailable: {query.error instanceof Error ? query.error.message : "unknown error"}.</p></main>;

  // ── Hub ─────────────────────────────────────────────────────────────────────
  if (!task && !catalogsMode && !destination) {
    return (
      <main className="cc-cx" data-testid="settings-page">
        <Header eyebrow="Command Center / Customize" title="Shape the board" lede="Choose the part of the board you want to change. The screen follows the way the board actually renders." onExit={() => navigate("/mission-control")} />
        <div className="cc-cx-destinations" aria-label="Customization destinations">
          {([
            { id: "rooms", title: "Rooms", copy: "Arrange the sections that cycle on the board.", Icon: LayoutGrid },
            { id: "looks", title: "Looks", copy: "Choose a palette without starting with code.", Icon: Palette },
            { id: "signals", title: "Signals & sources", copy: "See what each metric measures and where it comes from.", Icon: Radar },
            { id: "board", title: "Board settings", copy: "Set the cycle, transition, and rooms in rotation.", Icon: PanelsTopLeft },
            { id: "operator", title: "Operator mode", copy: "Open the complete catalog and JSON surface.", Icon: Layers },
          ] as Array<{ id: string; title: string; copy: string; Icon: typeof Palette }>).map(({ id, title, copy, Icon }) => <button key={id} type="button" className="cc-cx-destination" onClick={() => setParam({ destination: id, task: null, mode: id === "operator" ? "catalogs" : null, cat: id === "operator" ? "rooms" : null })}>
            <span className="cc-cx-destination-icon"><Icon size={20} strokeWidth={1.7} /></span><span><b>{title}</b><small>{copy}</small></span><ChevronRight size={18} />
          </button>)}
        </div>
      </main>
    );
  }

  if (!task && destination && destination !== "operator") {
    const common = { catalogs, setParam, saveCatalog: saveCatalogEntry, refetch: query.refetch };
    if (destination === "rooms") return <RoomsDestination {...common} onExit={() => navigate("/mission-control")} />;
    if (destination === "looks") return <LooksDestination {...common} onExit={() => navigate("/mission-control")} />;
    if (destination === "signals") return <SignalsDestination catalogs={catalogs} onExit={() => navigate("/mission-control")} onAgent={(target) => void copyAgentPrompt(target)} />;
    if (destination === "board") return <BoardSettingsDestination settings={boardSettingsQuery.data} rooms={catalogs.rooms ?? []} onExit={() => navigate("/mission-control")} onSave={async (value) => { await saveBoardSettings(value); await boardSettingsQuery.refetch(); }} />;
  }

  const chain: ChainNode[] = describeEntry(activeCatalog, draft, catalogs);
  const singular = CATALOG_SINGULAR[activeCatalog] ?? "entry";

  return (
    <main className="cc-cx" data-testid="settings-page">
      <Header
        eyebrow={catalogsMode ? "Command Center / Operator mode" : "Command Center / Customize"}
        title={catalogsMode ? "All catalogs" : task?.title ?? "Customize"}
        lede={catalogsMode ? CATALOG_BLURB[activeCatalog] : task?.intent}
        crumb={{ label: catalogsMode ? "Customize" : "Customize", onClick: () => setParam({ task: null, mode: null, entry: null, new: null }) }}
        onExit={() => navigate("/mission-control")}
      />

      <div className={catalogsMode ? "cc-cx-work cc-cx-work--rail" : "cc-cx-work"}>
        {catalogsMode ? (
          <nav className="cc-cx-rail" aria-label="Catalogs">
            {CATALOG_ORDER.map((name) => (
              <button key={name} type="button" className={name === activeCatalog ? "active" : ""} onClick={() => setParam({ cat: name, entry: null, new: null })}>
                <span>{name}</span><span className="cc-cx-rail-count">{catalogs[name]?.length ?? 0}</span>
              </button>
            ))}
          </nav>
        ) : null}

        <section className="cc-cx-editor" aria-label={`${activeCatalog} editor`}>
          {chain.length ? <RelationshipStrip chain={chain} /> : null}

          <div className="cc-cx-picker">
            <div className="cc-cx-searchbox">
              <Search size={15} strokeWidth={1.8} aria-hidden="true" />
              <input type="text" value={search} placeholder={`Search ${activeCatalog}…`} onChange={(event) => setSearch(event.target.value)} aria-label={`Search ${activeCatalog}`} spellCheck={false} />
            </div>
            <div className="cc-cx-list" role="list">
              {canCreate ? <button type="button" className={showingNew ? "cc-cx-new active" : "cc-cx-new"} onClick={startNew}><Plus size={14} strokeWidth={2.2} /> New {singular}</button> : null}
              {visibleEntries.map((entry) => {
                const key = `${activeCatalog}:${entry.id}`;
                const isDirty = drafts[key] && JSON.stringify(drafts[key]) !== JSON.stringify(entry);
                return (
                  <button key={entry.id} type="button" role="listitem" className={!showingNew && entry.id === effectiveId ? "active" : ""} onClick={() => setParam({ entry: entry.id, new: null })}>
                    {labelOf(entry)}{isDirty ? <span className="cc-cx-dot" aria-label="unsaved" /> : null}
                  </button>
                );
              })}
              {!visibleEntries.length && !canCreate ? <p className="cc-cx-hint">No {activeCatalog} match “{search}”.</p> : null}
            </div>
          </div>

          {draft ? (
            <>
              {showingNew ? (
                <label className="cc-field cc-cx-idfield">
                  <span className="cc-field-label">Identifier <em>lowercase-kebab · becomes the filename</em></span>
                  <input type="text" value={str(newDraft?.entry.id)} placeholder={`e.g. ${slugify(str(newDraft?.entry.title)) || "my-" + singular}`} onChange={(event) => patch({ id: slugify(event.target.value) })} spellCheck={false} />
                  {idError ? <span className="cc-cx-error">{idError}</span> : null}
                </label>
              ) : null}
              <CatalogForm catalog={activeCatalog} draft={draft} patch={patch} catalogs={catalogs} blockingError={blockingError} />
              <AdvancedJson key={draftKey + (showingNew ? ":new" : "")} draft={draft} onReplace={replaceDraft} />
            </>
          ) : <p className="cc-cx-hint">No entries in this catalog yet.{canCreate ? ` Use “New ${singular}”.` : ""}</p>}
        </section>

        <aside className="cc-cx-preview" aria-label="Live preview">
          <div className="cc-cx-preview-head">
            <p className="cc-cx-preview-title"><span className="cc-cx-live" /> Live preview{previewRoom ? ` · ${labelOf(previewRoom)}` : ""}</p>
            {!editingRoom && rooms.length ? (
              <label className="cc-cx-preview-room">
                <span>Room</span>
                <select value={previewRoomId ?? rooms[0]?.id ?? ""} onChange={(event) => setPreviewRoomId(event.target.value)}>
                  {rooms.map((room) => <option key={room.id} value={room.id}>{labelOf(room)}</option>)}
                </select>
              </label>
            ) : null}
          </div>
          <SettingsPreview catalogs={catalogs} room={previewRoom} theme={previewTheme} />
          <p className="cc-cx-preview-note">Runs the same scene engine as the board, driven by authored samples. This is exactly what saving renders.</p>
        </aside>
      </div>

      {pending.length ? (
        <div className="cc-cx-savebar" role="region" aria-label="Unsaved changes">
          <span className="cc-cx-savecount">{hasBlocking ? <span className="cc-cx-error">{idError ?? pending.find((change) => change.error)?.error}</span> : <><b>{pending.length}</b> unsaved change{pending.length === 1 ? "" : "s"} · <span className="cc-cx-savemeta">{pending.map((change) => `${CATALOG_SINGULAR[change.catalog] ?? change.catalog} “${change.entry.id || "new"}”`).slice(0, 3).join(", ")}{pending.length > 3 ? "…" : ""}</span></>}</span>
          <span className="cc-cx-spacer" />
          {saveState === "error" ? <span className="cc-cx-error">{saveError}</span> : null}
          <button type="button" className="cc-cx-discard" onClick={discardAll} disabled={saveState === "saving"}>Discard</button>
          <button type="button" className="cc-cx-save" disabled={hasBlocking || saveState === "saving"} onClick={() => void saveAll()}>
            {saveState === "saving" ? <><Loader2 size={15} className="cc-cx-spin" /> Saving…</> : `Save all`}
          </button>
        </div>
      ) : saveState === "saved" ? (
        <div className="cc-cx-savebar cc-cx-savebar--ok" role="status"><Check size={16} strokeWidth={2.4} /> Saved. Your changes are written as config and live on the board.</div>
      ) : null}
    </main>
  );
}

// ── Presentational pieces (pages/ is coverage-excluded) ───────────────────────

function Header({ eyebrow, title, lede, crumb, onExit }: { eyebrow?: string; title: string; lede?: string; crumb?: { label: string; onClick: () => void }; onExit: () => void }) {
  return (
    <header className="cc-cx-header">
      <div className="cc-cx-header-copy">
        {crumb ? <button type="button" className="cc-cx-crumb" onClick={crumb.onClick}><ArrowLeft size={14} strokeWidth={2} /> {crumb.label}</button> : eyebrow ? <p className="cc-cx-eyebrow">{eyebrow}</p> : null}
        <h1>{title}</h1>
        {lede ? <p className="cc-cx-lede">{lede}</p> : null}
      </div>
      <button type="button" className="cc-cx-exit" onClick={onExit}>Return to board</button>
    </header>
  );
}

function RelationshipStrip({ chain }: { chain: ChainNode[] }) {
  return (
    <div className="cc-cx-chain" aria-label="How this fits together">
      {chain.map((node, index) => (
        <span key={`${node.kind}-${index}`} className="cc-cx-chain-seg">
          <span className={`cc-cx-node cc-cx-node--${node.tone}`}>{node.label}<small>{node.kind}</small></span>
          {index < chain.length - 1 ? <ChevronRight size={13} className="cc-cx-chain-arw" aria-hidden="true" /> : null}
        </span>
      ))}
    </div>
  );
}

async function copyAgentPrompt(target: string): Promise<void> {
  const prompt = "Use the command-center-customize skill in scenarios/command-center/skills/command-center-customize/SKILL.md to " + target + ". Preserve the section model, validate room bindings, and add focused tests.";
  await navigator.clipboard?.writeText(prompt);
}

type DestinationProps = { catalogs: Record<string, CatalogEntry[]>; setParam: (patch: Record<string, string | null>) => void; saveCatalog: typeof saveCatalogEntry; refetch: () => Promise<unknown>; onExit: () => void };

function DestinationHeader({ title, copy, onExit, onBack }: { title: string; copy: string; onExit: () => void; onBack?: () => void }) {
  return <Header eyebrow="Command Center / Customize" title={title} lede={copy} crumb={onBack ? { label: "Customize", onClick: onBack } : undefined} onExit={onExit} />;
}

function RoomsDestination({ catalogs, setParam, saveCatalog, refetch, onExit }: DestinationProps) {
  const [params] = useSearchParams();
  const [rooms, setRooms] = useState(catalogs.rooms ?? []);
  const [dragged, setDragged] = useState<string | null>(null);
  const [message, setMessage] = useState("");
  const selected = params.get("room");
  useEffect(() => setRooms(catalogs.rooms ?? []), [catalogs.rooms]);
  if (selected) {
    const room = rooms.find((entry) => entry.id === selected);
    if (room) return <RoomDestination catalogs={catalogs} room={room} setParam={setParam} saveCatalog={saveCatalog} refetch={refetch} onExit={onExit} />;
  }
  const persistOrder = async (next: CatalogEntry[]) => {
    setRooms(next);
    const current = await fetchBoardSettings();
    await saveBoardSettings({ ...current, rooms: next.map((entry) => ({ id: entry.id, enabled: current.rooms.find((item) => item.id === entry.id)?.enabled ?? true })) });
    setMessage("Room order saved.");
  };
  const addRoom = async () => {
    const title = window.prompt("Name this room");
    if (!title?.trim()) return;
    const id = slugify(title);
    if (rooms.some((entry) => entry.id === id)) { setMessage("That room already exists."); return; }
    const entry: CatalogEntry = { id, title: title.trim(), theme: String(catalogs.themes?.[0]?.id ?? "cosmos"), composition: String(catalogs.compositions?.[0]?.id ?? "orbital-field"), beats: [], bind: {} };
    await saveCatalog("rooms", entry); await refetch(); setParam({ room: id });
  };
  const deleteRoom = async (entry: CatalogEntry) => {
    const typed = window.prompt("Type " + entry.id + " to delete this room.");
    if (typed !== entry.id) return;
    await deleteCatalogEntry("rooms", entry.id);
    const current = await fetchBoardSettings();
    await saveBoardSettings({ ...current, rooms: current.rooms.filter((item) => item.id !== entry.id) });
    await refetch();
    setRooms((currentRooms) => currentRooms.filter((item) => item.id !== entry.id));
    setMessage("Room deleted.");
  };
  return <main className="cc-cx" data-testid="settings-rooms">
    <DestinationHeader title="Rooms" copy="Arrange the sections that cycle on the board. Open a room to edit the moments it shows." onExit={onExit} onBack={() => setParam({ destination: null, room: null })} />
    <section className="cc-cx-destination-body"><div className="cc-cx-section-heading"><div><p className="cc-cx-eyebrow">Rotation</p><h2>Rooms in rotation</h2></div><button type="button" className="cc-cx-primary" onClick={() => void addRoom()}><Plus size={15} /> Add room</button></div>
      <div className="cc-cx-room-grid">{rooms.map((room) => <article key={room.id} className="cc-cx-room-card" draggable onDragStart={() => setDragged(room.id)} onDragOver={(event) => event.preventDefault()} onDrop={() => { if (!dragged || dragged === room.id) return; const from = rooms.findIndex((item) => item.id === dragged); const to = rooms.findIndex((item) => item.id === room.id); const next = [...rooms]; const [moved] = next.splice(from, 1); if (!moved) return; next.splice(to, 0, moved); void persistOrder(next); setDragged(null); }}>
        <button type="button" className="cc-cx-room-open" onClick={() => setParam({ room: room.id })}><span className="cc-cx-room-kicker">{String(room.category ?? "room")}</span><strong>{labelOf(room)}</strong><span>{Array.isArray(room.beats) ? room.beats.length : 0} sections · {String(room.composition ?? "no layout")}</span></button>
        <div className="cc-cx-room-actions"><button type="button" onClick={() => setParam({ room: room.id })}>Edit room</button><button type="button" onClick={() => void deleteRoom(room)}>Delete</button></div>
      </article>)}</div>{message ? <p className="cc-cx-inline-status" role="status">{message}</p> : null}</section>
  </main>;
}

function RoomDestination({ catalogs, room, setParam, saveCatalog, refetch, onExit }: DestinationProps & { room: CatalogEntry }) {
  const beats = Array.isArray(room.beats) ? room.beats as Record<string, unknown>[] : [];
  const [selected, setSelected] = useState(0);
  const [draft, setDraft] = useState(room);
  const section = (Array.isArray(draft.beats) ? draft.beats as Record<string, unknown>[] : [])[selected] ?? { hero: "", dwellSeconds: 10 };
  const patchSection = (changes: Record<string, unknown>) => { const next = [...(Array.isArray(draft.beats) ? draft.beats as Record<string, unknown>[] : [])]; next[selected] = { ...section, ...changes }; setDraft({ ...draft, beats: next }); };
  const save = async () => { await saveCatalog("rooms", draft); await refetch(); setParam({ room: null }); };
  const compositionId = String(section.composition ?? draft.composition ?? "");
  const composition = catalogs.compositions?.find((entry) => entry.id === compositionId);
  const signals = catalogs.signals ?? [];
  return <main className="cc-cx" data-testid="settings-room-editor"><DestinationHeader title={labelOf(room)} copy="Edit the board's real sections: each one has a hero metric, a dwell, a layout, and compatible supporting metrics." onExit={onExit} onBack={() => setParam({ room: null })} />
    <div className="cc-cx-room-editor"><aside className="cc-cx-preview cc-cx-preview--room"><p className="cc-cx-preview-title"><span className="cc-cx-live" /> Preview · {labelOf(draft)}</p><SettingsPreview catalogs={catalogs} room={draft} theme={catalogs.themes?.find((entry) => entry.id === draft.theme)} beatIndex={selected} /></aside>
      <section className="cc-cx-section-editor"><div className="cc-cx-section-heading"><div><p className="cc-cx-eyebrow">Sections</p><h2>What the board shows</h2></div><button type="button" className="cc-cx-primary" onClick={() => setDraft({ ...draft, beats: [...(Array.isArray(draft.beats) ? draft.beats : []), { hero: signals[0]?.id ?? "", dwellSeconds: 10 }] })}><Plus size={15} /> Add section</button></div>
        <div className="cc-cx-section-list">{(Array.isArray(draft.beats) ? draft.beats as Record<string, unknown>[] : []).map((beat, index) => <button type="button" key={index} className={index === selected ? "active" : ""} onClick={() => setSelected(index)}><span>{String(index + 1).padStart(2, "0")}</span><b>{labelOf(signals.find((entry) => entry.id === beat.hero) ?? { id: String(beat.hero ?? "section") })}</b><small>{String(beat.dwellSeconds ?? 10)} sec</small></button>)}</div>
        {beats.length ? <div className="cc-cx-section-controls"><label className="cc-field"><span className="cc-field-label">Hero</span><select value={String(section.hero ?? "")} onChange={(event) => patchSection({ hero: event.target.value })}>{signals.map((signal) => <option key={signal.id} value={signal.id}>{labelOf(signal)}</option>)}</select></label><div className="cc-field cc-cx-slider-field"><Slider label="Dwell" description="How long this section stays on screen." min={1} max={120} step={1} value={Number(section.dwellSeconds ?? 10)} formatValue={(value) => `${value} sec`} onChange={(value) => patchSection({ dwellSeconds: value })} /></div><label className="cc-field"><span className="cc-field-label">Layout</span><select value={String(section.layout ?? "standard")} onChange={(event) => patchSection({ layout: event.target.value })}><option value="standard">Standard</option><option value="wide">Wide</option></select></label>
          {composition ? <div className="cc-cx-bindings"><h3>Supporting metrics</h3>{Object.entries((composition.slots ?? {}) as Record<string, Record<string, unknown>>).map(([name, spec]) => {
            const bind = (draft.bind as Record<string, string> | undefined) ?? {};
            const options = signals.filter((signal) => signal.shape === spec.shape);
            if (spec.variadic === true) {
              const entries = Object.keys(bind).filter((key) => key.startsWith(name + ".")).sort();
              const count = entries.length;
              const setCount = (nextCount: number) => {
                const nextBind = { ...bind };
                for (let index = 0; index < 3; index += 1) {
                  const key = `${name}.${index}`;
                  if (index < nextCount) nextBind[key] = nextBind[key] ?? options[index]?.id ?? "";
                  else delete nextBind[key];
                }
                setDraft({ ...draft, bind: nextBind });
              };
              return <div className="cc-cx-variadic" key={name}><label className="cc-field"><span className="cc-field-label">Additional metrics <em>up to {String(spec.maxItems ?? 3)}</em></span><input type="number" min="0" max={Number(spec.maxItems ?? 3)} value={count} onChange={(event) => setCount(Math.max(0, Math.min(Number(spec.maxItems ?? 3), Number(event.target.value) || 0)))} /></label>{entries.map((key) => <label className="cc-field" key={key}><span className="cc-field-label">{key} <em>{measurementLabel(String(spec.shape ?? ""))}</em></span><select value={bind[key] ?? ""} onChange={(event) => setDraft({ ...draft, bind: { ...bind, [key]: event.target.value } })}><option value="">Choose a metric</option>{options.map((signal) => <option key={signal.id} value={signal.id}>{labelOf(signal)}</option>)}</select></label>)}</div>;
            }
            const current = bind[name] ?? "";
            return <label className="cc-field" key={name}><span className="cc-field-label">{name === "primary" ? "Main metric" : "Metric · " + name} <em>{measurementLabel(String(spec.shape ?? ""))}</em></span><select value={current} onChange={(event) => setDraft({ ...draft, bind: { ...bind, [name]: event.target.value } })}><option value="">Choose a metric</option>{options.map((signal) => <option key={signal.id} value={signal.id}>{labelOf(signal)}</option>)}</select></label>;
          })}</div> : null}
          <button type="button" className="cc-cx-primary" onClick={() => void save()}>Save room</button>
        </div> : <div className="cc-cx-empty-state"><h3>Ambient room</h3><p>This room has no sections. It remains a calm panorama surface.</p></div>}
      </section></div><button type="button" className="cc-cx-agent-link" onClick={() => void copyAgentPrompt("add a new layout for " + room.id)}>Hand to an agent · new layout</button>
  </main>;
}

const measurementLabel = (shape: string): string => ({ scalar: "a single number", series: "a trend", rows: "a table", meta: "a status" }[shape] ?? "a compatible metric");

function LooksDestination({ catalogs, setParam, saveCatalog, refetch, onExit }: DestinationProps) {
  const [selected, setSelected] = useState(catalogs.themes?.[0]?.id ?? "");
  const theme = catalogs.themes?.find((entry) => entry.id === selected);
  const update = async (changes: Record<string, unknown>) => { if (!theme) return; await saveCatalog("themes", { ...theme, ...changes }); await refetch(); };
  const add = async () => { const id = slugify(window.prompt("Name this look") ?? ""); if (!id) return; await saveCatalog("themes", { id, label: id.split("-").join(" "), accent: "#7ce8ff", primary: "#33d6ff" }); await refetch(); setSelected(id); };
  const duplicate = async () => { if (!theme) return; const id = slugify(theme.id + "-copy"); await saveCatalog("themes", { ...theme, id, label: labelOf(theme) + " copy" }); await refetch(); setSelected(id); };
  const remove = async () => { if (!theme || (window.prompt("Type " + theme.id + " to delete this look.") !== theme.id)) return; await deleteCatalogEntry("themes", theme.id); await refetch(); setSelected(catalogs.themes?.find((entry) => entry.id !== theme.id)?.id ?? ""); };
  return <main className="cc-cx" data-testid="settings-looks"><DestinationHeader title="Looks" copy="Choose a palette first. Fine-tune its colors only when you need a precise match." onExit={onExit} onBack={() => setParam({ destination: null })} /><section className="cc-cx-destination-body"><div className="cc-cx-section-heading"><div><p className="cc-cx-eyebrow">Palettes</p><h2>Looks</h2></div><button type="button" className="cc-cx-primary" onClick={() => void add()}><Plus size={15} /> Add look</button></div><div className="cc-cx-swatch-grid">{(catalogs.themes ?? []).map((entry) => <button type="button" key={entry.id} className={entry.id === selected ? "active" : ""} onClick={() => setSelected(entry.id)}><span style={{ background: String(entry.primary ?? entry.accent ?? "#7ce8ff") }} /><b>{labelOf(entry)}</b></button>)}</div>{theme ? <div className="cc-cx-look-controls"><h2>{labelOf(theme)}</h2><label className="cc-field"><span className="cc-field-label">Accent color</span><input type="color" value={String(theme.accent ?? "#7ce8ff")} onChange={(event) => void update({ accent: event.target.value })} /></label><div><button type="button" className="cc-cx-room-actions" onClick={() => void duplicate()}>Duplicate</button> <button type="button" className="cc-cx-room-actions" onClick={() => void remove()}>Delete</button></div><button type="button" className="cc-cx-agent-link" onClick={() => void copyAgentPrompt("create a new custom component for the " + theme.id + " look")}>Hand to an agent · custom component</button></div> : null}</section></main>;
}

function SignalsDestination({ catalogs, onExit, onAgent }: { catalogs: Record<string, CatalogEntry[]>; onExit: () => void; onAgent: (target: string) => void }) {
  return <main className="cc-cx" data-testid="settings-signals"><DestinationHeader title="Signals & sources" copy="Every metric has a measurement kind, a renderer, and a source scenario. Choose a compatible metric when editing a section." onExit={onExit} onBack={() => window.history.back()} /><section className="cc-cx-destination-body"><div className="cc-cx-signal-table" role="table">{(catalogs.signals ?? []).map((signal) => <div className="cc-cx-signal-row" role="row" key={signal.id}><strong>{labelOf(signal)}</strong><span>{measurementLabel(String(signal.shape ?? ""))}</span><span>{String(signal.component ?? signal.kind ?? "Default readout")}</span><span>{String((signal.source as Record<string, unknown> | undefined)?.integrationId ?? "Local source")}</span></div>)}</div><button type="button" className="cc-cx-agent-link" onClick={() => onAgent("connect a new upstream source")}>Hand to an agent · new source</button></section></main>;
}

function BoardSettingsDestination({ settings, rooms, onExit, onSave }: { settings?: BoardSettings; rooms: CatalogEntry[]; onExit: () => void; onSave: (settings: BoardSettings) => Promise<void> }) {
  const [draft, setDraft] = useState<BoardSettings | null>(settings ?? null);
  const [saved, setSaved] = useState(false);
  useEffect(() => { if (settings) setDraft(settings); }, [settings]);
  if (!draft) return <main className="cc-cx"><DestinationHeader title="Board settings" copy="Reading board settings…" onExit={onExit} /><p role="status" className="cc-cx-status">Reading board settings…</p></main>;
  return <main className="cc-cx" data-testid="settings-board"><DestinationHeader title="Board settings" copy="Set the cycle rhythm and decide which rooms take a turn." onExit={onExit} onBack={() => window.history.back()} /><section className="cc-cx-destination-body"><div className="cc-cx-settings-grid"><div className="cc-field cc-cx-slider-field"><Slider label="Default dwell" description="The fallback time for sections without their own dwell." min={5} max={300} step={5} value={draft.cycleSeconds} formatValue={(value) => `${value} sec`} onChange={(value) => setDraft({ ...draft, cycleSeconds: value })} /></div><label className="cc-field"><span className="cc-field-label">Transition</span><select value={draft.transition} onChange={(event) => setDraft({ ...draft, transition: event.target.value })}><option value="crossfade">Crossfade</option><option value="cut">Cut</option></select></label></div><div className="cc-cx-room-toggles">{rooms.map((room) => <label key={room.id}><input type="checkbox" checked={draft.rooms.find((item) => item.id === room.id)?.enabled ?? true} onChange={(event) => setDraft({ ...draft, rooms: draft.rooms.map((item) => item.id === room.id ? { ...item, enabled: event.target.checked } : item) })} />{labelOf(room)}</label>)}</div><button type="button" className="cc-cx-primary" onClick={() => void onSave(draft).then(() => setSaved(true))}>Save board settings</button>{saved ? <p role="status" className="cc-cx-inline-status">Board settings saved.</p> : null}</section></main>;
}
