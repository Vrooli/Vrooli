/**
 * ComponentDetailPage — full-width editor + preview for a component.
 *
 * Resolves the component id from the route param, fetches the
 * component record so the editor header shows the libraryId (instead
 * of the bare id), and hands the rest off to `<ComponentEditor />`.
 * Closing the editor returns the user to the components list.
 */
import { useQuery } from "@tanstack/react-query";
import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { adoptionsClient } from "../api/adoptions";
import { componentsClient } from "../api/components";
import { strings } from "../consts/strings";
import { ComponentEditor } from "../features/components/ComponentEditor";
import { VersionsCard } from "../features/versions/VersionsCard";
import { useTranslation } from "../i18n";

export function ComponentDetailPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [selectedVersion, setSelectedVersion] = useState<string | undefined>();

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

  return (
    <div data-testid="component-detail-page" className="flex min-h-0 flex-1 flex-col">
      <ComponentEditor
        id={component.id}
        libraryId={component.libraryId || component.id}
        onClose={() => {
          void navigate("/components");
        }}
        selectedVersion={selectedVersion}
        metadataSlot={(
          <div className="space-y-4">
            <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
              <h3 className="font-medium">{t("componentDetail.info.identity", { defaultValue: "Identity" })}</h3>
              <dl className="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
                <dt className="text-app-muted-foreground">{t(strings.components.editor.libraryIdLabel)}</dt><dd className="break-all font-mono">{component.libraryId}</dd>
                <dt className="text-app-muted-foreground">{t(strings.components.editor.slotLabel)}</dt><dd>{component.slot || "—"}</dd>
                <dt className="text-app-muted-foreground">{t(strings.components.editor.categoryLabel)}</dt><dd>{headers.category || "—"}</dd>
                <dt className="text-app-muted-foreground">{t(strings.components.editor.tagsLabel)}</dt><dd>{tags.join(", ") || "—"}</dd>
              </dl>
            </section>
            <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
              <h3 className="font-medium">{t("componentDetail.info.affinities", { defaultValue: "Design affinities" })}</h3>
              {designStyles.length === 0 ? (
                <p className="mt-2 text-xs text-app-muted-foreground">{t("componentDetail.info.noAffinities", { defaultValue: "No design affinities declared." })}</p>
              ) : (
                <ul className="mt-2 space-y-1 text-xs">
                  {designStyles.map((affinity) => <li key={affinity.styleId}>{affinity.styleId}: {affinity.reason || affinity.affinity}</li>)}
                </ul>
              )}
            </section>
            <section className="rounded-lg border border-app-border bg-app-surface-muted p-3 text-sm text-app-foreground">
              <h3 className="font-medium">{t("componentDetail.info.adoptions", { defaultValue: "Adoptions" })}</h3>
              <p className="mt-2 text-xs text-app-muted-foreground">
                {adoptionsQuery.isLoading
                  ? t("componentDetail.info.adoptionsLoading", { defaultValue: "Loading adoption count…" })
                  : t("componentDetail.info.adoptionsCount", { defaultValue: "{{count}} adoption(s)", count: adoptionsQuery.data?.adoptions.length ?? 0 })}
              </p>
              <Link className="mt-2 inline-flex text-xs text-app-primary underline" to={`/adoptions?componentId=${encodeURIComponent(component.id)}`}>
                {t("componentDetail.info.viewAdoptions", { defaultValue: "View adoptions" })}
              </Link>
            </section>
            <VersionsCard componentId={component.id} selectedVersion={selectedVersion} onSelectVersion={setSelectedVersion} />
          </div>
        )}
      />
    </div>
  );
}
