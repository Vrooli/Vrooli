import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { AssignmentsCard } from "../features/assignments/AssignmentsCard";
import { useTranslation } from "../i18n";

export function AssignmentsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.assignments}
      aria-labelledby="assignments-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="assignments-heading" className="text-2xl font-semibold">
        {t(strings.pages.assignments.title)}
      </h2>
      <AssignmentsCard />
    </section>
  );
}
