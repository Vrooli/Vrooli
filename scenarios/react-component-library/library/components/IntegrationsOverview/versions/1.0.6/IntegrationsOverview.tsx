/**
 * @libraryId react-component-library:IntegrationsOverview
 * @displayName Integrations overview
 * @description What a scenario is connected to and whether each piece works: a health summary, problems with one fix each, grouped rows of accounts, tools and apps, and a detail view per row.
 * @version 1.0.6
 * @tags ["integrations","settings","status"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { useEffect, useRef, useState, type ReactNode } from "react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { SettingsList } from "@vrooli/react-component-library/SettingsList/1";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { useStrings } from "@vrooli/react-component-library/useLocale/1";
import {
  INTEGRATIONS_OVERVIEW_STRINGS,
  type IntegrationsOverviewStrings,
} from "./integrationsOverviewStrings";

export { INTEGRATIONS_OVERVIEW_STRINGS, type IntegrationsOverviewStrings };

/**
 * What a scenario is connected to and whether each piece works: accounts,
 * tools, other apps it builds on. The scenario owns the data (checks,
 * actions, copy); this owns the layout — a one-line health summary, the
 * problems with one fix each, grouped rows, and a detail view per row.
 */

/** `missing` is something optional that is not installed; `off` is something that should run and does not. */
export type IntegrationStatus = "ready" | "limited" | "off" | "missing" | "checking" | "unknown";

export interface IntegrationAction {
  label: string;
  onRun: () => void;
  pending?: boolean;
  pendingLabel?: string;
}

export interface IntegrationFeature {
  id: string;
  label: string;
  /** Defaults to true. */
  available?: boolean;
}

export interface IntegrationFact {
  label: string;
  value: ReactNode;
  monospace?: boolean;
}

export interface IntegrationItem {
  id: string;
  name: string;
  /** One short line under the name: what it is for, a version, an account. */
  subtitle?: string;
  /** The app or service behind it, shown on the detail view. */
  provider?: string;
  icon?: ReactNode;
  /** Shown in a tile when there is no icon. */
  initials?: string;
  status: IntegrationStatus;
  statusLabel?: string;
  /** When set and the item is not ready, the item is listed under the summary with its action. */
  problem?: { title: string; description?: string };
  /** The fix. Shown only while the item is not ready. */
  action?: IntegrationAction;
  /** Show the action in the row itself, e.g. Install for a tool that is not installed. */
  inlineAction?: boolean;
  features?: IntegrationFeature[];
  featuresNote?: ReactNode;
  facts?: IntegrationFact[];
  /** Extra content on the detail view, e.g. account controls. */
  detail?: ReactNode;
  /** Feedback from the last action on this item. */
  message?: { text: string; tone?: "success" | "danger" | "neutral" };
  /** Defaults to true when there is anything to show on a detail view. */
  selectable?: boolean;
}

export interface IntegrationGroup {
  id: string;
  label: string;
  items: IntegrationItem[];
  /** Wide screens place two consecutive half-width groups side by side. */
  width?: "half" | "full";
  emptyText?: ReactNode;
  footer?: ReactNode;
}

export interface IntegrationsOverviewProps {
  groups: IntegrationGroup[];
  /** Rendered first, e.g. the page's intro. */
  header?: ReactNode;
  /** Rendered under the summary, e.g. a recommendation. */
  notices?: ReactNode;
  checkedAt?: string | number | Date | null;
  loading?: boolean;
  error?: string | null;
  onRefresh?: () => void;
  refreshing?: boolean;
  /** Controls which item's detail view is open; leave undefined to let the component manage it. */
  selectedId?: string | null;
  onSelectedIdChange?: (id: string | null) => void;
  strings?: Partial<IntegrationsOverviewStrings>;
  locale?: string;
  className?: string;
}

const styles = `
[data-rcl-integrations] { container: rcl-integrations / inline-size; min-inline-size: 0; color: var(--color-foreground); }
[data-rcl-integrations-view] { display: grid; gap: var(--space-lg); min-inline-size: 0; }
[data-rcl-integrations-view][hidden] { display: none; }
[data-rcl-integrations-summary] { display: grid; min-inline-size: 0; border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); overflow: hidden; }
[data-rcl-integrations-summary-row] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-sm); padding: var(--space-sm); }
[data-rcl-integrations-tone] { color: var(--color-muted-foreground); }
[data-rcl-integrations-tone="success"] { color: var(--color-success); }
[data-rcl-integrations-tone="warning"] { color: var(--color-warning); }
[data-rcl-integrations-tone="danger"] { color: var(--color-danger); }
[data-rcl-integrations-badge] { display: inline-flex; align-items: center; justify-content: center; flex: none; inline-size: 2.25rem; block-size: 2.25rem; border-radius: 999px; background: color-mix(in srgb, currentColor 14%, transparent); }
[data-rcl-integrations-summary-text] { display: grid; gap: 2px; flex: 1 1 9rem; min-inline-size: 0; }
[data-rcl-integrations-summary-title] { font: var(--text-body); font-weight: 600; color: var(--color-foreground); }
[data-rcl-integrations-meta] { font: var(--text-body-sm); color: var(--color-muted-foreground); }
[data-rcl-integrations-attention] { display: grid; grid-template-columns: auto minmax(0, 1fr); align-items: start; gap: var(--space-2xs) var(--space-xs); padding: var(--space-sm); border-block-start: 1px solid var(--color-border); background: color-mix(in srgb, var(--color-warning) 7%, transparent); }
[data-rcl-integrations-attention] > svg { margin-block-start: 2px; }
[data-rcl-integrations-attention-text] { display: grid; gap: 2px; min-inline-size: 0; }
[data-rcl-integrations-attention-title] { font: var(--text-body); font-weight: 600; color: var(--color-foreground); }
[data-rcl-integrations-attention-action] { grid-column: 1 / -1; display: grid; }
@container rcl-integrations (min-width: 34rem) { [data-rcl-integrations-attention] { grid-template-columns: auto minmax(0, 1fr) auto; align-items: center; } [data-rcl-integrations-attention] > svg { margin-block-start: 0; } [data-rcl-integrations-attention-action] { grid-column: auto; display: block; } }
[data-rcl-integrations-message] { margin: 0; font: var(--text-body-sm); color: var(--color-muted-foreground); }
[data-rcl-integrations-message][data-tone="success"] { color: var(--color-success); }
[data-rcl-integrations-message][data-tone="danger"] { color: var(--color-danger); }
[data-rcl-integrations-groups] { display: grid; gap: var(--space-lg); min-inline-size: 0; }
@container rcl-integrations (min-width: 52rem) { [data-rcl-integrations-groups] { grid-template-columns: repeat(2, minmax(0, 1fr)); align-items: start; } [data-rcl-integrations-group][data-width="full"] { grid-column: 1 / -1; } }
[data-rcl-integrations-group] { min-inline-size: 0; }
[data-rcl-integrations-status] { font: var(--text-body-sm); white-space: nowrap; color: var(--color-muted-foreground); }
[data-rcl-integrations-status][data-status="ready"] { color: var(--color-success); }
[data-rcl-integrations-status][data-status="limited"], [data-rcl-integrations-status][data-status="off"] { color: var(--color-warning); }
[data-rcl-integrations-media] { display: inline-flex; align-items: center; justify-content: center; flex: none; inline-size: 2rem; block-size: 2rem; border-radius: var(--radius-control); background: var(--color-surface-muted); color: var(--color-foreground); font: var(--text-caption); font-weight: 700; letter-spacing: 0.02em; }
[data-rcl-integrations-media] svg { inline-size: 1.125rem; block-size: 1.125rem; }
[data-rcl-integrations-media][data-size="lg"] { inline-size: 3.25rem; block-size: 3.25rem; border-radius: var(--radius-panel); font: var(--text-body); font-weight: 700; }
[data-rcl-integrations-media][data-size="lg"] svg { inline-size: 1.625rem; block-size: 1.625rem; }
[data-rcl-integrations-empty] { margin: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-integrations-back] { justify-self: start; }
[data-rcl-integrations-detail-header] { display: flex; align-items: center; gap: var(--space-sm); min-inline-size: 0; }
[data-rcl-integrations-detail-header] h2 { margin: 0; font: var(--text-title); outline: none; }
[data-rcl-integrations-facts] { border: 1px solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface-raised); overflow: hidden; }
[data-rcl-integrations-facts] > summary { display: flex; align-items: center; min-block-size: 44px; box-sizing: border-box; padding: 0 var(--space-sm); cursor: pointer; font: var(--text-body-sm); font-weight: 500; color: var(--color-muted-foreground); }
[data-rcl-integrations-facts] > dl { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1.6fr); gap: var(--space-2xs) var(--space-sm); margin: 0; padding: 0 var(--space-sm) var(--space-sm); font: var(--text-body-sm); }
[data-rcl-integrations-facts] dt { color: var(--color-muted-foreground); }
[data-rcl-integrations-facts] dd { margin: 0; min-inline-size: 0; overflow-wrap: anywhere; }
[data-rcl-integrations-facts] dd[data-monospace] { font-family: var(--font-mono, ui-monospace, SFMono-Regular, Menlo, monospace); font-size: 0.92em; }
[data-rcl-integrations-feature-icon] { display: inline-flex; color: var(--color-success); }
[data-rcl-integrations-feature-icon][data-available="false"] { color: var(--color-muted-foreground); }
`;

const VERSION = "1.0.5";
const ROW_CLASS = "rcl-integrations-row";

function fill(template: string, values: Record<string, string | number>): string {
  return template.replace(/\{(\w+)\}/g, (match, key: string) =>
    key in values ? String(values[key]) : match,
  );
}

function Icon({ path, size = 20 }: { path: ReactNode; size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      {path}
    </svg>
  );
}

const checkCircle = (
  <>
    <circle cx="12" cy="12" r="9" />
    <path d="m8.5 12 2.5 2.5 5-5" />
  </>
);
const warningTriangle = (
  <>
    <path d="M12 3 2 20h20L12 3z" />
    <path d="M12 10v4" />
    <path d="M12 17h.01" />
  </>
);
const neutralCircle = <circle cx="12" cy="12" r="9" />;
const checkMark = <path d="m5 12 4.5 4.5L19 7" />;
const minusMark = <path d="M6 12h12" />;
const chevronLeft = <path d="m15 6-6 6 6 6" />;

type Tone = "success" | "warning" | "danger" | "neutral";

function statusText(item: IntegrationItem, s: IntegrationsOverviewStrings): string {
  if (item.statusLabel) return item.statusLabel;
  switch (item.status) {
    case "ready":
      return s.statusReady;
    case "limited":
      return s.statusLimited;
    case "off":
      return s.statusOff;
    case "missing":
      return s.statusMissing;
    case "checking":
      return s.statusChecking;
    default:
      return s.statusUnknown;
  }
}

function statusTone(status: IntegrationStatus): Tone {
  if (status === "ready") return "success";
  if (status === "limited" || status === "off") return "warning";
  return "neutral";
}

function toneIcon(tone: Tone) {
  if (tone === "success") return checkCircle;
  if (tone === "warning" || tone === "danger") return warningTriangle;
  return neutralCircle;
}

function isSelectable(item: IntegrationItem): boolean {
  if (item.selectable !== undefined) return item.selectable;
  return Boolean(item.features?.length || item.facts?.length || item.detail || item.problem);
}

function needsAttention(item: IntegrationItem): boolean {
  return item.status !== "ready" && Boolean(item.problem);
}

function Media({ item, size }: { item: IntegrationItem; size?: "lg" }) {
  if (!item.icon && !item.initials) return null;
  return (
    <span data-rcl-integrations-media data-size={size} aria-hidden="true">
      {item.icon ?? item.initials}
    </span>
  );
}

function ActionButton({
  action,
  variant,
}: {
  action: IntegrationAction;
  variant: "primary" | "secondary";
}) {
  return (
    <Button
      type="button"
      variant={variant}
      size="sm"
      disabled={action.pending}
      onClick={action.onRun}
    >
      {action.pending ? (action.pendingLabel ?? action.label) : action.label}
    </Button>
  );
}

function Message({ message }: { message?: IntegrationItem["message"] }) {
  if (!message) return null;
  return (
    <p role="status" data-rcl-integrations-message data-tone={message.tone ?? "neutral"}>
      {message.text}
    </p>
  );
}

function formatTime(value: IntegrationsOverviewProps["checkedAt"], locale?: string): string | null {
  if (value === null || value === undefined || value === "") return null;
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(locale, { hour: "numeric", minute: "2-digit" }).format(date);
}

export function IntegrationsOverview({
  groups,
  header,
  notices,
  checkedAt,
  loading = false,
  error,
  onRefresh,
  refreshing = false,
  selectedId,
  onSelectedIdChange,
  strings,
  locale,
  className,
}: IntegrationsOverviewProps) {
  useLibraryStyleSheet("react-component-library:IntegrationsOverview", VERSION, styles);
  // Precedence: the `strings` prop, then the host's library strings provider
  // (keys `integrationsOverview.<name>`), then the English defaults.
  const translate = useStrings();
  const s = Object.fromEntries(
    (Object.keys(INTEGRATIONS_OVERVIEW_STRINGS) as (keyof IntegrationsOverviewStrings)[]).map(
      (key) => [
        key,
        strings?.[key] ??
          translate(`integrationsOverview.${key}`, INTEGRATIONS_OVERVIEW_STRINGS[key]),
      ],
    ),
  ) as unknown as IntegrationsOverviewStrings;
  const [ownSelection, setOwnSelection] = useState<string | null>(null);
  const selection = selectedId === undefined ? ownSelection : selectedId;
  const rootRef = useRef<HTMLDivElement>(null);
  const headingRef = useRef<HTMLHeadingElement>(null);
  const items = groups.flatMap((group) => group.items);
  const selected = selection ? items.find((item) => item.id === selection) : undefined;

  const select = (id: string | null) => {
    if (selectedId === undefined) setOwnSelection(id);
    onSelectedIdChange?.(id);
  };

  // Focus follows the view: the detail heading on open, the row that opened
  // it on the way back. The row is found by item, not by what had focus,
  // because Safari does not focus a button that is tapped.
  const selectedKey = selected?.id;
  const lastOpened = useRef<string | null>(null);
  useEffect(() => {
    if (selectedKey) {
      lastOpened.current = selectedKey;
      headingRef.current?.focus();
      return;
    }
    const opened = lastOpened.current;
    lastOpened.current = null;
    if (!opened) return;
    const index = items.findIndex((item) => item.id === opened);
    rootRef.current
      ?.querySelector<HTMLElement>(`.${ROW_CLASS}-${index} [data-rcl-settings-row-select]`)
      ?.focus();
    // Only a change of view moves focus; new data while browsing must not.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedKey]);

  const rootProps = { "data-rcl-integrations": "", className, ref: rootRef };

  const tone: Tone = selected ? statusTone(selected.status) : "neutral";
  const showAction = selected?.action && selected.status !== "ready";
  // The overview stays mounted (hidden) under a detail view so Back returns
  // to the same row with its scroll position and focus intact.
  const detail = selected ? (
    <div data-rcl-integrations-view="detail">
      <div data-rcl-integrations-back>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => {
            select(null);
          }}
        >
          <Icon path={chevronLeft} size={18} />
          {s.back}
        </Button>
      </div>
      <header data-rcl-integrations-detail-header>
        <Media item={selected} size="lg" />
        <div data-rcl-integrations-summary-text>
          <h2 ref={headingRef} tabIndex={-1}>
            {selected.name}
          </h2>
          {selected.provider && (
            <div data-rcl-integrations-meta>
              {fill(s.providedBy, { provider: selected.provider })}
            </div>
          )}
        </div>
      </header>
      <section data-rcl-integrations-summary aria-label={statusText(selected, s)}>
        <div data-rcl-integrations-summary-row>
          <span data-rcl-integrations-tone={tone}>
            <Icon path={toneIcon(tone)} />
          </span>
          <div data-rcl-integrations-summary-text>
            <div data-rcl-integrations-summary-title>
              {selected.status !== "ready" && selected.problem
                ? selected.problem.title
                : statusText(selected, s)}
            </div>
            {selected.status !== "ready" && selected.problem?.description ? (
              <div data-rcl-integrations-meta>{selected.problem.description}</div>
            ) : selected.subtitle ? (
              <div data-rcl-integrations-meta>{selected.subtitle}</div>
            ) : null}
            <Message message={selected.message} />
          </div>
          {showAction && selected.action && (
            <ActionButton action={selected.action} variant="primary" />
          )}
        </div>
      </section>
      {selected.features && selected.features.length > 0 && (
        <SettingsList>
          <SettingsList.Group label={s.featuresTitle}>
            {selected.features.map((feature) => {
              const available = feature.available !== false;
              return (
                <SettingsList.Row
                  key={feature.id}
                  label={feature.label}
                  leading={
                    <span data-rcl-integrations-feature-icon data-available={String(available)}>
                      <Icon path={available ? checkMark : minusMark} size={18} />
                    </span>
                  }
                >
                  {!available && <span data-rcl-integrations-status>{s.featureUnavailable}</span>}
                </SettingsList.Row>
              );
            })}
            {selected.featuresNote && <p data-rcl-integrations-empty>{selected.featuresNote}</p>}
          </SettingsList.Group>
        </SettingsList>
      )}
      {selected.detail}
      {selected.facts && selected.facts.length > 0 && (
        <details data-rcl-integrations-facts>
          <summary>{s.technicalDetails}</summary>
          <dl>
            {selected.facts.map((fact) => (
              <div key={fact.label} style={{ display: "contents" }}>
                <dt>{fact.label}</dt>
                <dd data-monospace={fact.monospace ? "" : undefined}>{fact.value}</dd>
              </div>
            ))}
          </dl>
        </details>
      )}
    </div>
  ) : null;

  const total = items.length;
  // Limited counts against the total: "All ready" beside a warning would contradict itself.
  const ready = items.filter((item) => item.status === "ready").length;
  const attention = items.filter(needsAttention);
  const summaryTone: Tone = error
    ? "danger"
    : loading && total === 0
      ? "neutral"
      : attention.length > 0
        ? "warning"
        : ready === total
          ? "success"
          : "neutral";
  const summaryTitle = error
    ? s.summaryError
    : loading && total === 0
      ? s.summaryChecking
      : ready === total
        ? fill(s.summaryAllReady, { total })
        : fill(s.summarySomeReady, { ready, total });
  const checkedTime = formatTime(checkedAt, locale);
  const summaryMeta = error ?? (checkedTime ? fill(s.checkedAt, { time: checkedTime }) : null);

  return (
    <div {...rootProps} data-view={selected ? "detail" : "overview"}>
      {detail}
      <div data-rcl-integrations-view="overview" hidden={Boolean(selected)}>
        {header}
        <section data-rcl-integrations-summary aria-label={summaryTitle}>
          <div data-rcl-integrations-summary-row>
            <span data-rcl-integrations-badge data-rcl-integrations-tone={summaryTone}>
              <Icon path={toneIcon(summaryTone)} />
            </span>
            <div data-rcl-integrations-summary-text aria-live="polite">
              <div data-rcl-integrations-summary-title>{summaryTitle}</div>
              {summaryMeta && <div data-rcl-integrations-meta>{summaryMeta}</div>}
            </div>
            {onRefresh && (
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={refreshing || loading}
                onClick={onRefresh}
              >
                {refreshing || loading ? s.refreshing : s.refresh}
              </Button>
            )}
          </div>
          {attention.map((item) => (
            <div key={item.id} data-rcl-integrations-attention data-item={item.id}>
              <span data-rcl-integrations-tone="warning">
                <Icon path={warningTriangle} size={18} />
              </span>
              <div data-rcl-integrations-attention-text>
                <div data-rcl-integrations-attention-title>{item.problem?.title}</div>
                {item.problem?.description && (
                  <div data-rcl-integrations-meta>{item.problem.description}</div>
                )}
                <Message message={item.message} />
              </div>
              {item.action && (
                <div data-rcl-integrations-attention-action>
                  <ActionButton action={item.action} variant="primary" />
                </div>
              )}
            </div>
          ))}
        </section>
        {notices}
        <div data-rcl-integrations-groups>
          {groups.map((group) => (
            <div
              key={group.id}
              data-rcl-integrations-group
              data-width={group.width ?? "full"}
              data-group={group.id}
            >
              <SettingsList>
                <SettingsList.Group label={group.label}>
                  {group.items.length === 0 && (
                    <p data-rcl-integrations-empty>{group.emptyText ?? s.emptyGroup}</p>
                  )}
                  {group.items.map((item) => {
                    const inline =
                      item.inlineAction &&
                      item.action &&
                      item.status !== "ready" &&
                      !needsAttention(item);
                    return (
                      <SettingsList.Row
                        key={item.id}
                        className={`${ROW_CLASS}-${items.indexOf(item)}`}
                        label={item.name}
                        hint={
                          item.message && !needsAttention(item) ? item.message.text : item.subtitle
                        }
                        leading={item.icon || item.initials ? <Media item={item} /> : undefined}
                        onSelect={
                          isSelectable(item)
                            ? () => {
                                select(item.id);
                              }
                            : undefined
                        }
                      >
                        {inline && item.action ? (
                          <ActionButton action={item.action} variant="secondary" />
                        ) : (
                          <span data-rcl-integrations-status data-status={item.status}>
                            {statusText(item, s)}
                          </span>
                        )}
                      </SettingsList.Row>
                    );
                  })}
                  {group.footer}
                </SettingsList.Group>
              </SettingsList>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default IntegrationsOverview;
