import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import { type PortabilityCell, type PortabilityRow } from "./model";

/**
 * The portability matrix: capabilities down, operating systems across.
 *
 * The distinction this grid exists to preserve is between "it compiles for that
 * platform and passes fixtures" and "nobody built it". Before the vocabulary was
 * closed, `internal/deployability` collapsed both into `unwired` with the reason
 * "a mechanism is named but no implementation is wired for this host OS" — which
 * was false for every `build-verified` declaration and understated macOS and
 * Windows coverage by an unknown margin.
 *
 * So the cell renders the QUALIFICATION rung, not just the status. Amber is
 * reserved for `qualified` — proven on real hardware of that platform. A
 * build-verified capability reads as cyan and says so, because presenting it as
 * equal to real-hardware qualification would be exactly the flattery this
 * instrument removes.
 */

/** Qualification rung -> presentation. Ordered most to least proven. */
const QUALIFICATION_PRESENTATION: Record<
  string,
  { label: string; short: string; mark: string; className: string }
> = {
  qualified: {
    label: "Qualified",
    short: "QUAL",
    mark: "●",
    className: "lamp lamp--covered",
  },
  "build-verified": {
    label: "Build verified",
    short: "BLD",
    mark: "◐",
    className: "lamp lamp--partial",
  },
  unqualified: {
    label: "Unqualified",
    short: "UNQ",
    mark: "◔",
    className: "lamp lamp--unmeasurable",
  },
  degraded: {
    label: "Degraded",
    short: "DEG",
    mark: "◑",
    className: "lamp lamp--partial",
  },
  ineligible: {
    label: "Ineligible",
    short: "N/A",
    mark: "⊘",
    className: "lamp lamp--unavailable",
  },
  undeclared: {
    label: "Nothing declared",
    short: "OFF",
    mark: "○",
    className: "lamp lamp--blind",
  },
};

const UNKNOWN_PRESENTATION = {
  label: "Unrecognised",
  short: "?",
  mark: "!",
  className: "lamp lamp--excursion",
} as const;

function presentationFor(cell: PortabilityCell) {
  return QUALIFICATION_PRESENTATION[cell.qualification] ?? UNKNOWN_PRESENTATION;
}

export interface PortabilityMatrixProps {
  rows: readonly PortabilityRow[];
  /** Column order, e.g. ["linux", "macos", "windows"]. */
  operatingSystems: readonly string[];
  /** Rendered as the table caption; must state the denominator. */
  caption: string;
  /** Column header for the capability column. */
  rowHeader: string;
}

export function PortabilityMatrix({
  rows,
  operatingSystems,
  caption,
  rowHeader,
}: PortabilityMatrixProps) {
  const { t } = useTranslation();
  return (
    <div className="scroller">
      <table className="annunciator">
        <caption>{caption}</caption>
        <thead>
          <tr>
            <th scope="col">{rowHeader}</th>
            {operatingSystems.map((os) => (
              <th key={os} scope="col" className="annunciator__lamp-cell">
                {os}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.capability}>
              <th scope="row" className="annunciator__device">
                <span className="annunciator__device-name">{row.capability}</span>
              </th>
              {operatingSystems.map((os) => {
                const cell = row.platforms[os];
                if (!cell) {
                  // No declaration at all for this OS. That is a real, distinct
                  // fact from every declared state and must not borrow one.
                  return (
                    <td key={os} className="annunciator__lamp-cell">
                      <span
                        className="lamp lamp--blind"
                        role="img"
                        aria-label={t(strings.pages.substrate.nothingDeclaredLabel, {
                          capability: row.capability,
                          os,
                        })}
                      >
                        <span aria-hidden="true">○</span>
                        <span aria-hidden="true">
                          {t(strings.pages.substrate.nothingDeclaredShort)}
                        </span>
                      </span>
                    </td>
                  );
                }
                const presentation = presentationFor(cell);
                const name = [
                  `${row.capability} on ${os}`,
                  presentation.label,
                  cell.implementer ? `implemented by ${cell.implementer}` : null,
                  cell.mechanism ? `via ${cell.mechanism}` : null,
                  cell.reason,
                ]
                  .filter(Boolean)
                  .join(", ");
                return (
                  <td key={os} className="annunciator__lamp-cell">
                    <span
                      className={presentation.className}
                      role="img"
                      aria-label={name}
                      title={cell.reason}
                      data-qualification={cell.qualification}
                    >
                      <span aria-hidden="true">{presentation.mark}</span>
                      <span aria-hidden="true">{presentation.short}</span>
                    </span>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** The key for the qualification vocabulary, rendered beside the matrix. */
export function PortabilityLegend({ rungs }: { rungs: readonly string[] }) {
  return (
    <ul className="flex flex-wrap gap-x-space-md gap-y-space-2xs list-none p-0 m-0">
      {rungs.map((rung) => {
        const presentation = QUALIFICATION_PRESENTATION[rung] ?? UNKNOWN_PRESENTATION;
        return (
          <li key={rung} className="legend-key">
            <span className={presentation.className} aria-hidden="true">
              <span>{presentation.mark}</span>
              <span>{presentation.short}</span>
            </span>
            <span>{presentation.label}</span>
          </li>
        );
      })}
    </ul>
  );
}

export const QUALIFICATION_ORDER: readonly string[] = [
  "qualified",
  "build-verified",
  "unqualified",
  "degraded",
  "ineligible",
  "undeclared",
];
