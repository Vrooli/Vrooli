/**
 * @libraryId react-component-library:SettingsList
 * @displayName Settings List
 * @description Reusable settings list component implementation for the React component library.
 * @version 1.1.0
 * @tags []
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource react-component-library:SettingsList
 * @vrooliComponentSourceSlot settings.settings-list */
import {
  Children,
  cloneElement,
  createContext,
  isValidElement,
  useContext,
  useId,
  type HTMLAttributes,
  type ReactElement,
  type ReactNode,
} from "react";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";

export type SettingsListVariant = "auto" | "card" | "flush";
export type SettingsListDensity = "comfortable" | "compact";
export type SettingsControlAppetite = "compact" | "wide";

export interface SettingsListProps extends Omit<HTMLAttributes<HTMLElement>, "children"> {
  children: ReactNode;
  variant?: SettingsListVariant;
  density?: SettingsListDensity;
}
export interface SettingsListIntroProps {
  eyebrow?: ReactNode;
  title: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
}
export interface SettingsListGroupProps {
  label?: ReactNode;
  children: ReactNode;
  className?: string;
}
export interface SettingsListRowProps {
  label: ReactNode;
  hint?: ReactNode;
  control?: SettingsControlAppetite;
  children?: ReactNode;
  className?: string;
  /** An icon, avatar or logo before the label. */
  leading?: ReactNode;
  /**
   * Makes the whole row open something (a detail view, a sub-page). The label
   * becomes the button, the row shows a chevron, and controls in the row stay
   * independently clickable.
   */
  onSelect?: () => void;
}

const SettingsRowLabelContext = createContext<string | undefined>(undefined);
export function useSettingsRowLabelId() {
  return useContext(SettingsRowLabelContext);
}

const styles = `
[data-rcl-settings-list] { --rcl-settings-section-gap: var(--space-lg); --rcl-settings-group-gap: var(--space-xs); --rcl-settings-row-block: var(--space-sm); --rcl-settings-row-inline: var(--space-md); display: grid; gap: var(--rcl-settings-section-gap); min-inline-size: 0; container: rcl-settings / inline-size; }
[data-rcl-settings-list][data-density="compact"] { --rcl-settings-section-gap: var(--space-md); --rcl-settings-group-gap: var(--space-2xs); --rcl-settings-row-block: var(--space-xs); --rcl-settings-row-inline: var(--space-sm); }
[data-rcl-settings-list][data-variant="flush"] { --rcl-settings-row-inline: 0px; }
[data-rcl-settings-intro] { display: grid; gap: var(--space-3xs); }
[data-rcl-settings-intro-eyebrow], [data-rcl-settings-group-label] { color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: var(--tracking-caps); text-transform: uppercase; }
[data-rcl-settings-intro-title] { margin: 0; color: var(--color-foreground); font: var(--text-title); }
[data-rcl-settings-intro-description] { margin: 0; max-inline-size: 68ch; color: var(--color-muted-foreground); font: var(--text-body-sm); }
[data-rcl-settings-group] { display: grid; gap: var(--rcl-settings-group-gap); min-inline-size: 0; }
[data-rcl-settings-group-surface] { display: grid; min-inline-size: 0; border: 0 solid transparent; border-radius: 0; background: transparent; overflow: hidden; }
[data-rcl-settings-list]:not([data-variant="flush"]) [data-rcl-settings-group-surface] { border-color: var(--color-border); border-width: 1px; border-radius: var(--radius-panel); background: var(--color-surface-raised); }
[data-rcl-settings-list]:not([data-variant="flush"]) [data-rcl-settings-group-label] { padding-inline: var(--rcl-settings-row-inline); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]) { margin-inline: var(--rcl-settings-row-inline); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]):first-child { margin-block-start: var(--rcl-settings-row-block); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]):last-child { margin-block-end: var(--rcl-settings-row-block); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]) + :not([data-rcl-settings-row]) { margin-block-start: var(--rcl-settings-group-gap); }
[data-rcl-settings-row] { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: "label control" "hint control"; align-items: center; column-gap: var(--space-md); min-block-size: 48px; padding: var(--rcl-settings-row-block) var(--rcl-settings-row-inline); color: var(--color-foreground); }
[data-rcl-settings-row] + [data-rcl-settings-row], [data-rcl-settings-group-surface] > :not([data-rcl-settings-row]) + [data-rcl-settings-row] { border-block-start: 1px solid var(--color-border); }
[data-rcl-settings-row] { position: relative; }
[data-rcl-settings-row][data-has-leading] { grid-template-columns: auto minmax(0, 1fr) auto; grid-template-areas: "leading label control" "leading hint control"; }
[data-rcl-settings-row][data-control="wide"][data-has-leading] { grid-template-columns: auto minmax(0, 1fr); grid-template-areas: "leading label" "leading hint" "leading control"; }
[data-rcl-settings-row-leading] { grid-area: leading; display: flex; align-items: center; margin-inline-end: var(--space-xs); }
[data-rcl-settings-row-select] { all: unset; box-sizing: border-box; cursor: pointer; color: inherit; text-align: start; overflow-wrap: anywhere; }
[data-rcl-settings-row-select]::after { content: ""; position: absolute; inset: 0; }
[data-rcl-settings-row-select]:focus-visible { outline: none; }
[data-rcl-settings-row-select]:focus-visible::after { outline: 2px solid var(--color-focus); outline-offset: -2px; }
[data-rcl-settings-row][data-selectable]:hover { background: color-mix(in srgb, var(--color-foreground) 4%, transparent); }
[data-rcl-settings-row][data-selectable] [data-rcl-settings-row-control] { display: flex; align-items: center; justify-content: flex-end; gap: var(--space-xs); }
[data-rcl-settings-row-control] { position: relative; z-index: 1; }
[data-rcl-settings-row-chevron] { display: inline-flex; color: var(--color-muted-foreground); pointer-events: none; }
[data-rcl-settings-row-text] { display: contents; }
[data-rcl-settings-row-label] { grid-area: label; min-inline-size: 0; font: var(--text-body); font-weight: 500; }
[data-rcl-settings-row-hint] { grid-area: hint; min-inline-size: 0; color: var(--color-muted-foreground); font: var(--text-body-sm); overflow-wrap: anywhere; }
[data-rcl-settings-row-control] { grid-area: control; min-inline-size: 0; max-inline-size: 100%; justify-self: end; }
[data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(0, 1fr); grid-template-areas: "label" "hint" "control"; row-gap: var(--space-xs); }
[data-rcl-settings-row][data-control="wide"] [data-rcl-settings-row-control] { inline-size: 100%; justify-self: stretch; }
@container rcl-settings (max-width: 33.99rem) { [data-rcl-settings-list][data-density="comfortable"]:not([data-variant="flush"]) [data-rcl-settings-group] { --rcl-settings-row-inline: var(--space-sm); } }
@container rcl-settings (max-width: 25.99rem) { [data-rcl-settings-row][data-control="compact"] { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-xs) var(--space-sm); } [data-rcl-settings-row][data-control="compact"] [data-rcl-settings-row-text] { display: grid; flex: 1 1 11rem; gap: var(--space-3xs); min-inline-size: 0; } [data-rcl-settings-row][data-control="compact"] [data-rcl-settings-row-text] > * { grid-area: auto; } [data-rcl-settings-row][data-control="compact"] [data-rcl-settings-row-control] { flex: 0 1 auto; margin-inline-start: auto; } [data-rcl-settings-row][data-control="compact"] [data-rcl-settings-row-leading] { flex: 0 0 auto; margin-inline-end: 0; } }
@container rcl-settings (min-width: 26rem) { [data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(0, 1fr) minmax(12rem, .65fr); grid-template-areas: "label control" "hint control"; row-gap: 0; } [data-rcl-settings-row][data-control="wide"] [data-rcl-settings-row-control] { inline-size: auto; justify-self: end; } [data-rcl-settings-row][data-control="wide"][data-has-leading] { grid-template-columns: auto minmax(0, 1fr) minmax(12rem, .65fr); grid-template-areas: "leading label control" "leading hint control"; } }
@container rcl-settings (min-width: 52rem) { [data-rcl-settings-row] { grid-template-columns: minmax(10rem, .32fr) minmax(0, 1fr) auto; grid-template-areas: "label hint control"; } [data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(10rem, .32fr) minmax(0, 1fr) minmax(14rem, .5fr); grid-template-areas: "label hint control"; } [data-rcl-settings-row][data-has-leading] { grid-template-columns: auto minmax(10rem, .32fr) minmax(0, 1fr) auto; grid-template-areas: "leading label hint control"; } [data-rcl-settings-row][data-control="wide"][data-has-leading] { grid-template-columns: auto minmax(10rem, .32fr) minmax(0, 1fr) minmax(14rem, .5fr); grid-template-areas: "leading label hint control"; } }
`;

function SettingsListRoot({
  children,
  variant = "auto",
  density = "comfortable",
  className,
  style,
  ...sectionProps
}: SettingsListProps) {
  return (
    <section
      {...sectionProps}
      className={className}
      style={style}
      data-rcl-settings-list
      data-variant={variant}
      data-density={density}
    >
      <StyleSheet libraryId="react-component-library:SettingsList" version="1.1.0" css={styles} />
      {children}
    </section>
  );
}

function Intro({ eyebrow, title, description, children }: SettingsListIntroProps) {
  return (
    <header data-rcl-settings-intro>
      {eyebrow && <div data-rcl-settings-intro-eyebrow>{eyebrow}</div>}
      <h2 data-rcl-settings-intro-title>{title}</h2>
      {description && <p data-rcl-settings-intro-description>{description}</p>}
      {children}
    </header>
  );
}

function Group({ label, children, className }: SettingsListGroupProps) {
  const generatedId = useId().replace(/:/g, "");
  const labelId = label ? `settings-group-${generatedId}` : undefined;
  return (
    <section className={className} data-rcl-settings-group aria-labelledby={labelId}>
      {label && (
        <div id={labelId} data-rcl-settings-group-label>
          {label}
        </div>
      )}
      <div data-rcl-settings-group-surface>{children}</div>
    </section>
  );
}

function Row({
  label,
  hint,
  control = "compact",
  children,
  className,
  leading,
  onSelect,
}: SettingsListRowProps) {
  const generatedId = useId().replace(/:/g, "");
  const labelId = `settings-row-${generatedId}`;
  const hintId = `${labelId}-hint`;
  const controlId = `${labelId}-control`;
  const labelledChildren = Children.map(children, (child) => {
    if (!isValidElement(child)) return child;
    const element = child as ReactElement<{ "aria-label"?: string; "aria-labelledby"?: string }>;
    if (element.props["aria-label"] || element.props["aria-labelledby"]) return child;
    return cloneElement(element, { "aria-labelledby": labelId });
  });
  const hasChildren = Children.count(children) > 0;
  const describedBy =
    [hint ? hintId : "", hasChildren ? controlId : ""].filter(Boolean).join(" ") || undefined;
  return (
    <div
      className={className}
      data-rcl-settings-row
      data-control={control}
      data-has-leading={leading ? "" : undefined}
      data-selectable={onSelect ? "" : undefined}
    >
      {leading && <div data-rcl-settings-row-leading>{leading}</div>}
      <div data-rcl-settings-row-text>
        {onSelect ? (
          <button
            type="button"
            id={labelId}
            data-rcl-settings-row-label
            data-rcl-settings-row-select
            aria-describedby={describedBy}
            onClick={onSelect}
          >
            {label}
          </button>
        ) : (
          <div id={labelId} data-rcl-settings-row-label>
            {label}
          </div>
        )}
        {hint && (
          <div id={hintId} data-rcl-settings-row-hint>
            {hint}
          </div>
        )}
      </div>
      {(hasChildren || onSelect) && (
        <div data-rcl-settings-row-control>
          {hasChildren && (
            <div id={controlId} style={{ display: "contents" }}>
              <SettingsRowLabelContext.Provider value={labelId}>
                {labelledChildren}
              </SettingsRowLabelContext.Provider>
            </div>
          )}
          {onSelect && (
            <span data-rcl-settings-row-chevron aria-hidden="true">
              <svg
                width="16"
                height="16"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="m9 6 6 6-6 6" />
              </svg>
            </span>
          )}
        </div>
      )}
    </div>
  );
}

export const SettingsList = Object.assign(SettingsListRoot, { Intro, Group, Row });
export default SettingsList;
