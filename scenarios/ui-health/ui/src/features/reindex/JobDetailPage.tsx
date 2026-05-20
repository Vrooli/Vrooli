import { useParams } from "react-router-dom";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function JobDetailPage() {
  const { t } = useTranslation();
  const { jobId } = useParams<{ jobId: string }>();
  return (
    <section
      data-testid={selectors.pages.reindexJob}
      aria-labelledby="reindex-job-heading"
      className="flex flex-col gap-3"
    >
      <h2 id="reindex-job-heading" className="text-2xl font-semibold tracking-tight">
        {t(strings.pages.reindex.job.title)}
      </h2>
      <p className="text-sm text-app-muted-foreground break-all">{jobId}</p>
    </section>
  );
}
