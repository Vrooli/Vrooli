import { useEffect, useState } from "react";
import { fetchClosure, fetchRecommendation } from "../../api/selection";
import type { GetClosureResponse, GetRecommendationResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";
import { PlanSummary } from "@vrooli/react-component-library/PlanSummary/0";
import { i18n } from "../../i18n";

export function StepPlan({ target = "local", onAccept, onAdjust }: { target?: string; onAccept: () => Promise<void>; onAdjust: () => void }) {
  const [recommendation, setRecommendation] = useState<GetRecommendationResponse | null>(null);
  const [closure, setClosure] = useState<GetClosureResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [accepting, setAccepting] = useState(false);

  useEffect(() => {
    let active = true;
    Promise.all([fetchRecommendation(target), fetchClosure(target)]).then(([nextRecommendation, nextClosure]) => {
      if (!active) return;
      if (!Array.isArray(nextRecommendation?.scenarios) || !Array.isArray(nextRecommendation?.resources)) {
        setError(i18n.t("onboarding.plan.unavailable"));
        return;
      }
      setRecommendation(nextRecommendation);
      setClosure(Array.isArray(nextClosure?.resources) ? nextClosure : null);
    }).catch(() => { if (active) setError(i18n.t("onboarding.plan.loadError")); });
    return () => { active = false; };
  }, [target]);

  if (error) return <p role="alert" data-testid="plan-error">{error}</p>;
  if (!recommendation) return <p role="status" data-testid="plan-loading">{i18n.t("onboarding.plan.loading")}</p>;

  const items = [
    ...recommendation.scenarios.map((name) => ({ label: name, implied: false })),
    ...(closure?.resources ?? []).map((resource) => ({ label: resource.name, implied: true })),
  ];
  return <PlanSummary
    kicker={i18n.t("onboarding.plan.kicker")}
    title={recommendation.profile}
    note={recommendation.explanation}
    facts={[
      { value: String(recommendation.scenarios.length), label: i18n.t("onboarding.plan.scenarios") },
      { value: String(closure?.resources.length ?? recommendation.resources.length), label: i18n.t("onboarding.plan.resources") },
    ]}
    items={items}
    onAccept={async () => { setAccepting(true); setError(null); try { await onAccept(); } catch { setError(i18n.t("onboarding.plan.acceptError")); } finally { setAccepting(false); } }}
    acceptLabel={accepting ? i18n.t("onboarding.plan.applying") : i18n.t("onboarding.plan.use")}
    onAdjust={onAdjust}
    adjustLabel={i18n.t("onboarding.plan.adjust")}
  />;
}
