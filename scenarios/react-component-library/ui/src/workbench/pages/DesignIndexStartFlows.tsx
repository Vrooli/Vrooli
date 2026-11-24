import { ArrowRight } from "lucide-react";
import type { WorkspaceSummary } from "../../api/workspaces";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { Panel, PanelBody, PanelHead, WbButton } from "../primitives";

export type DesignFlow = "brief" | "capture" | null;

export type PendingBriefRecovery = {
  workspaceId: string;
  templateId: string;
  templateRevision: number;
  templateName: string;
  name: string;
  purpose: string;
  user: string;
  action: string;
};

type Props = {
  curated: WorkspaceSummary[];
  selectedTemplateId: string;
  onTemplateChange: (id: string) => void;
  briefName: string;
  onBriefNameChange: (value: string) => void;
  briefPurpose: string;
  onBriefPurposeChange: (value: string) => void;
  briefUser: string;
  onBriefUserChange: (value: string) => void;
  briefAction: string;
  onBriefActionChange: (value: string) => void;
  hasPendingBrief: boolean;
  pendingBriefRecovery: PendingBriefRecovery | null;
  restoringBrief: boolean;
  captureManifest: File | null;
  onCaptureManifestChange: (file: File | null) => void;
  captureArtifacts: File[];
  onCaptureArtifactsChange: (files: File[]) => void;
  startingFlow: DesignFlow;
  onStartFromBrief: () => void;
  onStartFromCapture: () => void;
};

export function DesignIndexStartFlows({
  curated,
  selectedTemplateId,
  onTemplateChange,
  briefName,
  onBriefNameChange,
  briefPurpose,
  onBriefPurposeChange,
  briefUser,
  onBriefUserChange,
  briefAction,
  onBriefActionChange,
  hasPendingBrief,
  pendingBriefRecovery,
  restoringBrief,
  captureManifest,
  onCaptureManifestChange,
  captureArtifacts,
  onCaptureArtifactsChange,
  startingFlow,
  onStartFromBrief,
  onStartFromCapture,
}: Props) {
  const { t } = useTranslation();
  return (
    <section
      className="wb-design-paths"
      aria-label={t(strings.workbench.designIndex.startWorkflow)}
    >
      <Panel>
        <PanelHead
          title={t(strings.workbench.designIndex.briefHeading)}
          meta={t(strings.workbench.designIndex.briefMeta)}
        />
        <PanelBody>
          {pendingBriefRecovery ? (
            <div
              className="wb-design-recovery"
              role="status"
              aria-live="polite"
              aria-label={t(strings.workbench.designIndex.briefRecoveryLabel)}
            >
              <strong>
                {restoringBrief
                  ? t(strings.workbench.designIndex.briefRecoveryRestoring, {
                      name: pendingBriefRecovery.name,
                      templateName: pendingBriefRecovery.templateName,
                    })
                  : t(strings.workbench.designIndex.briefRecoveryLabel)}
              </strong>
              <span>
                {t(strings.workbench.designIndex.briefRecoveryStatus, {
                  name: pendingBriefRecovery.name,
                  workspaceId: pendingBriefRecovery.workspaceId,
                  templateName: pendingBriefRecovery.templateName,
                  templateId: pendingBriefRecovery.templateId,
                  revision: pendingBriefRecovery.templateRevision,
                })}
              </span>
            </div>
          ) : null}
          <label className="wb-design-template-choice wb-design-field">
            {t(strings.workbench.designIndex.reviewedTemplate)}
            <select
              aria-label={t(strings.workbench.designIndex.reviewedTemplate)}
              value={selectedTemplateId}
              onChange={(event) => onTemplateChange(event.target.value)}
              disabled={curated.length === 0 || hasPendingBrief}
            >
              {curated.map((item) => (
                <option value={item.id} key={item.id}>
                  {item.name}
                </option>
              ))}
            </select>
            <span className="wb-muted-2 wb-small">
              {t(strings.workbench.designIndex.templateForkNote)}
            </span>
          </label>
          <div className="wb-design-path-grid">
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.pageName)}
              <input
                value={briefName}
                onChange={(event) => onBriefNameChange(event.target.value)}
                placeholder={t(strings.workbench.designIndex.pageNamePlaceholder)}
              />
            </label>
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.purposeLabel)}
              <textarea
                value={briefPurpose}
                onChange={(event) => onBriefPurposeChange(event.target.value)}
                placeholder={t(strings.workbench.designIndex.purposePlaceholder)}
              />
            </label>
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.userLabel)}
              <input
                value={briefUser}
                onChange={(event) => onBriefUserChange(event.target.value)}
                placeholder={t(strings.workbench.designIndex.userPlaceholder)}
              />
            </label>
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.actionLabel)}
              <input
                value={briefAction}
                onChange={(event) => onBriefActionChange(event.target.value)}
                placeholder={t(strings.workbench.designIndex.actionPlaceholder)}
              />
            </label>
          </div>
          <p className="wb-muted-2 wb-small">{t(strings.workbench.designIndex.briefNote)}</p>
          <WbButton
            variant="primary"
            size="sm"
            disabled={
              startingFlow !== null ||
              restoringBrief ||
              (curated.length === 0 && !hasPendingBrief) ||
              !briefName.trim() ||
              !briefPurpose.trim() ||
              !briefUser.trim() ||
              !briefAction.trim()
            }
            onClick={onStartFromBrief}
          >
            {startingFlow === "brief"
              ? t(strings.workbench.designIndex.preparingPage)
              : t(
                  hasPendingBrief
                    ? strings.workbench.designIndex.retryBrief
                    : strings.workbench.designIndex.buildFromBrief,
                )}
            <ArrowRight aria-hidden size={13} />
          </WbButton>
        </PanelBody>
      </Panel>
      <Panel>
        <PanelHead
          title={t(strings.workbench.designIndex.captureHeading)}
          meta={t(strings.workbench.designIndex.captureMeta)}
        />
        <PanelBody>
          <div className="wb-design-path-grid">
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.manifestLabel)}
              <input
                type="file"
                accept="application/json,.json"
                onChange={(event) => onCaptureManifestChange(event.target.files?.[0] ?? null)}
              />
            </label>
            <label className="wb-design-field">
              {t(strings.workbench.designIndex.artifactsLabel)}
              <input
                type="file"
                multiple
                onChange={(event) => onCaptureArtifactsChange(Array.from(event.target.files ?? []))}
              />
            </label>
          </div>
          {(captureManifest || captureArtifacts.length > 0) && (
            <div
              className="wb-design-selected-files"
              role="status"
              aria-live="polite"
              aria-label={t(strings.workbench.designIndex.captureSelectedFiles)}
            >
              <strong>{t(strings.workbench.designIndex.captureSelectedFiles)}</strong>
              <ul>
                {captureManifest && <li>{captureManifest.name}</li>}
                {captureArtifacts.map((file, index) => (
                  <li key={`${file.name}-${index}`}>{file.name}</li>
                ))}
              </ul>
            </div>
          )}
          <p className="wb-muted-2 wb-small">{t(strings.workbench.designIndex.captureNote)}</p>
          <WbButton
            variant="primary"
            size="sm"
            disabled={startingFlow !== null || !captureManifest || captureArtifacts.length === 0}
            onClick={onStartFromCapture}
          >
            {startingFlow === "capture"
              ? t(strings.workbench.designIndex.validatingCapture)
              : t(strings.workbench.designIndex.createRevision)}
            <ArrowRight aria-hidden size={13} />
          </WbButton>
        </PanelBody>
      </Panel>
    </section>
  );
}
