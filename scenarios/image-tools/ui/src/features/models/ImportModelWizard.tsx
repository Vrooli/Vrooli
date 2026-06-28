import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";
import type { InspectModelSourceResponse } from "@vrooli/proto-types/image-tools/v1/models/models_pb";

/** The architectures an imported base model may declare (mirrors models.Architecture). */
const ARCHITECTURES = ["sd15", "sdxl", "flux", "instruct-pix2pix", "qwen-image-edit", "longcat-image-edit"];

const layoutLabel = (layout: number): string => {
  switch (layout) {
    case 1:
      return "single-file checkpoint";
    case 2:
      return "diffusers repo";
    default:
      return "unknown";
  }
};

/**
 * ImportModelWizard is the guided bring-your-own-model flow (plan capability D):
 * paste a source → Inspect (dry run: layout + inferred architecture + license/
 * NSFW + offered ops, installs nothing) → confirm the architecture/id → Import
 * (registers add-only + submits the install job). On success it invalidates the
 * models list so the new entry appears with its derived ops.
 */
export function ImportModelWizard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [source, setSource] = useState("");
  const [id, setId] = useState("");
  const [name, setName] = useState("");
  const [architecture, setArchitecture] = useState("");
  const [attest, setAttest] = useState(false);

  const inspectMutation = useMutation({
    mutationFn: (src: string) => modelsClient.inspectModelSource({ source: src }),
    onSuccess: (resp: InspectModelSourceResponse) => {
      // Prefill the confirmable fields from the proposal so the common path is
      // one click; the operator can still override before importing.
      setId(resp.proposed?.id ?? "");
      setArchitecture(resp.architecture?.architecture && resp.architecture.architecture !== "none" ? resp.architecture.architecture : "");
    },
  });

  const importMutation = useMutation({
    mutationFn: () =>
      modelsClient.importModel({
        source: source.trim(),
        id: id.trim(),
        name: name.trim(),
        architecture: architecture.trim(),
        attestCommercialRights: attest,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["models"] });
    },
  });

  const preview = inspectMutation.data;
  const inferred = preview?.architecture;
  const needsArchitecture = Boolean(preview) && !architecture.trim();

  const onInspect = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (source.trim()) {
      inspectMutation.mutate(source.trim());
    }
  };

  const fieldId = (key: string) => `models-import-${key}`;

  return (
    <section
      aria-label={t(strings.models.import.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4"
    >
      <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.models.import.title)}</h2>

      <form
        data-testid={selectors.models.import.form}
        onSubmit={onInspect}
        className="mt-3 flex flex-col gap-3"
      >
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("source")} className="text-xs text-app-muted-foreground">
            {t(strings.models.import.sourceLabel)}
          </label>
          <div className="flex gap-2">
            <Input
              id={fieldId("source")}
              data-testid={selectors.models.import.source}
              value={source}
              onChange={(e) => setSource(e.target.value)}
              required
            />
            <Button
              data-testid={selectors.models.import.inspect}
              type="submit"
              disabled={!source.trim() || inspectMutation.isPending}
            >
              {inspectMutation.isPending ? t(strings.models.import.inspecting) : t(strings.models.import.inspect)}
            </Button>
          </div>
        </div>
      </form>

      {preview && (
        <div data-testid={selectors.models.import.preview} className="mt-4 flex flex-col gap-3">
          <dl className="grid grid-cols-[max-content_1fr] gap-x-3 gap-y-1 text-xs">
            <dt className="text-app-muted-foreground">{t(strings.models.import.layoutLabel)}</dt>
            <dd>{layoutLabel(preview.layout)}</dd>
            <dt className="text-app-muted-foreground">{t(strings.models.import.architectureInferred)}</dt>
            <dd>
              {`${inferred?.architecture ?? "none"} (${inferred?.confidence ?? "none"}) — ${inferred?.evidence ?? ""}`}
            </dd>
            <dt className="text-app-muted-foreground">{t(strings.models.import.licenseLabel)}</dt>
            <dd>{preview.license || "unverified"}</dd>
            <dt className="text-app-muted-foreground">{t(strings.models.import.nsfwLabel)}</dt>
            <dd>{preview.nsfw ? "yes" : "no"}</dd>
            <dt className="text-app-muted-foreground">{t(strings.models.import.sizeLabel)}</dt>
            <dd>{`~${Number(preview.sizeBytes / BigInt(1024 * 1024))} MB`}</dd>
            <dt className="text-app-muted-foreground">{t(strings.models.import.offeredOpsLabel)}</dt>
            <dd>{preview.offeredOperations.join(", ")}</dd>
          </dl>

          <div className="flex flex-col gap-1">
            <label htmlFor={fieldId("id")} className="text-xs text-app-muted-foreground">
              {t(strings.models.import.idLabel)}
            </label>
            <Input id={fieldId("id")} data-testid={selectors.models.import.id} value={id} onChange={(e) => setId(e.target.value)} required />
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor={fieldId("name")} className="text-xs text-app-muted-foreground">
              {t(strings.models.import.nameLabel)}
            </label>
            <Input id={fieldId("name")} data-testid={selectors.models.import.name} value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor={fieldId("architecture")} className="text-xs text-app-muted-foreground">
              {t(strings.models.import.architectureLabel)}
            </label>
            <select
              id={fieldId("architecture")}
              data-testid={selectors.models.import.architecture}
              value={architecture}
              onChange={(e) => setArchitecture(e.target.value)}
              className="rounded-md border border-app-border bg-app-background p-2 text-sm"
            >
              <option value="">{t(strings.models.import.selectArchitecture)}</option>
              {ARCHITECTURES.map((arch) => (
                <option key={arch} value={arch}>
                  {arch}
                </option>
              ))}
            </select>
            {needsArchitecture && (
              <p className="text-xs text-app-warning">{t(strings.models.import.needsArchitecture)}</p>
            )}
          </div>

          <label className="flex items-center gap-2 text-xs text-app-muted-foreground">
            <input
              type="checkbox"
              data-testid={selectors.models.import.attest}
              checked={attest}
              onChange={(e) => setAttest(e.target.checked)}
            />
            {t(strings.models.import.attestLabel)}
          </label>

          <Button
            data-testid={selectors.models.import.submit}
            type="button"
            onClick={() => importMutation.mutate()}
            disabled={!id.trim() || needsArchitecture || importMutation.isPending}
          >
            {importMutation.isPending ? t(strings.models.import.importing) : t(strings.models.import.import)}
          </Button>
        </div>
      )}

      {importMutation.isSuccess && (
        <p data-testid={selectors.models.import.success} className="mt-2 text-xs text-app-success">
          {t(strings.models.import.success)}
        </p>
      )}
      {(inspectMutation.error || importMutation.error) && (
        <p data-testid={selectors.models.import.error} className="mt-2 text-xs text-app-danger">
          {errorMessage(inspectMutation.error ?? importMutation.error, t)}
        </p>
      )}
    </section>
  );
}
