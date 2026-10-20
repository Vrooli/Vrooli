import { useState } from "react";
import { Link } from "react-router-dom";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { projectsClient } from "../../api/projects";

const KEY = ["projects"] as const;
export function ProjectWorkspace() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [name, setName] = useState("");
  const query = useInfiniteQuery({
    queryKey: KEY,
    initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.listProjects({ pageSize: 50, pageToken: pageParam }),
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
  });
  const projects = query.data?.pages.flatMap((page) => page.projects) ?? [];
  const create = useMutation({
    mutationFn: (value: string) => projectsClient.createProject({ name: value }),
    onSuccess: async () => {
      setName("");
      await queryClient.invalidateQueries({ queryKey: KEY });
    },
  });
  const state = query.isLoading ? "loading" : query.error ? "error" : projects.length === 0 ? "empty" : "ready";

  return (
    <ExperienceSurface
      surfaceId="projects"
      state={state}
      statusMessage={query.isLoading ? t(strings.projects.loading) : query.error ? errorMessage(query.error, t) : undefined}
      data-testid={selectors.projects.surface}
    >
      <Card>
        <CardHeader>
          <CardTitle as="h2">{t(strings.projects.title)}</CardTitle>
          <p className="text-sm text-app-muted-foreground">{t(strings.projects.description)}</p>
        </CardHeader>
        <CardContent className="flex flex-col gap-space-md">
          <form className="flex flex-wrap items-end gap-space-sm" onSubmit={(event) => { event.preventDefault(); create.mutate(name); }}>
            <label className="flex min-w-56 flex-1 flex-col gap-1 text-sm font-medium">
              {t(strings.projects.nameLabel)}
              <input
                data-testid={selectors.projects.nameInput}
                className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground"
                value={name}
                onChange={(event) => setName(event.target.value)}
                required
              />
            </label>
            <Button className="min-h-11" data-testid={selectors.projects.createButton} type="submit" disabled={create.isPending || !name.trim()}>
              {t(strings.projects.create)}
            </Button>
          </form>
          {query.isLoading && <p data-testid={selectors.projects.loading} role="status">{t(strings.projects.loading)}</p>}
          {query.error && <p data-testid={selectors.projects.error} role="alert">{errorMessage(query.error, t)}</p>}
          {create.error && <p data-testid={selectors.projects.error} role="alert">{errorMessage(create.error, t)}</p>}
          {query.data && projects.length === 0 && <div data-testid={selectors.projects.empty}><EmptyState title={t(strings.projects.empty)} /></div>}
          {projects.length > 0 && <h3 className="text-base font-semibold">{t(strings.projects.recent)}</h3>}
          {projects.length > 0 && (
            <ul data-testid={selectors.projects.list} className="divide-y divide-app-border">
              {projects.map((project) => (
                <li key={project.id} className="flex items-center justify-between gap-space-sm py-space-sm">
                  <div className="min-w-0 flex-1">
                    <p className="break-words font-medium">{project.name}</p>
                    <p className="text-xs text-app-muted-foreground">{t(strings.projects.revision, { revision: project.revision.toString() })}</p>
                  </div>
                  <Link className="inline-flex min-h-11 shrink-0 items-center rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground" data-testid={selectors.projects.openButton({ id: project.id })} to={`/projects/${encodeURIComponent(project.id)}`}>
                    {t(strings.projects.open)}
                  </Link>
                </li>
              ))}
            </ul>
          )}
          {query.hasNextPage && <Button className="min-h-11 self-start" variant="outline" disabled={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>{t(strings.projects.loadMore)}</Button>}
        </CardContent>
      </Card>
    </ExperienceSurface>
  );
}
