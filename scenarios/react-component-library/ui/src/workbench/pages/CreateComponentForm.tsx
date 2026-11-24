import { CircleAlert, Plus } from "lucide-react";
import { useTranslation } from "../../i18n";
import { Banner, Field, TextArea, TextInput, WbButton } from "../primitives";
import type { MutationStatus, TextSetter } from "./CreateRouteFormTypes";

export function CreateComponentForm({
  slug,
  setSlug,
  libraryId,
  setLibraryId,
  displayName,
  setDisplayName,
  description,
  setDescription,
  tags,
  setTags,
  initialVersion,
  setInitialVersion,
  create,
}: {
  slug: string;
  setSlug: TextSetter;
  libraryId: string;
  setLibraryId: TextSetter;
  displayName: string;
  setDisplayName: TextSetter;
  description: string;
  setDescription: TextSetter;
  tags: string;
  setTags: TextSetter;
  initialVersion: string;
  setInitialVersion: TextSetter;
  create: MutationStatus;
}) {
  const { t } = useTranslation();
  return (
    <div className="wb-form-grid">
      <Field label={t("workbench.create.slug", { defaultValue: "Slug" })} htmlFor="create-slug">
        <TextInput
          id="create-slug"
          value={slug}
          onChange={(event) => setSlug(event.target.value)}
          placeholder={t("workbench.create.slugPlaceholder", {
            defaultValue: "AgentCard",
          })}
        />
      </Field>
      <Field
        label={t("workbench.create.libraryId", { defaultValue: "Library id" })}
        htmlFor="create-library"
      >
        <TextInput
          id="create-library"
          value={libraryId}
          onChange={(event) => setLibraryId(event.target.value)}
          placeholder={t("workbench.create.libraryPlaceholder", {
            defaultValue: "react-component-library:AgentCard",
          })}
        />
      </Field>
      <Field
        label={t("workbench.create.displayName", { defaultValue: "Display name" })}
        htmlFor="create-name"
      >
        <TextInput
          id="create-name"
          value={displayName}
          onChange={(event) => setDisplayName(event.target.value)}
        />
      </Field>
      <Field
        label={t("workbench.create.version", { defaultValue: "Initial version" })}
        htmlFor="create-version"
      >
        <TextInput
          id="create-version"
          value={initialVersion}
          onChange={(event) => setInitialVersion(event.target.value)}
        />
      </Field>
      <div className="is-full">
        <Field
          label={t("workbench.create.description", { defaultValue: "Description" })}
          htmlFor="create-desc"
        >
          <TextArea
            id="create-desc"
            value={description}
            onChange={(event) => setDescription(event.target.value)}
          />
        </Field>
      </div>
      <div className="is-full">
        <Field label={t("workbench.create.tags", { defaultValue: "Tags" })} htmlFor="create-tags">
          <TextInput
            id="create-tags"
            value={tags}
            onChange={(event) => setTags(event.target.value)}
            placeholder={t("workbench.create.tagsPlaceholder", {
              defaultValue: "data-display, card",
            })}
          />
        </Field>
      </div>
      {create.isError ? (
        <div className="is-full">
          <Banner tone="red" icon={<CircleAlert aria-hidden size={15} />}>
            {create.error instanceof Error ? create.error.message : String(create.error)}
          </Banner>
        </div>
      ) : null}
      <div className="is-full">
        <p className="wb-muted-2 wb-small" style={{ marginBottom: 8 }}>
          {t("workbench.create.requiredHint", {
            defaultValue: "Slug and display name are required before a draft can be created.",
          })}
        </p>
      </div>
      <div className="is-full wb-row">
        <WbButton
          variant="primary"
          size="md"
          onClick={() => create.mutate()}
          disabled={create.isPending || !slug || !displayName}
        >
          <Plus aria-hidden size={14} />
          {create.isPending
            ? t("workbench.create.creating", { defaultValue: "Creating…" })
            : t("workbench.create.createAction", { defaultValue: "Create draft" })}
        </WbButton>
      </div>
    </div>
  );
}
