import { useQuery } from "@tanstack/react-query";
import { AlertCircle, FileText } from "lucide-react";
import { Link, useParams } from "react-router-dom";

import { fetchDebtEntry } from "../api/templateDomain";
import { DefinitionList } from "../components/detail/DefinitionList";
import { DetailError, DetailLoading } from "../components/detail/DetailStates";
import { DetailPageHeader } from "../components/detail/DetailPageHeader";
import { DetailSection } from "../components/detail/DetailSection";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { debtStatusTone } from "../lib/templateLabels";
import { formatTimestamp } from "../lib/time";

const TITLE_ID = "debt-detail-heading";

export function DebtDetailPage() {
  const { t } = useTranslation();
  const { debtKey = "" } = useParams();
  const key = decodeURIComponent(debtKey);
  const query = useQuery({
    queryKey: ["debt-entry", key] as const,
    queryFn: () => fetchDebtEntry(key),
  });

  if (query.isLoading) {
    return <DetailLoading testId={selectors.debtDetail.loading} />;
  }
  if (query.isError || !query.data) {
    return <DetailError testId={selectors.debtDetail.error} />;
  }

  const entry = query.data;

  return (
    <section
      data-testid={selectors.pages.debtDetail}
      aria-labelledby={TITLE_ID}
      className="flex flex-col gap-4"
    >
      <DetailPageHeader
        testId={selectors.debtDetail.header}
        backTo="/debt"
        backLabel={t(strings.detail.back.debtList)}
        title={entry.title || entry.key}
        titleId={TITLE_ID}
        status={{ label: entry.status || "—", tone: debtStatusTone(entry.status) }}
        subtitle={`${entry.key} · ${entry.severity}`}
      />

      <DetailSection
        testId={selectors.debtDetail.overview}
        title={t(strings.debtDetail.overview)}
        icon={<AlertCircle aria-hidden className="h-4 w-4" />}
        hideDivider
      >
        <DefinitionList
          items={[
            { label: t(strings.debtDetail.fields.key), value: entry.key, full: true },
            {
              label: t(strings.debtDetail.fields.template),
              value: entry.templateId ? (
                <Link to={`/templates/${encodeURIComponent(entry.templateId)}`} className="underline-offset-2 hover:underline">
                  {entry.templateId}
                </Link>
              ) : (
                ""
              ),
            },
            { label: t(strings.debtDetail.fields.severity), value: entry.severity },
            { label: t(strings.debtDetail.fields.status), value: entry.status },
            { label: t(strings.debtDetail.fields.source), value: entry.source },
            { label: t(strings.debtDetail.fields.firstSeen), value: formatTimestamp(entry.firstSeenAt) },
            { label: t(strings.debtDetail.fields.lastSeen), value: formatTimestamp(entry.lastSeenAt) },
          ]}
        />
      </DetailSection>

      <DetailSection
        testId={selectors.debtDetail.message}
        title={t(strings.debtDetail.message)}
        icon={<FileText aria-hidden className="h-4 w-4" />}
      >
        {entry.detail ? (
          <p className="whitespace-pre-wrap break-words text-sm text-app-foreground">{entry.detail}</p>
        ) : (
          <p className="text-sm text-app-muted-foreground">{t(strings.debtDetail.messageEmpty)}</p>
        )}
      </DetailSection>
    </section>
  );
}
