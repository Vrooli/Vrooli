import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { Button } from "../../components/ui/button";
import { DataTable, type DataTableColumn } from "../../components/ui/data-table";
import { EmptyState } from "../../components/ui/empty-state";
import { Input } from "../../components/ui/input";
import { StatusBadge } from "../../components/ui/status-badge";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { componentsClient, type Component } from "../../api/components";
import { errorMessage } from "../../lib/errorMessage";
import { CreateComponentDialog } from "./CreateComponentDialog";

const DESIGN_AFFINITY_NATIVE = 1;
const DESIGN_AFFINITY_COMPATIBLE = 2;
const DESIGN_AFFINITY_DISCOURAGED = 3;

function isDiscouragedAffinity(affinity: unknown) {
  return affinity === DESIGN_AFFINITY_DISCOURAGED;
}

function componentSearchValue(component: Component) {
  return [
    component.libraryId,
    component.displayName,
    component.version,
    component.slot,
    component.category,
    component.description,
    ...component.tags,
    ...component.designStyles.map((style) => `${style.styleId} ${formatAffinity(style.affinity)}`),
  ]
    .filter(Boolean)
    .join(" ");
}

/**
 * ComponentsCard renders the indexed component registry. The user can
 * filter by name substring + tag, trigger a re-index from disk, and
 * inspect each entry (libraryId, displayName, version, tags). It is
 * the surface for requirements 01 (registry/header) and 07 (search +
 * filter) — see `requirements/`.
 */
export function ComponentsCard() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [match, setMatch] = useState("");
  const [tag, setTag] = useState("");
  const [tagsRaw, setTagsRaw] = useState("");
  const [category, setCategory] = useState("");
  const [styleId, setStyleId] = useState("");
  const [affinity, setAffinity] = useState("");
  const [showCreate, setShowCreate] = useState(false);

  const tags = tagsRaw
    .split(",")
    .map((t) => t.trim())
    .filter((t) => t !== "");

  const componentsQuery = useQuery({
    queryKey: ["components", { match, tag, tags, category, styleId, affinity }],
    queryFn: () =>
      componentsClient.listComponents({ match, tag, tags, category, styleId, affinity }),
  });

  const indexMutation = useMutation({
    mutationFn: () => componentsClient.indexComponents({}),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["components"] });
    },
  });

  const components = componentsQuery.data?.components ?? [];

  const columns: Array<DataTableColumn<Component>> = [
    {
      id: "component",
      header: "Component",
      sortValue: (component) => component.displayName,
      searchValue: componentSearchValue,
      accessor: (component) => (
        <div className="min-w-64">
          <div
            data-testid={selectors.components.itemDisplayName}
            className="font-medium text-app-foreground"
          >
            {component.displayName}
          </div>
          <div
            data-testid={selectors.components.itemLibraryId}
            className="mt-1 font-mono text-xs text-app-muted-foreground"
          >
            {component.libraryId}
          </div>
          {component.description && (
            <div className="mt-1 text-xs text-app-muted-foreground">{component.description}</div>
          )}
        </div>
      ),
    },
    {
      id: "version",
      header: "Version",
      sortValue: (component) => component.version,
      searchValue: (component) => component.version,
      accessor: (component) =>
        component.version ? (
          <span data-testid={selectors.components.itemVersion} className="text-xs">
            {t(strings.components.versionLabel, { version: component.version })}
          </span>
        ) : (
          <span className="text-xs text-app-muted-foreground">-</span>
        ),
    },
    {
      id: "slot",
      header: "Slot",
      sortValue: (component) => component.slot,
      searchValue: (component) => component.slot,
      accessor: (component) =>
        component.slot ? (
          <span data-testid={selectors.components.itemSlot} className="font-mono text-xs">
            {t(strings.components.slotLabel, { slot: component.slot })}
          </span>
        ) : (
          <span className="text-xs text-app-muted-foreground">-</span>
        ),
    },
    {
      id: "design",
      header: "Design",
      searchValue: (component) =>
        component.designStyles
          .map((style) => `${style.styleId}:${formatAffinity(style.affinity)}`)
          .join(" "),
      accessor: (component) => {
        const designStyles = component.designStyles;
        const designStylesSummary = designStyles
          .map((style) => `${style.styleId}:${formatAffinity(style.affinity)}`)
          .join(", ");
        if (designStyles.length === 0) {
          return <span className="text-xs text-app-muted-foreground">-</span>;
        }
        return (
          <div
            data-testid={selectors.components.itemDesignStyles}
            className="flex max-w-80 flex-wrap gap-1 text-xs"
            aria-label={t(strings.components.designStylesLabel, {
              styles: designStylesSummary,
            })}
          >
            {designStyles.map((style) => (
              <StatusBadge
                key={style.styleId}
                tone={isDiscouragedAffinity(style.affinity) ? "warning" : "neutral"}
              >
                {style.styleId}:{formatAffinity(style.affinity)}
              </StatusBadge>
            ))}
          </div>
        );
      },
    },
    {
      id: "tags",
      header: "Tags",
      searchValue: (component) => component.tags.join(" "),
      accessor: (component) => (
        <div data-testid={selectors.components.itemTags} className="max-w-64 text-xs">
          {component.tags.length > 0
            ? t(strings.components.tagsLabel, { tags: component.tags.join(", ") })
            : t(strings.components.noTags)}
        </div>
      ),
    },
    {
      id: "actions",
      header: "Actions",
      className: "text-right",
      accessor: (component) => (
        <div className="flex items-center justify-end gap-2">
          <Link
            to={`/components/${component.id}`}
            className="inline-flex min-h-8 items-center justify-center rounded-control border border-app-border bg-app-surface px-3 text-xs font-medium text-app-foreground transition hover:bg-app-surface-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-app-primary/50"
          >
            {t(strings.components.openAction)}
          </Link>
        </div>
      ),
    },
  ];

  return (
    <section
      data-testid={selectors.components.card}
      aria-label={t(strings.components.title)}
      className="mt-4 rounded-xl border border-app-border bg-app-surface p-4"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-app-muted-foreground">{t(strings.components.title)}</h2>
        <div className="flex items-center gap-2">
          <Button
            data-testid={selectors.components.create.button}
            onClick={() => setShowCreate(true)}
          >
            {t(strings.components.create.action)}
          </Button>
          <Button
            data-testid={selectors.components.indexButton}
            onClick={() => indexMutation.mutate()}
            disabled={indexMutation.isPending}
            variant="secondary"
          >
            {t(strings.components.indexAction)}
          </Button>
        </div>
      </div>

      {showCreate && <CreateComponentDialog onClose={() => setShowCreate(false)} />}

      <details
        data-testid="components-filters"
        className="group mt-3 rounded-control border border-app-border bg-app-surface-muted p-2"
      >
        <summary className="flex cursor-pointer list-none items-center justify-between text-xs font-medium text-app-foreground">
          {t("components.filters", { defaultValue: "Filters" })}
          <span className="text-app-muted-foreground group-open:hidden">
            {t("components.filtersShow", { defaultValue: "Show" })}
          </span>
          <span className="hidden text-app-muted-foreground group-open:inline">
            {t("components.filtersHide", { defaultValue: "Hide" })}
          </span>
        </summary>
        <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.searchLabel)}
            <Input
              data-testid={selectors.components.searchInput}
              value={match}
              onChange={(e) => setMatch(e.target.value)}
              placeholder={t(strings.components.searchPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.tagLabel)}
            <Input
              data-testid={selectors.components.tagInput}
              value={tag}
              onChange={(e) => setTag(e.target.value)}
              placeholder={t(strings.components.tagPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.tagsFilterLabel)}
            <Input
              data-testid={selectors.components.tagsInput}
              value={tagsRaw}
              onChange={(e) => setTagsRaw(e.target.value)}
              placeholder={t(strings.components.tagsFilterPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.categoryLabel)}
            <Input
              data-testid={selectors.components.categoryInput}
              value={category}
              onChange={(e) => setCategory(e.target.value)}
              placeholder={t(strings.components.categoryPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.styleLabel)}
            <Input
              data-testid={selectors.components.styleInput}
              value={styleId}
              onChange={(e) => setStyleId(e.target.value)}
              placeholder={t(strings.components.stylePlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.affinityLabel)}
            <Input
              data-testid={selectors.components.affinityInput}
              value={affinity}
              onChange={(e) => setAffinity(e.target.value)}
              placeholder={t(strings.components.affinityPlaceholder)}
              className="mt-1"
            />
          </label>
        </div>
      </details>

      {componentsQuery.isLoading && (
        <p data-testid={selectors.components.loading} className="mt-3 text-app-foreground">
          {t(strings.components.loading)}
        </p>
      )}
      {componentsQuery.error && (
        <p data-testid={selectors.components.error} className="mt-3 text-app-danger">
          {errorMessage(componentsQuery.error, t)}
        </p>
      )}
      {indexMutation.error && (
        <p data-testid={selectors.components.error} className="mt-3 text-app-danger">
          {errorMessage(indexMutation.error, t)}
        </p>
      )}

      {!componentsQuery.isLoading && components.length === 0 && (
        <div data-testid={selectors.components.empty} className="mt-3">
          <EmptyState
            title={t(strings.components.empty)}
            action={(
              <div className="flex flex-wrap gap-2">
                <Button onClick={() => setShowCreate(true)}>
                  {t(strings.components.create.action)}
                </Button>
                <Button
                  onClick={() => indexMutation.mutate()}
                  disabled={indexMutation.isPending}
                  variant="secondary"
                >
                  {t(strings.components.indexAction)}
                </Button>
              </div>
            )}
          />
        </div>
      )}

      {components.length > 0 && (
        <>
          <p data-testid={selectors.components.summary} className="mt-3 text-xs text-app-muted-foreground">
            {t(strings.components.summary, { count: components.length })}
          </p>
          <div data-testid={selectors.components.list} className="mt-2">
            <DataTable
              rows={components}
              columns={columns}
              getRowKey={(component) => component.id}
              caption={t(strings.components.title)}
              searchLabel={t(strings.components.searchLabel)}
              searchPlaceholder={t(strings.components.searchPlaceholder)}
              emptyMessage={t(strings.components.empty)}
            />
          </div>
        </>
      )}
    </section>
  );
}

function formatAffinity(affinity: number) {
  switch (affinity) {
    case DESIGN_AFFINITY_NATIVE:
      return "native";
    case DESIGN_AFFINITY_COMPATIBLE:
      return "compatible";
    case DESIGN_AFFINITY_DISCOURAGED:
      return "discouraged";
    default:
      return "unspecified";
  }
}
