/**
 * @libraryId react-component-library:SettingsList
 * @displayName Settings List
 * @version 0.1.6
 * @tags ["settings","layout","responsive","accessibility","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource settings.settings-list */
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
  children: ReactNode;
  className?: string;
}

const SettingsRowLabelContext = createContext<string | undefined>(undefined);
export function useSettingsRowLabelId() {
  return useContext(SettingsRowLabelContext);
}

const styles = `
[data-rcl-settings-list] { --rcl-settings-section-gap: var(--space-xl, 40px); --rcl-settings-group-gap: var(--space-md, 24px); --rcl-settings-row-block: var(--space-sm, 16px); --rcl-settings-row-inline: var(--space-md, 24px); display: grid; gap: var(--rcl-settings-section-gap); min-inline-size: 0; container: rcl-settings / inline-size; }
[data-rcl-settings-list][data-density="compact"] { --rcl-settings-section-gap: var(--space-lg, 32px); --rcl-settings-group-gap: var(--space-sm, 16px); --rcl-settings-row-block: var(--space-xs, 12px); --rcl-settings-row-inline: var(--space-sm, 16px); }
[data-rcl-settings-intro] { display: grid; gap: var(--space-3xs, 4px); }
[data-rcl-settings-intro-eyebrow], [data-rcl-settings-group-label] { color: var(--color-muted-foreground, #64748b); font: var(--text-overline, 700 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); letter-spacing: var(--tracking-caps, .08em); text-transform: uppercase; }
[data-rcl-settings-intro-title] { margin: 0; color: var(--color-foreground, #0f172a); font: var(--text-title, 700 var(--text-title-size) / var(--text-title-line) var(--font-sans)); }
[data-rcl-settings-intro-description] { margin: 0; max-inline-size: 68ch; color: var(--color-muted-foreground, #64748b); font: var(--text-body-sm, 400 var(--text-body-sm-size) / var(--text-body-sm-line) var(--font-sans)); }
[data-rcl-settings-group] { display: grid; gap: var(--rcl-settings-group-gap); min-inline-size: 0; }
[data-rcl-settings-group-surface] { display: grid; min-inline-size: 0; border: 0 solid transparent; border-radius: 0; background: transparent; overflow: hidden; }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]) { margin-inline: var(--rcl-settings-row-inline); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]):first-child { margin-block-start: var(--rcl-settings-row-block); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]):last-child { margin-block-end: var(--rcl-settings-row-block); }
[data-rcl-settings-group-surface] > :not([data-rcl-settings-row]) + :not([data-rcl-settings-row]) { margin-block-start: var(--rcl-settings-group-gap); }
[data-rcl-settings-row] { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: "label control" "hint control"; align-items: center; column-gap: var(--space-md, 24px); min-block-size: 48px; padding: var(--rcl-settings-row-block) var(--rcl-settings-row-inline); color: var(--color-foreground, #0f172a); }
[data-rcl-settings-row] + [data-rcl-settings-row] { border-block-start: 1px solid var(--color-border, #cbd5e1); }
[data-rcl-settings-row-label] { grid-area: label; min-inline-size: 0; font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
[data-rcl-settings-row-hint] { grid-area: hint; min-inline-size: 0; color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); overflow-wrap: anywhere; }
[data-rcl-settings-row-control] { grid-area: control; min-inline-size: 0; max-inline-size: 100%; justify-self: end; }
[data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(0, 1fr); grid-template-areas: "label" "hint" "control"; row-gap: var(--space-xs, 12px); }
[data-rcl-settings-row][data-control="wide"] [data-rcl-settings-row-control] { inline-size: 100%; justify-self: stretch; }
[data-rcl-settings-list][data-variant="card"] [data-rcl-settings-group-surface] { border-color: var(--color-border, #cbd5e1); border-width: 1px; border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); }
@container rcl-settings (max-width: 25.99rem) { [data-rcl-settings-row][data-control="compact"] { grid-template-columns: minmax(0, 1fr); grid-template-areas: "label" "hint" "control"; row-gap: var(--space-xs, 12px); } [data-rcl-settings-row][data-control="compact"] [data-rcl-settings-row-control] { inline-size: 100%; justify-self: stretch; } }
@container rcl-settings (min-width: 26rem) { [data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(0, 1fr) minmax(12rem, .65fr); grid-template-areas: "label control" "hint control"; row-gap: 0; } [data-rcl-settings-row][data-control="wide"] [data-rcl-settings-row-control] { inline-size: auto; justify-self: end; } }
@container rcl-settings (min-width: 34rem) { [data-rcl-settings-list][data-variant="auto"] [data-rcl-settings-group-surface] { border-color: var(--color-border, #cbd5e1); border-width: 1px; border-radius: var(--radius-panel, 0.5rem); background: var(--color-surface-raised, #ffffff); } [data-rcl-settings-list][data-variant="auto"] [data-rcl-settings-group-label] { padding-inline: var(--rcl-settings-row-inline); } }
@container rcl-settings (min-width: 52rem) { [data-rcl-settings-row] { grid-template-columns: minmax(10rem, .32fr) minmax(0, 1fr) auto; grid-template-areas: "label hint control"; } [data-rcl-settings-row][data-control="wide"] { grid-template-columns: minmax(10rem, .32fr) minmax(0, 1fr) minmax(14rem, .5fr); grid-template-areas: "label hint control"; } }
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
      <StyleSheet name="settings-list-0-1-5" css={styles} />
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

function Row({ label, hint, control = "compact", children, className }: SettingsListRowProps) {
  const generatedId = useId().replace(/:/g, "");
  const labelId = `settings-row-${generatedId}`;
  const labelledChildren = Children.map(children, (child) => {
    if (!isValidElement(child)) return child;
    const element = child as ReactElement<{ "aria-label"?: string; "aria-labelledby"?: string }>;
    if (element.props["aria-label"] || element.props["aria-labelledby"]) return child;
    return cloneElement(element, { "aria-labelledby": labelId });
  });
  return (
    <div className={className} data-rcl-settings-row data-control={control}>
      <div id={labelId} data-rcl-settings-row-label>
        {label}
      </div>
      {hint && <div data-rcl-settings-row-hint>{hint}</div>}
      <div data-rcl-settings-row-control>
        <SettingsRowLabelContext.Provider value={labelId}>
          {labelledChildren}
        </SettingsRowLabelContext.Provider>
      </div>
    </div>
  );
}

export const SettingsList = Object.assign(SettingsListRoot, { Intro, Group, Row });
export default SettingsList;
