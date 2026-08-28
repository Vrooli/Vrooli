import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, ChevronDown, Folder, FolderPlus, FolderX, Palette, Pencil, Search, SquareSlash, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { HEADER_COLORS } from "../../consts/config";
import { strings } from "../../consts/strings";
import { useGroupActions } from "../../hooks/useGroupActions";
import { useWorkspaceSync } from "../../hooks/useWorkspaceSync";
import { cn } from "../../lib/classnames";
import { useWorkspaceStore, type TabGroupMeta } from "../../stores/useWorkspaceStore";

// [REQ:P0-014a] Launcher Destination And Appearance Disclosure
// [REQ:P0-014c] Group Assignment And Administration Split

/**
 * Choosing a group is ONE surface, reached from two places.
 *
 * The launcher's destination and the session menu's "Add to group" are the
 * same decision — "which group?" — so they open the same overlay. They used
 * to be two different controls, which meant two create-a-group behaviours and
 * two places for the operator to learn the same thing.
 *
 * The layout follows the design record's picker mockup: one field that both
 * filters the list and names a new group, tight rows carrying a colour swatch
 * and a size, and a pinned create action that never scrolls away. Typing a
 * name that matches nothing is therefore the same gesture as picking one that
 * does — you never create an empty group first and assign to it afterwards.
 */

/** The sentinel value for "leave this session ungrouped". */
export const NO_GROUP_VALUE = "__none__";

interface GroupPickerOverlayProps {
  open: boolean;
  groups: TabGroupMeta[];
  /** The selected group id, or null for no group. */
  value: string | null;
  onChange: (groupId: string | null) => void;
  /** Called when the operator names a group that does not exist yet. */
  onCreate: (name: string) => void;
  onClose: () => void;
  /**
   * Hide the "No group" row. The assign flow keeps it (taking a session out
   * of its group is a real choice); a caller whose destination must be a
   * group drops it.
   */
  allowNoGroup?: boolean;
  title?: string;
}

/** How full a group is, counted from the live workspace rather than passed in. */
interface GroupSize {
  sessions: number;
  waiting: number;
}

/**
 * Size every group in one pass.
 *
 * Both callers render the same rows, and neither owns pane state, so the
 * count is read here instead of threaded through two prop chains that would
 * drift.
 */
function useGroupSizes(): Record<string, GroupSize> {
  const panes = useWorkspaceStore((s) => s.panes);
  const roles = useWorkspaceStore((s) => s.roles);
  return useMemo(() => {
    const sizes: Record<string, GroupSize> = {};
    const bucket = (groupId: string) => (sizes[groupId] ??= { sessions: 0, waiting: 0 });
    for (const pane of panes) {
      if (pane.groupId) bucket(pane.groupId).sessions += 1;
    }
    for (const role of roles) {
      if (!role.sessionId) bucket(role.groupId).waiting += 1;
    }
    return sizes;
  }, [panes, roles]);
}

/** The size summary shown on a row and on the destination trigger. */
function useSizeSummary() {
  const { t } = useTranslation();
  return useCallback((size: GroupSize | undefined) => {
    const sessions = size?.sessions ?? 0;
    const waiting = size?.waiting ?? 0;
    if (sessions === 0 && waiting === 0) return t(strings.groupPicker.emptyGroup);
    const parts = [t(strings.groupPicker.sessionCount, { count: sessions })];
    if (waiting > 0) parts.push(t(strings.groupPicker.waitingCount, { count: waiting }));
    return parts.join(" · ");
  }, [t]);
}

interface GroupRowProps {
  testId: string;
  selected: boolean;
  name: string;
  summary: string;
  swatch: React.ReactNode;
  muted?: boolean;
  onSelect: () => void;
}

/** One choosable group. "No group" uses the same row so neither reads as an afterthought. */
function GroupRow({ testId, selected, name, summary, swatch, muted = false, onSelect }: GroupRowProps) {
  return (
    <button
      type="button"
      role="option"
      aria-selected={selected}
      data-testid={testId}
      onClick={onSelect}
      className={cn(
        "flex min-h-11 w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-start transition",
        selected ? "bg-wc-accent/10" : "hover:bg-wc-surface-input",
      )}
    >
      {swatch}
      <span className="min-w-0 flex-1">
        <span className={cn("block truncate text-sm", muted ? "text-wc-text-muted" : "text-wc-text-primary")}>{name}</span>
        <span className="block truncate text-[11px] text-wc-text-faint">{summary}</span>
      </span>
      {selected && <Check className="h-4 w-4 shrink-0 text-wc-accent" aria-hidden />}
    </button>
  );
}

interface GroupEditRowProps {
  group: TabGroupMeta;
  summary: string;
  paletteOpen: boolean;
  onTogglePalette: () => void;
  onRename: (name: string) => void;
  onRecolor: (color: string) => void;
  onClose: () => void;
}

/**
 * The same group, in edit mode.
 *
 * Renaming, recolouring and closing are the three things you do to a group
 * once it exists, and none of them was reachable from the picker: the
 * operator had to know a separate manager existed. They live here, behind one
 * toggle, so the surface that lists groups is also the surface that maintains
 * them.
 */
function GroupEditRow({
  group,
  summary,
  paletteOpen,
  onTogglePalette,
  onRename,
  onRecolor,
  onClose,
}: GroupEditRowProps) {
  const { t } = useTranslation();
  return (
    <div
      data-testid={`group-picker-edit-${group.id}`}
      className="rounded-lg border border-wc-default bg-wc-surface-base/50"
    >
      <div className="flex items-center gap-1.5 p-1.5">
        <button
          type="button"
          data-testid={`group-picker-recolor-${group.id}`}
          aria-label={t(strings.manageGroups.recolorAriaLabel, { name: group.name })}
          title={t(strings.manageGroups.recolorAriaLabel, { name: group.name })}
          aria-expanded={paletteOpen}
          onClick={onTogglePalette}
          className="group/swatch flex h-8 w-8 shrink-0 items-center justify-center rounded-full border border-wc-default transition hover:ring-2 hover:ring-wc-accent/40"
          style={{ backgroundColor: group.color }}
        >
          <Palette className="h-3.5 w-3.5 text-black/50 opacity-0 transition group-hover/swatch:opacity-100" aria-hidden />
        </button>

        <input
          data-testid={`group-picker-rename-${group.id}`}
          value={group.name}
          aria-label={t(strings.manageGroups.renameAriaLabel, { name: group.name })}
          onChange={(event) => { onRename(event.target.value); }}
          className="min-h-11 min-w-0 flex-1 rounded-lg border border-transparent bg-transparent px-2 text-sm text-wc-text-primary outline-none transition hover:border-wc-default focus:border-wc-accent focus:bg-wc-surface-input"
        />

        <span className="shrink-0 text-[11px] text-wc-text-faint">{summary}</span>

        <button
          type="button"
          data-testid={`group-picker-close-${group.id}`}
          aria-label={t(strings.manageGroups.closeAriaLabelFor, { name: group.name })}
          title={t(strings.manageGroups.closeGroup)}
          onClick={onClose}
          className="flex h-11 w-9 shrink-0 items-center justify-center rounded-lg text-wc-text-muted transition hover:bg-rose-400/10 hover:text-rose-300"
        >
          <Trash2 className="h-4 w-4" aria-hidden />
        </button>
      </div>

      {paletteOpen && (
        <div data-testid={`group-picker-palette-${group.id}`} className="flex flex-wrap gap-1.5 border-t border-wc-default px-3 py-2">
          {HEADER_COLORS.map((color) => (
            <button
              key={color}
              type="button"
              data-testid={`group-picker-color-${color}`}
              title={color}
              onClick={() => { onRecolor(color); }}
              className={cn(
                "h-6 w-6 rounded-full border border-wc-default transition hover:scale-110",
                color === group.color && "ring-2 ring-wc-accent",
              )}
              style={{ backgroundColor: color }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export function GroupPickerOverlay({
  open,
  groups,
  value,
  onChange,
  onCreate,
  onClose,
  allowNoGroup = true,
  title,
}: GroupPickerOverlayProps) {
  const { t } = useTranslation();
  const sizes = useGroupSizes();
  const summarize = useSizeSummary();
  // One field, two jobs. Filtering a long list and naming a new group are the
  // same typing gesture, and splitting them into two inputs is what made the
  // old control read as a form rather than a chooser.
  const [query, setQuery] = useState("");
  // Maintenance is a mode, not a second surface. Off by default, because
  // picking a group is what the overlay is for.
  const [editing, setEditing] = useState(false);
  const [paletteId, setPaletteId] = useState<string | null>(null);

  const updateGroup = useWorkspaceStore((s) => s.updateGroup);
  const { syncUpdateGroup } = useWorkspaceSync();
  // closeGroup ungroups every member BEFORE removing the group, so closing
  // one releases its sessions rather than ending them, and it leaves an undo
  // snapshot behind. That is the whole reason this control routes through it
  // rather than deleting the row itself.
  const { closeGroup } = useGroupActions();

  useEffect(() => {
    if (open) return;
    setQuery("");
    setEditing(false);
    setPaletteId(null);
  }, [open]);

  const rename = useCallback((groupId: string, name: string) => {
    updateGroup(groupId, { name });
    syncUpdateGroup(groupId, { name });
  }, [syncUpdateGroup, updateGroup]);

  const recolor = useCallback((groupId: string, color: string) => {
    updateGroup(groupId, { color });
    syncUpdateGroup(groupId, { color });
    setPaletteId(null);
  }, [syncUpdateGroup, updateGroup]);

  if (!open) return null;

  const trimmed = query.trim();
  const needle = trimmed.toLowerCase();
  const matches = needle ? groups.filter((group) => group.name.toLowerCase().includes(needle)) : groups;
  const exact = groups.some((group) => group.name.trim().toLowerCase() === needle);
  const canCreate = trimmed.length > 0 && !exact;

  // Two sections, split on whether a group holds any session. Waiting roles
  // do NOT make a group active: a group of placeholders is still one you
  // might sweep, and its row says how many are waiting so the decision is
  // informed rather than blind.
  const isActive = (group: TabGroupMeta) => (sizes[group.id]?.sessions ?? 0) > 0;
  const active = matches.filter(isActive);
  const empty = matches.filter((group) => !isActive(group));

  const submitCreate = () => {
    if (!canCreate) return;
    onCreate(trimmed);
    setQuery("");
  };

  const renderGroup = (group: TabGroupMeta) => (
    editing
      ? (
        <GroupEditRow
          key={group.id}
          group={group}
          summary={summarize(sizes[group.id])}
          paletteOpen={paletteId === group.id}
          onTogglePalette={() => { setPaletteId((prev) => (prev === group.id ? null : group.id)); }}
          onRename={(name) => { rename(group.id, name); }}
          onRecolor={(color) => { recolor(group.id, color); }}
          onClose={() => { closeGroup(group.id); }}
        />
      )
      : (
        <GroupRow
          key={group.id}
          testId={`group-picker-option-${group.id}`}
          selected={value === group.id}
          name={group.name}
          summary={summarize(sizes[group.id])}
          swatch={
            <span
              className="h-4 w-4 shrink-0 rounded-full border border-wc-default"
              style={{ backgroundColor: group.color }}
              aria-hidden
            />
          }
          onSelect={() => { onChange(group.id); onClose(); }}
        />
      )
  );

  const sectionHeading = (icon: React.ReactNode, label: string, action?: React.ReactNode) => (
    <div className="flex items-center gap-2 px-1.5 pb-1 pt-2">
      <span className="flex flex-1 items-center gap-1.5 text-[11px] font-semibold uppercase tracking-wider text-wc-text-faint">
        {icon}
        {label}
      </span>
      {action}
    </div>
  );

  return (
    <ResponsiveDialog
      avoidKeyboard
      open
      onClose={onClose}
      size="sm"
      closeLabel={t(strings.roles.cancel)}
      title={title ?? t(strings.launcher.chooseDestination)}
      testId="group-assign-picker"
    >
      <div className="flex h-full flex-col">
        <div className="flex shrink-0 items-center gap-2 border-b border-wc-default p-3">
          <label className="relative block min-w-0 flex-1">
            <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-wc-text-faint" aria-hidden />
            <input
              data-testid="group-picker-filter"
              autoFocus
              value={query}
              aria-label={t(strings.groupPicker.filterOrCreate)}
              placeholder={t(strings.groupPicker.filterOrCreate)}
              onChange={(event) => { setQuery(event.target.value); }}
              onKeyDown={(event) => {
                if (event.key === "Enter") {
                  event.preventDefault();
                  submitCreate();
                }
              }}
              className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input py-2 ps-9 pe-3 text-sm text-wc-text-primary outline-none transition placeholder:text-wc-text-faint focus:border-wc-accent"
            />
          </label>
          {groups.length > 0 && (
            <button
              type="button"
              data-testid="group-picker-edit-toggle"
              aria-pressed={editing}
              onClick={() => { setEditing((prev) => !prev); setPaletteId(null); }}
              className={cn(
                "flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg border px-3 text-xs font-medium transition",
                editing
                  ? "border-wc-accent bg-wc-accent/10 text-wc-accent"
                  : "border-wc-default bg-wc-surface-input text-wc-text-secondary hover:border-wc-accent hover:text-wc-text-primary",
              )}
            >
              <Pencil className="h-3.5 w-3.5 shrink-0" aria-hidden />
              {editing ? t(strings.groupPicker.done) : t(strings.groupPicker.edit)}
            </button>
          )}
        </div>

        <div
          // In edit mode the rows are forms, not options: claiming listbox
          // semantics over a set of text fields would lie to a screen reader.
          role={editing ? undefined : "listbox"}
          aria-label={editing ? undefined : t(strings.launcher.destination)}
          data-testid="group-picker-list"
          className="min-h-0 flex-1 space-y-0.5 overflow-y-auto p-2"
        >
          {/* Groups holding work, then groups holding none. The split is the
              whole point of the list: the second section is the one you sweep,
              and mixing them made a long list of live and dead groups. */}
          {active.length > 0 && (
            <>
              {empty.length > 0 && sectionHeading(
                <Folder className="h-3 w-3" aria-hidden />,
                t(strings.manageGroups.sectionActive),
              )}
              <div data-testid="group-picker-section-active" className="space-y-0.5">
                {active.map(renderGroup)}
              </div>
            </>
          )}

          {empty.length > 0 && (
            <>
              {sectionHeading(
                <FolderX className="h-3 w-3" aria-hidden />,
                t(strings.manageGroups.sectionEmpty),
                <button
                  type="button"
                  data-testid="group-picker-close-all-empty"
                  onClick={() => { for (const group of empty) closeGroup(group.id); }}
                  className="shrink-0 rounded-lg px-2 py-1 text-[11px] font-medium text-wc-accent transition hover:bg-wc-surface-input"
                >
                  {t(strings.manageGroups.closeAllEmpty)}
                </button>,
              )}
              <div data-testid="group-picker-section-empty" className="space-y-0.5">
                {empty.map(renderGroup)}
              </div>
            </>
          )}

          {/* Ungrouping sits at the end of the list, muted: it is a real
              choice, but it is not the one the operator came here to make.
              It is not offered while editing — there is nothing to maintain
              about not being in a group. */}
          {allowNoGroup && !editing && (
            <GroupRow
              testId="group-picker-option-none"
              selected={value === null}
              name={t(strings.launcher.noGroup)}
              summary={t(strings.groupPicker.noGroupHint)}
              muted
              swatch={<SquareSlash className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />}
              onSelect={() => { onChange(null); onClose(); }}
            />
          )}

          {groups.length === 0 && (
            <p data-testid="group-picker-empty" className="py-6 text-center text-sm text-wc-text-muted">
              {t(strings.groupPicker.noGroupsYet)}
            </p>
          )}

          {groups.length > 0 && matches.length === 0 && !canCreate && (
            <p data-testid="group-picker-no-matches" className="py-6 text-center text-sm text-wc-text-muted">
              {t(strings.manageGroups.noMatches)}
            </p>
          )}
        </div>

        {/* Pinned, so it survives a list long enough to scroll — and OUTSIDE
            the listbox, because an action among the options breaks the
            relationship assistive tech relies on. */}
        <div className="shrink-0 border-t border-wc-default p-2">
          {editing ? (
            <p data-testid="group-picker-ungroup-note" className="px-2 py-1.5 text-[11px] text-wc-text-faint">
              {t(strings.groupPicker.ungroupNote)}
            </p>
          ) : (
          <button
            type="button"
            data-testid="group-picker-create-submit"
            disabled={!canCreate}
            onClick={submitCreate}
            className="flex min-h-11 w-full items-center gap-2.5 rounded-lg px-2.5 text-start text-sm font-medium text-wc-accent transition hover:bg-wc-surface-input disabled:cursor-not-allowed disabled:text-wc-text-faint disabled:hover:bg-transparent"
          >
            <FolderPlus className="h-4 w-4 shrink-0" aria-hidden />
            <span className="min-w-0 flex-1 truncate">
              {canCreate ? t(strings.launcher.createGroupNamed, { name: trimmed }) : t(strings.groupPicker.createHint)}
            </span>
          </button>
          )}
        </div>
      </div>
    </ResponsiveDialog>
  );
}

interface GroupDestinationTriggerProps {
  groups: TabGroupMeta[];
  value: string | null;
  onChange: (groupId: string | null) => void;
  onCreate: (name: string) => void;
  allowNoGroup?: boolean;
  label?: string;
  disabled?: boolean;
}

/**
 * The launcher's destination: one compact trigger stating where the session
 * will land, carrying the group's own colour.
 *
 * It is a trigger rather than an inline list because the dialog's vertical
 * space belongs to the choice the operator actually came to make. Pressing it
 * opens the same overlay the session menu opens.
 */
export default function GroupDestinationTrigger({
  groups,
  value,
  onChange,
  onCreate,
  allowNoGroup = true,
  label,
  disabled = false,
}: GroupDestinationTriggerProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const sizes = useGroupSizes();
  const summarize = useSizeSummary();

  const selected = groups.find((group) => group.id === value) ?? null;
  const heading = label ?? t(strings.launcher.destination);

  return (
    <div data-testid="launcher-destination" className="min-w-0 flex-1 basis-[13rem]">
      <button
        type="button"
        data-testid="launcher-destination-trigger"
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={`${heading}: ${selected ? selected.name : t(strings.launcher.noGroup)}`}
        onClick={() => { setOpen(true); }}
        className="flex min-h-11 w-full items-center gap-2 rounded-lg border bg-wc-surface-input px-3 text-start text-sm transition hover:border-wc-accent disabled:cursor-not-allowed disabled:opacity-50"
        // The destination wears the group's colour, so "which group" is
        // legible before the label is read.
        style={selected ? { borderColor: selected.color } : undefined}
      >
        {selected
          ? (
            <span
              className="h-3 w-3 shrink-0 rounded-full"
              style={{ backgroundColor: selected.color }}
              aria-hidden
            />
          )
          : <SquareSlash className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />}
        <span className="min-w-0 flex-1 truncate text-wc-text-primary">
          {selected ? t(strings.groupPicker.into, { name: selected.name }) : t(strings.launcher.noGroup)}
        </span>
        {selected && (
          <span className="shrink-0 text-[11px] text-wc-text-faint">{summarize(sizes[selected.id])}</span>
        )}
        <ChevronDown className="h-4 w-4 shrink-0 text-wc-text-faint" aria-hidden />
      </button>

      <GroupPickerOverlay
        open={open}
        groups={groups}
        value={value}
        allowNoGroup={allowNoGroup}
        title={heading}
        onChange={onChange}
        onCreate={(name) => { onCreate(name); setOpen(false); }}
        onClose={() => { setOpen(false); }}
      />
    </div>
  );
}
