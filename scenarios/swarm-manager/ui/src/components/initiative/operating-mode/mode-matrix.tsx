/**
 * ModeMatrix
 *
 * Side-by-side comparison grid driven entirely by the catalog response.
 * Adding a new mode auto-populates a new column. Rows: scope, run strategy,
 * unit of work, best for, not for, tradeoffs, capabilities.
 */

import { selectors } from "../../../consts/selectors";
import type { OperatingModeCatalogEntry } from "../../../types/operating-mode";
import { CapabilityList } from "./capability-list";
import { humanizeRunStrategy, humanizeScopeKind } from "./utils";

export interface ModeMatrixProps {
  catalog: OperatingModeCatalogEntry[];
}

export function ModeMatrix({ catalog }: ModeMatrixProps) {
  if (catalog.length === 0) {
    return (
      <p className="text-sm italic text-slate-500">
        No modes are registered.
      </p>
    );
  }
  return (
    <div
      className="overflow-x-auto"
      data-testid={selectors.initiativeDetails.howToChooseMatrix}
    >
      <table className="min-w-full border-collapse text-left text-sm text-slate-200">
        <thead>
          <tr className="border-b border-slate-800 text-xs uppercase tracking-wide text-slate-500">
            <th scope="col" className="px-3 py-2 font-medium">
              Aspect
            </th>
            {catalog.map((entry) => (
              <th
                scope="col"
                key={entry.mode}
                className="px-3 py-2 font-medium text-slate-200"
              >
                {entry.label}
                <p className="mt-0.5 text-[10px] font-normal text-slate-500">
                  {entry.mode}
                  {entry.default ? " · default" : ""}
                </p>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          <Row label="Scope" cells={catalog.map((e) => humanizeScopeKind(e.scopeKind))} />
          <Row
            label="Run strategy"
            cells={catalog.map((e) => humanizeRunStrategy(e.runStrategy))}
          />
          <ListRow label="Best for" cellLists={catalog.map((e) => e.bestFor)} />
          <ListRow label="Not for" cellLists={catalog.map((e) => e.notFor)} />
          <ListRow label="Tradeoffs" cellLists={catalog.map((e) => e.tradeoffs)} />
          <tr className="border-b border-slate-800/60 align-top">
            <th scope="row" className="px-3 py-2 font-medium text-slate-300">
              Capabilities
            </th>
            {catalog.map((entry) => (
              <td key={entry.mode} className="px-3 py-2">
                <CapabilityList capabilities={entry.capabilities} variant="compact" />
              </td>
            ))}
          </tr>
        </tbody>
      </table>
    </div>
  );
}

function Row({ label, cells }: { label: string; cells: string[] }) {
  return (
    <tr className="border-b border-slate-800/60 align-top">
      <th scope="row" className="px-3 py-2 font-medium text-slate-300">
        {label}
      </th>
      {cells.map((cell, idx) => (
        <td key={idx} className="px-3 py-2">
          {cell}
        </td>
      ))}
    </tr>
  );
}

function ListRow({ label, cellLists }: { label: string; cellLists: string[][] }) {
  return (
    <tr className="border-b border-slate-800/60 align-top">
      <th scope="row" className="px-3 py-2 font-medium text-slate-300">
        {label}
      </th>
      {cellLists.map((items, idx) => (
        <td key={idx} className="px-3 py-2">
          <ul className="space-y-1">
            {items.map((item, i) => (
              <li key={i} className="text-xs text-slate-300">
                {item}
              </li>
            ))}
          </ul>
        </td>
      ))}
    </tr>
  );
}
