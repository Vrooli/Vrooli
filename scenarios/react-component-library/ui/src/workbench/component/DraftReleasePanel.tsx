import { Check, Play, UploadCloud, X } from "lucide-react";

import type { ComponentVersionCheckResult } from "../../api/components";
import { useTranslation } from "../../i18n";
import { Chip, Empty, Field, TextArea, WbButton } from "../primitives";

type DraftCheck = ComponentVersionCheckResult["checks"][number];

interface DraftReleasePanelState {
  dirty: boolean;
  checks: DraftCheck[] | null;
  checksPassed: boolean;
  runCheckPending: boolean;
  publishing: boolean;
  publishPending: boolean;
  changelog: string;
}

interface DraftReleasePanelActions {
  runChecks: () => void;
  preparePublish: () => void;
  cancelPublish: () => void;
  publish: () => void;
  changeChangelog: (value: string) => void;
}

interface DraftReleasePanelProps {
  state: DraftReleasePanelState;
  actions: DraftReleasePanelActions;
}

export function DraftReleasePanel({ state, actions }: DraftReleasePanelProps) {
  const { t } = useTranslation();

  return (
    <div className="wb-source-actions">
      <div className="wb-row-between">
        <span className="wb-caption">
          {t("workbench.component.draftChecks", { defaultValue: "Draft checks" })}
        </span>
        <WbButton
          variant="ghost"
          size="sm"
          disabled={state.dirty || state.runCheckPending}
          onClick={actions.runChecks}
        >
          <Play aria-hidden size={12} />
          {state.runCheckPending
            ? t("workbench.component.runningChecks", { defaultValue: "Checking…" })
            : t("workbench.component.runChecks", { defaultValue: "Run checks" })}
        </WbButton>
      </div>
      {state.dirty ? (
        <p className="wb-small wb-muted-2">
          {t("workbench.component.checkDirtyHint", {
            defaultValue: "Save the draft before checking so the check reads the bytes you can see.",
          })}
        </p>
      ) : null}
      {state.checks ? (
        <ul className="wb-checklist">
          {state.checks.map((check, index) => (
            <li key={`${check.stage}-${index}`}>
              <span
                className={`wb-checkdot ${check.verdict === "passed" ? "is-pass" : "is-fail"}`}
              >
                {check.verdict === "passed" ? (
                  <Check aria-hidden size={11} />
                ) : (
                  <X aria-hidden size={11} />
                )}
              </span>
              <span style={{ minWidth: 0 }}>
                <b className="wb-small">{check.stage}</b>
                <small className="wb-muted-2 wb-small" style={{ display: "block" }}>
                  {check.message}
                  {check.verdict !== "passed" && check.remediation
                    ? ` — ${check.remediation}`
                    : ""}
                </small>
              </span>
            </li>
          ))}
        </ul>
      ) : null}
      {state.checks && state.checks.length === 0 ? (
        <Empty title={t("workbench.component.noChecksTitle", { defaultValue: "No checks declared" })}>
          {t("workbench.component.draftNoChecksBody", {
            defaultValue: "The check owner returned no stages for this draft.",
          })}
        </Empty>
      ) : null}

      {state.checksPassed ? (
        <div className="wb-source-publish">
          <div className="wb-row-between">
            <span className="wb-caption">
              {t("workbench.component.publishDraft", { defaultValue: "Publish draft" })}
            </span>
            <Chip tone="green">
              {t("workbench.component.checksPassed", { defaultValue: "Checks passed" })}
            </Chip>
          </div>
          {state.publishing ? (
            <Field
              label={t("workbench.component.changelog", { defaultValue: "Changelog" })}
              htmlFor="rcl-draft-changelog"
              hint={t("workbench.component.changelogHint", {
                defaultValue: "Stored with the release. Markdown is preserved.",
              })}
            >
              <TextArea
                id="rcl-draft-changelog"
                rows={4}
                value={state.changelog}
                placeholder={t("workbench.component.changelogPlaceholder", {
                  defaultValue: "What changed in this version?",
                })}
                onChange={(event) => actions.changeChangelog(event.target.value)}
              />
            </Field>
          ) : null}
          <div className="wb-row" style={{ gap: 8, flexWrap: "wrap" }}>
            {state.publishing ? (
              <>
                <WbButton variant="ghost" size="sm" onClick={actions.cancelPublish}>
                  {t("workbench.component.cancel", { defaultValue: "Cancel" })}
                </WbButton>
                <WbButton
                  variant="primary"
                  size="sm"
                  disabled={state.publishPending || state.dirty}
                  onClick={actions.publish}
                >
                  <UploadCloud aria-hidden size={13} />
                  {state.publishPending
                    ? t("workbench.component.publishing", { defaultValue: "Publishing…" })
                    : t("workbench.component.publish", { defaultValue: "Publish release" })}
                </WbButton>
              </>
            ) : (
              <WbButton
                variant="primary"
                size="sm"
                disabled={state.dirty}
                onClick={actions.preparePublish}
              >
                <UploadCloud aria-hidden size={13} />
                {t("workbench.component.preparePublish", { defaultValue: "Publish…" })}
              </WbButton>
            )}
          </div>
        </div>
      ) : null}
    </div>
  );
}
