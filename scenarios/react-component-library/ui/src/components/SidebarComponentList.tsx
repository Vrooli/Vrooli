/**
 * SidebarComponentList — the registered-component list shown inside the
 * sidebar, rendered as a two-level hierarchy (slot → category → component).
 *
 * Pulls the cached `components` query via React Query (filled by the
 * Components page) and renders leaves as links to /components/:id so users
 * can jump straight into the editor without leaving keyboard range of the
 * sidebar.
 *
 * Accessibility: this is a disclosure-style navigation tree. Slot and category
 * rows are native <button aria-expanded> toggles; component rows are links.
 * A roving tabindex keeps a single tab stop, and a keydown handler adds
 * tree-style arrow navigation: Up/Down move between visible rows, Left
 * collapses (or moves to the parent), Right expands (or moves to the first
 * child), and Home/End jump to the first/last visible row. Enter/Space use the
 * native button/link activation. Native <button>/<a> elements (rather than
 * role="treeitem" on <li>) keep the markup interactive and WCAG-clean.
 *
 * Taxonomy depth is intentionally fixed at two levels (slot, category) — the
 * catalog is a dozen components, so a flat slot/category string pair is the
 * right data model. Arbitrary-depth nesting is deferred until the catalog
 * outgrows two levels; when that happens, replace the two-string grouping
 * below with a real nested tree structure rather than adding a third string.
 */
import { useQuery } from "@tanstack/react-query";
import { NavLink, useParams } from "react-router-dom";
import { useMemo, useRef, useState, type KeyboardEvent } from "react";

import { componentsClient, type Component } from "../api/components";
import { EmptyState } from "./ui/empty-state";
import { useTranslation } from "../i18n";

interface Props {
  onNavigate?: () => void;
}

type NodeType = "slot" | "category" | "component";

interface TreeNode {
  key: string;
  type: NodeType;
  level: 1 | 2 | 3;
  label: string;
  count: number;
  slot: string;
  category?: string;
  component?: Component;
  parentKey?: string;
  hasChildren: boolean;
  testId: string;
}

const EMPTY_COMPONENTS: Component[] = [];
const humanize = (value: string) => value.split("-").join(" ");
const slotKeyOf = (slot: string) => `slot:${slot}`;
const categoryKeyOf = (slot: string, category: string) => `cat:${slot} ${category}`;

// Progressive indentation is what turns a flat list into a visible hierarchy:
// slots sit at the edge, categories tuck under them, components tuck under those.
const INDENT_BY_LEVEL: Record<TreeNode["level"], string> = {
  1: "pl-2",
  2: "pl-6",
  3: "pl-10",
};

export function SidebarComponentList({ onNavigate }: Props) {
  const { t } = useTranslation();
  const { id: active } = useParams();
  const { data, isLoading, error } = useQuery({
    queryKey: ["components", "list", "sidebar"],
    queryFn: () => componentsClient.listComponents({ limit: 100 }),
    staleTime: 30_000,
  });

  const components = data?.components ?? EMPTY_COMPONENTS;

  // Collapsed slot/category node keys. Default is fully expanded so the whole
  // catalog is visible at a glance. Kept in memory only (not persisted).
  const [collapsed, setCollapsed] = useState<ReadonlySet<string>>(() => new Set());
  const [focusedKey, setFocusedKey] = useState<string | null>(null);
  const elements = useRef(new Map<string, HTMLElement>());

  // The ordered list of rows currently visible given the collapse state. Built
  // in one pass, respecting the flat slot/category string pair per the taxonomy
  // note in the file header. Collapsed subtrees are simply skipped.
  const visible = useMemo(() => {
    const bySlot = new Map<string, Map<string, Component[]>>();
    for (const item of components) {
      const slot = item.slot || "other";
      const category = item.category || "uncategorized";
      let cats = bySlot.get(slot);
      if (!cats) {
        cats = new Map();
        bySlot.set(slot, cats);
      }
      cats.set(category, [...(cats.get(category) ?? []), item]);
    }

    const nameOf = (c: Component) => c.displayName || c.libraryId || c.id;
    const rows: TreeNode[] = [];
    for (const slot of [...bySlot.keys()].sort((a, b) => a.localeCompare(b))) {
      const cats = bySlot.get(slot);
      if (!cats) continue;
      const slotKey = slotKeyOf(slot);
      const slotCount = [...cats.values()].reduce((n, members) => n + members.length, 0);
      rows.push({
        key: slotKey,
        type: "slot",
        level: 1,
        label: humanize(slot),
        count: slotCount,
        slot,
        hasChildren: cats.size > 0,
        testId: `sidebar-component-slot-${slot}`,
      });
      if (collapsed.has(slotKey)) continue;

      for (const category of [...cats.keys()].sort((a, b) => a.localeCompare(b))) {
        const members = cats.get(category);
        if (!members) continue;
        const ordered = [...members].sort((a, b) => nameOf(a).localeCompare(nameOf(b)));
        const catKey = categoryKeyOf(slot, category);
        rows.push({
          key: catKey,
          type: "category",
          level: 2,
          label: humanize(category),
          count: ordered.length,
          slot,
          category,
          parentKey: slotKey,
          hasChildren: ordered.length > 0,
          testId: `sidebar-component-category-${slot}-${category}`,
        });
        if (collapsed.has(catKey)) continue;

        for (const component of ordered) {
          rows.push({
            key: `cmp:${component.id}`,
            type: "component",
            level: 3,
            label: nameOf(component),
            count: 0,
            slot,
            category,
            component,
            parentKey: catKey,
            hasChildren: false,
            testId: `sidebar-component-${component.id}`,
          });
        }
      }
    }
    return rows;
  }, [components, collapsed]);

  const visibleKeys = useMemo(() => new Set(visible.map((n) => n.key)), [visible]);
  const effectiveFocusedKey =
    focusedKey && visibleKeys.has(focusedKey) ? focusedKey : (visible[0]?.key ?? null);

  const focusKey = (key: string) => {
    setFocusedKey(key);
    elements.current.get(key)?.focus();
  };

  const toggle = (key: string) => {
    setFocusedKey(key);
    setCollapsed((current) => {
      const next = new Set(current);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  // Tree-style arrow navigation over the flat `visible` array. Enter/Space are
  // left to the native <button>/<a> activation of the focused row.
  const handleKeyDown = (event: KeyboardEvent<HTMLElement>) => {
    const key = effectiveFocusedKey;
    if (!key) return;
    const index = visible.findIndex((n) => n.key === key);
    if (index < 0) return;
    const node = visible[index];
    if (!node) return;

    switch (event.key) {
      case "ArrowDown": {
        event.preventDefault();
        const next = visible[index + 1];
        if (next) focusKey(next.key);
        break;
      }
      case "ArrowUp": {
        event.preventDefault();
        const prev = visible[index - 1];
        if (prev) focusKey(prev.key);
        break;
      }
      case "Home": {
        event.preventDefault();
        const first = visible[0];
        if (first) focusKey(first.key);
        break;
      }
      case "End": {
        event.preventDefault();
        const last = visible[visible.length - 1];
        if (last) focusKey(last.key);
        break;
      }
      case "ArrowRight": {
        event.preventDefault();
        if (node.hasChildren && collapsed.has(key)) {
          toggle(key); // expand; focus stays on the parent
        } else if (node.hasChildren) {
          const child = visible[index + 1];
          if (child && child.parentKey === key) focusKey(child.key);
        }
        break;
      }
      case "ArrowLeft": {
        event.preventDefault();
        if (node.hasChildren && !collapsed.has(key)) {
          toggle(key); // collapse; focus stays on the parent
        } else if (node.parentKey) {
          focusKey(node.parentKey);
        }
        break;
      }
      default:
        break;
    }
  };

  const setElementRef = (key: string) => (el: HTMLElement | null) => {
    if (el) elements.current.set(key, el);
    else elements.current.delete(key);
  };

  return (
    <div data-testid="sidebar-component-list">
      <h2 className="px-3 pb-1 text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
        {t("sidebar.components", { defaultValue: "Components" })}
      </h2>
      {isLoading && (
        <p
          data-testid="sidebar-component-list-loading"
          className="px-3 py-1 text-xs text-app-muted-foreground"
        >
          {t("sidebar.loading", { defaultValue: "Loading…" })}
        </p>
      )}
      {error && (
        <p
          data-testid="sidebar-component-list-error"
          className="px-3 py-1 text-xs text-app-danger"
        >
          {t("sidebar.error", { defaultValue: "Failed to load components" })}
        </p>
      )}
      {!isLoading && !error && visible.length === 0 && (
        <div data-testid="sidebar-component-list-empty">
          <EmptyState
            title={t("sidebar.empty", { defaultValue: "No components indexed yet" })}
            className="mx-3 mt-2 border-0 bg-transparent p-0 text-xs"
          />
        </div>
      )}
      {visible.length > 0 && (
        <ul
          aria-label={t("sidebar.components", { defaultValue: "Components" })}
          data-testid="sidebar-component-tree"
          className="flex flex-col"
        >
          {visible.map((node) => {
            const tabIndex = node.key === effectiveFocusedKey ? 0 : -1;
            const indent = INDENT_BY_LEVEL[node.level];

            if (node.type === "component") {
              const c = node.component;
              if (!c) return null;
              return (
                <li key={node.key}>
                  <NavLink
                    ref={setElementRef(node.key)}
                    to={`/components/${encodeURIComponent(c.id)}`}
                    tabIndex={tabIndex}
                    onFocus={() => setFocusedKey(node.key)}
                    onKeyDown={handleKeyDown}
                    onClick={onNavigate}
                    data-testid={node.testId}
                    data-active={active === c.id ? "true" : undefined}
                    className={({ isActive }) =>
                      [
                        "block truncate rounded-control py-1.5 pr-3 text-xs outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50",
                        indent,
                        isActive
                          ? "bg-app-surface-muted font-medium text-app-foreground"
                          : "text-app-muted-foreground hover:bg-app-surface-muted hover:text-app-foreground",
                      ].join(" ")
                    }
                    title={c.libraryId || c.displayName || c.id}
                  >
                    {node.label}
                  </NavLink>
                </li>
              );
            }

            const isExpanded = !collapsed.has(node.key);
            const headingSize = node.level === 1 ? "text-[11px]" : "text-[10px]";
            return (
              <li key={node.key}>
                <button
                  ref={setElementRef(node.key)}
                  type="button"
                  aria-expanded={isExpanded}
                  tabIndex={tabIndex}
                  onFocus={() => setFocusedKey(node.key)}
                  onKeyDown={handleKeyDown}
                  onClick={() => toggle(node.key)}
                  data-testid={node.testId}
                  className={[
                    "flex w-full items-center gap-1 rounded-control py-2 pr-3 text-left font-semibold uppercase tracking-wide text-app-muted-foreground outline-none hover:text-app-foreground focus-visible:ring-2 focus-visible:ring-app-primary/50",
                    headingSize,
                    indent,
                  ].join(" ")}
                >
                  <span aria-hidden className="w-3 shrink-0 text-center">
                    {isExpanded ? "▾" : "▸"}
                  </span>
                  <span className="truncate">{node.label}</span>
                  <span className="ml-auto tabular-nums text-app-muted-foreground/70">
                    {node.count}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
