import { useMemo, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CandidateStatus,
  type LogoCandidate,
} from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import {
  candidateThumbnailUrl,
  pickCandidate,
  rejectCandidate,
  restoreCandidate,
} from "../../api/candidates";
import { originLabel, statusLabelKey } from "./labels";

interface LogoCandidateGalleryProps {
  brandId: string;
  candidates: LogoCandidate[];
  isLoading: boolean;
  queryError: unknown;
  selectedForCompare: string[];
  onToggleCompare: (candidateId: string) => void;
  onSelectForRefine: (candidate: LogoCandidate) => void;
}

/**
 * LogoCandidateGallery is the operator's candidate surface: candidates grouped
 * by concept with lineage, a status chip, and the pick/reject/restore actions.
 * Compare selection (2-4) is owned by the page so the compare strip and the
 * gallery stay in sync. Thumbnails are served by the raw API endpoint, so an
 * SVG candidate renders crisply without a raster round-trip. The candidates
 * query is owned by the page; mutations invalidate it to refetch.
 */
export function LogoCandidateGallery({
  brandId,
  candidates,
  isLoading,
  queryError,
  selectedForCompare,
  onToggleCompare,
  onSelectForRefine,
}: LogoCandidateGalleryProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  // Optimistic status overrides so a pick/reject/restore updates the chip
  // immediately; the invalidated query then reconciles with the server.
  const [statusOverrides, setStatusOverrides] = useState<Record<string, CandidateStatus>>({});

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ["candidates", brandId] });
  };

  const pick = useMutation({
    mutationFn: (candidateId: string) => pickCandidate(candidateId),
    onSuccess: (_resp, candidateId) => {
      setStatusOverrides((current) => {
        const next: Record<string, CandidateStatus> = { ...current };
        for (const candidate of candidates) {
          if (next[candidate.id] === CandidateStatus.PICKED || candidate.status === CandidateStatus.PICKED) {
            next[candidate.id] = CandidateStatus.SUPERSEDED;
          }
        }
        next[candidateId] = CandidateStatus.PICKED;
        return next;
      });
      invalidate();
    },
  });
  const reject = useMutation({
    mutationFn: (candidateId: string) => rejectCandidate(candidateId, ""),
    onSuccess: (_resp, candidateId) => {
      setStatusOverrides((current) => ({ ...current, [candidateId]: CandidateStatus.REJECTED }));
      invalidate();
    },
  });
  const restore = useMutation({
    mutationFn: (candidateId: string) => restoreCandidate(candidateId),
    onSuccess: (_resp, candidateId) => {
      setStatusOverrides((current) => ({ ...current, [candidateId]: CandidateStatus.PROPOSED }));
      invalidate();
    },
  });

  const statusOf = (candidate: LogoCandidate): CandidateStatus =>
    statusOverrides[candidate.id] ?? candidate.status;

  const byId = useMemo(() => {
    const map = new Map<string, LogoCandidate>();
    for (const candidate of candidates) {
      map.set(candidate.id, candidate);
    }
    return map;
  }, [candidates]);

  const groups = useMemo(() => {
    const map = new Map<string, LogoCandidate[]>();
    for (const candidate of candidates) {
      const key = candidate.concept || t(strings.logo.conceptLabel);
      const list = map.get(key) ?? [];
      list.push(candidate);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [candidates, t]);

  if (brandId === "") {
    return null;
  }

  const busy = pick.isPending || reject.isPending || restore.isPending;
  const mutationError = pick.error ?? reject.error ?? restore.error;

  return (
    <section
      data-testid={selectors.logo.gallery}
      aria-label={t(strings.logo.title)}
      className="flex flex-col gap-3"
    >
      {isLoading && (
        <p data-testid={selectors.logo.loading} className="text-sm text-slate-400">
          {t(strings.logo.loading)}
        </p>
      )}
      {queryError != null && (
        <p data-testid={selectors.logo.error} className="text-sm text-red-400">
          {errorMessage(queryError, t)}
        </p>
      )}
      {mutationError && (
        <p className="text-sm text-red-400">{errorMessage(mutationError, t)}</p>
      )}
      {!isLoading && candidates.length === 0 && (
        <p data-testid={selectors.logo.empty} className="text-sm text-slate-400">
          {t(strings.logo.empty)}
        </p>
      )}

      {groups.map(([concept, candidates]) => (
        <div key={concept} className="flex flex-col gap-2">
          <h3 className="text-sm font-medium text-slate-400">{concept}</h3>
          <ul className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {candidates.map((candidate) => {
              const parent = candidate.parentId ? byId.get(candidate.parentId) : undefined;
              const compareSelected = selectedForCompare.includes(candidate.id);
              const status = statusOf(candidate);
              return (
                <li
                  key={candidate.id}
                  data-testid={selectors.logo.card}
                  className="flex flex-col gap-2 rounded-xl border border-white/10 bg-black/20 p-3"
                >
                  <div className="flex items-start gap-3">
                    <img
                      data-testid={selectors.logo.cardThumb}
                      src={candidateThumbnailUrl(candidate.id)}
                      alt={candidate.concept}
                      className="h-16 w-16 rounded-lg bg-slate-900 object-contain"
                    />
                    <div className="flex min-w-0 flex-1 flex-col gap-1">
                      <span
                        data-testid={selectors.logo.cardStatus}
                        className="w-fit rounded-full border border-white/10 px-2 py-0.5 text-xs text-slate-300"
                      >
                        {t(statusLabelKey(status))}
                      </span>
                      <span className="text-xs text-slate-500">{originLabel(candidate.origin)}</span>
                      <p
                        data-testid={selectors.logo.cardConcept}
                        className="truncate text-sm text-slate-300"
                        title={candidate.prompt}
                      >
                        {candidate.concept}
                      </p>
                      {parent && (
                        <button
                          type="button"
                          className="w-fit text-xs text-cyan-300 underline"
                          onClick={() => onSelectForRefine(parent)}
                        >
                          {t(strings.logo.lineage)}: {parent.concept}
                        </button>
                      )}
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {status !== CandidateStatus.PICKED && (
                      <button
                        type="button"
                        data-testid={selectors.logo.cardPick}
                        disabled={busy}
                        className="rounded-control border border-cyan-400/40 px-2 py-1 text-xs text-cyan-200 disabled:opacity-50"
                        onClick={() => pick.mutate(candidate.id)}
                      >
                        {pick.isPending ? t(strings.logo.picking) : t(strings.logo.pick)}
                      </button>
                    )}
                    {status !== CandidateStatus.REJECTED ? (
                      <button
                        type="button"
                        data-testid={selectors.logo.cardReject}
                        disabled={busy}
                        className="rounded-control border border-white/10 px-2 py-1 text-xs text-slate-300 disabled:opacity-50"
                        onClick={() => reject.mutate(candidate.id)}
                      >
                        {t(strings.logo.reject)}
                      </button>
                    ) : (
                      <button
                        type="button"
                        data-testid={selectors.logo.cardRestore}
                        disabled={busy}
                        className="rounded-control border border-white/10 px-2 py-1 text-xs text-slate-300 disabled:opacity-50"
                        onClick={() => restore.mutate(candidate.id)}
                      >
                        {t(strings.logo.restore)}
                      </button>
                    )}
                    <button
                      type="button"
                      data-testid={selectors.logo.cardCompare}
                      aria-pressed={compareSelected}
                      className={`rounded-control border px-2 py-1 text-xs ${compareSelected ? "border-cyan-400 text-cyan-200" : "border-white/10 text-slate-300"}`}
                      onClick={() => onToggleCompare(candidate.id)}
                    >
                      {t(strings.logo.compare)}
                    </button>
                    <button
                      type="button"
                      className="rounded-control border border-white/10 px-2 py-1 text-xs text-slate-300"
                      onClick={() => onSelectForRefine(candidate)}
                    >
                      {t(strings.logo.refineTitle)}
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </section>
  );
}
