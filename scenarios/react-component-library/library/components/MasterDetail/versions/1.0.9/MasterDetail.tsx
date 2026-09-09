/**
 * @libraryId react-component-library:MasterDetail
 * @displayName MasterDetail
 * @description The coordinated collection-and-detail layout preserving selection, scroll position, route state, and transition continuity, and drilling in rather than splitting on compact viewports.
 * @version 1.0.9
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

/** @vrooliComponentSource react-component-library:MasterDetail
 * @vrooliComponentSourceSlot navigation.master-detail */
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type ReactNode,
} from "react";
import { SplitView } from "@vrooli/react-component-library/SplitView/1";
import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";

export interface MasterDetailItem<T = unknown> {
  id: string;
  title: string;
  summary?: string;
  meta?: string;
  value: T;
  disabled?: boolean;
}

export type MasterDetailStatus = "default" | "loading" | "empty" | "partial" | "request-error";

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
  renderMaster?: (item: MasterDetailItem<T>, state: MasterDetailRenderState) => ReactNode;
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
[data-rcl-master-detail] { display: grid; gap: var(--space-md); min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-master-detail-header] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-master-detail-kicker] { color: var(--color-primary); font: var(--text-overline); letter-spacing: .1em; text-transform: uppercase; }
[data-rcl-master-detail-title] { margin: 0; font: var(--text-title); }
[data-rcl-master-detail-description] { max-inline-size: 68ch; color: var(--color-muted-foreground); font: var(--text-body); }
[data-rcl-master-detail-panel] { min-inline-size: 0; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); box-shadow: var(--elev-raised); }
[data-rcl-master-detail-list] { display: grid; gap: var(--space-2xs); max-block-size: min(62vh, 38rem); overflow: auto; padding: var(--space-xs); overscroll-behavior: contain; }
[data-rcl-master-detail-list]::-webkit-scrollbar { inline-size: .55rem; }
[data-rcl-master-detail-list]::-webkit-scrollbar-thumb { border-radius: 999px; background: var(--color-border-strong); }
[data-rcl-master-detail-item] { display: grid; gap: var(--space-3xs); inline-size: 100%; min-block-size: var(--tap-target-min); padding: var(--space-sm); border: var(--border-hairline) solid transparent; border-radius: var(--radius-control); background: transparent; color: inherit; text-align: start; cursor: pointer; transition: background var(--dur-quick) var(--ease-standard), border-color var(--dur-quick) var(--ease-standard), transform var(--dur-quick) var(--ease-standard); }
[data-rcl-master-detail-item]:hover { border-color: var(--color-border); background: var(--color-surface-muted); }
[data-rcl-master-detail-item][aria-current="true"] { border-color: color-mix(in srgb, var(--color-primary) 42%, var(--color-border)); background: color-mix(in srgb, var(--color-primary) 9%, var(--color-surface-raised)); }
[data-rcl-master-detail-item]:disabled { cursor: not-allowed; opacity: .5; }
[data-rcl-master-detail-item-title] { overflow-wrap: anywhere; font: var(--text-label); }
[data-rcl-master-detail-item-summary] { overflow-wrap: anywhere; color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-master-detail-item-meta] { color: var(--color-muted-foreground); font: var(--text-caption); }
[data-rcl-master-detail-detail] { display: grid; align-content: start; gap: var(--space-md); min-block-size: min(34rem, 62vh); padding: clamp(var(--space-md), 3vw, var(--space-xl)); }
[data-rcl-master-detail-detail-header] { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--space-sm); }
[data-rcl-master-detail-detail-copy] { display: grid; gap: var(--space-2xs); min-inline-size: 0; }
[data-rcl-master-detail-detail-title] { margin: 0; overflow-wrap: break-word; font: var(--text-heading); }
[data-rcl-master-detail-detail-body] { min-inline-size: 0; overflow-wrap: anywhere; font: var(--text-body); }
[data-rcl-master-detail-back] { display: inline-flex; align-items: center; justify-content: center; min-block-size: var(--tap-target-min); padding: var(--space-2xs) var(--space-sm); border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-control); background: var(--color-surface-muted); color: inherit; font: var(--text-label); cursor: pointer; }
[data-rcl-master-detail-partial] { min-block-size: 0; padding: var(--space-xs) 0; color: var(--color-muted-foreground); text-align: start; }
[data-rcl-master-detail-state] { display: grid; place-items: center; min-block-size: 12rem; gap: var(--space-xs); padding: var(--space-xl); color: var(--color-muted-foreground); text-align: center; font: var(--text-body); }
[data-rcl-master-detail-state="error"] { color: var(--color-danger); }
[data-rcl-master-detail-state="loading"]::before { content: ""; inline-size: 1.3rem; block-size: 1.3rem; border: 2px solid currentColor; border-block-start-color: transparent; border-radius: 50%; animation: rcl-master-detail-spin .8s linear infinite; }
@keyframes rcl-master-detail-spin { to { transform: rotate(360deg); } }
@media (max-width: 52rem) { [data-rcl-master-detail] { gap: var(--space-sm); } [data-rcl-master-detail-list] { max-block-size: none; } [data-rcl-master-detail-detail] { min-block-size: min(32rem, 70vh); padding: var(--space-md); } }


`;

function State({ status, children }: { status: MasterDetailStatus; children: ReactNode }) {
  return (
    <div
      data-rcl-master-detail-state={status === "request-error" ? "error" : status}
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
  label,
  breakpoint = "(max-width: 52rem)",
  className,
  style,
}: MasterDetailProps<T>) {
  const libraryStrings = useStrings();
  label =
    label ??
    libraryStrings("navigation.master-detail.master-detail-workspace", "Master detail workspace");
  const compact = useMediaQuery(breakpoint);
  const controlled = selectedId !== undefined;
  const [internalSelectedId, setInternalSelectedId] = useState(defaultSelectedId);
  const currentId = controlled ? (selectedId ?? undefined) : internalSelectedId;
  const [listScrollTop, setListScrollTop] = useState(0);
  const listRef = useRef<HTMLDivElement>(null);
  const selected = useMemo(() => items.find((item) => item.id === currentId), [currentId, items]);

  useEffect(() => {
    if (compact && !selected && listRef.current) listRef.current.scrollTop = listScrollTop;
  }, [compact, listScrollTop, selected]);

  const choose = useCallback(
    (item: MasterDetailItem<T>) => {
      if (item.disabled) return;
      if (compact && listRef.current) setListScrollTop(listRef.current.scrollTop);
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
      aria-describedby={item.summary ? `master-detail-summary-${item.id}` : undefined}
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
            <span id={`master-detail-summary-${item.id}`} data-rcl-master-detail-item-summary>
              {item.summary}
            </span>
          ) : null}
          {item.meta ? <span data-rcl-master-detail-item-meta>{item.meta}</span> : null}
        </>
      )}
    </button>
  );

  const masterState =
    status === "loading" ? (
      <State status={status}>{statusMessage ?? "Loading records…"}</State>
    ) : status === "empty" || (status === "default" && items.length === 0) ? (
      <State status="empty">{statusMessage ?? "Nothing needs your attention yet."}</State>
    ) : status === "request-error" ? (
      <State status={status}>{errorMessage}</State>
    ) : (
      <div
        ref={listRef}
        data-rcl-master-detail-list
        role="list"
        aria-label={libraryStrings("navigation.master-detail.records", "Records")}
      >
        {items.map(renderMasterItem)}
      </div>
    );

  const detailState =
    status === "loading" ? (
      <State status={status}>{statusMessage ?? "Loading detail…"}</State>
    ) : status === "request-error" ? (
      <State status="empty">
        {libraryStrings(
          "navigation.master-detail.detail-is-unavailable-until-the-collection-recon",
          "Detail is unavailable until the collection reconnects.",
        )}
      </State>
    ) : !selected ? (
      <State status="empty">{statusMessage ?? "Choose a record to inspect its details."}</State>
    ) : (
      <div data-rcl-master-detail-detail-body>
        {renderDetail ? renderDetail(selected) : <p>{selected.summary ?? selected.title}</p>}
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
            <span data-rcl-master-detail-kicker>{compact ? "Selected record" : "Detail"}</span>
            <h2 data-rcl-master-detail-detail-title>{selected?.title ?? "Detail"}</h2>
          </div>
          {compact && selected ? (
            <button
              data-testid="navigation.master-detail"
              data-rcl-master-detail-back
              type="button"
              onClick={back}
            >
              {libraryStrings("navigation.master-detail.back-to-list", "Back to list")}
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
      aria-label={libraryStrings("navigation.master-detail.record-collection", "Record collection")}
    >
      {masterState}
    </section>
  );

  return (
    <section data-rcl-master-detail className={className} style={style} aria-label={label}>
      <StyleSheet libraryId="react-component-library:MasterDetail" version="1.0.8" css={styles} />
      <header data-rcl-master-detail-header>
        <span data-rcl-master-detail-kicker>
          {libraryStrings("navigation.master-detail.collection-workspace", "Collection workspace")}
        </span>
        <h1 data-rcl-master-detail-title>{label}</h1>
        <span data-rcl-master-detail-description>
          {libraryStrings(
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
