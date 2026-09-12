/** @vrooliComponentSource forms.input */
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { componentsClient } from "../../api/components";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Input } from "@vrooli/react-component-library/Input/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { workflowsClient } from "../../api/workflows";

export function IngestComponentForm() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [scenario, setScenario] = useState("");
  const [sourceFile, setSourceFile] = useState("");
  const [slug, setSlug] = useState("");
  const [tags, setTags] = useState("");
  const [acceptBehaviorLoss, setAcceptBehaviorLoss] = useState(false);
  const ingest = useMutation({
    mutationFn: () =>
      componentsClient.ingestComponent({
        scenario,
        sourceFile,
        slug,
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        acceptBehaviorLoss,
      }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["components"] }),
  });
  const acceptedLosses = ingest.data?.parityReport?.acknowledged
    ? ingest.data.parityReport.findings.length
    : 0;
  const readiness = useQuery({
    queryKey: [
      "promotion-readiness",
      ingest.data?.component?.id,
      scenario,
      ingest.data?.draftVersion,
    ],
    queryFn: () =>
      workflowsClient.getPromotionReadiness({
        assetId: ingest.data!.component!.id,
        originScenario: scenario,
        version: ingest.data!.draftVersion,
      }),
    enabled: Boolean(ingest.data?.component?.id && scenario),
  });

  return (
    <details
      data-testid={selectors.components.ingest.details}
      className="rounded-xl border border-app-border bg-app-surface p-space-sm"
    >
      <summary className="cursor-pointer text-sm font-medium text-app-foreground">
        {t(strings.components.ingest.title)}
      </summary>
      <p className="mt-space-2xs text-xs text-app-muted-foreground">
        {t(strings.components.ingest.description)}
      </p>
      <form
        className="mt-space-xs grid gap-space-2xs md:grid-cols-2"
        onSubmit={(event) => {
          event.preventDefault();
          ingest.mutate();
        }}
      >
        <Input
          data-testid={selectors.components.ingest.scenario}
          value={scenario}
          onChange={(event) => setScenario(event.target.value)}
          placeholder={t(strings.components.ingest.scenario)}
          required
        />
        <Input
          data-testid={selectors.components.ingest.sourceFile}
          value={sourceFile}
          onChange={(event) => setSourceFile(event.target.value)}
          placeholder={t(strings.components.ingest.sourceFile)}
          required
        />
        <Input
          data-testid={selectors.components.ingest.slug}
          value={slug}
          onChange={(event) => setSlug(event.target.value)}
          placeholder={t(strings.components.ingest.slug)}
          required
        />
        <Input
          data-testid={selectors.components.ingest.tags}
          value={tags}
          onChange={(event) => setTags(event.target.value)}
          placeholder={t(strings.components.ingest.tags)}
        />
        <label className="flex items-center gap-space-2xs text-xs text-app-muted-foreground md:col-span-2">
          <input
            data-testid={selectors.components.ingest.acceptLoss}
            type="checkbox"
            checked={acceptBehaviorLoss}
            onChange={(event) => setAcceptBehaviorLoss(event.target.checked)}
          />
          {t(strings.components.ingest.acceptBehaviorLoss)}
        </label>
        <div className="md:col-span-2">
          <Button
            data-testid={selectors.components.ingest.submit}
            type="submit"
            disabled={ingest.isPending}
          >
            {ingest.isPending
              ? t(strings.components.ingest.running)
              : t(strings.components.ingest.submit)}
          </Button>
        </div>
      </form>
      {ingest.error && (
        <p
          data-testid={selectors.components.ingest.error}
          className="mt-space-2xs text-xs text-app-danger"
        >
          {errorMessage(ingest.error, t)}
        </p>
      )}
      {ingest.data && (
        <div className="mt-space-2xs space-y-space-2xs text-xs">
          <p data-testid={selectors.components.ingest.success} className="text-app-success">
            {t(strings.components.ingest.success, {
              version: ingest.data.draftVersion,
              findings: ingest.data.findings.length,
            })}
            {acceptedLosses > 0 &&
              ` ${t(strings.components.ingest.acceptedNotice, { findings: acceptedLosses })}`}
          </p>
          {readiness.data?.readiness && (
            <section
              aria-label={t("components.ingest.promotionReadiness", {
                defaultValue: "Promotion readiness",
              })}
              className="rounded-control border border-app-border p-space-2xs text-app-muted-foreground"
            >
              <p className="font-medium text-app-foreground">
                {readiness.data.readiness.ready
                  ? t("components.ingest.promotionReady", {
                      defaultValue: "Promotion evidence is complete.",
                    })
                  : t("components.ingest.promotionBlocked", {
                      defaultValue: "Promotion evidence is incomplete.",
                    })}
              </p>
              <p>
                {t("components.ingest.promotionExamples", {
                  defaultValue: "Examples: {{available}}/{{required}}",
                  available: readiness.data.readiness.availableExampleCount,
                  required: readiness.data.readiness.requiredExampleCount,
                })}
              </p>
              {readiness.data.readiness.blockers.map((blocker) => (
                <p key={blocker} className="text-app-danger">
                  {blocker}
                </p>
              ))}
              <p className="mt-space-3xs font-mono">
                {readiness.data.readiness.nextValidationCommand}
              </p>
            </section>
          )}
        </div>
      )}
    </details>
  );
}
