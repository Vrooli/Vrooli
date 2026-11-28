import { useTranslation } from "../../i18n";
import { Panel, PanelBody, PanelHead } from "../primitives";

type Route = "create" | "extract" | "adopt";
const DOC_PATH = "docs/concepts/ASSET-DERIVATION.md";

export function CreateGuidance({
  initialVersion,
  route,
  adoptScenario,
  slot,
}: {
  initialVersion: string;
  route: Route;
  adoptScenario: string;
  slot: string;
}) {
  const { t } = useTranslation();
  return (
    <aside
      className="wb-stack"
      aria-label={t("workbench.create.help", { defaultValue: "Create guidance" })}
    >
      <Panel>
        <PanelHead
          title={t("workbench.create.next", { defaultValue: "What happens next" })}
          meta={t("workbench.create.steps", { defaultValue: "4 steps" })}
        />
        <PanelBody flush>
          <div className="wb-steps">
            {[
              [
                t("workbench.create.step1", { defaultValue: "Draft created" }),
                t("workbench.create.step1Body", {
                  defaultValue: "A mutable version folder and manifest pointer.",
                }),
              ],
              [
                t("workbench.create.step2", { defaultValue: "Derived artifacts" }),
                t("workbench.create.step2Body", {
                  defaultValue: "Story contract, dependency lock, exports.",
                }),
              ],
              [
                t("workbench.create.step3", { defaultValue: "Focused checks" }),
                t("workbench.create.step3Body", {
                  defaultValue: "Types, stories and catalog gates.",
                }),
              ],
              [
                t("workbench.create.step4", { defaultValue: "Publish successor" }),
                t("workbench.create.step4Body", {
                  defaultValue: "Immutable release with provenance.",
                }),
              ],
            ].map(([title, body], index) => (
              <div key={title} className="wb-step">
                <b>{index + 1}</b>
                <span>
                  {title}
                  <small>{body}</small>
                </span>
              </div>
            ))}
          </div>
        </PanelBody>
      </Panel>

      <Panel>
        <PanelHead
          title={t("workbench.create.summary", { defaultValue: "Summary" })}
          meta={t("workbench.create.beforeAnalysis", { defaultValue: "Before analysis" })}
        />
        <PanelBody>
          <div className="wb-summary">
            <div className="wb-row-between">
              <span>{t("workbench.create.owner", { defaultValue: "Owner" })}</span>
              <b>{t("workbench.create.ownerValue", { defaultValue: "Local operator" })}</b>
            </div>
            <div className="wb-row-between">
              <span>{t("workbench.create.policy", { defaultValue: "Policy" })}</span>
              <b>{t("workbench.create.policyValue", { defaultValue: "Governed" })}</b>
            </div>
            <div className="wb-row-between">
              <span>{t("workbench.create.draft", { defaultValue: "Draft" })}</span>
              <b>{initialVersion}</b>
            </div>
            <div className="wb-row-between">
              <span>{t("workbench.create.target", { defaultValue: "Target" })}</span>
              <b>
                {route === "adopt" ? adoptScenario || "—" : route === "extract" ? slot : "catalog"}
              </b>
            </div>
          </div>
          <p className="wb-muted-2 wb-small" style={{ marginTop: 10 }}>
            {t("workbench.create.summaryNote", {
              defaultValue:
                "No route publishes directly. The draft is inspectable, and the release is a separate governed step.",
            })}
          </p>
        </PanelBody>
      </Panel>

      <Panel>
        <PanelHead title={t("workbench.create.docs", { defaultValue: "Reference" })} />
        <PanelBody>
          <p className="wb-mono wb-small wb-muted">{DOC_PATH}</p>
          <p className="wb-small wb-muted" style={{ marginTop: 6 }}>
            {t("workbench.create.docsBody", {
              defaultValue:
                "The authored source loop — draft, validate, publish, compile, inspect — is unchanged by the workbench redesign.",
            })}
          </p>
        </PanelBody>
      </Panel>
    </aside>
  );
}
