import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import type { ReactNode } from "react";

import { RUNG_ORDER, rungToken, type Rung, type SignalState } from "../../theme/instrument";
import { Lamp } from "./Lamp";

const cn = (...inputs: ClassValue[]) => twMerge(clsx(inputs));

export interface AnnunciatorRow {
  /** Stable key — use the device's durable identity, not its display name. */
  id: string;
  /** Display name, e.g. "Samsung SSD 990 PRO 2TB". */
  name: string;
  /** The real reference under the name: a PCI address, a device node, a cell id. */
  tag: string;
  states: Readonly<Record<Rung, SignalState>>;
  /** Per-rung reasons. Required wherever a state is UNMEASURABLE or UNAVAILABLE. */
  reasons?: Readonly<Partial<Record<Rung, string>>>;
  /** Per-rung blindness age, from the coverage model's `gap_open_days`. */
  blindDays?: Readonly<Partial<Record<Rung, number>>>;
  /** Invoked when the row is activated; omit for a non-interactive grid. */
  onSelect?: () => void;
}

export interface AnnunciatorGridProps {
  rows: readonly AnnunciatorRow[];
  /**
   * Rendered below the grid as a `<caption>`. Say what the grid is measuring
   * and against which denominator — a matrix without its denominator is the
   * kind of unqualified number this instrument refuses to print.
   */
  caption: ReactNode;
  /** Header for the first column, e.g. "Device". */
  rowHeader: string;
  className?: string;
}

/**
 * The device-by-rung annunciator matrix: rows are devices, columns are the
 * five ladder rungs, cells are lamps.
 *
 * This is a real `<table>` with two header axes because it is real tabular
 * data. A screen reader announcing "Samsung SSD 990 PRO, Anticipation,
 * unmeasurable, permission denied" is the whole point; a grid of divs could
 * not say that.
 */
export function AnnunciatorGrid({ rows, caption, rowHeader, className }: AnnunciatorGridProps) {
  return (
    <div className={cn("scroller", className)}>
      <table className="annunciator">
        <caption>{caption}</caption>
        <thead>
          <tr>
            <th scope="col">{rowHeader}</th>
            {RUNG_ORDER.map((rung) => {
              const token = rungToken(rung);
              return (
                <th key={rung} scope="col" className="annunciator__lamp-cell" title={token.question}>
                  <span className="font-mono text-signal-covered">{token.tag}</span> {token.label}
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr
              key={row.id}
              onClick={row.onSelect}
              className={row.onSelect ? "cursor-pointer hover:bg-app-surface-raised" : undefined}
            >
              <th scope="row" className="annunciator__device">
                {row.onSelect ? (
                  <button
                    type="button"
                    onClick={row.onSelect}
                    className="text-left bg-transparent border-0 p-0 text-app-foreground"
                  >
                    <span className="annunciator__device-name">{row.name}</span>
                    <span className="annunciator__device-tag">{row.tag}</span>
                  </button>
                ) : (
                  <>
                    <span className="annunciator__device-name">{row.name}</span>
                    <span className="annunciator__device-tag">{row.tag}</span>
                  </>
                )}
              </th>
              {RUNG_ORDER.map((rung) => (
                <td key={rung} className="annunciator__lamp-cell">
                  <Lamp
                    state={row.states[rung]}
                    subject={`${row.name}, ${rungToken(rung).label}`}
                    reason={row.reasons?.[rung]}
                    blindDays={row.blindDays?.[rung]}
                  />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
