import type { LogoCandidate } from "@vrooli/proto-types/brand-manager/v1/candidates/candidates_pb";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { candidateThumbnailUrl } from "../../api/candidates";

const COMPARE_SIZES = [64, 32, 16] as const;

/** LogoCompare shows 2-4 candidates side by side with a size strip. */
export function LogoCompare({ candidates }: { candidates: LogoCandidate[] }) {
  const { t } = useTranslation();
  if (candidates.length < 2) {
    return null;
  }
  return (
    <section
      data-testid={selectors.logo.compare}
      aria-label={t(strings.logo.compareTitle)}
      className="flex flex-col gap-2 rounded-xl border border-white/10 bg-black/20 p-3"
    >
      <h3 className="text-sm font-medium text-slate-400">{t(strings.logo.compareTitle)}</h3>
      <div className="flex flex-wrap gap-4">
        {candidates.map((candidate) => (
          <div
            key={candidate.id}
            data-testid={selectors.logo.compareItem}
            className="flex flex-col items-start gap-2"
          >
            <img
              src={candidateThumbnailUrl(candidate.id)}
              alt={candidate.concept}
              className="h-64 w-64 rounded-xl bg-slate-900 object-contain"
            />
            <p className="max-w-64 truncate text-xs text-slate-400">{candidate.concept}</p>
            <div className="flex items-end gap-2">
              {COMPARE_SIZES.map((size) => (
                <img
                  key={size}
                  src={candidateThumbnailUrl(candidate.id)}
                  alt={`${candidate.concept} at ${size}px`}
                  style={{ width: size, height: size }}
                  className="rounded bg-slate-900 object-contain"
                />
              ))}
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
