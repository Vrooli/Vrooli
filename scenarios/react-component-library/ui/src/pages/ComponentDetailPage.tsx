/**
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";

import { adoptionsClient } from "../api/adoptions";
import { Button } from "../components/ui/button";
import { EmptyState } from "../components/ui/empty-state";
import { StatusBadge } from "../components/ui/status-badge";
import { componentsClient } from "../api/components";
import { strings } from "../consts/strings";
import { CreateAdoptionDialog } from "../features/adoptions/CreateAdoptionDialog";
import { ComponentEditor, type ComparisonSession } from "../features/components/ComponentEditor";
import { VersionsCard } from "../features/versions/VersionsCard";
import { useTranslation } from "../i18n";

type InfoTab = "overview" | "versions" | "adoptions";

function statusTone(library: number, local: number): "success" | "warning" | "danger" | "info" | "neutral" {
  if (local === 2 || library === 2 || library === 3) return "warning";
  if (local === 3 || library === 4) return "danger";
  if (local === 4 || library === 5) return "info";
  return library === 1 && local === 1 ? "success" : "neutral";
}

function statusLabel(library: number, local: number) {
  const libraryLabel = ["Unspecified", "Current", "Behind", "Deprecated", "Missing", "Unknown"][library] ?? "Unknown";
  const localLabel = ["Unspecified", "Clean", "Modified", "Missing", "Unknown"][local] ?? "Unknown";
  return `${libraryLabel} / ${localLabel}`;
}

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { id } = useParams<{ id: string }>();
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>();
  const [comparison, setComparison] = useState<ComparisonSession | null>(null);
  const [infoTab, setInfoTab] = useState<InfoTab>("overview");
  const [selectedAdoptionID, setSelectedAdoptionID] = useState("");
  const [createTarget, setCreateTarget] = useState<{ componentId: string; scenario: string } | null>(null);

  const { data, isLoading, error } = useQuery({
    queryKey: ["components", "get", id],
    queryFn: () => componentsClient.getComponent({ id: id ?? "" }),
    enabled: !!id,
  });

  const adoptionsQuery = useQuery({
    queryKey: ["adoptions", "component", id],
    queryFn: () => adoptionsClient.listAdoptions({ componentId: id ?? "", limit: 0 }),
    enabled: Boolean(id),
  });
  const suggestionsQuery = useQuery({
    queryKey: ["adoptions", "suggestions", "component", id],
    queryFn: () => adoptionsClient.suggestAdoptions({ componentId: id ?? "", limit: 8 }),
    enabled: Boolean(id) && infoTab === "adoptions",
  });
  const refreshMutation = useMutation({
    mutationFn: () => adoptionsClient.refreshAdoptions({ componentId: id ?? "" }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ["adoptions", "component", id] }),
  });

  if (!id) {
    return (
      <p data-testid="component-detail-missing-id" className="text-sm text-app-danger">
        {t("componentDetail.missingId", { defaultValue: "No component id in URL." })}
      </p>
    );
  }

  if (isLoading) {
    return (
      <p data-testid="component-detail-loading" className="text-sm text-app-muted-foreground">
        {t("componentDetail.loading", { defaultValue: "Loading component…" })}
      </p>
    );
  }

  if (error || !data?.component) {
    return (
      <p data-testid="component-detail-error" className="text-sm text-app-danger">
        {t("componentDetail.notFound", { defaultValue: "Component not found." })}
      </p>
    );
  }

  const component = data.component;
  // Proto clients provide empty collections, while test and older persisted
  // projections can omit these optional-shaped values entirely.
  const headers = (component.headers as Record<string, string> | undefined) ?? {};
  const tags = (component.tags as string[] | undefined) ?? [];
  const designStyles = (component.designStyles as typeof component.designStyles | undefined) ?? [];
  const adoptions = adoptionsQuery.data?.adoptions ?? [];
  const selectedAdoption = adoptions.find((adoption) => adoption.id === selectedAdoptionID) ?? adoptions[0];
  const suggestions = suggestionsQuery.data?.suggestions ?? [];

  const tabs: Array<{ id: InfoTab; label: string }> = [
    { id: "overview", label: t("componentDetail.info.overview", { defaultValue: "Overview" }) },
    { id: "versions", label: t("componentDetail.info.versions", { defaultValue: "Versions" }) },
    { id: "adoptions", label: t("componentDetail.info.adoptions", { defaultValue: "Adoptions" }) },
  ];

  return (
    <div data-testid="component-detail-page" className="flex min-h-0 flex-1 flex-col">
      <ComponentEditor
        id={component.id}
        libraryId={component.libraryId || component.id}
        onClose={() => {
          void navigate("/components");
        }}
        selectedVersion={selectedVersion}
        comparison={comparison}
        onCloseComparison={() => setComparison(null)}
        metadataSlot={(
          <div className="space-y-4">
            <div role="tablist" aria-label={t("componentDetail.info.tabs", { defaultValue: "Component information" })} className="flex border-b border-app-border">
              {tabs.map((tab) => (
                <Button key={tab.id} type="button" role="tab" variant="secondary" aria-selected={infoTab === tab.id} onClick={() => setInfoTab(tab.id)} className={`min-h-0 rounded-none border-0 px-2 py-2 text-xs font-medium ${infoTab === tab.id ? "border-b-2 border-app-primary text-app-foreground" : "text-app-muted-foreground"}`}>
                  {tab.label}
                </Button>
              ))}
            </div>
            {infoTab === "overview" && <>
              <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
                <h3 className="font-medium">{t("componentDetail.info.identity", { defaultValue: "Identity" })}</h3>
                <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.libraryIdLabel)}</dt><dd className="break-all font-mono">{component.libraryId}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.slotLabel)}</dt><dd>{component.slot || "—"}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.categoryLabel)}</dt><dd>{component.category || headers.category || "—"}</dd>
                  <dt className="text-app-muted-foreground">{t(strings.components.editor.tagsLabel)}</dt><dd>{tags.join(", ") || "—"}</dd>
                </dl>
              </section>
              <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
                <h3 className="font-medium">{t("componentDetail.info.affinities", { defaultValue: "Design affinities" })}</h3>
                {designStyles.length === 0 ? <p className="mt-2 text-xs text-app-muted-foreground">{t("componentDetail.info.noAffinities", { defaultValue: "No design affinities declared." })}</p> : <ul className="mt-2 space-y-1 text-xs">{designStyles.map((affinity) => <li key={affinity.styleId}>{affinity.styleId}: {affinity.reason || affinity.affinity}</li>)}</ul>}
              </section>
            </>}
            {infoTab === "versions" && <VersionsCard componentId={component.id} selectedVersion={selectedVersion} onSelectVersion={setSelectedVersion} onCompare={setComparison} />}
            {infoTab === "adoptions" && <section data-testid="component-detail-adoptions" className="space-y-3 text-sm text-app-foreground">
              <div className="flex items-center justify-between"><h3 className="font-medium">{t("componentDetail.info.adoptions", { defaultValue: "Adoptions" })}</h3><Button size="sm" onClick={() => refreshMutation.mutate()} disabled={refreshMutation.isPending}>{refreshMutation.isPending ? t(strings.adoptions.refreshing) : t(strings.adoptions.refreshAction)}</Button></div>
              {adoptionsQuery.isLoading ? <p className="text-xs text-app-muted-foreground">{t("componentDetail.info.adoptionsLoading", { defaultValue: "Loading adoptions…" })}</p> : adoptions.length === 0 ? <EmptyState className="p-2 text-xs" title={t("componentDetail.info.noAdoptions", { defaultValue: "No scenarios have adopted this component yet." })} /> : <div className="space-y-2">{adoptions.map((adoption) => <Button key={adoption.id} type="button" variant="secondary" onClick={() => setSelectedAdoptionID(adoption.id)} className={`h-auto w-full rounded-control border p-2 text-left ${selectedAdoption?.id === adoption.id ? "border-app-primary" : "border-app-border"}`}><div className="flex items-center justify-between gap-2"><span className="font-medium">{adoption.scenario}</span><StatusBadge tone={statusTone(adoption.libraryVersionStatus, adoption.localStatus)}>{statusLabel(adoption.libraryVersionStatus, adoption.localStatus)}</StatusBadge></div><p className="mt-1 font-mono text-xs text-app-muted-foreground">{adoption.adoptedVersion} · {adoption.adoptedPath}</p></Button>)}</div>}
              {selectedAdoption && <ul data-testid="component-detail-adoption-file-tree" className="space-y-1 rounded-control bg-app-background p-2 font-mono text-xs text-app-muted-foreground">{(selectedAdoption.files.length > 0 ? selectedAdoption.files.map((file) => file.adoptedPath) : [selectedAdoption.adoptedPath]).map((path) => <li key={path}>{path}</li>)}</ul>}
              <div className="border-t border-app-border pt-3"><h3 className="font-medium">{t(strings.adoptions.suggestions.title)}</h3>{suggestionsQuery.isLoading ? <p className="mt-1 text-xs text-app-muted-foreground">{t("componentDetail.info.suggestionsLoading", { defaultValue: "Finding candidates…" })}</p> : <div className="mt-2 space-y-2">{suggestions.map((suggestion) => <div key={suggestion.scenario} className="rounded-control border border-app-border p-2"><p className="font-medium">{suggestion.scenario}</p><p className="mt-1 text-xs text-app-muted-foreground">{suggestion.reasons.join(" · ")}</p><Button size="sm" className="mt-2" onClick={() => setCreateTarget({ componentId: component.id, scenario: suggestion.scenario })}>{t(strings.adoptions.suggestions.adoptAction)}</Button></div>)}{suggestions.length === 0 && <p className="mt-1 text-xs text-app-muted-foreground">{t(strings.adoptions.suggestions.empty)}</p>}</div>}</div>
            </section>}
          </div>
        )}
      />
      <CreateAdoptionDialog open={Boolean(createTarget)} initial={createTarget} onClose={() => setCreateTarget(null)} />
    </div>
  );
}
