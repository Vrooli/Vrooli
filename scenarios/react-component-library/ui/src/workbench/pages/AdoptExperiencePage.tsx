import { useState } from "react";
import { create } from "@bufbuild/protobuf";
import { useNavigate, useParams } from "react-router-dom";
import { ArrowRight, CircleAlert, Layers } from "lucide-react";

import {
  DisplayDevice,
  StateStatus,
  WorkspaceDisplaySchema,
  WorkspacePageSchema,
  WorkspaceStateSchema,
  createWorkspace,
  getWorkspace,
  listWorkspaces,
} from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { workspacePath } from "../../routes";
import { adoptExperiencePage, useDesignPages, type ExperienceAdoptionResult } from "../data";
import { Banner, Empty, Panel, PanelBody, PanelHead, Skeleton, WbButton } from "../primitives";

/** Creates the workspace composition for an experience page that has not been adopted yet. */
export function AdoptExperiencePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { scenario = "", page = "" } = useParams<{ scenario?: string; page?: string }>();
  const pages = useDesignPages(scenario || null);
  const intent = pages.data?.find((entry) => entry.page === page);
  const hasLegacyComposition = intent?.status === "composition-present";
  const adoptable = intent && ["contract-only", "region-declared"].includes(intent.status);
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [imported, setImported] = useState<ExperienceAdoptionResult | null>(null);

  const adopt = async () => {
    if (!intent || creating) return;
    setCreating(true);
    setError(null);
    try {
      const workspace = await createWorkspace({
        name: intent.title || page,
        summary: `Composition for ${scenario}/${page}`,
        bindingScenario: scenario,
        bindingPage: page,
        pages: [
          create(WorkspacePageSchema, {
            name: intent.title || page,
            route: intent.route,
            specJson: "{}",
            displays: [
              create(WorkspaceDisplaySchema, {
                device: DisplayDevice.DESKTOP,
                width: 1180,
                height: 780,
                states: [
                  create(WorkspaceStateSchema, {
                    name: "Default",
                    argsJson: "{}",
                    scenarioDataJson: "{}",
                    status: StateStatus.DRAFT,
                  }),
                ],
              }),
            ],
          }),
        ],
      });
      void navigate(workspacePath(workspace.id), { replace: true });
    } catch (cause) {
      // A unique binding constraint is the cross-tab race guard. If another
      // adoption won, resolve its workspace and continue there idempotently.
      try {
        const conflictID =
          cause instanceof Error
            ? cause.message.match(
                /experience page is already bound to a workspace:\s*(ws_[\w-]+)/,
              )?.[1]
            : undefined;
        if (conflictID) {
          const existing = await getWorkspace(conflictID);
          if (existing) {
            void navigate(workspacePath(existing.id), { replace: true });
            return;
          }
        }
        const workspaces = await listWorkspaces({ limit: 500 });
        const existing = workspaces.find(
          (entry) => entry.bindingScenario === scenario && entry.bindingPage === page,
        );
        if (existing) {
          void navigate(workspacePath(existing.id), { replace: true });
          return;
        }
      } catch {
        // Keep the original failure visible if the recovery read also fails.
      }
      setError(cause instanceof Error ? cause.message : "The workspace could not be created.");
    } finally {
      setCreating(false);
    }
  };

  const importComposition = async () => {
    if (!intent || creating) return;
    setCreating(true);
    setError(null);
    try {
      const result = await adoptExperiencePage(scenario, page);
      if (!result.workspace?.id || !result.receiptId || !result.sourceHash || !result.targetHash) {
        throw new Error("The import completed without its workspace and recovery receipt.");
      }
      if (!result.receiptJson || !JSON.parse(result.receiptJson)) {
        throw new Error("The import completed without a readable recovery receipt.");
      }
      setImported(result);
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "The experience page could not be imported.");
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="wb-page" data-testid="adopt-experience-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">{t(strings.workbench.adopt.kicker)}</p>
          <h1 className="wb-h1">{t(strings.workbench.adopt.title)}</h1>
          <p className="wb-lede">{t(strings.workbench.adopt.lede)}</p>
        </div>
      </div>
      {error ? (
        <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
          {error}
        </Banner>
      ) : null}
      {imported ? (
        <Panel>
          <PanelHead
            title={t(strings.workbench.adopt.importedTitle)}
            meta={t(imported.alreadyAdopted ? strings.workbench.adopt.existingWorkspace : strings.workbench.adopt.workspaceReady)}
          />
          <PanelBody>
            {(() => {
              const receipt = JSON.parse(imported.receiptJson) as { renderParity?: boolean; renderGap?: string };
              return (
                <>
                  <p>{t(strings.workbench.adopt.nativeParity)} <code>{t(receipt.renderParity ? strings.workbench.adopt.parityConfirmed : strings.workbench.adopt.parityPartial)}</code>.</p>
                  {receipt.renderGap ? <p>{t(strings.workbench.adopt.renderGap)} <span data-testid="adopt-render-gap">{receipt.renderGap}</span></p> : null}
                </>
              );
            })()}
            <p>{t(strings.workbench.adopt.sourceIntact)} {t(strings.workbench.adopt.receiptLabel)} <code data-testid="adopt-receipt-id">{imported.receiptId}</code></p>
            <p>{t(strings.workbench.adopt.sourceHash)} <code data-testid="adopt-source-hash">{imported.sourceHash}</code></p>
            <p>{t(strings.workbench.adopt.targetHash)} <code data-testid="adopt-target-hash">{imported.targetHash}</code></p>
            <a href={`data:application/json;charset=utf-8,${encodeURIComponent(imported.receiptJson)}`} download={`${scenario}-${page}-recovery-receipt.json`}>
              {t(strings.workbench.adopt.downloadReceipt)}
            </a>
            <WbButton variant="primary" onClick={() => imported.workspace && void navigate(workspacePath(imported.workspace.id))}>
              {t(strings.workbench.adopt.openWorkspace)} <ArrowRight aria-hidden size={14} />
            </WbButton>
          </PanelBody>
        </Panel>
      ) : null}
      {pages.isLoading ? (
        <Skeleton height={160} />
      ) : pages.isError ? (
        <Empty title={t(strings.workbench.adopt.unavailable)}>
          {t(strings.workbench.adopt.loadError)}
        </Empty>
      ) : !intent ? (
        <Empty title={t(strings.workbench.adopt.missing)}>
          {t(strings.workbench.adopt.missingBody)}
        </Empty>
      ) : hasLegacyComposition ? (
        <Panel>
          <PanelHead title={intent.title || page} meta={`${scenario} · ${page}`} />
          <PanelBody>
            <p>{t(strings.workbench.adopt.compositionPresentBody)}</p>
            <WbButton data-testid="adopt-import-experience" variant="primary" disabled={creating || Boolean(imported)} onClick={() => void importComposition()}>
              <Layers aria-hidden size={14} />
              {creating ? t(strings.workbench.adopt.importing) : t(strings.workbench.adopt.importAction)}
              <ArrowRight aria-hidden size={14} />
            </WbButton>
          </PanelBody>
        </Panel>
      ) : !adoptable ? (
        <Empty title={t(strings.workbench.adopt.unavailable)}>
          {t(strings.workbench.adopt.invalidIntent)}
        </Empty>
      ) : (
        <Panel>
          <PanelHead title={intent.title || page} meta={`${scenario} · ${page}`} />
          <PanelBody>
            <p className="wb-lede">{intent.route || t(strings.workbench.adopt.noRoute)}</p>
            <WbButton
              data-testid="adopt-create-workspace"
              variant="primary"
              disabled={creating}
              onClick={() => void adopt()}
            >
              <Layers aria-hidden size={14} />
              {creating ? t(strings.workbench.adopt.creating) : t(strings.workbench.adopt.create)}
              <ArrowRight aria-hidden size={14} />
            </WbButton>
          </PanelBody>
        </Panel>
      )}
    </div>
  );
}
