import { useEffect, useMemo, useState } from "react";
import { useParams, useSearchParams } from "react-router-dom";

import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import {
  getWorkspace,
  type Workspace,
  type WorkspaceDisplay,
  type WorkspacePage,
  type WorkspaceState,
} from "../../api/workspaces";
import { useComponentBundle } from "../data";
import { FixtureFrame } from "../component/FixtureFrame";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { walk, type TemplateNode } from "../templates/spec";
import { pageDocument } from "../workspaces/spec";
import { frameTemplate } from "../studio/model";

interface Selection {
  workspace: Workspace;
  page: WorkspacePage;
  display: WorkspaceDisplay;
  state: WorkspaceState;
}

/** A chrome-free, read-only owner route for BAS state capture. */
export function WorkspaceStateCapturePage() {
  const { t } = useTranslation();
  const { id = "" } = useParams<{ id: string }>();
  const [params] = useSearchParams();
  const revision = Number(params.get("revision"));
  const pageId = params.get("page") ?? "";
  const displayId = params.get("display") ?? "";
  const stateId = params.get("state") ?? "";
  const [selection, setSelection] = useState<Selection | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [readySlots, setReadySlots] = useState<Set<string>>(() => new Set());

  useEffect(() => {
    let active = true;
    setSelection(null);
    setError(null);
    setReadySlots(new Set());
    void getWorkspace(id)
      .then((workspace) => {
        if (!active) return;
        const page = workspace?.pages.find((item) => item.id === pageId);
        const display = page?.displays.find((item) => item.id === displayId);
        const state = display?.states.find((item) => item.id === stateId);
        if (!workspace || workspace.revision !== revision || !page || !display || !state) {
          setError("The selected workspace revision or state is stale.");
          return;
        }
        setSelection({ workspace, page, display, state });
      })
      .catch((cause: unknown) => {
        if (active)
          setError(cause instanceof Error ? cause.message : "Workspace state could not be loaded.");
      });
    return () => {
      active = false;
    };
  }, [displayId, id, pageId, revision, stateId]);

  const document = useMemo(
    () =>
      selection
        ? pageDocument(selection.workspace, selection.page, {
            [selection.display.id]: selection.state.id,
          })
        : null,
    [selection],
  );
  const frame = document?.frames.find((entry) => entry.id === displayId);
  const componentSlots = useMemo(() => {
    const slots: TemplateNode[] = [];
    if (frame)
      walk(frame.spec, (node) => {
        if (
          node.type === "slot" &&
          typeof node.props.componentId === "string" &&
          node.props.componentId
        )
          slots.push(node);
      });
    return slots;
  }, [frame]);
  const slotReady = componentSlots.every((node) => readySlots.has(node.id));

  if (error)
    return (
      <div role="alert" data-testid="workspace-capture-error">
        {error}
      </div>
    );
  if (!selection || !document || !frame)
    return <div role="status">{t(strings.workbench.workspace.capturePageLoading)}</div>;

  const renderComponentSlot = (node: TemplateNode) => {
    const componentId = typeof node.props.componentId === "string" ? node.props.componentId : "";
    if (!componentId) return undefined;
    return (
      <WorkspaceCaptureComponentSlot
        key={node.id}
        componentId={componentId}
        storyId={typeof node.props.storyId === "string" ? node.props.storyId : ""}
        componentVersion={
          typeof node.props.componentVersion === "string" ? node.props.componentVersion : ""
        }
        onReady={(ready) =>
          setReadySlots((current) => {
            const next = new Set(current);
            if (ready) next.add(node.id);
            else next.delete(node.id);
            return next;
          })
        }
      />
    );
  };
  return (
    <main
      data-testid="workspace-state-composition"
      data-capture-ready={slotReady ? "true" : undefined}
      style={{
        width: selection.display.width,
        height: selection.display.height,
        overflow: "hidden",
        margin: 0,
      }}
    >
      <TemplateRenderer
        template={frameTemplate(document, frame)}
        spec={frame.spec}
        mode="preview"
        renderComponentSlot={renderComponentSlot}
      />
    </main>
  );
}

function WorkspaceCaptureComponentSlot({
  componentId,
  storyId,
  componentVersion,
  onReady,
}: {
  componentId: string;
  storyId: string;
  componentVersion: string;
  onReady: (ready: boolean) => void;
}) {
  const { t } = useTranslation();
  const bundle = useComponentBundle(componentId, componentVersion || undefined);
  // A selected state can pin a story. If that story disappears from the
  // current bundle, keep capture blocked instead of silently recording the
  // first remaining story as evidence for a different composition.
  const story = storyId
    ? bundle.data?.stories.find((entry) => entry.id === storyId)
    : bundle.data?.stories[0];
  useEffect(() => {
    if (!bundle.isLoading && (!bundle.data || !story)) onReady(false);
  }, [bundle.data, bundle.isLoading, onReady, story]);
  if (bundle.isLoading) {
    return (
      <div role="status" data-capture-pending="true">
        {t(strings.workbench.workspace.captureComponentLoading)}
      </div>
    );
  }
  if (!bundle.data) {
    return (
      <div role="alert" data-capture-error="true">
        {t(strings.workbench.workspace.captureComponentUnavailable)}
      </div>
    );
  }
  if (!story) {
    return (
      <div role="alert" data-capture-error="true">
        {t(strings.workbench.workspace.captureStoryUnavailable)}
      </div>
    );
  }
  return (
    <FixtureFrame
      assetId={componentId}
      storyId={story.id}
      version={story.version}
      height={64}
      title={story.name || story.id}
      onStatus={(status) => onReady(status === "ready" || status === "applied")}
    />
  );
}
