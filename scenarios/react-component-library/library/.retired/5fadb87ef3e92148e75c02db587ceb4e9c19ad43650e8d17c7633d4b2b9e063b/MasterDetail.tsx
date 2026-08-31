/**
 * @libraryId react-component-library:MasterDetail
 * @displayName MasterDetail
 * @description A coordinated collection-and-detail layout that preserves selection and list position while drilling into detail on compact screens.
 * @version 1.0.4
 * @tags ["navigation","responsive","collection","detail","routing","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

/** @vrooliComponentSource navigation.master-detail */
import { resolveStrings } from "@vrooli/react-component-library/useLocale/1.0.1";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { SplitView } from "@vrooli/react-component-library/SplitView/1.0.0";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1.0.0";

export interface MasterDetailItem<T = unknown> {
  id: string;
  title: string;
  summary?: string;
  meta?: string;
  value: T;
  disabled?: boolean;
}

export type MasterDetailStatus =
  | "default"
  | "loading"
  | "empty"
  | "partial"
  | "request-error";

export interface MasterDetailRenderState {
  selected: boolean;
  select: () => void;
}

export interface MasterDetailProps<T = unknown> {
  items: MasterDetailItem<T>[];
  selectedId?: string | null;
  defaultSelectedId?: string;
  onSelect?: (item: MasterDetailItem<T>) => void;
  onNavigate?: (item: MasterDetailItem<T>) => void;
  onBack?: () => void;
  renderMaster?: (
    item: MasterDetailItem<T>,
    state: MasterDetailRenderState,
  ) => ReactNode;
  renderDetail?: (item: MasterDetailItem<T>) => ReactNode;
  status?: MasterDetailStatus;
  statusMessage?: ReactNode;
  errorMessage?: ReactNode;
  label?: string;
  breakpoint?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-master-detail] { display: grid; gap: var(--space-md, 1rem); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
[data-rcl-master-detail-header] { display: grid; gap: var(--space-2xs, .35rem); min-inline-size: 0; }
[data-rcl-master-detail-kicker] { color: var(--color-primary, #2563eb); font: var(--text-overline, 700 .68rem/1.1 system-ui, sans-serif); letter-spacing: .1em; text-transform: uppercase; }
[data-rcl-master-detail-title] { margin: 0; font: var(--text-title, 700 1.35rem/1.2 system-ui, sans-serif); }
[data-rcl-master-detail-description] { max-inline-size: 68ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body, 400 .95rem/1.5 system-ui, sans-serif); }
[data-rcl-master-detail-panel] { min-inline-size: 0; overflow: hidden; border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-panel, 1rem); background: var(--color-surface-raised, #fff); box-shadow: var(--elev-raised, 0 3px 12px rgb(15 23 42 / .06)); }
[data-rcl-master-detail-list] { display: grid; gap: var(--space-2xs, .35rem); max-block-size: min(62vh, 38rem); overflow: auto; padding: var(--space-xs, .625rem); overscroll-behavior: contain; }
[data-rcl-master-detail-list]::-webkit-scrollbar { inline-size: .55rem; }
[data-rcl-master-detail-list]::-webkit-scrollbar-thumb { border-radius: 999px; background: var(--color-border-strong, #94a3b8); }
[data-rcl-master-detail-item] { display: grid; gap: var(--space-3xs, .2rem); inline-size: 100%; min-block-size: var(--tap-target-min, 44px); padding: var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid transparent; border-radius: var(--radius-control, .625rem); background: transparent; color: inherit; text-align: start; cursor: pointer; transition: background var(--dur-quick, 160ms) var(--ease-standard, ease), border-color var(--dur-quick, 160ms) var(--ease-standard, ease), transform var(--dur-quick, 160ms) var(--ease-standard, ease); }
[data-rcl-master-detail-item]:hover { border-color: var(--color-border, #cbd5e1); background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-master-detail-item][aria-current="true"] { border-color: color-mix(in srgb, var(--color-primary, #2563eb) 42%, var(--color-border, #cbd5e1)); background: color-mix(in srgb, var(--color-primary, #2563eb) 9%, var(--color-surface-raised, #fff)); }
[data-rcl-master-detail-item]:disabled { cursor: not-allowed; opacity: .5; }
[data-rcl-master-detail-item-title] { overflow-wrap: anywhere; font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); }
[data-rcl-master-detail-item-summary] { overflow-wrap: anywhere; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.35 system-ui, sans-serif); }
[data-rcl-master-detail-item-meta] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 .75rem/1.2 system-ui, sans-serif); }
[data-rcl-master-detail-detail] { display: grid; align-content: start; gap: var(--space-md, 1rem); min-block-size: min(34rem, 62vh); padding: clamp(var(--space-md, 1rem), 3vw, var(--space-xl, 2rem)); }
[data-rcl-master-detail-detail-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm, .75rem); }
[data-rcl-master-detail-detail-copy] { display: grid; gap: var(--space-2xs, .35rem); min-inline-size: 0; }
[data-rcl-master-detail-detail-title] { margin: 0; overflow-wrap: break-word; font: var(--text-heading, 750 1.5rem/1.15 system-ui, sans-serif); }
[data-rcl-master-detail-detail-body] { min-inline-size: 0; overflow-wrap: anywhere; font: var(--text-body, 400 .95rem/1.5 system-ui, sans-serif); }
[data-rcl-master-detail-back] { display: inline-flex; align-items: center; justify-content: center; min-block-size: var(--tap-target-min, 44px); padding: var(--space-2xs, .35rem) var(--space-sm, .75rem); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, .625rem); background: var(--color-surface-muted, #f1f5f9); color: inherit; font: var(--text-label, 650 .9rem/1.35 system-ui, sans-serif); cursor: pointer; }
[data-rcl-master-detail-partial] { min-block-size: 0; padding: var(--space-xs, .625rem) 0; color: var(--color-muted-foreground, #64748b); text-align: start; }
[data-rcl-master-detail-state] { display: grid; place-items: center; min-block-size: 12rem; gap: var(--space-xs, .625rem); padding: var(--space-xl, 2rem); color: var(--color-muted-foreground, #64748b); text-align: center; font: var(--text-body, 400 .95rem/1.5 system-ui, sans-serif); }
[data-rcl-master-detail-state="error"] { color: var(--color-danger, #b42318); }
[data-rcl-master-detail-state="loading"]::before { content: ""; inline-size: 1.3rem; block-size: 1.3rem; border: 2px solid currentColor; border-block-start-color: transparent; border-radius: 50%; animation: rcl-master-detail-spin .8s linear infinite; }
@keyframes rcl-master-detail-spin { to { transform: rotate(360deg); } }
@media (max-width: 52rem) { [data-rcl-master-detail] { gap: var(--space-sm, .75rem); } [data-rcl-master-detail-list] { max-block-size: none; } [data-rcl-master-detail-detail] { min-block-size: min(32rem, 70vh); padding: var(--space-md, 1rem); } }


`;

function State({
  status,
  children,
}: {
  status: MasterDetailStatus;
  children: ReactNode;
}) {
  return (
    <div
      data-rcl-master-detail-state={
        status === "request-error" ? "error" : status
      }
      role={status === "request-error" ? "alert" : "status"}
    >
      {children}
    </div>
  );
}

export const MasterDetail = withClassName(function MasterDetail<T>({
  items,
  selectedId,
  defaultSelectedId,
  onSelect,
  onNavigate,
  onBack,
  renderMaster,
  renderDetail,
  status = "default",
  statusMessage,
  errorMessage = "We couldn’t load this collection. Try again when the connection is available.",
  label = resolveStrings(
    "navigation.master-detail.master-detail-workspace",
    "Master detail workspace",
  ),
  breakpoint = "(max-width: 52rem)",
  className,
  style,
}: MasterDetailProps<T>) {
  const compact = useMediaQuery(breakpoint);
  const controlled = selectedId !== undefined;
  const [internalSelectedId, setInternalSelectedId] =
    useState(defaultSelectedId);
  const currentId = controlled ? (selectedId ?? undefined) : internalSelectedId;
  const [listScrollTop, setListScrollTop] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const selected = useMemo(
    () => items.find((item) => item.id === currentId),
    [currentId, items],
  );

  useEffect(() => {
    if (compact && !selected && listRef.current)
      listRef.current.scrollTop = listScrollTop;
  }, [compact, listScrollTop, selected]);

  const choose = useCallback(
    (item: MasterDetailItem<T>) => {
      if (item.disabled) return;
      if (compact && listRef.current)
        setListScrollTop(listRef.current.scrollTop);
      if (!controlled) setInternalSelectedId(item.id);
      onSelect?.(item);
      onNavigate?.(item);
    },
    [compact, controlled, onNavigate, onSelect],
  );

  const back = useCallback(() => {
    if (!controlled) setInternalSelectedId(undefined);
    onBack?.();
  }, [controlled, onBack]);

  const renderMasterItem = (item: MasterDetailItem<T>) => (
    <button
      data-testid="navigation.master-detail"
      key={item.id}
      type="button"
      data-rcl-master-detail-item
      aria-label={item.title}
      aria-describedby={
        item.summary ? `master-detail-summary-${item.id}` : undefined
      }
      aria-current={item.id === currentId ? "true" : undefined}
      disabled={item.disabled}
      onClick={() => choose(item)}
    >
      {renderMaster ? (
        renderMaster(item, {
          selected: item.id === currentId,
          select: () => choose(item),
        })
      ) : (
        <>
          <span data-rcl-master-detail-item-title>{item.title}</span>
          {item.summary ? (
            <span
              id={`master-detail-summary-${item.id}`}
              data-rcl-master-detail-item-summary
            >
              {item.summary}
            </span>
          ) : null}
          {item.meta ? (
            <span data-rcl-master-detail-item-meta>{item.meta}</span>
          ) : null}
        </>
      )}
    </button>
  );

  const masterState =
    status === "loading" ? (
      <State status={status}>{statusMessage ?? "Loading records…"}</State>
    ) : status === "empty" || (status === "default" && items.length === 0) ? (
      <State status="empty">
        {statusMessage ?? "Nothing needs your attention yet."}
      </State>
    ) : status === "request-error" ? (
      <State status={status}>{errorMessage}</State>
    ) : (
      <div
        ref={listRef}
        data-rcl-master-detail-list
        role="list"
        aria-label={resolveStrings(
          "navigation.master-detail.records",
          "Records",
        )}
      >
        {items.map(renderMasterItem)}
      </div>
    );

  const detailState =
    status === "loading" ? (
      <State status={status}>{statusMessage ?? "Loading detail…"}</State>
    ) : status === "request-error" ? (
      <State status="empty">
        {resolveStrings(
          "navigation.master-detail.detail-is-unavailable-until-the-collection-recon",
          "Detail is unavailable until the collection reconnects.",
        )}
      </State>
    ) : !selected ? (
      <State status="empty">
        {statusMessage ?? "Choose a record to inspect its details."}
      </State>
    ) : (
      <div data-rcl-master-detail-detail-body>
        {renderDetail ? (
          renderDetail(selected)
        ) : (
          <p>{selected.summary ?? selected.title}</p>
        )}
      </div>
    );

  const detailPanel = (
    <section
      data-rcl-master-detail-panel
      aria-label={selected ? `${selected.title} details` : "Detail"}
    >
      <div data-rcl-master-detail-detail>
        <div data-rcl-master-detail-detail-header>
          <div data-rcl-master-detail-detail-copy>
            <span data-rcl-master-detail-kicker>
              {compact ? "Selected record" : "Detail"}
            </span>
            <h2 data-rcl-master-detail-detail-title>
              {selected?.title ?? "Detail"}
            </h2>
          </div>
          {compact && selected ? (
            <button
              data-testid="navigation.master-detail"
              data-rcl-master-detail-back
              type="button"
              onClick={back}
            >
              {resolveStrings(
                "navigation.master-detail.back-to-list",
                "Back to list",
              )}
            </button>
          ) : null}
        </div>
        {status === "partial" && (
          <div data-rcl-master-detail-partial role="status">
            {statusMessage ?? "Some detail fields are still arriving."}
          </div>
        )}
        {detailState}
      </div>
    </section>
  );

  const collectionPanel = (
    <section
      data-rcl-master-detail-panel
      aria-label={resolveStrings(
        "navigation.master-detail.record-collection",
        "Record collection",
      )}
    >
      {masterState}
    </section>
  );

  return (
    <section
      data-rcl-master-detail
      className={className}
      style={style}
      aria-label={label}
    >
      <StyleSheet name="masterdetail-1-0-4-1" css={styles} />
      <header data-rcl-master-detail-header>
        <span data-rcl-master-detail-kicker>
          {resolveStrings(
            "navigation.master-detail.collection-workspace",
            "Collection workspace",
          )}
        </span>
        <h1 data-rcl-master-detail-title>{label}</h1>
        <span data-rcl-master-detail-description>
          {resolveStrings(
            "navigation.master-detail.keep-the-collection-in-reach-while-inspecting-on",
            "Keep the collection in reach while inspecting one record at a time.",
          )}
        </span>
      </header>
      {compact && selected ? (
        detailPanel
      ) : compact ? (
        collectionPanel
      ) : (
        <SplitView
          primaryLabel="Record collection"
          secondaryLabel="Record detail"
          primary={collectionPanel}
          secondary={detailPanel}
        />
      )}
    </section>
  );
});
