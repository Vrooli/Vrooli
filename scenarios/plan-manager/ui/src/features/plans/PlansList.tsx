import { useState } from "react";
import { Link } from "react-router-dom";
import { ArrowRight, Plus } from "lucide-react";

import { AsyncBoundary } from "../../components/AsyncBoundary";
import { StatusBadge } from "../../components/StatusBadge";
import { SectionPanel } from "../../components/Surfaces";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { formatDate } from "../../i18n/format";
import { useTranslation } from "../../i18n";
import { countPhases, planStatusDescriptor } from "../../lib/planStatus";
import { useCreateFromTemplate, usePlansList, useTemplates } from "./usePlans";

/**
 * PlansList — the plans board. Lists every plan (including archived) with its
 * status, phase progress, and last-updated, plus a compact "create from
 * template" form. Rows link to the plan detail route.
 */
export function PlansList() {
  const { t } = useTranslation();
  const plans = usePlansList();
  const templates = useTemplates();
  const create = useCreateFromTemplate();

  const [templateId, setTemplateId] = useState("");
  const [title, setTitle] = useState("");

  const canCreate = templateId.length > 0 && title.trim().length > 0 && !create.isPending;

  const handleCreate = (e: React.FormEvent) => {
    e.preventDefault();
    if (!canCreate) return;
    create.mutate(
      { templateId, title: title.trim() },
      {
        onSuccess: () => {
          setTitle("");
          setTemplateId("");
        },
      },
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <SectionPanel
        title={t(strings.pages.plans.newFromTemplate)}
        headingId="plans-create-heading"
      >
        <form
          data-testid={selectors.plans.createForm}
          onSubmit={handleCreate}
          className="flex flex-col gap-3 sm:flex-row sm:items-end"
        >
          <label className="flex flex-1 flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.plans.templateLabel)}
            </span>
            <select
              data-testid={selectors.plans.templateSelect}
              value={templateId}
              onChange={(e) => setTemplateId(e.target.value)}
              className="h-10 rounded-control border border-app-border bg-app-surface px-3 text-app-foreground"
            >
              <option value="">{t(strings.common.selectPlaceholder)}</option>
              {(templates.data ?? []).map((tpl) => (
                <option key={tpl.id} value={tpl.id}>
                  {tpl.name}
                </option>
              ))}
            </select>
          </label>
          <label className="flex flex-1 flex-col gap-1 text-sm">
            <span className="text-xs font-medium text-app-muted-foreground">
              {t(strings.pages.plans.titleLabel)}
            </span>
            <Input
              data-testid={selectors.plans.titleInput}
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder={t(strings.pages.plans.titlePlaceholder)}
            />
          </label>
          <Button
            type="submit"
            data-testid={selectors.plans.createButton}
            disabled={!canCreate}
            className="shrink-0"
          >
            <Plus aria-hidden="true" className="me-2 h-4 w-4" />
            {t(strings.pages.plans.create)}
          </Button>
        </form>
      </SectionPanel>

      <AsyncBoundary
        isLoading={plans.isLoading}
        error={plans.error}
        isEmpty={(plans.data?.length ?? 0) === 0}
        testIdPrefix={selectors.plans.list}
        emptyLabel={t(strings.pages.plans.empty)}
      >
        <div
          data-testid={selectors.plans.list}
          className="overflow-x-auto rounded-panel border border-app-border bg-app-surface"
        >
          <table className="w-full min-w-[40rem] border-collapse text-sm">
            <caption className="sr-only">{t(strings.pages.plans.title)}</caption>
            <thead>
              <tr className="border-b border-app-border text-left text-xs uppercase tracking-wide text-app-muted-foreground">
                <th scope="col" className="px-4 py-2 font-medium">
                  {t(strings.pages.plans.titleLabel)}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t(strings.planStatus.active)}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t(strings.pages.plans.phasesLabel)}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  {t(strings.pages.plans.updatedLabel)}
                </th>
                <th scope="col" className="px-4 py-2 font-medium">
                  <span className="sr-only">{t(strings.pages.plans.viewDetail)}</span>
                </th>
              </tr>
            </thead>
            <tbody>
              {(plans.data ?? []).map((plan) => {
                const counts = countPhases(plan.phases.map((p) => p.status));
                return (
                  <tr
                    key={plan.id}
                    data-testid={selectors.plans.row({ id: plan.id })}
                    className="border-b border-app-border last:border-0 hover:bg-app-surface-muted"
                  >
                    <th scope="row" className="px-4 py-3 text-left font-medium text-app-foreground">
                      <Link
                        to={`/plans/${plan.id}`}
                        className="rounded-control underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                      >
                        {plan.title}
                      </Link>
                    </th>
                    <td className="px-4 py-3">
                      <StatusBadge descriptor={planStatusDescriptor(plan.status)} />
                    </td>
                    <td className="px-4 py-3 text-app-muted-foreground">
                      {t(strings.pages.plans.phaseProgress, {
                        done: counts.done,
                        total: counts.total,
                      })}
                    </td>
                    <td className="px-4 py-3 text-app-muted-foreground">
                      {plan.updatedAt
                        ? formatDate(new Date(plan.updatedAt), { dateStyle: "medium" })
                        : t(strings.common.unknownDate)}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <Link
                        to={`/plans/${plan.id}`}
                        aria-label={`${t(strings.pages.plans.viewDetail)}: ${plan.title}`}
                        className="inline-flex rounded-control p-1 text-app-muted-foreground hover:text-app-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
                      >
                        <ArrowRight aria-hidden="true" className="h-4 w-4" />
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </AsyncBoundary>
    </div>
  );
}
