import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchClosure, fetchRecommendation } from "../../api/selection";
import type { GetClosureResponse, GetRecommendationResponse } from "@vrooli/proto-types/vrooli-onboarding/v1/selection/selection_pb";
import { PlanSummary } from "@vrooli/react-component-library/PlanSummary/0";
import { i18n } from "../../i18n";

type PlanResult = { recommendation: GetRecommendationResponse; closure: GetClosureResponse | null };
type AcceptanceState = { accepting: boolean; error: string | null };

export function StepPlan({ target = "local", onAccept, onAdjust }: { target?: string; onAccept: () => Promise<void>; onAdjust: () => void }) {
  const [acceptanceByTarget, setAcceptanceByTarget] = useState<Record<string, AcceptanceState>>({});
  const acceptance = acceptanceByTarget[target] ?? { accepting: false, error: null };
  const plan = useQuery<PlanResult>({
    queryKey: ["onboarding-step-plan", target],
    queryFn: async () => {
      const [recommendation, closure] = await Promise.all([fetchRecommendation(target), fetchClosure(target)]);
      if (!Array.isArray(recommendation.scenarios) || !Array.isArray(recommendation.resources)) {
        throw new Error("recommendation-invalid");
      }
      return {
        recommendation,
        closure: Array.isArray(closure.resources) ? closure : null,
      };
    },
    retry: false,
  });

  const accept = async () => {
    setAcceptanceByTarget((current) => ({ ...current, [target]: { accepting: true, error: null } }));
    try {
      await onAccept();
      setAcceptanceByTarget((current) => ({ ...current, [target]: { accepting: false, error: null } }));
    } catch {
      setAcceptanceByTarget((current) => ({ ...current, [target]: { accepting: false, error: i18n.t("onboarding.plan.acceptError") } }));
    }
  };

  if (plan.error) {
    const message = plan.error.message === "recommendation-invalid"
      ? i18n.t("onboarding.plan.unavailable")
      : i18n.t("onboarding.plan.loadError");
    return <p role="alert" data-testid="plan-error">{message}</p>;
  }
  if (plan.isLoading || !plan.data) return <p role="status" data-testid="plan-loading">{i18n.t("onboarding.plan.loading")}</p>;

  const { recommendation, closure } = plan.data;
  const items = [
    ...recommendation.scenarios.map((name) => ({ label: name, implied: false })),
    ...(closure?.resources ?? []).map((resource) => ({ label: resource.name, implied: true })),
  ];
  return <>
    {acceptance.error && <p role="alert" data-testid="plan-accept-error">{acceptance.error}</p>}
    <PlanSummary
    kicker={i18n.t("onboarding.plan.kicker")}
    title={recommendation.profile}
    note={recommendation.explanation}
    facts={[
      { value: String(recommendation.scenarios.length), label: i18n.t("onboarding.plan.scenarios") },
      { value: String(closure?.resources.length ?? recommendation.resources.length), label: i18n.t("onboarding.plan.resources") },
    ]}
    items={items}
    onAccept={() => { void accept(); }}
    acceptLabel={acceptance.accepting ? i18n.t("onboarding.plan.applying") : i18n.t("onboarding.plan.use")}
    onAdjust={onAdjust}
    adjustLabel={i18n.t("onboarding.plan.adjust")}
    />
  </>;
}
