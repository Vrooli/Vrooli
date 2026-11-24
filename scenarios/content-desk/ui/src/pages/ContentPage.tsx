import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";

import type { Draft } from "@vrooli/proto-types/content-desk/v1/artifacts/artifacts_pb";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";

import { artifactsClient } from "../api/artifacts";
import { projectsClient } from "../api/projects";
import { relationshipsClient } from "../api/relationships";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { DraftAssetAttachments } from "./DraftAssetAttachments";

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
  const [collections, setCollections] = useState<Array<{ id: string; projectId: string; name: string }>>([]);
  const [collectionItems, setCollectionItems] = useState<Array<{ collectionId: string; draftId: string; revisionId: string }>>([]);
  const [collectionName, setCollectionName] = useState("");
  const [collectionProject, setCollectionProject] = useState("");
  const [selectedCollections, setSelectedCollections] = useState<Record<string, string>>({});
  const [collectionMessage, setCollectionMessage] = useState("");
  const [selectedRevision, setSelectedRevision] = useState<{ draftId: string; revisionId: string; body: string } | null>(null);
  const [collectionBusy, setCollectionBusy] = useState(false);
  const [projectError, setProjectError] = useState(false);
  const [collectionsError, setCollectionsError] = useState(false);
  const search = searchParams.get("q") ?? "";
  const requestedStatus = searchParams.get("status") as StatusFilter | null;
  const status = requestedStatus && STATUS_OPTIONS.includes(requestedStatus) ? requestedStatus : "all";
  const projectFilter = searchParams.get("project") ?? "all";
  const collectionFilter = searchParams.get("collection") ?? "all";
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
    void projectsClient.listProjects({}).then(async (response) => {
      const loadedProjects = response.projects.map(({ id, name }) => ({ id, name }));
      setProjects(loadedProjects);
      setCollectionProject((current) => current || loadedProjects[0]?.id || "");
      try {
        const grouped = await Promise.all(loadedProjects.map(async (project) => {
          const result = await relationshipsClient.listCollections({ projectId: project.id });
          return result.collections;
        }));
        const loadedCollections = grouped.flatMap((items) => items.map(({ id, projectId, name }) => ({ id, projectId, name })));
        setCollections(loadedCollections);
        const memberships = await Promise.all(loadedCollections.map(async (collection) => {
          const result = await relationshipsClient.listCollectionItems({ collectionId: collection.id });
          return result.items;
        }));
        setCollectionItems(memberships.flatMap((items) => items.map(({ collectionId, draftId, revisionId }) => ({ collectionId, draftId, revisionId }))));
      } catch {
        setCollectionsError(true);
      }
    }).catch(() => setProjectError(true));
  }, []);

  async function createCollection() {
    if (!collectionProject || !collectionName.trim()) return;
    setCollectionBusy(true);
    setCollectionMessage("");
    try {
      const response = await relationshipsClient.createCollection({ projectId: collectionProject, name: collectionName.trim() });
      const created = response.collection;
      if (created) setCollections((current) => [...current, { id: created.id, projectId: created.projectId, name: created.name }]);
      setCollectionName("");
      setCollectionMessage(t(strings.pages.content.collectionCreated));
    } catch (caught) {
      setCollectionMessage(caught instanceof Error ? caught.message : t(strings.pages.content.collectionError));
    } finally {
      setCollectionBusy(false);
    }
  }

  async function pinDraft(draft: Draft) {
    const collectionId = selectedCollections[draft.id];
    if (!collectionId) return;
    setCollectionBusy(true);
    setCollectionMessage("");
    try {
      const current = await artifactsClient.getDraftCurrentRevision({ id: draft.id });
      const revision = current.currentRevision;
      if (!revision || revision.body !== draft.body) throw new Error(t(strings.pages.content.staleRevision));
      if (collectionItems.some((item) => item.collectionId === collectionId && item.draftId === draft.id && item.revisionId === revision.revisionId)) {
        setCollectionMessage(t(strings.pages.content.revisionAlreadyPinned));
        return;
      }
      await relationshipsClient.addCollectionItem({ item: { collectionId, draftId: draft.id, revisionId: revision.revisionId } });
      setCollectionItems((items) => items.some((item) => item.collectionId === collectionId && item.draftId === draft.id && item.revisionId === revision.revisionId)
        ? items
        : [...items, { collectionId, draftId: draft.id, revisionId: revision.revisionId }]);
      setCollectionMessage(t(strings.pages.content.revisionPinned));
    } catch (caught) {
      setCollectionMessage(caught instanceof Error ? caught.message : t(strings.pages.content.collectionError));
    } finally {
      setCollectionBusy(false);
    }
  }

  async function openSavedRevision(draftId: string, revisionId: string) {
    setCollectionMessage("");
    try {
      const response = await artifactsClient.getDraftRevision({ draftId, revisionId });
      if (!response.revision) throw new Error(t(strings.pages.content.collectionError));
      setSelectedRevision({ draftId: response.revision.draftId, revisionId: response.revision.revisionId, body: response.revision.body });
    } catch (caught) {
      setCollectionMessage(caught instanceof Error ? caught.message : t(strings.pages.content.collectionError));
    }
  }

  async function removeSavedRevision(item: { collectionId: string; draftId: string; revisionId: string }) {
    setCollectionBusy(true);
    setCollectionMessage("");
    try {
      await relationshipsClient.removeCollectionItem({ item });
      setCollectionItems((items) => items.filter((saved) => !(saved.collectionId === item.collectionId && saved.draftId === item.draftId && saved.revisionId === item.revisionId)));
      setCollectionMessage(t(strings.pages.content.revisionRemoved));
    } catch (caught) {
      setCollectionMessage(caught instanceof Error ? caught.message : t(strings.pages.content.collectionError));
    } finally {
      setCollectionBusy(false);
    }
  }

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
      && (collectionFilter === "all" || collectionItems.some((item) => item.collectionId === collectionFilter && item.draftId === draft.id))
      && (!query || searchableText(draft).includes(query)));
  }, [collectionFilter, collectionItems, drafts, projectFilter, search, status]);

  const projectOptions = useMemo(() => {
    const names = new Map(projects.map((project) => [project.id, project.name || project.id]));
    return [...new Set(drafts.map((draft) => draft.projectId).filter((id): id is string => !!id && id !== "unassigned"))]
      .map((id) => ({ id, name: names.get(id) ?? id }));
  }, [drafts, projects]);

  return <section className="mx-auto w-full max-w-7xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="content" data-testid="page-content">
    <header>
      <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.pages.content.eyebrow)}</p>
      <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.pages.content.title)}</h1>
      <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.content.description)}</p>
    </header>
    <Card>
      <CardHeader><CardTitle as="h2">{t(strings.pages.content.collectionTitle)}</CardTitle></CardHeader>
      <CardContent className="space-y-4">
        <section aria-labelledby="saved-collections-title" className="rounded-control border border-app-border p-3">
          <h3 id="saved-collections-title" className="font-semibold">{t(strings.pages.content.savedCollections)}</h3>
          <p className="mt-1 text-sm text-app-muted-foreground">{t(strings.pages.content.collectionsDescription)}</p>
          <div className="mt-3 grid gap-2 sm:grid-cols-[minmax(10rem,1fr)_minmax(10rem,1fr)_auto]">
            <label className="grid gap-1 text-sm"><span>{t(strings.pages.content.collectionProject)}</span><Select aria-label={t(strings.pages.content.collectionProject)} value={collectionProject} onChange={(event) => setCollectionProject(event.target.value)} options={[{ value: "", label: t(strings.pages.content.chooseProject) }, ...projects.map((project) => ({ value: project.id, label: project.name || project.id }))]} /></label>
            <label className="grid gap-1 text-sm"><span>{t(strings.pages.content.collectionName)}</span><Input value={collectionName} onChange={(event) => setCollectionName(event.target.value)} /></label>
            <Button className="self-end" disabled={!collectionProject || !collectionName.trim() || collectionBusy} onClick={() => void createCollection()}>{t(strings.pages.content.createCollection)}</Button>
          </div>
          {projectError ? <p role="status" className="mt-2 text-xs text-app-muted-foreground">{t(strings.pages.content.projectScopeUnavailable)}</p> : null}
          {collectionsError ? <p role="status" className="mt-2 text-xs text-app-muted-foreground">{t(strings.pages.content.collectionsUnavailable)}</p> : null}
          {collectionMessage ? <p role="status" aria-live="polite" className="mt-2 text-sm text-app-muted-foreground">{collectionMessage}</p> : null}
          <ul className="mt-3 flex flex-wrap gap-2">{collections.map((collection) => <li key={collection.id} className="rounded-full border border-app-border px-3 py-1 text-xs">{collection.name} · {collectionItems.filter((item) => item.collectionId === collection.id).length}</li>)}</ul>
        </section>
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
            <Select aria-label={t(strings.pages.content.projectLabel)} value={projectFilter} onChange={(event) => changeParam("project", event.target.value, "all")} options={[{ value: "all", label: t(strings.pages.content.allProjects) }, { value: "unassigned", label: t(strings.pages.content.unassigned) }, ...projectOptions.map((project) => ({ value: project.id, label: project.name }))]} />
            {projectError ? <span role="status" className="text-xs font-normal text-app-muted-foreground">{t(strings.pages.content.projectScopeUnavailable)}</span> : null}
          </label>
          <label className="grid gap-1 text-sm font-medium">
            <span>{t(strings.pages.content.collectionFilter)}</span>
            <Select aria-label={t(strings.pages.content.collectionFilter)} value={collectionFilter} onChange={(event) => changeParam("collection", event.target.value, "all")} options={[{ value: "all", label: t(strings.pages.content.allCollections) }, ...collections.map((collection) => ({ value: collection.id, label: collection.name }))]} />
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
              <DraftAssetAttachments draft={draft} />
              {draft.projectId && draft.projectId !== "unassigned" ? <div className="mt-3 flex flex-wrap items-end gap-2">
                <label className="grid min-w-48 gap-1 text-xs"><span>{t(strings.pages.content.pinRevision)}</span><Select aria-label={t(strings.pages.content.pinRevisionFor, { id: draft.id })} value={selectedCollections[draft.id] ?? ""} onChange={(event) => setSelectedCollections((current) => ({ ...current, [draft.id]: event.target.value }))} options={[{ value: "", label: t(strings.pages.content.chooseCollection) }, ...collections.filter((collection) => collection.projectId === draft.projectId).map((collection) => ({ value: collection.id, label: collection.name }))]} /></label>
                <Button size="sm" variant="secondary" disabled={!selectedCollections[draft.id] || collectionBusy} onClick={() => void pinDraft(draft)}>{t(strings.pages.content.pinCurrentRevision)}</Button>
                <ul className="w-full flex flex-wrap gap-2">{collectionItems.filter((item) => item.draftId === draft.id).map((item) => <li key={`${item.collectionId}:${item.revisionId}`} className="flex items-center gap-1"><Button size="sm" variant="secondary" onClick={() => void openSavedRevision(item.draftId, item.revisionId)}>{t(strings.pages.content.viewSavedRevision, { collection: collections.find((collection) => collection.id === item.collectionId)?.name ?? item.collectionId, revision: item.revisionId })}</Button><Button size="sm" variant="secondary" disabled={collectionBusy} onClick={() => void removeSavedRevision(item)}>{t(strings.pages.content.removeSavedRevision)}</Button></li>)}</ul>
                {selectedRevision?.draftId === draft.id ? <section aria-label={t(strings.pages.content.savedRevisionTitle)} className="w-full rounded-control border border-app-border bg-app-background p-3"><h4 className="text-sm font-semibold">{t(strings.pages.content.savedRevisionTitle)}</h4><p className="mt-1 text-xs text-app-muted-foreground">{selectedRevision.revisionId} · {t(strings.pages.content.savedRevisionLimit)}</p><p className="mt-2 whitespace-pre-wrap break-words text-sm">{selectedRevision.body || t(strings.pages.content.emptyBody)}</p></section> : null}
              </div> : null}
            </article>
          </li>)}
        </ul> : null}
        <p className="text-xs text-app-muted-foreground">{t(strings.pages.content.localNote)}</p>
      </CardContent>
    </Card>
  </section>;
}
