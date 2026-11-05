import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import type { AdoptionSuggestion } from "../../api/adoptions";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export function AdoptionSuggestions({
  suggestions,
  loading,
  onAdopt,
  compact = false,
}: {
  suggestions: AdoptionSuggestion[];
  loading: boolean;
  onAdopt: (suggestion: AdoptionSuggestion) => void;
  compact?: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className={compact ? "mt-space-xs space-y-space-2xs" : "mt-space-xs space-y-space-2xs"}>
      {suggestions.map((item) => (
        <article
          key={`${item.scenario}-${item.componentId}`}
          className="flex items-start justify-between gap-space-xs rounded-control border border-app-border p-space-xs text-xs"
        >
          <div>
            <div className="font-medium text-app-foreground">
              {item.displayName}{" "}
              <span className="text-app-muted-foreground">→ {item.scenario}</span>
            </div>
            {compact ? (
              <StatusBadge tone="neutral">
                {item.classification === 1
                  ? t("adoptions.suggestions.heuristic", {
                      defaultValue: "Heuristic candidate — review before adopting",
                    })
                  : t("adoptions.suggestions.unavailable", {
                      defaultValue: "Unavailable candidate",
                    })}
              </StatusBadge>
            ) : (
              <p className="mt-space-3xs text-app-muted-foreground">
                {item.classification === 1
                  ? t("adoptions.suggestions.heuristic", {
                      defaultValue: "Heuristic candidate — review before adopting",
                    })
                  : t("adoptions.suggestions.unavailable", {
                      defaultValue: "Unavailable candidate",
                    })}
              </p>
            )}
            <ul className="mt-space-3xs list-disc space-y-space-4xs ps-space-sm text-app-muted-foreground">
              {item.reasons.map((reason) => (
                <li key={reason}>{reason}</li>
              ))}
            </ul>
          </div>
          <Button size="sm" onClick={() => onAdopt(item)}>
            {t(strings.adoptions.suggestions.adoptAction)}
          </Button>
        </article>
      ))}
      {!loading && suggestions.length === 0 && (
        <p className="text-xs text-app-muted-foreground">
          {t(strings.adoptions.suggestions.empty)}
        </p>
      )}
    </div>
  );
}
