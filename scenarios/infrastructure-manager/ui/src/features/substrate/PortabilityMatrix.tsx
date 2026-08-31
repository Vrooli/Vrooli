import { strings } from "../../consts/strings.generated";
import { useTranslation } from "../../i18n";
import {
  type PlatformSkipBudget,
  type PortabilityCell,
  type PortabilityRow,
  type ResourceClaim,
} from "./model";

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

const CONTROLS_INCOMPLETE_PRESENTATION = {
  label: "Controls incomplete",
  short: "CTRL",
  mark: "⚠",
  className: "lamp lamp--excursion",
} as const;

function presentationFor(cell: PortabilityCell) {
	if (cell.status === "controls_incomplete") return CONTROLS_INCOMPLETE_PRESENTATION;
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
                  cell.absent?.length ? `absent: ${cell.absent.join(", ")}` : null,
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
					{cell.absent?.length ? (
						<span className="block max-w-32 text-[10px] leading-tight text-left" data-testid="absent-declarers">
							absent: {cell.absent.join(", ")}
						</span>
					) : null}
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

export interface ResourceClaimsProps {
  resources: readonly ResourceClaim[];
  skipBudget: PlatformSkipBudget | null;
}

/**
 * The resource side of the portability readout. It stays separate from the
 * capability matrix because resources have a different denominator: one row
 * per resource, with an explicit acquisition kind and six OS/architecture
 * cells. The component never substitutes a zero when the skip-budget source
 * did not answer.
 */
export function ResourceClaims({ resources, skipBudget }: ResourceClaimsProps) {
  const { t } = useTranslation();
  const budgetEntries = skipBudget
    ? Object.entries(skipBudget.budgets).sort(([left], [right]) => left.localeCompare(right))
    : [];
  const budgetSummary = budgetEntries.map(([os, budget]) => `${os}=${budget}`).join(", ");

  return (
    <div className="flex flex-col gap-space-md">
      <div className="scroller">
        <table className="annunciator">
          <caption>{t(strings.pages.substrate.resourceClaimsCaption, { count: resources.length })}</caption>
          <thead>
            <tr>
              <th scope="col">{t(strings.pages.substrate.resourceNameColumn)}</th>
              <th scope="col">{t(strings.pages.substrate.resourceDriverColumn)}</th>
              <th scope="col">{t(strings.pages.substrate.resourceAcquisitionColumn)}</th>
              <th scope="col">{t(strings.pages.substrate.resourcePlatformColumn)}</th>
            </tr>
          </thead>
          <tbody>
            {resources.map((resource) => (
              <tr key={resource.name}>
                <th scope="row" className="annunciator__device">{resource.name}</th>
                <td className="font-mono text-body-sm">{resource.driver}</td>
                <td className="font-mono text-body-sm">{resource.acquisitionKind}</td>
                <td>
                  <div className="flex flex-col gap-space-2xs">
                    {resource.platforms.map((platform) => (
                      <div key={platform.hostOs} className="flex flex-wrap gap-x-space-sm gap-y-space-2xs">
                        <span className="font-mono uppercase">{platform.hostOs}</span>
                        <span>{platform.support}</span>
                        {platform.mismatch ? <span className="text-app-danger">mismatch: {platform.reason}</span> : null}
                        {platform.architectures.map((architecture) => (
                          <span key={architecture.architecture} className="font-mono text-body-sm" title={architecture.reason}>
                            {architecture.architecture}={architecture.support}
                          </span>
                        ))}
                      </div>
                    ))}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <section aria-labelledby="substrate-skip-budget" className="flex flex-col gap-space-2xs">
        <h3 id="substrate-skip-budget" className="font-mono text-body-sm uppercase tracking-[0.12em]">
          {t(strings.pages.substrate.skipBudgetHeading)}
        </h3>
        <p className="max-w-[66ch] text-body-sm text-app-muted-foreground">
          {t(strings.pages.substrate.skipBudgetNote)}
        </p>
        {!skipBudget || !skipBudget.available ? (
          <p className="text-body-sm text-app-danger">
            {t(strings.pages.substrate.skipBudgetUnavailable, {
              reason: skipBudget?.reason ?? "the portability grid did not supply a measurement",
            })}
          </p>
        ) : (
          <div className="flex flex-wrap gap-x-space-md gap-y-space-2xs text-body-sm">
            <span>{t(strings.pages.substrate.skipBudgetSummary, {
              measured: skipBudget.measured,
              budgets: budgetSummary,
              ratchet: skipBudget.ratchetDirection ?? "unspecified",
            })}</span>
            <span className={skipBudget.lastRunWithinBudget ? "text-app-foreground" : "text-app-danger"}>
              {skipBudget.lastRunWithinBudget
                ? t(strings.pages.substrate.skipBudgetWithin)
                : t(strings.pages.substrate.skipBudgetOver)}
            </span>
            {skipBudget.reason ? <span>{skipBudget.reason}</span> : null}
          </div>
        )}
      </section>
    </div>
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
