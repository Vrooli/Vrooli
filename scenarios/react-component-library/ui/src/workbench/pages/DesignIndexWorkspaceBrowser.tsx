import { useMemo } from "react";
import type { TFunction } from "i18next";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ArrowRight, CircleAlert, Layers } from "lucide-react";

import {
  DisplayDevice,
  WorkspaceProvenance,
  getWorkspace,
  type WorkspaceSummary,
} from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
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
import { displaySpec, parseOptions, parsePalette } from "../workspaces/spec";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { isTemplateCategory, type Template } from "../templates/spec";

type Props = {
  authored: WorkspaceSummary[];
  curated: WorkspaceSummary[];
  isLoading: boolean;
  isError: boolean;
  forking: string | null;
  onFork: (id: string) => void;
};

export function DesignIndexWorkspaceBrowser({
  authored,
  curated,
  isLoading,
  isError,
  forking,
  onFork,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="wb-stack">
      <Panel>
        <PanelHead
          title={t(strings.workbench.designIndex.workspaces)}
          meta={t(strings.workbench.designIndex.workspaceCount, { count: authored.length })}
        />
        <PanelBody flush>
          {isLoading ? (
            <div className="wb-stack" style={{ padding: 16 }} data-testid="design-index-loading">
              <Skeleton height={42} />
              <Skeleton height={42} />
              <Skeleton height={42} />
            </div>
          ) : isError ? (
            <div style={{ padding: 16 }}>
              <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
                <b>{t(strings.workbench.designIndex.errorTitle)}</b>{" "}
                {t(strings.workbench.designIndex.errorBody)}
              </Banner>
            </div>
          ) : authored.length === 0 ? (
            <Empty title={t(strings.workbench.designIndex.emptyTitle)}>
              {t(strings.workbench.designIndex.emptyBody)}
            </Empty>
          ) : (
            <ul className="wb-home-list">
              {authored.map((workspace) => (
                <li key={workspace.id} data-testid="design-workspace-row">
                  <div className="wb-home-row">
                    <span className="wb-home-thumb" aria-hidden>
                      <Layers size={14} />
                    </span>
                    <span className="wb-home-row-copy">
                      <b>{workspace.name}</b>
                      <small>
                        {workspaceCounts(workspace, t)}
                        {" · "}
                        {t(strings.workbench.designIndex.updated, {
                          relative: formatRelative(timestampToDate(workspace.updatedAt)),
                        })}
                      </small>
                    </span>
                    <Chip tone="ice">{provenanceLabel(workspace.provenance, t)}</Chip>
                    <Link to={workspacePath(workspace.id)} className="wb-btn ghost sm">
                      {t(strings.workbench.designIndex.open)}
                      <ArrowRight aria-hidden size={13} />
                    </Link>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </PanelBody>
      </Panel>

      <section
        id="curated-templates"
        aria-label={t(strings.workbench.designIndex.templates)}
        tabIndex={-1}
      >
        <Panel>
          <PanelHead
            title={t(strings.workbench.designIndex.templates)}
            meta={t(strings.workbench.designIndex.templateCount, { count: curated.length })}
          />
          <PanelBody>
            {curated.length === 0 ? (
              <Empty title={t(strings.workbench.designIndex.noTemplates)}>
                {t(strings.workbench.designIndex.noTemplatesBody)}
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
                    <small>{workspace.summary || workspaceCounts(workspace, t)}</small>
                    <span className="wb-muted-2 wb-small">
                      {t(strings.workbench.designIndex.templatePageCount, {
                        pages: plural(
                          workspace.pageCount,
                          t(strings.workbench.designIndex.page),
                          t(strings.workbench.designIndex.pages),
                        ),
                        displays: plural(
                          workspace.displayCount,
                          t(strings.workbench.designIndex.display),
                          t(strings.workbench.designIndex.displays),
                        ),
                      })}
                    </span>
                    <div className="wb-row" style={{ gap: 6, marginTop: 6 }}>
                      <Link to={workspacePath(workspace.id)} className="wb-btn ghost sm">
                        {t(strings.workbench.designIndex.open)}
                      </Link>
                      <WbButton
                        variant="primary"
                        size="sm"
                        disabled={forking === workspace.id}
                        onClick={() => onFork(workspace.id)}
                      >
                        {forking === workspace.id
                          ? t(strings.workbench.designIndex.forking)
                          : t(strings.workbench.designIndex.useTemplate)}
                      </WbButton>
                    </div>
                  </article>
                ))}
              </div>
            )}
          </PanelBody>
        </Panel>
      </section>
    </div>
  );
}

function WorkspaceTemplatePreview({ workspaceId }: { workspaceId: string }) {
  const query = useQuery({
    queryKey: ["workspace", workspaceId],
    queryFn: () => getWorkspace(workspaceId),
    staleTime: 60_000,
  });
  const workspace = query.data;
  const page = workspace?.pages[0];
  const display =
    page?.displays.find((entry) => entry.device === DisplayDevice.PHONE) ?? page?.displays[0];
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

function plural(count: number, one: string, many: string): string {
  return `${count} ${count === 1 ? one : many}`;
}

function workspaceCounts(workspace: WorkspaceSummary, t: TFunction): string {
  return [
    plural(
      workspace.pageCount,
      t(strings.workbench.designIndex.page),
      t(strings.workbench.designIndex.pages),
    ),
    plural(
      workspace.displayCount,
      t(strings.workbench.designIndex.display),
      t(strings.workbench.designIndex.displays),
    ),
    plural(
      workspace.stateCount,
      t(strings.workbench.designIndex.state),
      t(strings.workbench.designIndex.states),
    ),
  ].join(" · ");
}

function provenanceLabel(provenance: WorkspaceProvenance, t: TFunction): string {
  switch (provenance) {
    case WorkspaceProvenance.CURATED:
      return t(strings.workbench.designIndex.provenanceCurated);
    case WorkspaceProvenance.IMPORTED:
      return t(strings.workbench.designIndex.provenanceImported);
    case WorkspaceProvenance.FORKED:
      return t(strings.workbench.designIndex.provenanceForked);
    default:
      return t(strings.workbench.designIndex.provenanceAuthored);
  }
}
