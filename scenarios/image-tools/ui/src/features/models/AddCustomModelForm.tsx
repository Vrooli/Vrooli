import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { modelsClient } from "../../api/models";
import { errorMessage } from "../../lib/errorMessage";

/** Split a comma-separated operations field into a trimmed, non-empty list. */
const parseOperations = (raw: string): string[] =>
  raw
    .split(",")
    .map((s) => s.trim())
    .filter((s) => s.length > 0);

/**
 * AddCustomModelForm registers an operator-provided local/custom model via
 * AddCustomModel. The id, operations, and backend land on the embedded Model
 * message; local path / download URL are top-level request fields. On success
 * it invalidates the models list so the new [custom] entry appears.
 */
export function AddCustomModelForm() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();

  const [id, setId] = useState("");
  const [operations, setOperations] = useState("");
  const [backend, setBackend] = useState("");
  const [localPath, setLocalPath] = useState("");
  const [downloadUrl, setDownloadUrl] = useState("");

  const addMutation = useMutation({
    mutationFn: () =>
      modelsClient.addCustomModel({
        model: { id: id.trim(), operations: parseOperations(operations), backend: backend.trim() },
        localPath: localPath.trim(),
        downloadUrl: downloadUrl.trim(),
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["models"] });
      setId("");
      setOperations("");
      setBackend("");
      setLocalPath("");
      setDownloadUrl("");
    },
  });

  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (id.trim()) {
      addMutation.mutate();
    }
  };

  const fieldId = (key: string) => `add-custom-${key}`;

  return (
    <section
      aria-label={t(strings.models.addCustom.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.models.addCustom.title)}</h2>
      <form
        data-testid={selectors.models.addCustom.form}
        onSubmit={onSubmit}
        className="mt-3 flex flex-col gap-3"
      >
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("id")} className="text-xs text-slate-400">
            {t(strings.models.addCustom.idLabel)}
          </label>
          <Input
            id={fieldId("id")}
            data-testid={selectors.models.addCustom.id}
            value={id}
            onChange={(e) => setId(e.target.value)}
            required
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("operations")} className="text-xs text-slate-400">
            {t(strings.models.addCustom.operationsLabel)}
          </label>
          <Input
            id={fieldId("operations")}
            data-testid={selectors.models.addCustom.operations}
            value={operations}
            onChange={(e) => setOperations(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("backend")} className="text-xs text-slate-400">
            {t(strings.models.addCustom.backendLabel)}
          </label>
          <Input
            id={fieldId("backend")}
            data-testid={selectors.models.addCustom.backend}
            value={backend}
            onChange={(e) => setBackend(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("local-path")} className="text-xs text-slate-400">
            {t(strings.models.addCustom.localPathLabel)}
          </label>
          <Input
            id={fieldId("local-path")}
            data-testid={selectors.models.addCustom.localPath}
            value={localPath}
            onChange={(e) => setLocalPath(e.target.value)}
          />
        </div>
        <div className="flex flex-col gap-1">
          <label htmlFor={fieldId("download-url")} className="text-xs text-slate-400">
            {t(strings.models.addCustom.downloadUrlLabel)}
          </label>
          <Input
            id={fieldId("download-url")}
            data-testid={selectors.models.addCustom.downloadUrl}
            value={downloadUrl}
            onChange={(e) => setDownloadUrl(e.target.value)}
          />
        </div>
        <Button
          data-testid={selectors.models.addCustom.submit}
          type="submit"
          disabled={!id.trim() || addMutation.isPending}
        >
          {addMutation.isPending
            ? t(strings.models.addCustom.submitting)
            : t(strings.models.addCustom.submit)}
        </Button>
      </form>
      {addMutation.isSuccess && (
        <p data-testid={selectors.models.addCustom.success} className="mt-2 text-xs text-emerald-400">
          {t(strings.models.addCustom.success)}
        </p>
      )}
      {addMutation.error && (
        <p data-testid={selectors.models.addCustom.error} className="mt-2 text-xs text-red-400">
          {errorMessage(addMutation.error, t)}
        </p>
      )}
    </section>
  );
}
