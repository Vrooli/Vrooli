import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ScanLine } from "lucide-react";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Card, CardContent, CardHeader, CardTitle } from "@vrooli/react-component-library/Card/1";
import { ExperienceSurface } from "@vrooli/react-component-library/ExperienceSurface/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { projectConflict, projectsClient } from "../../api/projects";

export function ProjectDetailPage() {
  const { projectId = "" } = useParams();
  return <ProjectDetailEditor key={projectId} projectId={projectId} />;
}

function ProjectDetailEditor({ projectId }: { projectId: string }) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<string>();
  const [baseRevision, setBaseRevision] = useState<bigint>();
  const query = useQuery({ queryKey: ["project", projectId], queryFn: () => projectsClient.getProject({ id: projectId }), enabled: Boolean(projectId) });
  const project = query.data?.project;
  useEffect(() => { if (project && baseRevision === undefined) setBaseRevision(project.revision); }, [project, baseRevision]);
  const rename = useMutation({
    mutationFn: (name: string) => projectsClient.renameProject({ id: projectId, name, expectedRevision: baseRevision ?? 0n }),
    onSuccess: async (response) => {
      setDraft(undefined);
      if (response.project) {
        setBaseRevision(response.project.revision);
        queryClient.setQueryData(["project", projectId], response);
      }
      await queryClient.invalidateQueries({ queryKey: ["project", projectId] });
      await queryClient.invalidateQueries({ queryKey: ["projects"] });
    },
  });
  const conflict = projectConflict(rename.error);
  const status = query.isLoading ? t(strings.project.loading) : query.error ? errorMessage(query.error, t) : undefined;
  return <section data-testid={selectors.pages.project} className="flex flex-col gap-space-md">
    <ExperienceSurface surfaceId="project-detail" state={query.isLoading ? "loading" : query.error ? "error" : "ready"} statusMessage={status}>
      {query.isLoading && <p data-testid={selectors.projects.loading} role="status">{t(strings.project.loading)}</p>}
      {query.error && <p data-testid={selectors.projects.error} role="alert">{errorMessage(query.error, t)}</p>}
      {project && <Card className="studio-detail-card">
        <CardHeader><div className="flex items-start justify-between gap-3"><CardTitle as="h2">{project.name}</CardTitle><Link className="inline-flex min-h-11 shrink-0 items-center rounded-control border border-app-border px-3 text-sm font-medium text-app-foreground" to="/">{t(strings.project.back)}</Link></div><p className="text-sm text-app-muted-foreground">{t(strings.project.revision, { revision: project.revision.toString() })}</p></CardHeader>
        <CardContent className="flex flex-col gap-space-md">
          <div className="studio-detail-preview" role="note"><ScanLine size={19} aria-hidden="true" /><span>{t(strings.projects.noShotPreviewDescription)}</span></div>
          <form className="flex flex-wrap items-end gap-space-sm" onSubmit={(event) => { event.preventDefault(); rename.mutate(draft ?? project.name); }}>
            <label className="flex min-w-56 flex-1 flex-col gap-1 text-sm font-medium">{t(strings.project.nameLabel)}<input data-testid={selectors.projects.renameInput} className="min-h-11 rounded-control border border-app-border bg-app-surface px-3 py-2 text-base text-app-foreground" value={draft ?? project.name} onChange={(event) => setDraft(event.target.value)} disabled={rename.isPending} required /></label>
            <Button className="min-h-11" data-testid={selectors.projects.renameButton} type="submit" disabled={rename.isPending || baseRevision === undefined || !(draft ?? project.name).trim()}>{t(strings.project.rename)}</Button>
          </form>
          {rename.error && (conflict ? <div data-testid={selectors.projects.conflict} role="alert" className="rounded-panel border border-app-warning/40 bg-app-warning/10 p-space-sm text-sm"><p>{t(strings.project.conflict)}</p><Button data-testid={selectors.projects.reloadButton} variant="outline" className="mt-2 min-h-11" onClick={async () => { const result = await query.refetch(); if (result.data?.project) setBaseRevision(result.data.project.revision); }}>{t(strings.project.reload)}</Button></div> : <p data-testid={selectors.projects.error} role="alert">{errorMessage(rename.error, t) || t(strings.project.renameError)}</p>)}
        </CardContent>
      </Card>}
    </ExperienceSurface>
  </section>;
}
