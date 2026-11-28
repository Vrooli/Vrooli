import { CircleAlert, FileCode2 } from "lucide-react";
import { useTranslation } from "../../i18n";
import { Banner, Field, Select, TextArea, TextInput, WbButton } from "../primitives";
import type { MutationStatus, ScenarioOption, TextSetter } from "./CreateRouteFormTypes";

export function ExtractComponentForm({
  sourceScenario,
  setSourceScenario,
  scenarios,
  sourceFile,
  setSourceFile,
  displayName,
  setDisplayName,
  slot,
  setSlot,
  initialVersion,
  setInitialVersion,
  slug,
  setSlug,
  description,
  setDescription,
  tags,
  setTags,
  extract,
}: {
  sourceScenario: string;
  setSourceScenario: TextSetter;
  scenarios: ScenarioOption[];
  sourceFile: string;
  setSourceFile: TextSetter;
  displayName: string;
  setDisplayName: TextSetter;
  slot: string;
  setSlot: TextSetter;
  initialVersion: string;
  setInitialVersion: TextSetter;
  slug: string;
  setSlug: TextSetter;
  description: string;
  setDescription: TextSetter;
  tags: string;
  setTags: TextSetter;
  extract: MutationStatus;
}) {
  const { t } = useTranslation();
  return (
    <div className="wb-form-grid">
      <Field
        label={t("workbench.create.sourceScenario", { defaultValue: "Source scenario" })}
        htmlFor="extract-scenario"
      >
        <Select
          id="extract-scenario"
          value={sourceScenario}
          onChange={(event) => setSourceScenario(event.target.value)}
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
        label={t("workbench.create.sourcePath", { defaultValue: "Source path" })}
        htmlFor="extract-path"
      >
        <TextInput
          id="extract-path"
          value={sourceFile}
          onChange={(event) => setSourceFile(event.target.value)}
          placeholder={t("workbench.create.sourcePlaceholder", {
            defaultValue: "ui/src/components/AgentCard.tsx",
          })}
        />
      </Field>
      <Field
        label={t("workbench.create.assetName", { defaultValue: "Asset name" })}
        htmlFor="extract-name"
      >
        <TextInput
          id="extract-name"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
        />
      </Field>
      <Field label={t("workbench.create.slot", { defaultValue: "Slot" })} htmlFor="extract-slot">
        <TextInput
          id="extract-slot"
          value={slot}
          onChange={(event) => setSlot(event.target.value)}
        />
      </Field>
      <Field
        label={t("workbench.create.version", { defaultValue: "Initial version" })}
        htmlFor="extract-version"
      >
        <TextInput
          id="extract-version"
          value={initialVersion}
          onChange={(event) => setInitialVersion(event.target.value)}
        />
      </Field>
      <Field label={t("workbench.create.slug", { defaultValue: "Slug" })} htmlFor="extract-slug">
        <TextInput
          id="extract-slug"
          value={slug}
          onChange={(event) => setSlug(event.target.value)}
        />
      </Field>
      <div className="is-full">
        <Field
          label={t("workbench.create.description", { defaultValue: "Description" })}
          htmlFor="extract-desc"
        >
          <TextArea
            id="extract-desc"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>
      </div>
      <div className="is-full">
        <Field
          label={t("workbench.create.provenance", {
            defaultValue: "Why this belongs in the library",
          })}
          htmlFor="extract-tags"
        >
          <TextInput
            id="extract-tags"
            value={tags}
            onChange={(event) => setTags(event.target.value)}
            placeholder={t("workbench.create.tagsPlaceholder", {
              defaultValue: "data-display, card",
            })}
          />
        </Field>
      </div>
      {extract.isError ? (
        <div className="is-full">
          <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
            {extract.error instanceof Error ? extract.error.message : String(extract.error)}
          </Banner>
        </div>
      ) : null}
      <div className="is-full wb-row">
        <WbButton
          variant="primary"
          size="md"
          onClick={() => extract.mutate()}
          disabled={extract.isPending || !sourceScenario || !sourceFile || !displayName}
        >
          <FileCode2 aria-hidden size={14} />
          {extract.isPending
            ? t("workbench.create.extracting", { defaultValue: "Extracting…" })
            : t("workbench.create.extractAction", { defaultValue: "Extract draft" })}
        </WbButton>
      </div>
    </div>
  );
}
