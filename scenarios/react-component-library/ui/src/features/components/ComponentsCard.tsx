import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { componentsClient } from "../../api/components";
import { errorMessage } from "../../lib/errorMessage";
import { ComponentEditor } from "./ComponentEditor";
import { CreateComponentDialog } from "./CreateComponentDialog";

const DESIGN_AFFINITY_NATIVE = 1;
const DESIGN_AFFINITY_COMPATIBLE = 2;
const DESIGN_AFFINITY_DISCOURAGED = 3;

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
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedLibraryId, setSelectedLibraryId] = useState<string>("");
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

  if (selectedId) {
    return (
      <ComponentEditor
        id={selectedId}
        libraryId={selectedLibraryId}
        onClose={() => {
          setSelectedId(null);
          setSelectedLibraryId("");
        }}
      />
    );
  }

  return (
    <section
      data-testid={selectors.components.card}
      aria-label={t(strings.components.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-sm font-medium text-slate-400">{t(strings.components.title)}</h2>
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
            variant="outline"
          >
            {t(strings.components.indexAction)}
          </Button>
        </div>
      </div>

      {showCreate && <CreateComponentDialog onClose={() => setShowCreate(false)} />}

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <label className="block text-xs text-slate-400">
          {t(strings.components.searchLabel)}
          <Input
            data-testid={selectors.components.searchInput}
            value={match}
            onChange={(e) => setMatch(e.target.value)}
            placeholder={t(strings.components.searchPlaceholder)}
            className="mt-1"
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.components.tagLabel)}
          <Input
            data-testid={selectors.components.tagInput}
            value={tag}
            onChange={(e) => setTag(e.target.value)}
            placeholder={t(strings.components.tagPlaceholder)}
            className="mt-1"
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.components.tagsFilterLabel)}
          <Input
            data-testid={selectors.components.tagsInput}
            value={tagsRaw}
            onChange={(e) => setTagsRaw(e.target.value)}
            placeholder={t(strings.components.tagsFilterPlaceholder)}
            className="mt-1"
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.components.categoryLabel)}
          <Input
            data-testid={selectors.components.categoryInput}
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            placeholder={t(strings.components.categoryPlaceholder)}
            className="mt-1"
          />
        </label>
        <label className="block text-xs text-slate-400">
          {t(strings.components.styleLabel)}
          <Input
            data-testid={selectors.components.styleInput}
            value={styleId}
            onChange={(e) => setStyleId(e.target.value)}
            placeholder={t(strings.components.stylePlaceholder)}
            className="mt-1"
          />
        </label>
        <label className="block text-xs text-slate-400">
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

      {componentsQuery.isLoading && (
        <p data-testid={selectors.components.loading} className="mt-3 text-slate-200">
          {t(strings.components.loading)}
        </p>
      )}
      {componentsQuery.error && (
        <p data-testid={selectors.components.error} className="mt-3 text-red-400">
          {errorMessage(componentsQuery.error, t)}
        </p>
      )}
      {indexMutation.error && (
        <p data-testid={selectors.components.error} className="mt-3 text-red-400">
          {errorMessage(indexMutation.error, t)}
        </p>
      )}

      {!componentsQuery.isLoading && components.length === 0 && (
        <div data-testid={selectors.components.empty} className="mt-3 rounded-lg border border-dashed border-white/15 p-4 text-slate-200">
          <p>{t(strings.components.empty)}</p>
          <div className="mt-3 flex flex-wrap gap-2">
            <Button onClick={() => setShowCreate(true)}>
              {t(strings.components.create.action)}
            </Button>
            <Button
              onClick={() => indexMutation.mutate()}
              disabled={indexMutation.isPending}
              variant="outline"
            >
              {t(strings.components.indexAction)}
            </Button>
          </div>
        </div>
      )}

      {components.length > 0 && (
        <>
          <p data-testid={selectors.components.summary} className="mt-3 text-xs text-slate-400">
            {t(strings.components.summary, { count: components.length })}
          </p>
          <ul data-testid={selectors.components.list} className="mt-2 space-y-2 text-sm text-slate-200">
            {components.map((c) => {
              const designStyles = c.designStyles ?? [];
              const componentTags = c.tags ?? [];
              return (
                <li
                  key={c.id}
                  data-testid={selectors.components.item}
                  className="rounded-lg border border-white/10 p-3"
                >
                  <div className="flex items-baseline justify-between gap-3">
                    <span
                      data-testid={selectors.components.itemLibraryId}
                      className="font-mono text-xs text-slate-400"
                    >
                      {c.libraryId}
                    </span>
                    {c.version && (
                      <span
                        data-testid={selectors.components.itemVersion}
                        className="text-xs text-slate-500"
                      >
                        {t(strings.components.versionLabel, { version: c.version })}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 flex items-center justify-between gap-2">
                    <div
                      data-testid={selectors.components.itemDisplayName}
                      className="font-medium"
                    >
                      {c.displayName}
                    </div>
                    <Button
                      data-testid={selectors.components.itemEditButton}
                      onClick={() => {
                        setSelectedId(c.id);
                        setSelectedLibraryId(c.libraryId);
                      }}
                      className="h-7 px-3 text-xs"
                    >
                      {t(strings.components.editAction)}
                    </Button>
                    <Button
                      asChild
                      className="h-7 px-3 text-xs"
                      variant="outline"
                    >
                      <Link to={`/components/${c.id}`}>{t(strings.components.openAction)}</Link>
                    </Button>
                  </div>
                  {c.description && (
                    <div className="mt-1 text-xs text-slate-400">{c.description}</div>
                  )}
                  {c.slot && (
                    <div
                      data-testid={selectors.components.itemSlot}
                      className="mt-1 font-mono text-xs text-slate-500"
                    >
                      slot={c.slot}
                    </div>
                  )}
                  {designStyles.length > 0 && (
                    <div
                      data-testid={selectors.components.itemDesignStyles}
                      className="mt-2 flex flex-wrap gap-1 text-xs"
                    >
                      {designStyles.map((style) => (
                        <span
                          key={style.styleId}
                          className={
                            style.affinity === DESIGN_AFFINITY_DISCOURAGED
                              ? "rounded border border-amber-400/50 px-2 py-0.5 text-amber-200"
                              : "rounded border border-white/10 px-2 py-0.5 text-slate-300"
                          }
                        >
                          {style.styleId}:{formatAffinity(style.affinity)}
                        </span>
                      ))}
                    </div>
                  )}
                  <div
                    data-testid={selectors.components.itemTags}
                    className="mt-1 text-xs text-slate-500"
                  >
                    {componentTags.length > 0
                      ? t(strings.components.tagsLabel, { tags: componentTags.join(", ") })
                      : t(strings.components.noTags)}
                  </div>
                </li>
              );
            })}
          </ul>
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
