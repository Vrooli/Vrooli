import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { componentsClient } from "../../api/components";
import { errorMessage } from "../../lib/errorMessage";
import { ComponentEditor } from "./ComponentEditor";

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
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [selectedLibraryId, setSelectedLibraryId] = useState<string>("");

  const tags = tagsRaw
    .split(",")
    .map((t) => t.trim())
    .filter((t) => t !== "");

  const componentsQuery = useQuery({
    queryKey: ["components", { match, tag, tags, category }],
    queryFn: () =>
      componentsClient.listComponents({ match, tag, tags, category }),
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
        <Button
          data-testid={selectors.components.indexButton}
          onClick={() => indexMutation.mutate()}
          disabled={indexMutation.isPending}
        >
          {t(strings.components.indexAction)}
        </Button>
      </div>

      <div className="mt-3 grid grid-cols-1 gap-2 sm:grid-cols-2">
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

      {componentsQuery.data && components.length === 0 && !componentsQuery.isLoading && (
        <p data-testid={selectors.components.empty} className="mt-3 text-slate-200">
          {t(strings.components.empty)}
        </p>
      )}

      {components.length > 0 && (
        <>
          <p data-testid={selectors.components.summary} className="mt-3 text-xs text-slate-400">
            {t(strings.components.summary, { count: components.length })}
          </p>
          <ul data-testid={selectors.components.list} className="mt-2 space-y-2 text-sm text-slate-200">
            {components.map((c) => (
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
                </div>
                {c.description && (
                  <div className="mt-1 text-xs text-slate-400">{c.description}</div>
                )}
                <div
                  data-testid={selectors.components.itemTags}
                  className="mt-1 text-xs text-slate-500"
                >
                  {c.tags.length > 0
                    ? t(strings.components.tagsLabel, { tags: c.tags.join(", ") })
                    : t(strings.components.noTags)}
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
}
