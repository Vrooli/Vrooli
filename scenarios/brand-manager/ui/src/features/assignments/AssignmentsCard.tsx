import { useQuery } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { assignmentsClient } from "../../api/assignments";
import { errorMessage } from "../../lib/errorMessage";

const ASSIGNMENTS_QUERY_KEY = ["assignments"] as const;

/**
 * AssignmentsCard lists brand→scenario assignments (newest-applied first). It is
 * a read surface: assignments are created from the CLI/wizard where a brand id
 * and scenario name are supplied. Mirrors the canonical BrandsCard structure but
 * wired to the AssignmentsService Connect client.
 */
export function AssignmentsCard() {
  const { t } = useTranslation();

  const assignmentsQuery = useQuery({
    queryKey: ASSIGNMENTS_QUERY_KEY,
    queryFn: () => assignmentsClient.listAssignments({}),
  });

  return (
    <section
      data-testid={selectors.assignments.card}
      aria-label={t(strings.assignments.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.assignments.title)}</h2>
      {assignmentsQuery.isLoading && (
        <p data-testid={selectors.assignments.loading} className="mt-2 text-slate-200">
          {t(strings.assignments.loading)}
        </p>
      )}
      {assignmentsQuery.error && (
        <p data-testid={selectors.assignments.error} className="mt-2 text-red-400">
          {errorMessage(assignmentsQuery.error, t)}
        </p>
      )}
      {assignmentsQuery.data && assignmentsQuery.data.assignments.length === 0 && (
        <p data-testid={selectors.assignments.empty} className="mt-2 text-slate-200">
          {t(strings.assignments.empty)}
        </p>
      )}
      {assignmentsQuery.data && assignmentsQuery.data.assignments.length > 0 && (
        <ul data-testid={selectors.assignments.list} className="mt-2 space-y-1 text-sm text-slate-200">
          {assignmentsQuery.data.assignments.map((assignment) => (
            <li key={assignment.id} className="rounded-lg border border-white/10 p-3">
              <div className="flex items-center justify-between">
                <span data-testid={selectors.assignments.scenario} className="font-medium">
                  {assignment.scenarioName}
                </span>
                <span data-testid={selectors.assignments.version} className="text-xs text-slate-400">
                  {`v${assignment.brandVersion}`}
                </span>
              </div>
              <div className="mt-1 text-xs text-slate-400">
                <span>{t(strings.assignments.brandLabel)}</span>{" "}
                <span data-testid={selectors.assignments.brand}>{assignment.brandId}</span>
              </div>
              {assignment.elements.length > 0 && (
                <div className="mt-1 text-xs text-slate-400">
                  <span>{t(strings.assignments.elementsLabel)}</span> {assignment.elements.join(", ")}
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
