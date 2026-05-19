import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { ArrowLeft } from "lucide-react";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { formatDate } from "../../i18n/format";
import { goldenClient, type Golden } from "../../api/golden";
import { reportClient, type TupleVerdict } from "../../api/report";
import { TupleKind } from "../../api/validationRecord";
import { errorMessage } from "../../lib/errorMessage";
import {
  summarizeVerdicts,
  summaryToVariant,
  tupleKindToSegment,
  verdictToKind,
} from "../../lib/verdict";
import { ROUTES } from "../../routes.generated";
import { Button } from "../../shared/ui/primitives/Button";
import { Badge, type BadgeProps } from "../../shared/ui/primitives/Badge";
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "../../shared/ui/primitives/Dialog";
import { PanelHeader } from "../../shared/ui/composites/PanelHeader";
import { LoadingSkeleton } from "../../shared/ui/composites/LoadingSkeleton";
import { EmptyState } from "../../shared/ui/composites/EmptyState";
import { MetadataList } from "../../shared/ui/composites/MetadataList";
import { VerdictGrid, type VerdictGridRow } from "../../shared/ui/composites/VerdictGrid";
import { usePreferencesStore } from "../../shared/stores/preferencesStore";

const GOLDENS_QUERY_KEY = ["goldens"] as const;

const KIND_TO_BADGE_VARIANT: Record<string, BadgeProps["variant"]> = {
  pass: "verdict-pass",
  stale: "verdict-stale",
  unexpected: "verdict-unexpected",
  failure: "verdict-failure",
  neutral: "neutral",
};

function tupleVerdictsToGridRows(
  tuples: readonly TupleVerdict[],
): import("../../shared/ui/composites/VerdictGrid").VerdictGridRow[] {
  return tuples.map((tv) => ({
    id: tv.subjectId,
    label: tv.subjectId,
    subLabel: tv.stale ? "stale" : undefined,
    kind: verdictToKind(tv.latestVerdict, tv.stale),
  }));
}

/**
 * Golden detail surface.
 *
 * Looks up the golden by slug from the index query (no per-slug RPC ships
 * today); renders a header, metadata list, and two verdict grids. The
 * grids degrade gracefully — until verdicts.proto ships, both grids show
 * an empty state with neutral placeholders.
 */
export function GoldenDetail() {
  const { t } = useTranslation();
  const params = useParams<{ slug: string }>();
  const slug = params.slug ?? "";
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const setLastVisited = usePreferencesStore((s) => s.setLastVisitedGoldenSlug);
  const [confirmRegenerate, setConfirmRegenerate] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  useEffect(() => {
    if (slug) setLastVisited(slug);
  }, [slug, setLastVisited]);

  const listQuery = useQuery({
    queryKey: GOLDENS_QUERY_KEY,
    queryFn: () => goldenClient.listGoldens({}),
  });

  const summaryQuery = useQuery({
    queryKey: ["goldenSummary", slug] as const,
    queryFn: () => reportClient.getGoldenSummary({ goldenSlug: slug }),
    enabled: slug.length > 0,
  });

  const regenerateMutation = useMutation({
    mutationFn: (s: string) => goldenClient.regenerateGolden({ slug: s }),
    onSuccess: (resp) => {
      const respSlug = resp.golden?.slug ?? slug;
      setStatusMessage(t(strings.goldens.regenerateSuccess, { slug: respSlug }));
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (s: string) => goldenClient.deleteGolden({ slug: s }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: GOLDENS_QUERY_KEY });
      void navigate(ROUTES.goldensIndex);
    },
  });

  const golden: Golden | undefined = listQuery.data?.goldens.find((g) => g.slug === slug);

  if (listQuery.isLoading) {
    return <LoadingSkeleton data-testid={selectors.goldens.loading} variant="card" count={4} />;
  }

  if (listQuery.error) {
    return (
      <p data-testid={selectors.goldens.error} className="text-sm text-status-failure">
        {errorMessage(listQuery.error, t)}
      </p>
    );
  }

  if (!golden) {
    return (
      <EmptyState
        testId={selectors.goldens.empty}
        title={t(strings.goldens.empty)}
        description={t(strings.goldens.emptyDescription)}
        action={
          <Button size="sm" variant="outline" onClick={() => void navigate(ROUTES.goldensIndex)}>
            {t(strings.goldens.backToIndex)}
          </Button>
        }
      />
    );
  }

  const summary = summaryQuery.data?.summary;
  const skillTuples = summary?.skillVerdicts ?? [];
  const toolTuples = summary?.toolVerdicts ?? [];
  const skillsRows: readonly VerdictGridRow[] = tupleVerdictsToGridRows(skillTuples);
  const toolsRows: readonly VerdictGridRow[] = tupleVerdictsToGridRows(toolTuples);

  const allTuples = [...skillTuples, ...toolTuples];
  const summaryCounts = summarizeVerdicts(allTuples);
  const summaryKind = summaryToVariant(summaryCounts);
  const summaryBadgeVariant: BadgeProps["variant"] =
    KIND_TO_BADGE_VARIANT[summaryKind] ?? "neutral";
  const summaryChipText =
    summaryCounts.total === 0
      ? t(strings.goldens.verdictSummaryPending)
      : t(strings.goldens.verdictSummaryCounts, {
          pass: summaryCounts.pass,
          stale: summaryCounts.stale,
          unexpected: summaryCounts.unexpected + summaryCounts.failure,
        });

  const handleTupleClick = (tupleKind: TupleKind, subjectId: string) => {
    void navigate(ROUTES.tupleDetail(golden.slug, tupleKindToSegment(tupleKind), subjectId));
  };

  return (
    <section
      data-testid={selectors.goldens.detail}
      aria-labelledby={selectors.goldens.detailHeading}
      className="flex flex-col gap-5"
    >
      <PanelHeader
        title={<span data-testid={selectors.goldens.detailHeading} id={selectors.goldens.detailHeading}>{golden.slug}</span>}
        description={t(strings.goldens.rowLabel, {
          slug: golden.slug,
          template: golden.templateId,
          version: golden.templateVersionPinned,
        })}
        badge={
          <Badge
            data-testid={selectors.goldens.rowVerdictSummary}
            variant={summaryBadgeVariant}
          >
            {summaryQuery.isLoading ? "…" : summaryChipText}
          </Badge>
        }
        actions={
          <div className="flex gap-2">
            <Button
              data-testid={selectors.goldens.detailBack}
              size="sm"
              variant="ghost"
              onClick={() => void navigate(ROUTES.goldensIndex)}
            >
              <ArrowLeft className="h-4 w-4" />
              {t(strings.goldens.backToIndex)}
            </Button>
            <Button
              data-testid={selectors.goldens.detailRegenerate}
              size="sm"
              onClick={() => setConfirmRegenerate(true)}
              disabled={regenerateMutation.isPending}
            >
              {regenerateMutation.isPending
                ? t(strings.goldens.regenerating)
                : t(strings.goldens.regenerate)}
            </Button>
            <Button
              data-testid={selectors.goldens.detailDelete}
              size="sm"
              variant="danger"
              onClick={() => setConfirmDelete(true)}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t(strings.goldens.deleting) : t(strings.goldens.delete)}
            </Button>
          </div>
        }
      />

      <MetadataList
        items={[
          { label: t(strings.goldens.templateLabel), value: `${golden.templateId}@${golden.templateVersionPinned}` },
          { label: t(strings.goldens.pathLabel), value: golden.path, mono: true },
          ...(golden.lastRegeneratedAt
            ? [
                {
                  label: t(strings.goldens.lastRegeneratedLabel),
                  value: formatDate(timestampDate(golden.lastRegeneratedAt), {
                    dateStyle: "medium",
                    timeStyle: "short",
                  }),
                },
              ]
            : []),
        ]}
      />

      <div className="grid gap-4 lg:grid-cols-2">
        <VerdictGrid
          testId={selectors.goldens.skillsGrid}
          caption={t(strings.goldens.skillsGridCaption)}
          rows={skillsRows}
          onRowClick={(id) => handleTupleClick(TupleKind.SKILL, id)}
          emptyState={
            <EmptyState
              title={t(strings.goldens.skillsGridCaption)}
              description={t(strings.goldens.noSkillVerdicts)}
            />
          }
        />
        <VerdictGrid
          testId={selectors.goldens.toolsGrid}
          caption={t(strings.goldens.toolsGridCaption)}
          rows={toolsRows}
          onRowClick={(id) => handleTupleClick(TupleKind.TOOL, id)}
          emptyState={
            <EmptyState
              title={t(strings.goldens.toolsGridCaption)}
              description={t(strings.goldens.noToolVerdicts)}
            />
          }
        />
      </div>

      {statusMessage ? (
        <p data-testid={selectors.goldens.detailStatus} className="text-xs text-status-pass">
          {statusMessage}
        </p>
      ) : null}

      <Dialog
        open={confirmRegenerate}
        onOpenChange={setConfirmRegenerate}
        ariaLabel={t(strings.goldens.confirmRegenerate, { slug: golden.slug })}
      >
        <DialogTitle>{t(strings.goldens.regenerate)}</DialogTitle>
        <DialogDescription>{t(strings.goldens.confirmRegenerate, { slug: golden.slug })}</DialogDescription>
        <DialogFooter>
          <Button size="sm" variant="ghost" onClick={() => setConfirmRegenerate(false)}>
            {t(strings.goldens.close)}
          </Button>
          <Button
            size="sm"
            onClick={() => {
              setConfirmRegenerate(false);
              regenerateMutation.mutate(golden.slug);
            }}
          >
            {t(strings.goldens.regenerate)}
          </Button>
        </DialogFooter>
      </Dialog>

      <Dialog
        open={confirmDelete}
        onOpenChange={setConfirmDelete}
        ariaLabel={t(strings.goldens.confirmDelete, { slug: golden.slug })}
      >
        <DialogTitle>{t(strings.goldens.delete)}</DialogTitle>
        <DialogDescription>{t(strings.goldens.confirmDelete, { slug: golden.slug })}</DialogDescription>
        <DialogFooter>
          <Button size="sm" variant="ghost" onClick={() => setConfirmDelete(false)}>
            {t(strings.goldens.close)}
          </Button>
          <Button
            size="sm"
            variant="danger"
            onClick={() => {
              setConfirmDelete(false);
              deleteMutation.mutate(golden.slug);
            }}
          >
            {t(strings.goldens.delete)}
          </Button>
        </DialogFooter>
      </Dialog>
    </section>
  );
}
