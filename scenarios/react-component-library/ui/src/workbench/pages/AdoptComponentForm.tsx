import { CircleAlert, Layers, Sparkles } from "lucide-react";
import { useTranslation } from "../../i18n";
import { Banner, Chip, Field, Select, WbButton } from "../primitives";
import type {
  AdoptionAssetOption,
  AdoptionPreflight,
  MutationStatus,
  ScenarioOption,
  TextSetter,
} from "./CreateRouteFormTypes";

export function AdoptComponentForm({
  adoptComponent,
  setAdoptComponent,
  setPreflight,
  assets,
  adoptScenario,
  setAdoptScenario,
  scenarios,
  adoptVersion,
  setAdoptVersion,
  versions,
  check,
  link,
  preflight,
}: {
  adoptComponent: string;
  setAdoptComponent: TextSetter;
  setPreflight: (value: AdoptionPreflight | null) => void;
  assets: AdoptionAssetOption[];
  adoptScenario: string;
  setAdoptScenario: TextSetter;
  scenarios: ScenarioOption[];
  adoptVersion: string;
  setAdoptVersion: TextSetter;
  versions: string[];
  check: MutationStatus;
  link: MutationStatus;
  preflight: AdoptionPreflight | null;
}) {
  const { t } = useTranslation();
  return (
    <div className="wb-form-grid">
      <Field label={t("workbench.create.asset", { defaultValue: "Asset" })} htmlFor="adopt-asset">
        <Select
          id="adopt-asset"
          value={adoptComponent}
          onChange={(event) => {
            setAdoptComponent(event.target.value);
            setPreflight(null);
            const asset = assets.find((entry) => entry.id === event.target.value);
            setAdoptVersion(asset?.latestVersion || asset?.version || "");
          }}
        >
          <option value="">
            {t("workbench.create.chooseAsset", { defaultValue: "Choose an asset" })}
          </option>
          {assets.slice(0, 200).map((asset) => (
            <option key={asset.id} value={asset.id}>
              {asset.displayName} · {asset.latestVersion || asset.version || "draft"}
            </option>
          ))}
        </Select>
      </Field>
      <Field
        label={t("workbench.create.targetScenario", { defaultValue: "Target scenario" })}
        htmlFor="adopt-scenario"
      >
        <Select
          id="adopt-scenario"
          value={adoptScenario}
          onChange={(event) => {
            setAdoptScenario(event.target.value);
            setPreflight(null);
          }}
        >
          <option value="">
            {t("workbench.create.chooseScenario", { defaultValue: "Choose a scenario" })}
          </option>
          {scenarios.map((entry) => (
            <option key={entry.scenario} value={entry.scenario}>
              {entry.scenario}
            </option>
          ))}
        </Select>
      </Field>
      <Field
        label={t("workbench.create.versionLabel", { defaultValue: "Version" })}
        htmlFor="adopt-version"
      >
        <Select
          id="adopt-version"
          value={adoptVersion}
          onChange={(event) => setAdoptVersion(event.target.value)}
        >
          {versions.map((value) => (
            <option key={value} value={value}>
              {value}
            </option>
          ))}
        </Select>
      </Field>
      <div className="is-full wb-row">
        <WbButton
          variant="ghost"
          size="md"
          onClick={() => check.mutate()}
          disabled={check.isPending || !adoptComponent || !adoptScenario}
        >
          <Sparkles aria-hidden size={14} />
          {check.isPending
            ? t("workbench.create.checking", { defaultValue: "Checking…" })
            : t("workbench.create.preflight", { defaultValue: "Preflight adoption" })}
        </WbButton>
        <WbButton
          variant="primary"
          size="md"
          onClick={() => link.mutate()}
          disabled={link.isPending || !preflight || preflight.blocking}
        >
          <Layers aria-hidden size={14} />
          {link.isPending
            ? t("workbench.create.linking", { defaultValue: "Linking…" })
            : t("workbench.create.link", { defaultValue: "Link adoption" })}
        </WbButton>
      </div>
      {check.isError || link.isError ? (
        <div className="is-full">
          <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
            {(check.error ?? link.error) instanceof Error
              ? ((check.error ?? link.error) as Error).message
              : String(check.error ?? link.error)}
          </Banner>
        </div>
      ) : null}
      {preflight ? (
        <div className="is-full">
          <div className="wb-row" style={{ flexWrap: "wrap" }}>
            <Chip tone={preflight.blocking ? "red" : "green"}>
              {preflight.blocking
                ? t("workbench.create.blocked", { defaultValue: "Blocked" })
                : t("workbench.create.ready", { defaultValue: "Ready" })}
            </Chip>
            <Chip tone="ice">{preflight.styleFit}</Chip>
            <Chip>{preflight.dependencyVerdict}</Chip>
          </div>
          {preflight.unsatisfiedTokens.length > 0 ? (
            <Banner tone="amber" icon={<CircleAlert aria-hidden size={15} />}>
              {t("workbench.create.tokensMissing", {
                defaultValue: "Unsatisfied tokens: {{tokens}}",
                tokens: preflight.unsatisfiedTokens.join(", "),
              })}
            </Banner>
          ) : null}
          {preflight.warnings.map((warning) => (
            <p key={warning} className="wb-small wb-muted">
              {warning}
            </p>
          ))}
        </div>
      ) : null}
    </div>
  );
}
