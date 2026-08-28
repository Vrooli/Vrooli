import { useCallback, useEffect, useMemo, useState } from "react";
import { ArrowDownUp, Folder, FolderX, Palette, Search, X } from "lucide-react";
import { useTranslation } from "react-i18next";
import { BulkActionBar } from "@vrooli/react-component-library/BulkActionBar/1";
import { Checkbox } from "@vrooli/react-component-library/Checkbox/1";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { HEADER_COLORS } from "../consts/config";
import { strings } from "../consts/strings";
import { useGroupActions } from "../hooks/useGroupActions";
import { useWorkspaceSync } from "../hooks/useWorkspaceSync";
import { cn } from "../lib/classnames";
import { useWorkspaceStore, type TabGroupMeta } from "../stores/useWorkspaceStore";
import { Button } from "./ui/button";

// [REQ:P0-014c] Group Assignment And Administration Split
// [REQ:P0-014f] Group Auto-Close With Undo

/**
 * The group administration surface.
 *
 * This is a MANAGER, not a picker. It used to be both: opened with a session
 * context it offered per-group assign toggles, and opened without one it did
 * CRUD. That dual role is why it felt heavy — choosing a group for a single
 * session opened the whole list. Assignment now happens in an anchored picker
 * beside the tab, and this surface exists for the thing a picker cannot do:
 * seeing every group at once and acting on several.
 *
 * "Close" rather than "Delete" throughout, because that is the consequence the
 * operator experiences — the sessions survive; the folder goes away. Every
 * close routes through closeGroup, so the undo banner covers all of them and
 * no per-row confirm is needed.
 */
export default function ManageGroupsDrawer() {
  const { t } = useTranslation();
  const open = useWorkspaceStore((s) => s.manageGroupsOpen);
  const setOpen = useWorkspaceStore((s) => s.setManageGroupsOpen);
  const groups = useWorkspaceStore((s) => s.groups);
  const panes = useWorkspaceStore((s) => s.panes);
  const roles = useWorkspaceStore((s) => s.roles);
  const autoCloseEmptyGroups = useWorkspaceStore((s) => s.autoCloseEmptyGroups);
  const setAutoCloseEmptyGroups = useWorkspaceStore((s) => s.setAutoCloseEmptyGroups);
  const updateGroup = useWorkspaceStore((s) => s.updateGroup);
  const { syncUpdateGroup } = useWorkspaceSync();
  const { closeGroup, createGroup } = useGroupActions();

  const [filter, setFilter] = useState("");
  // Recent-first is the honest default: groups are transient here, so the one
  // you just made is the one you are most likely to act on.
  const [sort, setSort] = useState<"recent" | "name">("recent");
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [paletteId, setPaletteId] = useState<string | null>(null);

  const close = useCallback(() => { setOpen(false); }, [setOpen]);

  useEffect(() => {
    if (!open) return;
    setFilter("");
    setSort("recent");
    setSelectedIds([]);
    setPaletteId(null);
  }, [open]);

  const counts = useMemo(() => {
    const paneCounts = new Map<string, number>();
    const roleCounts = new Map<string, number>();
    for (const pane of panes) {
      if (pane.groupId) paneCounts.set(pane.groupId, (paneCounts.get(pane.groupId) ?? 0) + 1);
    }
    for (const role of roles) {
      if (role.sessionId === null) roleCounts.set(role.groupId, (roleCounts.get(role.groupId) ?? 0) + 1);
    }
    return { paneCounts, roleCounts };
  }, [panes, roles]);

  const filtered = useMemo(() => {
    const query = filter.trim().toLowerCase();
    const matched = query ? groups.filter((g) => g.name.toLowerCase().includes(query)) : groups;
    // "Recent" is store order, which is creation order — the list is not
    // re-sorted, so a group does not move while you are aiming at it.
    return sort === "name"
      ? [...matched].sort((a, b) => a.name.localeCompare(b.name))
      : matched;
  }, [filter, groups, sort]);

  // A group with no sessions is "empty" whether or not it holds waiting roles.
  // The roles matter for auto-close, not for this grouping — an operator
  // scanning for clutter wants to see both, with the waiting count telling
  // them which ones are deliberate.
  const active = filtered.filter((g) => (counts.paneCounts.get(g.id) ?? 0) > 0);
  const empty = filtered.filter((g) => (counts.paneCounts.get(g.id) ?? 0) === 0);
  // Counted across every group, not the filtered view: the header describes
  // the workspace, and a filter that hides clutter must not hide its count.
  const emptyTotal = groups.filter((g) => (counts.paneCounts.get(g.id) ?? 0) === 0).length;

  const toggleSelected = (groupId: string) => {
    setSelectedIds((prev) => (prev.includes(groupId) ? prev.filter((id) => id !== groupId) : [...prev, groupId]));
  };

  const closeMany = (ids: string[]) => {
    for (const id of ids) closeGroup(id);
    setSelectedIds((prev) => prev.filter((id) => !ids.includes(id)));
  };

  const handleRecolor = (groupId: string, color: string) => {
    updateGroup(groupId, { color });
    syncUpdateGroup(groupId, { color });
    setPaletteId(null);
  };

  if (!open) return null;

  const renderRow = (group: TabGroupMeta) => {
    const paneCount = counts.paneCounts.get(group.id) ?? 0;
    const waitingCount = counts.roleCounts.get(group.id) ?? 0;
    return (
      <div
        key={group.id}
        data-testid={`manage-groups-row-${group.id}`}
        className="rounded-lg border border-wc-default bg-wc-surface-base/50"
      >
        <div className="flex items-center gap-2 px-3 py-2">
          <Checkbox
            label=""
            aria-label={t(strings.manageGroups.selectGroup, { name: group.name })}
            data-testid={`manage-groups-select-${group.id}`}
            checked={selectedIds.includes(group.id)}
            onChange={() => { toggleSelected(group.id); }}
          />

          <button
            type="button"
            data-testid={`manage-groups-recolor-${group.id}`}
            aria-label={t(strings.manageGroups.recolorAriaLabel, { name: group.name })}
            title={t(strings.manageGroups.recolorAriaLabel, { name: group.name })}
            aria-expanded={paletteId === group.id}
            className="group/swatch relative flex h-7 w-7 shrink-0 items-center justify-center rounded-full border border-wc-default transition hover:ring-2 hover:ring-wc-accent/40"
            style={{ backgroundColor: group.color }}
            onClick={() => { setPaletteId((prev) => (prev === group.id ? null : group.id)); }}
          >
            <Palette className="h-3.5 w-3.5 text-black/50 opacity-0 transition group-hover/swatch:opacity-100" aria-hidden />
          </button>

          <span className="min-w-0 flex-1 truncate text-sm font-medium text-wc-text-primary">
            {group.name}
          </span>

          {waitingCount > 0 && (
            <span
              data-testid={`manage-groups-waiting-${group.id}`}
              data-waiting-count={waitingCount}
              className="shrink-0 rounded border border-dashed border-wc-default px-1.5 py-0.5 text-[11px] text-wc-text-faint"
            >
              {t(strings.roles.waitingCount, { count: waitingCount })}
            </span>
          )}

          <span
            data-testid={`manage-groups-count-${group.id}`}
            data-session-count={paneCount}
            className="shrink-0 rounded bg-wc-surface-input px-1.5 py-0.5 text-[11px] text-wc-text-secondary"
          >
            {t(strings.manageGroups.sessionCount, { count: paneCount })}
          </span>

          <button
            type="button"
            data-testid={`manage-groups-close-${group.id}`}
            aria-label={t(strings.manageGroups.closeAriaLabelFor, { name: group.name })}
            title={t(strings.manageGroups.closeGroup)}
            className="shrink-0 rounded-full p-1.5 text-wc-text-muted transition hover:bg-wc-surface-input hover:text-wc-text-primary"
            onClick={() => { closeGroup(group.id); }}
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {paletteId === group.id && (
          <div data-testid="manage-groups-palette" className="flex flex-wrap gap-1.5 border-t border-wc-default px-3 py-2">
            {HEADER_COLORS.map((color) => (
              <button
                key={color}
                type="button"
                data-testid={`manage-groups-color-${color}`}
                className={cn(
                  "h-5 w-5 rounded-full border border-wc-default transition hover:scale-110",
                  color === group.color && "ring-2 ring-wc-accent",
                )}
                style={{ backgroundColor: color }}
                title={color}
                onClick={() => { handleRecolor(group.id, color); }}
              />
            ))}
          </div>
        )}
      </div>
    );
  };

  return (
    <ResponsiveDialog
      avoidKeyboard
      open
      onClose={close}
      size="md"
      closeLabel={t(strings.manageGroups.closeAriaLabel)}
      title={t(strings.manageGroups.title)}
      testId="manage-groups-drawer"
    >
      <div className="flex h-full flex-col">
        <div className="shrink-0 space-y-2 border-b border-wc-default p-3">
          {/* What the list contains, before it is read. The empty count is
              the number that decides whether this surface is worth opening. */}
          <p data-testid="manage-groups-summary" className="text-[11px] text-wc-text-faint">
            {t(strings.manageGroups.groupCount, { count: groups.length })}
            {emptyTotal > 0 && ` · ${t(strings.manageGroups.emptyCount, { count: emptyTotal })}`}
          </p>
          <div className="flex items-center gap-2">
          <label className="relative block min-w-0 flex-1">
            <Search className="pointer-events-none absolute start-3 top-1/2 h-4 w-4 -translate-y-1/2 text-wc-text-faint" aria-hidden />
            <input
              data-testid="manage-groups-filter"
              aria-label={t(strings.manageGroups.filter)}
              placeholder={t(strings.manageGroups.filter)}
              value={filter}
              onChange={(event) => { setFilter(event.target.value); }}
              className="min-h-11 w-full rounded-lg border border-wc-default bg-wc-surface-input py-2 ps-9 pe-3 text-sm text-wc-text-primary outline-none transition placeholder:text-wc-text-faint focus:border-wc-accent"
            />
          </label>
          <button
            type="button"
            data-testid="manage-groups-sort"
            aria-label={t(strings.manageGroups.sort)}
            onClick={() => { setSort((prev) => (prev === "recent" ? "name" : "recent")); }}
            className="flex min-h-11 shrink-0 items-center gap-1.5 rounded-lg border border-wc-default bg-wc-surface-input px-3 text-xs text-wc-text-secondary transition hover:border-wc-accent hover:text-wc-text-primary"
          >
            <ArrowDownUp className="h-3.5 w-3.5 shrink-0" aria-hidden />
            {sort === "recent" ? t(strings.manageGroups.sortRecent) : t(strings.manageGroups.sortName)}
          </button>
          </div>
        </div>

        <div className="min-h-0 flex-1 space-y-4 overflow-y-auto p-4">
          {groups.length === 0 && (
            <p data-testid="manage-groups-empty" className="py-6 text-center text-sm text-wc-text-muted">
              {t(strings.manageGroups.empty)}
            </p>
          )}

          {groups.length > 0 && filtered.length === 0 && (
            <p data-testid="manage-groups-no-matches" className="py-6 text-center text-sm text-wc-text-muted">
              {t(strings.manageGroups.noMatches)}
            </p>
          )}

          {active.length > 0 && (
            <section data-testid="manage-groups-section-active" className="space-y-2">
              <div className="flex items-center gap-2 px-1 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                <Folder className="h-3.5 w-3.5" aria-hidden />
                {t(strings.manageGroups.sectionActive)}
              </div>
              {active.map(renderRow)}
            </section>
          )}

          {empty.length > 0 && (
            <section data-testid="manage-groups-section-empty" className="space-y-2">
              <div className="flex items-center gap-2 px-1">
                <div className="flex flex-1 items-center gap-2 text-xs font-semibold uppercase tracking-wider text-wc-text-faint">
                  <FolderX className="h-3.5 w-3.5" aria-hidden />
                  {t(strings.manageGroups.sectionEmpty)}
                </div>
                <button
                  type="button"
                  data-testid="manage-groups-close-all-empty"
                  onClick={() => { closeMany(empty.map((g) => g.id)); }}
                  className="min-h-11 rounded-lg px-2 text-xs font-medium text-wc-accent transition hover:bg-wc-surface-input"
                >
                  {t(strings.manageGroups.closeAllEmpty)}
                </button>
              </div>
              <p className="px-1 text-[11px] text-wc-text-faint">{t(strings.manageGroups.sectionEmptyHint)}</p>
              {empty.map(renderRow)}
            </section>
          )}
        </div>

        {selectedIds.length > 0 && (
          <div data-testid="manage-groups-bulk-bar" className="shrink-0 border-t border-wc-default p-3">
            <BulkActionBar
              selectedCount={selectedIds.length}
              totalCount={filtered.length}
              actionLabel={t(strings.manageGroups.closeSelected, { count: selectedIds.length })}
              clearLabel={t(strings.manageGroups.clearSelection)}
              selectAllLabel={t(strings.manageGroups.selectAll)}
              onAction={() => { closeMany(selectedIds); }}
              onClear={() => { setSelectedIds([]); }}
              onSelectAll={() => { setSelectedIds(filtered.map((g) => g.id)); }}
            />
          </div>
        )}

        <div className="shrink-0 space-y-3 border-t border-wc-default p-3">
          <label className="flex items-start gap-2 text-xs text-wc-text-secondary">
            <input
              type="checkbox"
              data-testid="manage-groups-auto-close"
              checked={autoCloseEmptyGroups}
              onChange={(event) => { setAutoCloseEmptyGroups(event.target.checked); }}
              className="mt-0.5 h-4 w-4 accent-[rgb(var(--wc-accent))]"
            />
            <span>
              <span className="block">{t(strings.manageGroups.autoClose)}</span>
              <span className="block text-[11px] text-wc-text-faint">{t(strings.manageGroups.autoCloseHint)}</span>
            </span>
          </label>

          <Button
            data-testid="manage-groups-create"
            variant="outline"
            className="w-full"
            onClick={() => { void createGroup(); }}
          >
            {t(strings.manageGroups.create)}
          </Button>
        </div>
      </div>
    </ResponsiveDialog>
  );
}
