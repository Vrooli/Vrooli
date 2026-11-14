import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FileCode, Play, Plus, Save, Trash2 } from "lucide-react";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  createPlannedScenario,
  deletePlannedProtoFile,
  getPlannedScenario,
  listPlannedScenarios,
  materializePlannedScenario,
  putPlannedProtoFile,
  validatePlannedScenario,
  type PlannedScenario,
  type ValidatePlannedScenarioResponse,
} from "../../api/techTree";

const starterProto = (slug: string) => `syntax = "proto3";

package vrooli.${slug.replace(/-/g, "_")}.v1;

// @stability experimental

message ${toPascal(slug)}Request {
  string scenario_id = 1;
}
`;

const toPascal = (slug: string) =>
  slug
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.slice(0, 1).toUpperCase() + part.slice(1))
    .join("") || "PlannedScenario";

export function PlanningPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const scenariosQuery = useQuery({ queryKey: ["planned-scenarios"], queryFn: listPlannedScenarios });
  const scenarios = useMemo(() => scenariosQuery.data ?? [], [scenariosQuery.data]);
  const [selectedSlug, setSelectedSlug] = useState("");
  const [newSlug, setNewSlug] = useState("");
  const [filePath, setFilePath] = useState("");
  const [fileText, setFileText] = useState("");
  const [validation, setValidation] = useState<ValidatePlannedScenarioResponse | null>(null);

  useEffect(() => {
    if (!selectedSlug && scenarios[0]) setSelectedSlug(scenarios[0].slug);
  }, [scenarios, selectedSlug]);

  const selectedQuery = useQuery({
    queryKey: ["planned-scenario", selectedSlug],
    queryFn: () => getPlannedScenario(selectedSlug),
    enabled: Boolean(selectedSlug),
  });

  const selected = selectedQuery.data;
  const activeFile = useMemo(() => selected?.files.find((file) => file.path === filePath), [filePath, selected]);

  useEffect(() => {
    if (!selected) return;
    const firstPath = selected.files[0]?.path ?? `packages/proto/schemas/${selected.slug}/v1/${selected.slug}.proto`;
    if (!filePath || !selected.files.some((file) => file.path === filePath)) {
      setFilePath(firstPath);
      setFileText(selected.files[0]?.text ?? starterProto(selected.slug));
      return;
    }
    setFileText(activeFile?.text ?? fileText);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected?.slug, selected?.files.length, filePath]);

  const invalidatePlanning = async (slug = selectedSlug) => {
    await queryClient.invalidateQueries({ queryKey: ["planned-scenarios"] });
    if (slug) await queryClient.invalidateQueries({ queryKey: ["planned-scenario", slug] });
    await queryClient.invalidateQueries({ queryKey: ["tech-tree-graph"] });
  };

  const createMutation = useMutation({
    mutationFn: () => createPlannedScenario({
      slug: newSlug.trim(),
      displayName: toPascal(newSlug.trim()).replace(/([A-Z])/g, " $1").trim(),
      sector: "engineering",
      tier: "integration",
    }),
    onSuccess: async (scenario) => {
      setSelectedSlug(scenario.slug);
      setNewSlug("");
      await invalidatePlanning(scenario.slug);
    },
  });

  const saveMutation = useMutation({
    mutationFn: () => putPlannedProtoFile({ slug: selectedSlug, path: filePath.trim(), text: fileText }),
    onSuccess: async () => invalidatePlanning(),
  });

  const deleteMutation = useMutation({
    mutationFn: () => deletePlannedProtoFile({ slug: selectedSlug, path: filePath }),
    onSuccess: async () => {
      setFilePath("");
      setFileText("");
      await invalidatePlanning();
    },
  });

  const validateMutation = useMutation({
    mutationFn: () => validatePlannedScenario(selectedSlug),
    onSuccess: setValidation,
  });

  const materializeMutation = useMutation({
    mutationFn: () => materializePlannedScenario(selectedSlug),
  });

  return (
    <section data-testid={selectors.pages.planning} className="grid gap-5 xl:grid-cols-[320px_minmax(0,1fr)]">
      <aside className="rounded-lg border border-app-border bg-app-surface p-4">
        <p className="text-sm font-medium uppercase text-app-muted-foreground">{t(strings.planning.eyebrow)}</p>
        <h2 className="mt-1 text-2xl font-semibold">{t(strings.planning.title)}</h2>
        <div className="mt-4 flex gap-2">
          <Input
            data-testid={selectors.planning.slugInput}
            value={newSlug}
            onChange={(event) => setNewSlug(event.target.value)}
            placeholder={t(strings.planning.slugPlaceholder)}
          />
          <Button
            size="sm"
            disabled={!newSlug.trim() || createMutation.isPending}
            onClick={() => createMutation.mutate()}
            aria-label={t(strings.planning.actions.create)}
          >
            <Plus aria-hidden className="h-4 w-4" />
          </Button>
        </div>
        {scenariosQuery.isLoading && <p className="mt-4 text-sm text-app-muted-foreground">{t(strings.planning.states.loading)}</p>}
        <div className="mt-4 space-y-2">
          {scenarios.map((scenario) => (
            <button
              key={scenario.slug}
              type="button"
              onClick={() => {
                setSelectedSlug(scenario.slug);
                setValidation(null);
              }}
              className={[
                "w-full rounded-md border px-3 py-2 text-left text-sm transition",
                selectedSlug === scenario.slug
                  ? "border-app-primary bg-app-primary/10"
                  : "border-app-border hover:bg-app-surface-muted",
              ].join(" ")}
            >
              <span className="block truncate font-medium">{scenario.displayName || scenario.slug}</span>
              <span className="block truncate text-xs text-app-muted-foreground">{scenario.slug}</span>
            </button>
          ))}
          {!scenariosQuery.isLoading && scenarios.length === 0 && (
            <p className="rounded-md border border-dashed border-app-border p-3 text-sm text-app-muted-foreground">
              {t(strings.planning.states.emptyList)}
            </p>
          )}
        </div>
      </aside>

      <div className="min-w-0 rounded-lg border border-app-border bg-app-surface p-4">
        {!selected && (
          <div className="flex min-h-[420px] items-center justify-center text-center text-app-muted-foreground">
            {t(strings.planning.states.noSelection)}
          </div>
        )}
        {selected && (
          <Editor
            scenario={selected}
            filePath={filePath}
            fileText={fileText}
            validation={validation}
            onFilePathChange={setFilePath}
            onFileTextChange={setFileText}
            onSelectFile={(path, text) => {
              setFilePath(path);
              setFileText(text);
            }}
            onSave={() => saveMutation.mutate()}
            onDelete={() => deleteMutation.mutate()}
            onValidate={() => validateMutation.mutate()}
            onMaterialize={() => materializeMutation.mutate()}
            busy={saveMutation.isPending || validateMutation.isPending || materializeMutation.isPending}
            materializedPaths={materializeMutation.data?.writtenPaths ?? []}
          />
        )}
      </div>
    </section>
  );
}

function Editor(props: {
  scenario: PlannedScenario;
  filePath: string;
  fileText: string;
  validation: ValidatePlannedScenarioResponse | null;
  busy: boolean;
  materializedPaths: string[];
  onFilePathChange: (path: string) => void;
  onFileTextChange: (text: string) => void;
  onSelectFile: (path: string, text: string) => void;
  onSave: () => void;
  onDelete: () => void;
  onValidate: () => void;
  onMaterialize: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="grid min-h-[620px] gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
      <div className="rounded-md border border-app-border bg-app-background/45 p-3">
        <p className="font-medium">{props.scenario.displayName || props.scenario.slug}</p>
        <p className="text-xs text-app-muted-foreground">
          {props.scenario.sector || t(strings.planning.fallbacks.unassigned)} / {props.scenario.tier || t(strings.planning.fallbacks.untiered)}
        </p>
        <div className="mt-4 space-y-2">
          {props.scenario.files.map((file) => (
            <button
              key={file.path}
              type="button"
              onClick={() => props.onSelectFile(file.path, file.text)}
              className={[
                "flex w-full items-center gap-2 rounded-md px-2 py-2 text-left text-xs",
                file.path === props.filePath ? "bg-app-primary text-app-primary-foreground" : "hover:bg-app-surface-muted",
              ].join(" ")}
            >
              <FileCode aria-hidden className="h-4 w-4 shrink-0" />
              <span className="min-w-0 break-all">{file.path}</span>
            </button>
          ))}
          {props.scenario.files.length === 0 && (
            <p className="text-sm text-app-muted-foreground">{t(strings.planning.states.noFiles)}</p>
          )}
        </div>
      </div>
      <div className="min-w-0 space-y-3">
        <Input value={props.filePath} onChange={(event) => props.onFilePathChange(event.target.value)} />
        <Textarea
          data-testid={selectors.planning.editor}
          value={props.fileText}
          onChange={(event) => props.onFileTextChange(event.target.value)}
          className="min-h-[360px] font-mono"
        />
        <div className="flex flex-wrap gap-2">
          <Button disabled={props.busy || !props.filePath.trim()} onClick={props.onSave}>
            <Save aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.planning.actions.save)}
          </Button>
          <Button variant="outline" disabled={props.busy || !props.filePath.trim()} onClick={props.onValidate}>
            <Play aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.planning.actions.validate)}
          </Button>
          <Button variant="outline" disabled={props.busy || !props.filePath.trim()} onClick={props.onMaterialize}>
            {t(strings.planning.actions.materialize)}
          </Button>
          <Button variant="outline" disabled={props.busy || !props.filePath.trim()} onClick={props.onDelete}>
            <Trash2 aria-hidden className="mr-2 h-4 w-4" />
            {t(strings.planning.actions.delete)}
          </Button>
        </div>
        {props.validation && (
          <div data-testid={selectors.planning.findings} className="rounded-md border border-app-border p-3">
            <p className="font-medium">
              {props.validation.passed ? t(strings.planning.validation.passed) : t(strings.planning.validation.findings)}
            </p>
            <div className="mt-2 space-y-2">
              {props.validation.findings.length === 0 && (
                <p className="text-sm text-app-muted-foreground">{t(strings.planning.validation.none)}</p>
              )}
              {props.validation.findings.map((finding) => (
                <div key={`${finding.code}-${finding.location}-${finding.message}`} className="rounded-md bg-black/10 p-2 text-sm">
                  <p className="font-medium">{finding.code} <span className="text-app-muted-foreground">{finding.location}</span></p>
                  <p>{finding.message}</p>
                  {finding.suggestion && <p className="text-app-muted-foreground">{finding.suggestion}</p>}
                </div>
              ))}
            </div>
          </div>
        )}
        {props.materializedPaths.length > 0 && (
          <div className="rounded-md border border-emerald-500/35 bg-emerald-950/20 p-3 text-sm text-emerald-100">
            {t(strings.planning.materialized, { count: props.materializedPaths.length })}
          </div>
        )}
      </div>
    </div>
  );
}
