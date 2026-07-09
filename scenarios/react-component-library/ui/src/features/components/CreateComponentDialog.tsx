import { FormEvent, useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { componentsClient } from "../../api/components";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Textarea } from "../../components/ui/textarea";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { errorMessage } from "../../lib/errorMessage";

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
  const [tags, setTags] = useState("");
  const [initialVersion, setInitialVersion] = useState("0.1.0");
  const [fileName, setFileName] = useState("");
  const [initialSource, setInitialSource] = useState("");

  const mutation = useMutation({
    mutationFn: () =>
      componentsClient.initializeComponent({
        slug,
        libraryId,
        displayName,
        description,
        tags: tags
          .split(",")
          .map((tag) => tag.trim())
          .filter(Boolean),
        initialVersion,
        fileName,
        initialSource,
      }),
    onSuccess: async (resp) => {
      await queryClient.invalidateQueries({ queryKey: ["components"] });
      onClose();
      if (resp.component?.id) {
        void navigate(`/components/${resp.component.id}`);
      }
    },
  });

  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate();
  };

  return (
    <div
      data-testid={selectors.components.create.dialog}
      className="mt-4 rounded-lg border border-app-border bg-app-surface p-4"
    >
      <form onSubmit={submit} className="grid gap-3">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-sm font-medium text-app-foreground">
            {t(strings.components.create.title)}
          </h3>
          <Button
            type="button"
            variant="secondary"
            className="h-8 px-3 text-xs"
            onClick={onClose}
            data-testid={selectors.components.create.cancel}
          >
            {t(strings.components.create.cancel)}
          </Button>
        </div>
        <div className="grid grid-cols-1 gap-2 md:grid-cols-2">
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.slug)}
            <Input
              data-testid={selectors.components.create.slug}
              value={slug}
              onChange={(e) => setSlug(e.target.value)}
              required
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.libraryId)}
            <Input
              data-testid={selectors.components.create.libraryId}
              value={libraryId}
              onChange={(e) => setLibraryId(e.target.value)}
              placeholder={t(strings.components.create.libraryIdPlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.displayName)}
            <Input
              data-testid={selectors.components.create.displayName}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.version)}
            <Input
              data-testid={selectors.components.create.version}
              value={initialVersion}
              onChange={(e) => setInitialVersion(e.target.value)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.fileName)}
            <Input
              data-testid={selectors.components.create.fileName}
              value={fileName}
              onChange={(e) => setFileName(e.target.value)}
              placeholder={t(strings.components.create.fileNamePlaceholder)}
              className="mt-1"
            />
          </label>
          <label className="block text-xs text-app-muted-foreground">
            {t(strings.components.create.tags)}
            <Input
              data-testid={selectors.components.create.tags}
              value={tags}
              onChange={(e) => setTags(e.target.value)}
              placeholder={t(strings.components.create.tagsPlaceholder)}
              className="mt-1"
            />
          </label>
        </div>
        <label className="block text-xs text-app-muted-foreground">
          {t(strings.components.create.description)}
          <Textarea
            data-testid={selectors.components.create.description}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="mt-1 min-h-20"
          />
        </label>
        <label className="block text-xs text-app-muted-foreground">
          {t(strings.components.create.initialSource)}
          <Textarea
            data-testid={selectors.components.create.initialSource}
            value={initialSource}
            onChange={(e) => setInitialSource(e.target.value)}
            placeholder={t(strings.components.create.initialSourcePlaceholder)}
            className="mt-1 min-h-32 font-mono text-xs"
          />
        </label>
        {mutation.error && (
          <p data-testid={selectors.components.create.error} className="text-sm text-app-danger">
            {errorMessage(mutation.error, t)}
          </p>
        )}
        <div className="flex justify-end">
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
  );
}
