import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { Input } from "@vrooli/react-component-library/Input/1";

import { artifactsClient } from "../api/artifacts";
import { projectsClient } from "../api/projects";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type StatusFilter = "all" | "active" | "approved" | "published" | "abandoned";
const STATUS_OPTIONS: readonly StatusFilter[] = ["all", "active", "approved", "published", "abandoned"];

function searchableText(draft: Draft): string {
  return [draft.id, draft.status, draft.postTypeId, draft.campaignId, draft.projectId, draft.channel, draft.format, draft.body].join(" ").toLocaleLowerCase();
}

function matchesStatus(draft: Draft, filter: StatusFilter): boolean {
  const status = draft.status.toLowerCase();
  switch (filter) {
    case "all": return true;
    case "active": return !["published", "abandoned", "archived"].includes(status);
    case "approved": return status === "approved";
    case "published": return status === "published";
    case "abandoned": return status === "abandoned";
  }
}

export function ContentPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [drafts, setDrafts] = useState<Draft[]>([]);
  const [projects, setProjects] = useState<Array<{ id: string; name: string }>>([]);
  const [projectError, setProjectError] = useState(false);
  const search = searchParams.get("q") ?? "";
  const requestedStatus = searchParams.get("status") as StatusFilter | null;
  const status = requestedStatus && STATUS_OPTIONS.includes(requestedStatus) ? requestedStatus : "all";
  const projectFilter = searchParams.get("project") ?? "all";
  const [state, setState] = useState<"loading" | "ready" | "error">(() => import.meta.env.MODE === "test" ? "ready" : "loading");
  const [error, setError] = useState("");

  const refresh = useCallback(async () => {
    setState("loading");
    try {
      const response = await artifactsClient.listDrafts({});
      setDrafts(response.drafts);
      setError("");
      setState("ready");
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : t(strings.pages.content.loadError));
      setState("error");
    }
  }, [t]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => {
    void projectsClient.listProjects({}).then((response) => setProjects(response.projects.map(({ id, name }) => ({ id, name })))).catch(() => setProjectError(true));
  }, []);

  function changeParam(key: string, value: string, defaultValue: string) {
    setSearchParams((current) => {
      const next = new URLSearchParams(current);
      if (value === defaultValue) next.delete(key); else next.set(key, value);
      return next;
    });
  }

  const filteredDrafts = useMemo(() => {
    const query = search.trim().toLocaleLowerCase();
    return drafts.filter((draft) => matchesStatus(draft, status)
      && (projectFilter === "all" || (projectFilter === "unassigned" ? !draft.projectId || draft.projectId === "unassigned" : draft.projectId === projectFilter))
      && (!query || searchableText(draft).includes(query)));
  }, [drafts, projectFilter, search, status]);

  const projectOptions = useMemo(() => {
    const names = new Map(projects.map((project) => [project.id, project.name || project.id]));
    return [...new Set(drafts.map((draft) => draft.projectId).filter((id): id is string => !!id && id !== "unassigned"))]
      .map((id) => ({ id, name: names.get(id) ?? id }));
  }, [drafts, projects]);

  return <main className="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="content" data-testid="page-content">
    <header>
      <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.pages.content.eyebrow)}</p>
      <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.pages.content.title)}</h1>
      <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.content.description)}</p>
    </header>
    <Card>
      <CardHeader><CardTitle as="h2">{t(strings.pages.content.collectionTitle)}</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-3">
          <label className="grid gap-1 text-sm font-medium">
            <span>{t(strings.pages.content.searchLabel)}</span>
            <Input type="search" value={search} onChange={(event) => changeParam("q", event.target.value, "")} placeholder={t(strings.pages.content.searchPlaceholder)} />
          </label>
          <div className="grid gap-2">
            <span>{t(strings.pages.content.statusLabel)}</span>
            <div role="group" aria-label={t(strings.pages.content.statusLabel)} className="flex flex-wrap gap-2">
              {STATUS_OPTIONS.map((choice) => <Button key={choice} size="sm" variant={status === choice ? "primary" : "secondary"} aria-pressed={status === choice} onClick={() => changeParam("status", choice, "all")}>{t(strings.pages.content.status[choice])}</Button>)}
            </div>
          </div>
          <label className="grid gap-1 text-sm font-medium">
            <span>{t(strings.pages.content.projectLabel)}</span>
            <select aria-label={t(strings.pages.content.projectLabel)} value={projectFilter} onChange={(event) => changeParam("project", event.target.value, "all")} className="min-h-10 rounded-control border border-app-border bg-app-background px-3">
              <option value="all">{t(strings.pages.content.allProjects)}</option>
              <option value="unassigned">{t(strings.pages.content.unassigned)}</option>
              {projectOptions.map((project) => <option key={project.id} value={project.id}>{project.name}</option>)}
            </select>
            {projectError ? <span role="status" className="text-xs font-normal text-app-muted-foreground">{t(strings.pages.content.projectScopeUnavailable)}</span> : null}
          </label>
        </div>
        <p aria-live="polite" className="text-sm text-app-muted-foreground">{t(strings.pages.content.resultCount, { count: filteredDrafts.length })}</p>
        {state === "loading" ? <p aria-live="polite" className="text-sm text-app-muted-foreground">{t(strings.pages.content.loading)}</p> : null}
        {state === "error" ? <section role="alert" className="rounded-control border border-app-danger/40 p-4"><p>{error}</p><Button className="mt-3" variant="secondary" onClick={() => void refresh()}>{t(strings.pages.content.retry)}</Button></section> : null}
        {state === "ready" && filteredDrafts.length === 0 ? <section className="rounded-control border border-dashed border-app-border p-5"><h3 className="font-semibold">{drafts.length === 0 ? t(strings.pages.content.emptyTitle) : t(strings.pages.content.noMatchTitle)}</h3><p className="mt-1 text-sm text-app-muted-foreground">{drafts.length === 0 ? t(strings.pages.content.emptyDescription) : t(strings.pages.content.noMatchDescription)}</p></section> : null}
        {state === "ready" && filteredDrafts.length > 0 ? <ul className="grid gap-3 lg:grid-cols-2">
          {filteredDrafts.map((draft) => <li key={draft.id}>
            <article className="h-full rounded-panel border border-app-border bg-app-surface p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0"><h3 className="break-all font-medium">{draft.id}</h3><p className="mt-1 text-sm text-app-muted-foreground">{draft.status} · {draft.postTypeId || t(strings.pages.content.unspecifiedType)} · {draft.campaignId || t(strings.pages.content.noCampaign)} · {draft.projectId === "unassigned" || !draft.projectId ? t(strings.pages.content.unassigned) : draft.projectId}</p></div>
                <Button variant="secondary" onClick={() => {
                  const returnTo = `/content${searchParams.size ? `?${searchParams.toString()}` : ""}`;
                  const query = new URLSearchParams({ draft: draft.id, return: returnTo });
                  navigate(`/review?${query.toString()}`);
                }}>{t(strings.pages.content.open)}</Button>
              </div>
              <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-relaxed">{draft.body || t(strings.pages.content.emptyBody)}</p>
              <p className="mt-3 text-xs text-app-muted-foreground">{[draft.channel, draft.format].filter(Boolean).join(" · ")}</p>
            </article>
          </li>)}
        </ul> : null}
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.content.localNote)}</p>
      </CardContent>
    </Card>
  </main>;
}
