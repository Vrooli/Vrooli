import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useNavigate } from "react-router-dom";
import { ArrowRight, CircleAlert, Layers, PenTool } from "lucide-react";

import {
  DisplayDevice,
  WorkspaceProvenance,
  createWorkspace,
  forkWorkspace,
  getWorkspace,
  type WorkspaceSummary,
} from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { workspacePath } from "../../routes";
import { formatRelative, timestampToDate } from "../data";
import {
  Banner,
  Chip,
  Empty,
  Panel,
  PanelBody,
  PanelHead,
  Skeleton,
  WbButton,
} from "../primitives";
import { useWorkspaces } from "../workspaces/useWorkspaces";
import { displaySpec, parseOptions, parsePalette } from "../workspaces/spec";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { isTemplateCategory, type Template } from "../templates/spec";

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`;
}

function counts(workspace: WorkspaceSummary): string {
  return [
    plural(workspace.pageCount, "page", "pages"),
    plural(workspace.displayCount, "display", "displays"),
    plural(workspace.stateCount, "state", "states"),
  ].join(" · ");
}

/**
 * Design is the workspace index. A curated template is a workspace snapshot
 * with provenance curated; using it forks a new editable workspace and leaves
 * the original immutable.
 */
export function DesignIndexPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const workspaces = useWorkspaces();
  const [forking, setForking] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const rows = workspaces.data ?? [];
  const curated = rows.filter((row) => row.provenance === WorkspaceProvenance.CURATED);
  const authored = rows.filter((row) => row.provenance !== WorkspaceProvenance.CURATED);

  const newWorkspace = async () => {
    setCreating(true);
    setError(null);
    try {
      const workspace = await createWorkspace({
        name: t("workbench.designIndex.newWorkspaceName", { defaultValue: "Untitled workspace" }),
        provenance: WorkspaceProvenance.AUTHORED,
      });
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : t("workbench.designIndex.createFailed", {
              defaultValue: "The workspace could not be created.",
            }),
      );
    } finally {
      setCreating(false);
    }
  };

  const forkTemplate = async (id: string) => {
    setForking(id);
    setError(null);
    try {
      const workspace = await forkWorkspace(id);
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : t("workbench.designIndex.forkFailed", { defaultValue: "The template could not be forked." }),
      );
    } finally {
      setForking(null);
    }
  };

  return (
    <div className="wb-page" data-testid="design-index-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.designIndex.kicker", { defaultValue: "Design · Workspaces" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.designIndex.title", { defaultValue: "Your workspaces" })}
          </h1>
          <p className="wb-lede">
            {t("workbench.designIndex.lede", {
              defaultValue:
                "A workspace holds pages, a page holds device displays, and a display holds named states. Curated templates fork on use.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <WbButton
            variant="primary"
            size="md"
            disabled={creating}
            onClick={() => void newWorkspace()}
          >
            <PenTool aria-hidden size={14} />
            {creating
              ? t("workbench.designIndex.creating", { defaultValue: "Creating…" })
              : t("workbench.designIndex.newWorkspace", { defaultValue: "New workspace" })}
          </WbButton>
        </div>
      </div>

      {error ? (
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
          {error}
        </Banner>
      ) : null}

      <div className="wb-stack">
        <Panel>
          <PanelHead
            title={t("workbench.designIndex.workspaces", { defaultValue: "Workspaces" })}
            meta={t("workbench.designIndex.workspaceCount", {
              defaultValue: "{{count}} workspaces",
              count: authored.length,
            })}
          />
          <PanelBody flush>
            {workspaces.isLoading ? (
              <div className="wb-stack" style={{ padding: 16 }} data-testid="design-index-loading">
                <Skeleton height={42} />
                <Skeleton height={42} />
                <Skeleton height={42} />
              </div>
            ) : workspaces.isError ? (
              <div style={{ padding: 16 }}>
                <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
                  <b>
                    {t("workbench.designIndex.errorTitle", {
                      defaultValue: "The workspace list did not answer.",
                    })}
                  </b>{" "}
                  {t("workbench.designIndex.errorBody", {
                    defaultValue:
                      "Existing workspaces are not shown as an empty list. Retry by reloading the page.",
                  })}
                </Banner>
              </div>
            ) : authored.length === 0 ? (
              <Empty
                title={t("workbench.designIndex.emptyTitle", {
                  defaultValue: "No workspaces yet",
                })}
              >
                {t("workbench.designIndex.emptyBody", {
                  defaultValue:
                    "Use a curated template or create a workspace in the studio. Imported legacy designs appear here after the boot migration.",
                })}
              </Empty>
            ) : (
              <ul className="wb-home-list">
                {authored.map((workspace) => (
                  <li key={workspace.id} data-testid="design-workspace-row">
                    <div className="wb-home-row">
                      <span className="wb-home-thumb" aria-hidden>
                        <Layers size={14} />
                      </span>
                      <span className="wb-home-row-copy" style={{ minWidth: 0, flex: 1 }}>
                        <b>{workspace.name}</b>
                        <small>
                          {counts(workspace)}
                          {" · "}
                          {formatRelative(timestampToDate(workspace.updatedAt))}
                        </small>
                      </span>
                      <Chip tone="ice">{provenanceLabel(workspace.provenance)}</Chip>
                      <Link to={workspacePath(workspace.id)} className="wb-btn ghost sm">
                        {t("workbench.designIndex.open", { defaultValue: "Open" })}
                        <ArrowRight aria-hidden size={13} />
                      </Link>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </PanelBody>
        </Panel>

        <Panel>
          <PanelHead
            title={t("workbench.designIndex.templates", { defaultValue: "Curated templates" })}
            meta={t("workbench.designIndex.templateCount", {
              defaultValue: "{{count}} templates",
              count: curated.length,
            })}
          />
          <PanelBody>
            {curated.length === 0 ? (
              <Empty
                title={t("workbench.designIndex.noTemplates", {
                  defaultValue: "No curated templates yet",
                })}
              >
                {t("workbench.designIndex.noTemplatesBody", {
                  defaultValue:
                    "A curated template is a reviewed workspace snapshot. It remains immutable; using it creates a fork.",
                })}
              </Empty>
            ) : (
              <div className="wb-new-grid">
                {curated.map((workspace) => (
                  <article
                    key={workspace.id}
                    className="wb-new-card"
                    data-testid="design-template-card"
                  >
                    <WorkspaceTemplatePreview workspaceId={workspace.id} />
                    <b>{workspace.name}</b>
                    <small>{workspace.summary || counts(workspace)}</small>
                    <span className="wb-muted-2 wb-small">
                      {t("workbench.designIndex.templatePageCount", {
                        defaultValue: "{{pages}} · {{displays}}",
                        pages: plural(workspace.pageCount, "page", "pages"),
                        displays: plural(workspace.displayCount, "display", "displays"),
                      })}
                    </span>
                    <div className="wb-row" style={{ gap: 6, marginTop: 6 }}>
                      <Link to={workspacePath(workspace.id)} className="wb-btn ghost sm">
                        {t("workbench.designIndex.open", { defaultValue: "Open" })}
                      </Link>
                      <WbButton
                        variant="primary"
                        size="sm"
                        disabled={forking === workspace.id}
                        onClick={() => void forkTemplate(workspace.id)}
                      >
                        {forking === workspace.id
                          ? t("workbench.designIndex.forking", { defaultValue: "Forking…" })
                          : t("workbench.designIndex.useTemplate", { defaultValue: "Use template" })}
                      </WbButton>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </PanelBody>
        </Panel>
      </div>
    </div>
  );
}

/**
 * A curated template is a real composition, so the shelf shows it rather than
 * a placeholder bar that reads as a loading skeleton. The thumbnail is the
 * first page rendered at a reduced scale in the shelf's own colours.
 */
function WorkspaceTemplatePreview({ workspaceId }: { workspaceId: string }) {
  const query = useQuery({
    queryKey: ["workspace", workspaceId],
    queryFn: () => getWorkspace(workspaceId),
    staleTime: 60_000,
  });
  const workspace = query.data;
  const page = workspace?.pages[0];
  const display =
    page?.displays.find((entry) => entry.device === DisplayDevice.DESKTOP) ?? page?.displays[0];
  const template = useMemo<Template | null>(() => {
    if (!workspace || !page || !display) return null;
    const options = parseOptions(workspace.optionsJson);
    return {
      id: workspace.id,
      name: workspace.name,
      tagline: workspace.summary,
      category: workspace.tags.find(isTemplateCategory) ?? "analytics",
      idea: "",
      slotHint: "",
      display: options.display ?? "sans",
      radius: options.radius ?? 12,
      palette: parsePalette(workspace.themeJson),
      spec: displaySpec(page, display),
    };
  }, [display, page, workspace]);

  if (!template || !page) {
    return (
      <span className="wb-template-thumb is-pending" aria-hidden>
        <span className="wb-new-art-bar" />
      </span>
    );
  }
  return (
    <span className="wb-template-thumb" aria-hidden>
      <span className="wb-template-thumb-stage">
        <TemplateRenderer template={template} spec={template.spec} mode="thumb" />
      </span>
    </span>
  );
}

function provenanceLabel(provenance: WorkspaceProvenance): string {
  switch (provenance) {
    case WorkspaceProvenance.CURATED:
      return "curated";
    case WorkspaceProvenance.IMPORTED:
      return "imported";
    case WorkspaceProvenance.FORKED:
      return "forked";
    default:
      return "authored";
  }
}
