import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { ArrowLeft } from "lucide-react";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { reportClient } from "../../api/report";
import { manifestClient } from "../../api/manifest";
import {
  TupleKind,
  Verdict,
  type ValidationRecord,
} from "../../api/validationRecord";
import { errorMessage } from "../../lib/errorMessage";
import { segmentToTupleKind, verdictToKind } from "../../lib/verdict";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { Badge, type BadgeProps } from "../../shared/ui/primitives/Badge";
import { Card, CardHeader, CardTitle } from "../../shared/ui/primitives/Card";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "../../shared/ui/primitives/Tabs";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { MetadataList } from "../../shared/ui/composites/MetadataList";

const KIND_TO_BADGE_VARIANT: Record<string, BadgeProps["variant"]> = {
  pass: "verdict-pass",
  stale: "verdict-stale",
  unexpected: "verdict-unexpected",
  failure: "verdict-failure",
  neutral: "neutral",
};

/**
 * Tuple detail surface — drill-down for a single (skill|tool, golden)
 * tuple. Wires three tabs: Diff (placeholder pending the
 * workspace-sandbox bridge), Manifest (active manifest pinning), and
 * History (chronological validation records).
 */
export function TupleDetail() {
  const { t } = useTranslation();
  const params = useParams<{ slug: string; tupleKind: string; subjectId: string }>();
  const slug = params.slug ?? "";
  const subjectId = params.subjectId ?? "";
  const tupleKind = segmentToTupleKind(params.tupleKind ?? "skill");
  const navigate = useNavigate();
  const [tab, setTab] = useState<"diff" | "manifest" | "history">("diff");

  const historyQuery = useQuery({
    queryKey: ["tupleHistory", slug, tupleKind, subjectId] as const,
    queryFn: () =>
      reportClient.getTupleHistory({
        goldenSlug: slug,
        subjectId,
        tupleKind,
        pageSize: 25,
      }),
    enabled: slug.length > 0 && subjectId.length > 0,
  });

  const manifestQuery = useQuery({
    queryKey: ["manifest", subjectId, slug] as const,
    queryFn: () =>
      manifestClient.getManifest({ skillId: subjectId, goldenSlug: slug }),
    enabled: tupleKind === TupleKind.SKILL && slug.length > 0 && subjectId.length > 0,
    // 404 / not-found is expected for tool tuples; don't retry storms.
    retry: false,
  });

  const history = historyQuery.data?.history;
  const records: readonly ValidationRecord[] = history?.records ?? [];
  const latest = records[0];

  if (historyQuery.isLoading) {
    return (
      <LoadingSkeleton
        data-testid={selectors.goldens.loading}
        variant="card"
        count={3}
      />
    );
  }

  if (historyQuery.error) {
    return (
      <p data-testid={selectors.goldens.error} className="text-sm text-status-failure">
        {errorMessage(historyQuery.error, t)}
      </p>
    );
  }

  const latestKind = latest
    ? verdictToKind(latest.verdict, false)
    : "neutral";
  const latestBadgeVariant: BadgeProps["variant"] =
    KIND_TO_BADGE_VARIANT[latestKind] ?? "neutral";

  return (
    <section
      data-testid={selectors.goldens.tupleDetail}
      aria-labelledby={selectors.goldens.tupleDetailHeading}
      className="flex flex-col gap-5"
    >
      <PanelHeader
        title={
          <span
            data-testid={selectors.goldens.tupleDetailHeading}
            id={selectors.goldens.tupleDetailHeading}
          >
            {t(strings.goldens.tupleDetailTitle, { subject: subjectId, slug })}
          </span>
        }
        actions={
          <Button
            data-testid={selectors.goldens.tupleDetailBack}
            size="sm"
            variant="ghost"
            onClick={() => void navigate(ROUTES.goldenDetail(slug))}
          >
            <ArrowLeft className="h-4 w-4" />
            {t(strings.goldens.tupleBack, { slug })}
          </Button>
        }
      />

      {latest ? (
        <Card surface="raised" data-testid={selectors.goldens.tupleDetailRunSummary}>
          <CardHeader>
            <CardTitle>
              <span className="mr-2">{t(strings.goldens.tupleRunSummary)}</span>
              <Badge variant={latestBadgeVariant}>
                {Verdict[latest.verdict] || "—"}
              </Badge>
            </CardTitle>
          </CardHeader>
          <MetadataList
            items={[
              ...(latest.startedAt
                ? [
                    {
                      label: t(strings.goldens.lastRegeneratedLabel),
                      value: formatDate(timestampDate(latest.startedAt), {
                        dateStyle: "medium",
                        timeStyle: "short",
                      }),
                    },
                  ]
                : []),
              {
                label: t(strings.goldens.tupleDurationLabel),
                value: `${latest.durationMs.toString()} ms`,
              },
              {
                label: t(strings.goldens.tupleTokensLabel),
                value: latest.tokensUsed.toString(),
              },
              {
                label: t(strings.goldens.tupleCostLabel),
                value: latest.costUsdMicro.toString(),
              },
              ...(latest.agentManagerRunId
                ? [
                    {
                      label: t(strings.goldens.tupleAgentRunLabel),
                      value: latest.agentManagerRunId,
                      mono: true,
                    },
                  ]
                : []),
            ]}
          />
        </Card>
      ) : (
        <EmptyState
          title={t(strings.goldens.tupleRunSummary)}
          description={t(strings.goldens.tupleNoRuns)}
        />
      )}

      <Tabs
        value={tab}
        onValueChange={(v) => setTab(v as "diff" | "manifest" | "history")}
      >
        <TabsList>
          <TabsTrigger value="diff" data-testid={selectors.goldens.tupleDetailDiff}>
            {t(strings.goldens.tupleTabDiff)}
          </TabsTrigger>
          <TabsTrigger value="manifest" data-testid={selectors.goldens.tupleDetailManifest}>
            {t(strings.goldens.tupleTabManifest)}
          </TabsTrigger>
          <TabsTrigger value="history" data-testid={selectors.goldens.tupleDetailHistory}>
            {t(strings.goldens.tupleTabHistory)}
          </TabsTrigger>
        </TabsList>
        <TabsContent value="diff">
          {latest && latest.diffHash ? (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.goldens.tupleDiffPlaceholder, { hash: latest.diffHash })}
            </p>
          ) : (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.goldens.tupleNoDiff)}
            </p>
          )}
        </TabsContent>
        <TabsContent value="manifest">
          {tupleKind === TupleKind.SKILL && manifestQuery.data?.manifest ? (
            <MetadataList
              items={[
                {
                  label: t(strings.manifests.allowedPathsLabel),
                  value:
                    manifestQuery.data.manifest.allowedPaths.join(", ") || "—",
                  mono: true,
                },
                {
                  label: t(strings.manifests.wildcardAllowedLabel),
                  value: manifestQuery.data.manifest.wildcardAllowed
                    ? "yes"
                    : "no",
                },
                {
                  label: t(strings.manifests.pinningLabel, {
                    template:
                      manifestQuery.data.manifest.templateVersionPinned || "—",
                    skill:
                      manifestQuery.data.manifest.skillVersionPinned || "—",
                  }),
                  value: "",
                },
              ]}
            />
          ) : (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.goldens.tupleManifestEmpty)}
            </p>
          )}
        </TabsContent>
        <TabsContent value="history">
          {records.length === 0 ? (
            <p className="text-xs text-app-muted-foreground">
              {t(strings.goldens.tupleHistoryEmpty)}
            </p>
          ) : (
            <ul className="flex flex-col gap-2">
              {records.map((r) => {
                const kind = verdictToKind(r.verdict, false);
                const variant: BadgeProps["variant"] =
                  KIND_TO_BADGE_VARIANT[kind] ?? "neutral";
                return (
                  <li
                    key={r.id}
                    className="flex items-center justify-between rounded-control border border-app-border bg-app-surface p-2"
                  >
                    <span className="font-mono text-xs text-app-muted-foreground">
                      {r.startedAt
                        ? formatDate(timestampDate(r.startedAt), {
                            dateStyle: "short",
                            timeStyle: "short",
                          })
                        : r.id}
                    </span>
                    <Badge variant={variant}>{Verdict[r.verdict] || "—"}</Badge>
                  </li>
                );
              })}
            </ul>
          )}
        </TabsContent>
      </Tabs>
    </section>
  );
}
