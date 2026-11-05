import { useQuery } from "@tanstack/react-query";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../../components/ui/card";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { fetchConformancePlan, fetchReadiness, fetchTargets } from "../../api/delivery";

export function DeliveryReadinessCard() {
  const { t } = useTranslation();
  const readiness = useQuery({ queryKey: ["ios-readiness"], queryFn: fetchReadiness });
  const targets = useQuery({ queryKey: ["ios-targets"], queryFn: fetchTargets });
  const plan = useQuery({ queryKey: ["ios-conformance-plan"], queryFn: fetchConformancePlan });

  const firstBlocked = readiness.data?.rungs.find((rung) => rung.state !== "ready");
  const status = firstBlocked ? firstBlocked.state : "ready";

  return (
    <Card data-testid={selectors.delivery.targetMatrix} className="h-full lg:col-span-2">
      <CardHeader>
        <div className="flex items-start justify-between gap-3">
          <div>
            <CardTitle>{t(strings.delivery.title)}</CardTitle>
            <CardDescription>{t(strings.delivery.description)}</CardDescription>
          </div>
          <StatusBadge tone={status === "ready" ? "success" : "warning"}>{status}</StatusBadge>
        </div>
      </CardHeader>
      <CardContent className="grid gap-4 text-sm md:grid-cols-3">
        <section aria-labelledby="ios-readiness-heading">
          <h3 id="ios-readiness-heading" className="font-medium">{t(strings.delivery.readiness)}</h3>
          {readiness.isLoading && <p className="text-app-muted-foreground">{t(strings.delivery.loading)}</p>}
          {readiness.error && <p className="text-app-danger">{t(strings.delivery.unavailable)}</p>}
          {firstBlocked && (
            <dl className="mt-2 space-y-1">
              <div><dt className="text-app-muted-foreground">{t(strings.delivery.blockedAt)}</dt><dd>{firstBlocked.title}</dd></div>
              <div><dt className="text-app-muted-foreground">{t(strings.delivery.nextAction)}</dt><dd>{firstBlocked.next_action}</dd></div>
              {firstBlocked.missing_capability && <div><dt className="text-app-muted-foreground">{t(strings.delivery.missingCapability)}</dt><dd>{firstBlocked.missing_capability}</dd></div>}
            </dl>
          )}
        </section>
        <section aria-labelledby="ios-targets-heading">
          <h3 id="ios-targets-heading" className="font-medium">{t(strings.delivery.targets)}</h3>
          {targets.data?.targets.map((target) => (
            <div key={target.id} className="mt-2 border-b border-app-border pb-2 last:border-0">
              <div className="flex justify-between gap-2"><span>{target.label}</span><span>{target.available ? "ready" : "unavailable"}</span></div>
              {!target.available && <p className="text-app-muted-foreground">{target.next_action ?? target.reason}</p>}
            </div>
          ))}
          {targets.error && <p className="text-app-danger">{t(strings.delivery.unavailable)}</p>}
        </section>
        <section aria-labelledby="ios-plan-heading">
          <h3 id="ios-plan-heading" className="font-medium">{t(strings.delivery.conformance)}</h3>
          <p className="mt-2 text-2xl font-semibold">{plan.data?.chapters.length ?? "--"}</p>
          <p className="text-app-muted-foreground">{t(strings.delivery.chapters)}</p>
          {plan.error && <p className="text-app-danger">{t(strings.delivery.unavailable)}</p>}
        </section>
      </CardContent>
    </Card>
  );
}
