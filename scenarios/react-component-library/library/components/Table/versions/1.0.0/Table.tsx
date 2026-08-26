/** @vrooliComponentSource data-display.table */
import { withClassName } from "@vrooli/react-component-library/ClassMerge/1.0.1";

import type { CSSProperties, ReactNode } from "react";

export interface TableProps {
  rows?: Array<Record<string, string>>;
  children?: ReactNode;
  caption?: string;
  className?: string;
  style?: CSSProperties;
}

const styles = `
[data-rcl-table] { min-inline-size: 0; overflow: hidden; border: var(--border-hairline) solid var(--color-border); border-radius: var(--radius-panel); background: var(--color-surface); color: var(--color-foreground); box-shadow: var(--elev-raised); }
[data-rcl-table-scroll] { max-inline-size: 100%; overflow-x: auto; }
[data-rcl-table] table { inline-size: 100%; table-layout: auto; border-collapse: separate; border-spacing: 0; text-align: start; }
[data-rcl-table] caption { padding: var(--space-sm) var(--space-md); color: var(--color-muted-foreground); font: var(--text-caption); text-align: start; }
[data-rcl-table] th { padding: var(--space-sm) var(--space-md); border-block-end: var(--border-hairline) solid var(--color-border); background: color-mix(in srgb, var(--color-surface-muted) 72%, var(--color-primary)); color: var(--color-muted-foreground); font: var(--text-overline); letter-spacing: .06em; overflow-wrap: anywhere; text-align: start; text-transform: uppercase; }
[data-rcl-table] td { padding: var(--space-md); border-block-end: var(--border-hairline) solid var(--color-border); color: var(--color-foreground); font: var(--text-body); overflow-wrap: anywhere; vertical-align: middle; }
[data-rcl-table] tbody tr:last-child td { border-block-end: 0; }
[data-rcl-table] tbody tr { transition: background-color 160ms ease; }
[data-rcl-table] tbody tr:hover { background: color-mix(in srgb, var(--color-primary) 5%, var(--color-surface)); }
@media (prefers-reduced-motion: reduce) { [data-rcl-table] tbody tr { transition-duration: .01ms; } }
@media (forced-colors: active) { [data-rcl-table] { border-color: CanvasText; background: Canvas; color: CanvasText; box-shadow: none; } [data-rcl-table] th, [data-rcl-table] td { border-color: CanvasText; background: Canvas; color: CanvasText; } }
`;

export const Table = withClassName(function Table({
  rows = [],
  children,
  caption,
  className,
  style,
}: TableProps) {
  const columns = Object.keys(rows[0] || {});
  return (
    <div data-testid="data-display.table" data-rcl-table className={className} style={style}>
      <style
        data-rcl-table-styles
        dangerouslySetInnerHTML={{ __html: styles }}
      />
      <div data-rcl-table-scroll>
        {children ?? (
          <table>
            {caption ? <caption>{caption}</caption> : null}
            <thead>
              <tr>
                {columns.map((column) => (
                  <th key={column} scope="col">
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {rows.map((row, index) => (
                <tr key={index}>
                  {columns.map((column) => (
                    <td key={column}>{row[column]}</td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
});
