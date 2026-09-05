/** @vrooliComponentSource overlays.dialog */
import { FormEvent, KeyboardEvent, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { componentsClient } from "../../api/components";
import { Button } from "@vrooli/react-component-library/Button/2";
import { Dialog } from "@vrooli/react-component-library/Dialog/1";
import { Input } from "@vrooli/react-component-library/Input/1";
import { Textarea } from "@vrooli/react-component-library/Textarea/1";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";
import { assetPath } from "../../routes";

interface CreateComponentDialogProps {
  onClose: () => void;
}

export function CreateComponentDialog({ onClose }: CreateComponentDialogProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [slug, setSlug] = useState("");
  const [libraryId, setLibraryId] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [description, setDescription] = useState("");
  const [tags, setTags] = useState<string[]>([]);
  const [tagDraft, setTagDraft] = useState("");
  const [initialVersion, setInitialVersion] = useState("0.1.0");
  const [fileName, setFileName] = useState("");
  const [initialSource, setInitialSource] = useState("");

  const mutation = useMutation({
    mutationFn: (submittedTags: string[]) =>
      componentsClient.initializeComponent({
        slug,
        libraryId,
        displayName,
        description,
        tags: submittedTags,
        initialVersion,
        fileName,
        initialSource,
      }),
    onSuccess: async (resp) => {
      await queryClient.invalidateQueries({ queryKey: ["components"] });
      onClose();
      if (resp.component?.id) {
        void navigate(assetPath(resp.component.id));
      }
    },
  });

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const submittedTags = [
      ...new Set([
        ...tags,
        ...tagDraft
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
      ]),
    ];
    setTags(submittedTags);
    setTagDraft("");
    mutation.mutate(submittedTags);
  };

  const addTags = (value: string) => {
    const additions = value
      .split(",")
      .map((tag) => tag.trim())
      .filter(Boolean);
    if (additions.length > 0) setTags((current) => [...new Set([...current, ...additions])]);
    setTagDraft("");
  };
  const tagKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      addTags(tagDraft);
    }
  };

  return (
    <Dialog
      open
      onClose={onClose}
      closeLabel={t(strings.components.create.cancel)}
      title={t(strings.components.create.title)}
      className="max-w-2xl"
    >
      <div data-testid={selectors.components.create.dialog}>
        <form onSubmit={submit} className="grid gap-space-xs">
          <div className="grid grid-cols-1 gap-space-2xs md:grid-cols-2">
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.slug)}
              <Input
                data-testid={selectors.components.create.slug}
                value={slug}
                onChange={(e) => setSlug(e.target.value)}
                required
                className="mt-space-3xs"
              />
            </label>
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.libraryId)}
              <Input
                data-testid={selectors.components.create.libraryId}
                value={libraryId}
                onChange={(e) => setLibraryId(e.target.value)}
                placeholder={t(strings.components.create.libraryIdPlaceholder)}
                className="mt-space-3xs"
              />
            </label>
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.displayName)}
              <Input
                data-testid={selectors.components.create.displayName}
                value={displayName}
                onChange={(e) => setDisplayName(e.target.value)}
                className="mt-space-3xs"
              />
            </label>
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.version)}
              <Input
                data-testid={selectors.components.create.version}
                value={initialVersion}
                onChange={(e) => setInitialVersion(e.target.value)}
                className="mt-space-3xs"
              />
            </label>
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.fileName)}
              <Input
                data-testid={selectors.components.create.fileName}
                value={fileName}
                onChange={(e) => setFileName(e.target.value)}
                placeholder={t(strings.components.create.fileNamePlaceholder)}
                className="mt-space-3xs"
              />
            </label>
            <label className="block text-xs text-app-muted-foreground">
              {t(strings.components.create.tags)}
              <div className="mt-space-3xs flex flex-wrap gap-space-3xs rounded-control border border-app-border bg-app-background p-space-3xs">
                {tags.map((tag) => (
                  <button
                    key={tag}
                    type="button"
                    onClick={() => setTags((current) => current.filter((value) => value !== tag))}
                    aria-label={`Remove ${tag}`}
                    className="rounded bg-app-surface-muted px-space-2xs py-space-3xs text-xs"
                  >
                    {tag} ×
                  </button>
                ))}
                <Input
                  data-testid={selectors.components.create.tags}
                  value={tagDraft}
                  onChange={(e) => setTagDraft(e.target.value)}
                  onKeyDown={tagKeyDown}
                  onBlur={() => addTags(tagDraft)}
                  placeholder={t(strings.components.create.tagsPlaceholder)}
                  className="min-w-field-compact flex-1 border-0 bg-transparent"
                />
              </div>
            </label>
          </div>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.description)}
            <Textarea
              data-testid={selectors.components.create.description}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="mt-space-3xs min-h-surface-tiny"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.initialSource)}
            <Textarea
              data-testid={selectors.components.create.initialSource}
              value={initialSource}
              onChange={(e) => setInitialSource(e.target.value)}
              placeholder={t(strings.components.create.initialSourcePlaceholder)}
              className="mt-space-3xs min-h-surface font-mono text-xs"
            />
          </label>
          {mutation.error && (
            <p data-testid={selectors.components.create.error} className="text-sm text-app-danger">
              {errorMessage(mutation.error, t)}
            </p>
          )}
          <div className="flex justify-end gap-space-2xs">
            <Button
              type="button"
              variant="secondary"
              data-testid={selectors.components.create.cancel}
              onClick={onClose}
            >
              {t(strings.components.create.cancel)}
            </Button>
            <Button
              data-testid={selectors.components.create.submit}
              type="submit"
              disabled={mutation.isPending || slug.trim() === ""}
            >
              {mutation.isPending
                ? t(strings.components.create.creating)
                : t(strings.components.create.submit)}
            </Button>
          </div>
        </form>
      </div>
    </Dialog>
  );
}
