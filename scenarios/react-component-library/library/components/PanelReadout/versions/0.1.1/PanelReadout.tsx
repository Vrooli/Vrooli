/**
 * @libraryId react-component-library:PanelReadout
 * @displayName PanelReadout
 * @description Ranked panel reading for ambient displays
 * @version 0.1.1
 * @tags ["data-display","ambient"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1";

export interface PanelRow {
  key: string;
  label: string;
  value: number;
  share: number;
  detail?: string;
  ink?: "solid" | "reduced" | "hollow" | "dotted";
}

export interface PanelReading {
  id?: string;
  label: string;
  rows?: PanelRow[];
}

type Props = { reading: PanelReading; maxRows?: number; className?: string };

const styles = `
[data-rcl-panel-readout] { display: grid; gap: var(--space-2xs, .5rem); min-inline-size: 0; color: var(--color-foreground, #e8ecf3); }
[data-rcl-panel-heading] { color: var(--color-muted-foreground, #94a3b8); font: var(--text-label, 500 .8rem/1.25 var(--font-sans, sans-serif)); letter-spacing: .14em; text-transform: uppercase; }
[data-rcl-panel-row] { position: relative; display: grid; grid-template-columns: minmax(0, 1fr) auto; gap: var(--space-xs, .75rem); align-items: center; min-block-size: 2rem; overflow: hidden; }
[data-rcl-panel-label], [data-rcl-panel-value] { position: relative; z-index: 1; font: var(--text-body, 400 1rem/1.4 var(--font-sans, sans-serif)); }
[data-rcl-panel-value] { font-variant-numeric: tabular-nums; }
[data-rcl-panel-bar] { position: absolute; inset: 0 auto 0 0; max-inline-size: 100%; background: color-mix(in srgb, var(--color-primary, #7ce8ff) 12%, transparent); }
[data-rcl-panel-row][data-ink="hollow"] [data-rcl-panel-label], [data-rcl-panel-row][data-ink="hollow"] [data-rcl-panel-value] { opacity: .72; }
[data-rcl-panel-row][data-ink="dotted"] [data-rcl-panel-bar] { background: repeating-linear-gradient(90deg, color-mix(in srgb, var(--color-gap, #b49cff) 28%, transparent) 0 3px, transparent 3px 7px); }
[data-rcl-panel-placeholder] { color: var(--color-muted-foreground, #94a3b8); font: var(--text-body, 400 1rem/1.4 var(--font-sans, sans-serif)); }
`;

/** Compact ranked figure for panel readings. Material, not hue, carries row provenance. */
export const PanelReadout = withClassName(function PanelReadout({
  reading,
  maxRows = 6,
  className,
}: Props) {
  const rows = (reading.rows ?? []).slice(0, maxRows);
  return (
    <>
      <StyleSheet name="panel-readout-0-1-0" css={styles} />
      <section
        className={className}
        aria-label={reading.label}
        data-kind="panel"
        data-rcl-panel-readout
      >
        <div data-rcl-panel-heading>{reading.label}</div>
        {rows.length === 0 ? (
          <div data-rcl-panel-placeholder aria-label="No observations">
            ···
          </div>
        ) : (
          rows.map((row: PanelRow) => (
            <div data-rcl-panel-row data-ink={row.ink ?? "solid"} key={row.key}>
              <div data-rcl-panel-label>{row.label}</div>
              <div data-rcl-panel-value>{row.value.toLocaleString()}</div>
              <div
                data-rcl-panel-bar
                style={{ width: `${Math.max(0, Math.min(100, row.share * 100))}%` }}
              />
            </div>
          ))
        )}
      </section>
    </>
  );
});
