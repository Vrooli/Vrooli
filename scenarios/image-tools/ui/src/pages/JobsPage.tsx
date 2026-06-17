import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { JobsCard } from "../features/jobs/JobsCard";
import { useTranslation } from "../i18n";

export function JobsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.jobs}
      aria-labelledby="jobs-heading"
      className="flex flex-col gap-4"
    >
      <h2 id="jobs-heading" className="text-2xl font-semibold">
        {t(strings.pages.jobs.title)}
      </h2>
      <JobsCard />
    </section>
  );
}
