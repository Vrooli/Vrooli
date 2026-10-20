import { useRef, useState } from "react";
import { Link } from "react-router-dom";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";
import { ArrowUpRight, Plus, ScanLine } from "lucide-react";
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
  const [creating, setCreating] = useState(false);
  const newProjectRef = useRef<HTMLButtonElement>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  const query = useInfiniteQuery({
    queryKey: KEY,
    initialPageParam: "",
    queryFn: ({ pageParam }) => projectsClient.listProjects({ pageSize: 50, pageToken: pageParam }),
    getNextPageParam: (lastPage) => lastPage.nextPageToken || undefined,
  });
  const projects = query.data?.pages.flatMap((page) => page.projects) ?? [];
  const featured = projects[0];
  const remaining = projects.slice(1);
  const create = useMutation({
    mutationFn: (value: string) => projectsClient.createProject({ name: value }),
    onSuccess: async () => {
      setName("");
      setCreating(false);
      await queryClient.invalidateQueries({ queryKey: KEY });
    },
  });
  const state = query.isLoading ? "loading" : query.error ? "error" : projects.length === 0 ? "empty" : "ready";

  return (
    <ExperienceSurface surfaceId="projects" state={state} statusMessage={query.isLoading ? t(strings.projects.loading) : query.error ? errorMessage(query.error, t) : undefined} data-testid={selectors.projects.surface}>
      <div className="studio-projects">
        <div className="studio-library__heading">
          <div><span className="studio-eyebrow">{t(strings.projects.workspaceEyebrow).toLocaleUpperCase()}</span><h1 id="saved-projects-heading">{t(strings.projects.workspaceTitle)}</h1></div>
          <Button ref={newProjectRef} className="studio-primary min-h-11" data-testid={selectors.projects.createButton} onClick={() => { create.reset(); setCreating(true); requestAnimationFrame(() => nameRef.current?.focus()); }}><Plus size={17} aria-hidden="true" />{t(strings.projects.newProject)}</Button>
        </div>
        {query.isLoading && <p data-testid={selectors.projects.loading} role="status">{t(strings.projects.loading)}</p>}
        {query.error && <p data-testid={selectors.projects.error} role="alert">{errorMessage(query.error, t)}</p>}

        {featured && <article className="studio-featured" aria-labelledby="featured-project-title">
          <div className="studio-featured__preview" role="img" aria-label={t(strings.projects.noSavedPreview)}><ScanLine size={22} aria-hidden="true" /><span>{t(strings.projects.noSavedPreview)}</span></div>
          <div className="studio-featured__detail"><span className="studio-eyebrow">{t(strings.projects.featuredProject).toLocaleUpperCase()}</span><h3 id="featured-project-title">{featured.name}</h3><p>{t(strings.projects.revision, { revision: featured.revision.toString() })} · {t(strings.projects.noPreview)}</p><Link className="studio-continue" data-testid={selectors.projects.openButton({ id: featured.id })} to={`/projects/${encodeURIComponent(featured.id)}`}>{t(strings.projects.continueEditing)}<ArrowUpRight size={17} aria-hidden="true" /></Link></div>
        </article>}

        {!featured && query.data && <div className="studio-empty-row" data-testid={selectors.projects.empty}><span className="studio-empty-row__icon"><ScanLine size={19} aria-hidden="true" /></span><p>{t(strings.projects.empty)}</p></div>}

        {remaining.length > 0 && <section className="studio-library" aria-labelledby="more-projects-heading"><div className="studio-library__subhead"><h3 id="more-projects-heading">{t(strings.projects.moreProjects)}</h3></div>
          <ul data-testid={selectors.projects.list} className="studio-project-list">{remaining.map((project) => (
            <li key={project.id} className="studio-project-row">
              <span className="studio-project-row__copy"><span className="studio-project-row__name">{project.name}</span><span className="studio-project-row__revision">{t(strings.projects.revision, { revision: project.revision.toString() })} · {t(strings.projects.noPreview)}</span></span>
              <Link className="studio-open-link" data-testid={selectors.projects.openButton({ id: project.id })} to={`/projects/${encodeURIComponent(project.id)}`}>{t(strings.projects.open)}<ArrowUpRight size={17} aria-hidden="true" /></Link>
            </li>
          ))}</ul>
        </section>}
        {query.hasNextPage && <Button className="min-h-11 self-start" variant="outline" disabled={query.isFetchingNextPage} onClick={() => void query.fetchNextPage()}>{t(strings.projects.loadMore)}</Button>}
      </div>
      <Dialog open={creating} title={t(strings.projects.newProject)} description={t(strings.projects.description)} onClose={() => setCreating(false)} closeLabel={t(strings.layout.closeDialog)} returnFocusRef={newProjectRef} initialFocusRef={nameRef}>
        <form className="studio-create-form" onSubmit={(event) => { event.preventDefault(); create.mutate(name.trim()); }}>
          <label className="studio-field" htmlFor="new-project-name">{t(strings.projects.nameLabel)}<input ref={nameRef} id="new-project-name" data-testid={selectors.projects.nameInput} className="min-h-12 rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground" value={name} onChange={(event) => setName(event.target.value)} required /></label>
          {create.error && <p data-testid={selectors.projects.error} role="alert">{errorMessage(create.error, t)}</p>}
          <Button className="studio-primary min-h-11" data-testid="confirm-create-project" type="submit" disabled={create.isPending || !name.trim()}>{t(strings.projects.create)}</Button>
        </form>
      </Dialog>
    </ExperienceSurface>
  );
}
