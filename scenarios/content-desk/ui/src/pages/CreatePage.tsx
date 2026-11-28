import { useEffect, useRef, useState, type FormEvent } from "react";
import { Link } from "react-router-dom";

import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Select } from "@vrooli/react-component-library/Select/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";

import { artifactsClient } from "../api/artifacts";
import { posttypesClient } from "../api/posttypes";
import { projectsClient } from "../api/projects";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

type PostType = { id: string; status: string };
type Project = { id: string; name: string };
const CREATE_OPERATION_PREFIX = "content-desk.create-operation.v1:";

async function createOperationStorageKey(scope: string): Promise<string> {
  try {
    const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(scope));
    const hex = Array.from(new Uint8Array(digest), (byte) => byte.toString(16).padStart(2, "0")).join("");
    return CREATE_OPERATION_PREFIX + hex;
  } catch {
    return "";
  }
}

function readSessionOperation(key: string): string {
  if (!key) return "";
  try { return sessionStorage.getItem(key) ?? ""; } catch { return ""; }
}

function writeSessionOperation(key: string, operationID: string): void {
  if (!key) return;
  try { sessionStorage.setItem(key, operationID); } catch { /* In-memory retry remains available when session storage is disabled. */ }
}

function clearSessionOperation(key: string): void {
  if (!key) return;
  try { sessionStorage.removeItem(key); } catch { /* Successful response still clears the active-page key. */ }
}

export function CreatePage() {
  const { t } = useTranslation();
  const [postTypes, setPostTypes] = useState<PostType[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [postTypeID, setPostTypeID] = useState("");
  const [projectID, setProjectID] = useState("unassigned");
  const [projectName, setProjectName] = useState("");
  const [projectState, setProjectState] = useState<"loading" | "ready" | "saving" | "error">("loading");
  const [projectMessage, setProjectMessage] = useState("");
  const [body, setBody] = useState("");
  const [state, setState] = useState<"loading" | "ready" | "saving" | "saved" | "error">("loading");
  const [message, setMessage] = useState("");
  const [savedDraftID, setSavedDraftID] = useState("");
  const createOperations = useRef(new Map<string, string>());

  useEffect(() => {
    void posttypesClient.listPostTypes({}).then((response) => {
      const active = response.postTypes.filter((item) => item.status === "active");
      setPostTypes(active);
      setPostTypeID(active[0]?.id ?? "");
      setState("ready");
    }).catch((caught: unknown) => {
      setMessage(caught instanceof Error ? caught.message : t(strings.pages.create.loadError));
      setState("error");
    });
  }, [t]);

  useEffect(() => {
    void projectsClient.listProjects({}).then((response) => {
      setProjects(response.projects);
      setProjectState("ready");
    }).catch(() => {
      setProjectMessage(t(strings.pages.create.projectLoadError));
      setProjectState("error");
    });
  }, [t]);

  async function createProject() {
    const name = projectName.trim();
    if (!name) return;
    setProjectState("saving");
    try {
      const response = await projectsClient.createProject({ name });
      const project = response.project;
      if (!project) throw new Error(t(strings.pages.create.projectCreateError));
      setProjects((current) => [...current, project]);
      setProjectID(project.id);
      setProjectName("");
      setProjectMessage("");
      setProjectState("ready");
    } catch (caught) {
      setProjectMessage(caught instanceof Error ? caught.message : t(strings.pages.create.projectCreateError));
      setProjectState("error");
    }
  }

  async function saveDraft(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!postTypeID) return;
    setState("saving");
    try {
      const projectId = projectID === "unassigned" ? "" : projectID;
      const scope = JSON.stringify({ campaignId: "", projectId, postTypeId: postTypeID, body, channel: "x-twitter", format: "post" });
      const storageKey = await createOperationStorageKey(scope);
      let idempotencyKey = createOperations.current.get(scope);
      if (!idempotencyKey) {
        idempotencyKey = readSessionOperation(storageKey) || crypto.randomUUID();
        createOperations.current.set(scope, idempotencyKey);
        writeSessionOperation(storageKey, idempotencyKey);
      }
      const response = await artifactsClient.createDraft({
        campaignId: "",
        projectId,
        postTypeId: postTypeID,
        body,
        channel: "x-twitter",
        format: "post",
        idempotencyKey,
      });
      setSavedDraftID(response.draft?.id ?? "");
      setMessage(t(strings.pages.create.savedDetail, { id: response.draft?.id ?? "" }));
      if (response.draft?.id) {
        createOperations.current.delete(scope);
        clearSessionOperation(storageKey);
      }
      setState("saved");
    } catch (caught) {
      setMessage(caught instanceof Error ? caught.message : t(strings.pages.create.saveError));
      setState("error");
    }
  }

  return <section className="mx-auto w-full max-w-3xl space-y-6 px-4 py-6 sm:px-6 lg:px-8" data-experience-surface="create" data-testid="page-create">
    <header>
      <p className="text-sm font-semibold uppercase tracking-[0.1em] text-app-primary">{t(strings.pages.create.eyebrow)}</p>
      <h1 className="mt-1 font-serif text-4xl font-medium tracking-tight">{t(strings.pages.create.title)}</h1>
      <p className="mt-2 max-w-2xl text-sm text-app-muted-foreground">{t(strings.pages.create.description)}</p>
      <Link className="mt-3 inline-flex min-h-11 items-center rounded-control border border-app-border px-3 text-sm font-medium hover:bg-app-muted" to="/intake">{t(strings.pages.intake.title)}</Link>
    </header>
    <Card>
      <CardHeader><CardTitle as="h2">{t(strings.pages.create.formTitle)}</CardTitle></CardHeader>
      <CardContent>
        {state === "loading" ? <p aria-live="polite">{t(strings.pages.create.loading)}</p> : null}
        {state === "error" && !postTypes.length ? <p role="alert" className="text-sm text-app-danger">{message}</p> : null}
        {postTypes.length === 0 && state !== "loading" && state !== "error" ? <p>{t(strings.pages.create.noTypes)}</p> : null}
        {postTypes.length > 0 ? <form className="space-y-4" onSubmit={(event) => void saveDraft(event)}>
          <label className="block space-y-1 text-sm font-medium">
            <span>{t(strings.pages.create.postType)}</span>
            <Select value={postTypeID} onChange={(event) => setPostTypeID(event.target.value)} options={postTypes.map((item) => ({ value: item.id, label: item.id }))} required />
          </label>
          <label className="block space-y-1 text-sm font-medium">
            <span>{t(strings.pages.create.project)}</span>
            <Select value={projectID} onChange={(event) => setProjectID(event.target.value)} options={[{ value: "unassigned", label: t(strings.pages.create.unassigned) }, ...projects.map((item) => ({ value: item.id, label: item.name }))]} />
          </label>
          {projectState === "error" ? <p role="status" className="text-sm text-app-muted-foreground">{projectMessage}</p> : null}
          <details className="rounded-control border border-app-border p-3">
            <summary className="cursor-pointer text-sm font-medium">{t(strings.pages.create.newProject)}</summary>
            <div className="mt-3 flex flex-wrap gap-2">
              <label className="sr-only" htmlFor="create-project-name">{t(strings.pages.create.newProjectName)}</label>
              <Input id="create-project-name" className="min-w-0 flex-1" value={projectName} onChange={(event) => setProjectName(event.target.value)} placeholder={t(strings.pages.create.newProjectName)} />
              <Button type="button" variant="secondary" onClick={() => void createProject()} disabled={projectState === "saving" || !projectName.trim()}>{projectState === "saving" ? t(strings.pages.create.creatingProject) : t(strings.pages.create.createProject)}</Button>
            </div>
            {projectState === "error" && projects.length > 0 ? <p role="alert" className="mt-2 text-sm text-app-danger">{projectMessage}</p> : null}
          </details>
          <p className="rounded-control border border-app-border bg-app-surface p-3 text-sm text-app-muted-foreground">{t(strings.pages.create.scopeNote)}</p>
          <label className="block space-y-1 text-sm font-medium">
            <span>{t(strings.pages.create.body)}</span>
            <Textarea value={body} onChange={(event) => setBody(event.target.value)} rows={8} className="font-normal" required />
          </label>
          <div className="flex flex-wrap items-center gap-3">
            <Button type="submit" disabled={state === "saving" || state === "loading" || body.trim().length === 0}>{state === "saving" ? t(strings.pages.create.saving) : t(strings.pages.create.save)}</Button>
            <span className="text-sm text-app-muted-foreground">{t(strings.pages.create.draftOnly)}</span>
          </div>
          {state === "saved" ? <div role="status" className="space-y-2 text-sm text-app-primary"><p>{message}</p>{savedDraftID ? <Link className="inline-flex min-h-11 items-center rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground hover:bg-app-muted" to={`/review?draft=${encodeURIComponent(savedDraftID)}`}>{t(strings.pages.create.reviewDraft)}</Link> : null}</div> : null}
          {state === "error" ? <p role="alert" className="text-sm text-app-danger">{message}</p> : null}
        </form> : null}
      </CardContent>
    </Card>
  </section>;
}
