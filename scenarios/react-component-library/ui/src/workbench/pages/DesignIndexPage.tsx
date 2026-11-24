import "../design-index.css";
import { useEffect, useState } from "react";
import { create } from "@bufbuild/protobuf";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { CircleAlert, PenTool } from "lucide-react";

import {
  DisplayDevice,
  StateStatus,
  WorkspaceDisplaySchema,
  WorkspacePageSchema,
  WorkspaceProvenance,
  WorkspaceStateSchema,
  type Workspace,
  createWorkspace,
  forkWorkspace,
  getWorkspace,
  saveWorkspace,
  uploadWorkspaceAsset,
} from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { workspacePath } from "../../routes";
import { Banner, WbButton } from "../primitives";
import { useWorkspaces } from "../workspaces/useWorkspaces";
import { validateCapture, readTextFile } from "./designIndexCapture";
import { DesignIndexStartFlows } from "./DesignIndexStartFlows";
import { htmlTemplate, parseHtml } from "../studio/htmlImport";
import { writeStateSpec } from "../workspaces/spec";
import { DesignIndexWorkspaceBrowser } from "./DesignIndexWorkspaceBrowser";

type PendingBriefDraft = {
  workspaceId: string;
  templateId: string;
  templateRevision: number;
  templateName: string;
  name: string;
  purpose: string;
  user: string;
  action: string;
};

const PENDING_BRIEF_STORAGE_KEY = "rcl.workbench.design-brief.pending.v1";

function readPendingBriefDraft(): PendingBriefDraft | null {
  try {
    if (typeof window === "undefined") return null;
    const raw = window.sessionStorage.getItem(PENDING_BRIEF_STORAGE_KEY);
    if (!raw) return null;
    const value = JSON.parse(raw) as Partial<PendingBriefDraft>;
    if (
      typeof value.workspaceId !== "string" ||
      typeof value.templateId !== "string" ||
      typeof value.templateRevision !== "number" ||
      typeof value.templateName !== "string" ||
      typeof value.name !== "string" ||
      typeof value.purpose !== "string" ||
      typeof value.user !== "string" ||
      typeof value.action !== "string"
    ) return null;
    return value as PendingBriefDraft;
  } catch {
    return null;
  }
}

function writePendingBriefDraft(draft: PendingBriefDraft): void {
  try {
    window.sessionStorage.setItem(PENDING_BRIEF_STORAGE_KEY, JSON.stringify(draft));
  } catch {
    // Brief recovery is a session convenience; the workspace remains the source of truth.
  }
}

function clearPendingBriefDraft(): void {
  try {
    window.sessionStorage.removeItem(PENDING_BRIEF_STORAGE_KEY);
  } catch {
    // Nothing to clear when session storage is unavailable.
  }
}

/**
 * Design is the workspace index. A curated template is a workspace snapshot
 * with provenance curated; using it forks a new editable workspace and leaves
 * the original immutable.
 */
export function DesignIndexPage() {
  const { t } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const workspaces = useWorkspaces();
  const [pendingBriefDraft, setPendingBriefDraft] = useState<PendingBriefDraft | null>(
    readPendingBriefDraft,
  );
  const [forking, setForking] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [startingFlow, setStartingFlow] = useState<"brief" | "capture" | null>(null);
  const [briefName, setBriefName] = useState(() => pendingBriefDraft?.name ?? "");
  const [briefPurpose, setBriefPurpose] = useState(() => pendingBriefDraft?.purpose ?? "");
  const [briefUser, setBriefUser] = useState(() => pendingBriefDraft?.user ?? "");
  const [briefAction, setBriefAction] = useState(() => pendingBriefDraft?.action ?? "");
  const [pendingBriefWorkspace, setPendingBriefWorkspace] = useState<Workspace | null>(null);
  const [restoringPendingBrief, setRestoringPendingBrief] = useState(() => pendingBriefDraft !== null);
  const [captureManifest, setCaptureManifest] = useState<File | null>(null);
  const [captureArtifacts, setCaptureArtifacts] = useState<File[]>([]);
  const [templateId, setTemplateId] = useState(() => pendingBriefDraft?.templateId ?? "");
  const [params] = useSearchParams();
  const [error, setError] = useState<string | null>(null);

  const rows = workspaces.data ?? [];
  const curated = rows.filter((row) => row.provenance === WorkspaceProvenance.CURATED);
  const authored = rows.filter((row) => row.provenance !== WorkspaceProvenance.CURATED);
  const selectedTemplate = curated.find((item) => item.id === templateId) ?? curated[0];

  const pendingBriefWorkspaceId = pendingBriefDraft?.workspaceId;

  useEffect(() => {
    if (!pendingBriefWorkspaceId || pendingBriefWorkspace) {
      setRestoringPendingBrief(false);
      return;
    }
    let active = true;
    setRestoringPendingBrief(true);
    void getWorkspace(pendingBriefWorkspaceId)
      .then((workspace) => {
        if (!active) return;
        if (workspace) setPendingBriefWorkspace(workspace);
        else setError(t(strings.workbench.designIndex.briefRecoveryUnavailable));
      })
      .catch(() => {
        if (active) setError(t(strings.workbench.designIndex.briefRecoveryUnavailable));
      })
      .finally(() => {
        if (active) setRestoringPendingBrief(false);
      });
    return () => {
      active = false;
    };
  }, [pendingBriefWorkspaceId, pendingBriefWorkspace, t]);

  const persistPendingBriefField = (
    field: "name" | "purpose" | "user" | "action",
    value: string,
  ) => {
    setPendingBriefDraft((current) => {
      if (!current) return current;
      const next = { ...current, [field]: value };
      writePendingBriefDraft(next);
      return next;
    });
  };

  const newWorkspace = async () => {
    setCreating(true);
    setError(null);
    try {
      const workspace = await createWorkspace({
        name: t(strings.workbench.designIndex.newWorkspaceName),
        provenance: WorkspaceProvenance.AUTHORED,
      });
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : t(strings.workbench.designIndex.createFailed),
      );
    } finally {
      setCreating(false);
    }
  };

  useEffect(() => {
    // N lands here from anywhere; focusing the primary action completes the
    // shortcut without creating a workspace the person did not name.
    if (params.get("new") === "1") {
      document.querySelector<HTMLButtonElement>('[data-new-workspace="true"]')?.focus();
    }
  }, [params]);

  useEffect(() => {
    if (location.hash !== "#curated-templates") return;
    const target = document.getElementById("curated-templates");
    target?.focus({ preventScroll: true });
    target?.scrollIntoView({ block: "start" });
  }, [location.hash, workspaces.data]);

  const forkTemplate = async (id: string) => {
    setForking(id);
    setError(null);
    try {
      const workspace = await forkWorkspace(id);
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : t(strings.workbench.designIndex.forkFailed),
      );
    } finally {
      setForking(null);
    }
  };

  const startFromBrief = async () => {
    const template = selectedTemplate;
    if (
      (!pendingBriefDraft && !template) ||
      !briefName.trim() ||
      !briefPurpose.trim() ||
      !briefUser.trim() ||
      !briefAction.trim()
    )
      return;
    setStartingFlow("brief");
    setError(null);
    try {
      const sourceTemplate = pendingBriefDraft
        ? {
            id: pendingBriefDraft.templateId,
            revision: pendingBriefDraft.templateRevision,
            name: pendingBriefDraft.templateName,
          }
        : template;
      if (!sourceTemplate) return;

      let workspace = pendingBriefDraft
        ? (await getWorkspace(pendingBriefDraft.workspaceId)) ?? pendingBriefWorkspace
        : null;
      if (pendingBriefDraft) {
        if (!workspace) throw new Error(t(strings.workbench.designIndex.briefRecoveryUnavailable));
        setPendingBriefWorkspace(workspace);
        // A save can reach the server even when its response is lost. Re-read
        // the same fork before retrying so we neither fork a duplicate nor
        // overwrite a successfully saved draft with a stale revision.
        const expectedSummary = t(strings.workbench.designIndex.briefSummary, {
          id: sourceTemplate.id,
          revision: sourceTemplate.revision,
          purpose: briefPurpose.trim(),
          user: briefUser.trim(),
          action: briefAction.trim(),
        });
        if (workspace.tags.includes("design-brief") && workspace.summary === expectedSummary) {
          clearPendingBriefDraft();
          setPendingBriefDraft(null);
          setPendingBriefWorkspace(null);
          void navigate(workspacePath(workspace.id));
          return;
        }
      } else {
        workspace = await forkWorkspace(sourceTemplate.id, briefName.trim());
        const pending: PendingBriefDraft = {
          workspaceId: workspace.id,
          templateId: sourceTemplate.id,
          templateRevision: sourceTemplate.revision,
          templateName: sourceTemplate.name,
          name: briefName.trim(),
          purpose: briefPurpose.trim(),
          user: briefUser.trim(),
          action: briefAction.trim(),
        };
        writePendingBriefDraft(pending);
        setPendingBriefDraft(pending);
        setPendingBriefWorkspace(workspace);
      }
      if (workspace.pages[0]) workspace.pages[0].name = briefName.trim();
      workspace.tags = [...new Set([...workspace.tags, "design-brief"])];
      workspace.summary = t(strings.workbench.designIndex.briefSummary, {
        id: sourceTemplate.id,
        revision: sourceTemplate.revision,
        purpose: briefPurpose.trim(),
        user: briefUser.trim(),
        action: briefAction.trim(),
      });
      const pending: PendingBriefDraft = {
        workspaceId: workspace.id,
        templateId: sourceTemplate.id,
        templateRevision: sourceTemplate.revision,
        templateName: sourceTemplate.name,
        name: briefName.trim(),
        purpose: briefPurpose.trim(),
        user: briefUser.trim(),
        action: briefAction.trim(),
      };
      writePendingBriefDraft(pending);
      setPendingBriefDraft(pending);
      setPendingBriefWorkspace(workspace);
      const saved = await saveWorkspace(workspace, workspace.revision);
      clearPendingBriefDraft();
      setPendingBriefDraft(null);
      setPendingBriefWorkspace(null);
      void navigate(workspacePath(saved.id));
    } catch (cause) {
      setError(
        cause instanceof Error ? cause.message : t(strings.workbench.designIndex.briefCreateFailed),
      );
    } finally {
      setStartingFlow(null);
    }
  };

  const startFromCapture = async () => {
    if (!captureManifest || captureArtifacts.length === 0) return;
    setStartingFlow("capture");
    setError(null);
    try {
      const { manifest, pixels, frozenPage } = await validateCapture(
        captureManifest,
        captureArtifacts,
        t,
      );
      const html = await readTextFile(frozenPage, t);
      const parsed = parseHtml(html, frozenPage.name);
      if (parsed.layers === 0)
        throw new Error(t(strings.workbench.designIndex.captureFrozenPageNoContent));
      const imported = htmlTemplate(parsed, null, parsed.title, parsed.sourcePreview(null));
      const pixelDeclaration = manifest.artifacts.find((artifact) => artifact.kind === "pixels");
      if (!pixelDeclaration) throw new Error(t(strings.workbench.designIndex.capturePngRequired));
      const frozenPageDeclaration = manifest.artifacts.find(
        (artifact) => artifact.kind === "frozen-page",
      );
      if (!frozenPageDeclaration)
        throw new Error(t(strings.workbench.designIndex.captureFrozenPageRequired));
      const manifestDigest = new Uint8Array(
        await crypto.subtle.digest("SHA-256", await captureManifest.arrayBuffer()),
      );
      const manifestSha256 = Array.from(manifestDigest, (byte) =>
        byte.toString(16).padStart(2, "0"),
      ).join("");
      const lensFidelity = Object.entries(manifest.lenses)
        .map(
          ([lens, evidence]) =>
            `${lens}=${evidence.status}${evidence.reason ? ` (${evidence.reason})` : ""}`,
        )
        .join(", ");
      const suffix = manifest.id.replace(/-/g, "");
      const state = create(WorkspaceStateSchema, {
        id: `st_${suffix}`,
        name: "EX9 captured source",
        argsJson: "{}",
        scenarioDataJson: "{}",
        status: StateStatus.CAPTURED,
        captureId: manifest.id,
        sortOrder: 0,
      });
      writeStateSpec(state, imported.spec);
      const device =
        manifest.viewport.width <= 480
          ? DisplayDevice.PHONE
          : manifest.viewport.width <= 1024
            ? DisplayDevice.TABLET
            : DisplayDevice.DESKTOP;
      const workspace = await createWorkspace({
        name: `${manifest.source.scenario} · ${parsed.title} revision`,
        provenance: WorkspaceProvenance.IMPORTED,
        tags: ["ex9-capture", "editable-reference"],
        pages: [
          create(WorkspacePageSchema, {
            id: `pg_${suffix}`,
            name: parsed.title,
            route: manifest.source.route ?? "/",
            pageTypeId: "",
            sortOrder: 0,
            specJson: JSON.stringify(imported.spec),
            displays: [
              create(WorkspaceDisplaySchema, {
                id: `dp_${suffix}`,
                device,
                width: manifest.viewport.width,
                height: manifest.viewport.height,
                sortOrder: 0,
                states: [state],
              }),
            ],
          }),
        ],
        themeJson: JSON.stringify(imported.palette),
        optionsJson: JSON.stringify({ radius: imported.radius, display: imported.display }),
        summary: t(strings.workbench.designIndex.captureSummary, {
          schemaVersion: manifest.schema_version,
          captureId: manifest.id,
          manifestSha256,
          sessionId: manifest.session.id,
          capturedAt: manifest.captured_at,
          scenario: manifest.source.scenario,
          route: manifest.source.route,
          buildId: manifest.source.build_id,
          sourceRevision: manifest.source.revision ?? t(strings.workbench.designIndex.notSupplied),
          width: manifest.viewport.width,
          height: manifest.viewport.height,
          pixelSha256: pixelDeclaration.sha256,
          frozenPageSha256: frozenPageDeclaration.sha256,
          fidelity: lensFidelity,
        }),
      });
      await uploadWorkspaceAsset(workspace.id, pixels, "captured");
      void navigate(workspacePath(workspace.id));
    } catch (cause) {
      setError(
        cause instanceof Error
          ? cause.message
          : t(strings.workbench.designIndex.captureImportFailed),
      );
    } finally {
      setStartingFlow(null);
    }
  };

  return (
    <div
      className="wb-page"
      data-experience-surface="design-overview"
      data-testid="design-index-page"
    >
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t(strings.workbench.designIndex.kicker)}</p>
          <h1 className="wb-h1">{t(strings.workbench.designIndex.title)}</h1>
          <p className="wb-lede">{t(strings.workbench.designIndex.lede)}</p>
        </div>
        <div className="wb-actions">
          <WbButton
            data-new-workspace="true"
            variant="primary"
            size="md"
            disabled={creating}
            onClick={() => void newWorkspace()}
          >
            <PenTool aria-hidden size={14} />
            {creating
              ? t(strings.workbench.designIndex.creating)
              : t(strings.workbench.designIndex.newWorkspace)}
          </WbButton>
        </div>
      </div>

      {error ? (
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
          {error}
        </Banner>
      ) : null}

      <DesignIndexStartFlows
        curated={curated}
        selectedTemplateId={selectedTemplate?.id ?? ""}
        onTemplateChange={setTemplateId}
        briefName={briefName}
        onBriefNameChange={(value) => {
          setBriefName(value);
          persistPendingBriefField("name", value);
        }}
        briefPurpose={briefPurpose}
        onBriefPurposeChange={(value) => {
          setBriefPurpose(value);
          persistPendingBriefField("purpose", value);
        }}
        briefUser={briefUser}
        onBriefUserChange={(value) => {
          setBriefUser(value);
          persistPendingBriefField("user", value);
        }}
        briefAction={briefAction}
        onBriefActionChange={(value) => {
          setBriefAction(value);
          persistPendingBriefField("action", value);
        }}
        hasPendingBrief={pendingBriefDraft !== null}
        pendingBriefRecovery={pendingBriefDraft}
        restoringBrief={restoringPendingBrief}
        captureManifest={captureManifest}
        onCaptureManifestChange={setCaptureManifest}
        captureArtifacts={captureArtifacts}
        onCaptureArtifactsChange={setCaptureArtifacts}
        startingFlow={startingFlow}
        onStartFromBrief={() => void startFromBrief()}
        onStartFromCapture={() => void startFromCapture()}
      />

      <DesignIndexWorkspaceBrowser
        authored={authored}
        curated={curated}
        isLoading={workspaces.isLoading}
        isError={workspaces.isError}
        forking={forking}
        onFork={(id) => void forkTemplate(id)}
      />
    </div>
  );
}
