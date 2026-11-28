import { useTranslation } from "../../i18n";
import { Banner, Chip, Panel, PanelBody, PanelHead } from "../primitives";
import { CircleAlert } from "lucide-react";

export function CoveragePreparing({ embedded }: { embedded: boolean }) {
  const { t } = useTranslation();

  return (
    <div className={embedded ? "wb-page is-embedded" : "wb-page"} data-testid="coverage-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.coverage.kicker", { defaultValue: "Library · Coverage" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.coverage.title", { defaultValue: "How mature is the library?" })}
          </h1>
        </div>
        <div className="wb-actions">
          <Chip tone="amber">
            {t("workbench.coverage.preparingChip", { defaultValue: "Preparing" })}
          </Chip>
        </div>
      </div>
      <Banner tone="amber" icon={<CircleAlert aria-hidden size={16} />}>
        <b>
          {t("workbench.coverage.preparingTitle", {
            defaultValue: "Coverage is being prepared.",
          })}
        </b>{" "}
        {t("workbench.coverage.preparingBody", {
          defaultValue:
            "The first full gate pass runs the TypeScript toolchain and the capture evidence join; it can take a few minutes on a cold cache. This page retries automatically. Nothing is rendered as zero while it waits.",
        })}
      </Banner>
      <Panel>
        <PanelHead
          title={t("workbench.coverage.preparingPanel", {
            defaultValue: "What is being measured",
          })}
          meta={t("workbench.coverage.preparingMeta", { defaultValue: "64 gates · 3 tiers" })}
        />
        <PanelBody>
          <p className="wb-muted">
            {t("workbench.coverage.preparingDetail", {
              defaultValue:
                "Every catalog asset is checked against release integrity, asset quality and consumer use. The report stores which rung each asset achieved, which gates failed, and how much downstream work each failure blocks.",
            })}
          </p>
        </PanelBody>
      </Panel>
    </div>
  );
}
