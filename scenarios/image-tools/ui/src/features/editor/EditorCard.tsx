import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import {
  listOperations,
  runOp,
  type OpParamValues,
  type OperationInfo,
  type RunOpResult,
} from "../../api/ops";
import { errorMessage } from "../../lib/errorMessage";
import { OP_SPECS, opSpec, type OpField } from "./opSpecs";

const OPS_QUERY_KEY = ["operations"] as const;

/** Build the controlled-form default state for an op from its spec. */
const defaultParamsFor = (operation: string): OpParamValues => {
  const spec = opSpec(operation);
  const values: OpParamValues = {};
  for (const field of spec?.fields ?? []) {
    values[field.name] = field.default;
  }
  return values;
};

/**
 * Coerce the form's controlled values to the protojson shape the server
 * expects. `target_bytes` is an int64 → protojson encodes it as a string;
 * everything else maps 1:1 from the typed form state.
 */
const toRequestParams = (values: OpParamValues): OpParamValues => {
  const out: OpParamValues = {};
  for (const [key, value] of Object.entries(values)) {
    out[key] = key === "target_bytes" ? String(value) : value;
  }
  return out;
};

/**
 * EditorCard is the deterministic-ops vertical slice: upload an image,
 * pick an operation discovered via `ListOperations`, fill the typed params
 * form, run it, and compare the original against the result (or read the
 * returned metadata JSON). Object URLs are revoked on replacement and
 * unmount so blobs don't leak.
 */
export function EditorCard() {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [file, setFile] = useState<File | null>(null);
  const [originalUrl, setOriginalUrl] = useState<string | null>(null);
  const [overlay, setOverlay] = useState<File | null>(null);
  const [operation, setOperation] = useState<string>("");
  const [params, setParams] = useState<OpParamValues>({});
  const [result, setResult] = useState<RunOpResult | null>(null);

  const opsQuery = useQuery({
    queryKey: OPS_QUERY_KEY,
    queryFn: listOperations,
  });

  const operations: OperationInfo[] = useMemo(
    () => (opsQuery.data?.operations ?? []).filter((op) => op.name in OP_SPECS),
    [opsQuery.data],
  );

  // Pick the first known operation once discovery resolves.
  useEffect(() => {
    if (!operation && operations.length > 0) {
      const first = operations[0]?.name ?? "";
      setOperation(first);
      setParams(defaultParamsFor(first));
    }
  }, [operation, operations]);

  // Revoke object URLs when they are replaced or the component unmounts.
  useEffect(() => () => {
    if (originalUrl) {
      URL.revokeObjectURL(originalUrl);
    }
  }, [originalUrl]);

  useEffect(() => () => {
    if (result?.kind === "image") {
      URL.revokeObjectURL(result.url);
    }
  }, [result]);

  const spec = opSpec(operation);

  const runMutation = useMutation({
    mutationFn: () => {
      if (!file) {
        throw new Error("no file");
      }
      return runOp(operation, file, toRequestParams(params), {
        overlay: spec?.acceptsOverlay ? overlay ?? undefined : undefined,
      });
    },
    onSuccess: (next) => {
      setResult((prev) => {
        if (prev?.kind === "image") {
          URL.revokeObjectURL(prev.url);
        }
        return next;
      });
    },
  });

  const onSelectFile = (next: File | null) => {
    if (!next) {
      return;
    }
    setFile(next);
    setOriginalUrl((prev) => {
      if (prev) {
        URL.revokeObjectURL(prev);
      }
      return URL.createObjectURL(next);
    });
    setResult((prev) => {
      if (prev?.kind === "image") {
        URL.revokeObjectURL(prev.url);
      }
      return null;
    });
  };

  const onSelectOperation = (next: string) => {
    setOperation(next);
    setParams(defaultParamsFor(next));
  };

  const setParam = (name: string, value: string | number | boolean) =>
    setParams((prev) => ({ ...prev, [name]: value }));

  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (file) {
      runMutation.mutate();
    }
  };

  const onDrop = (event: React.DragEvent<HTMLElement>) => {
    event.preventDefault();
    onSelectFile(event.dataTransfer.files[0] ?? null);
  };

  const fieldId = (name: string) => `editor-field-${name}`;

  const renderField = (field: OpField) => {
    const value = params[field.name];
    const id = fieldId(field.name);
    const testId = selectors.editor.fieldInput({ name: field.name });

    if (field.kind === "checkbox") {
      return (
        <label key={field.name} className="flex items-center gap-2 text-sm text-slate-200">
          <input
            id={id}
            data-testid={testId}
            type="checkbox"
            checked={Boolean(value)}
            onChange={(e) => setParam(field.name, e.target.checked)}
            className="h-4 w-4"
          />
          {t(field.labelKey)}
        </label>
      );
    }

    return (
      <div key={field.name} className="flex flex-col gap-1">
        <label htmlFor={id} className="text-xs text-slate-400">
          {t(field.labelKey)}
        </label>
        {field.kind === "select" ? (
          <select
            id={id}
            data-testid={testId}
            value={String(value ?? "")}
            onChange={(e) => setParam(field.name, e.target.value)}
            className="h-10 rounded-md border border-white/20 bg-white/5 px-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-white/40"
          >
            {field.options?.map((option) => (
              <option key={option} value={option} className="bg-slate-900">
                {option}
              </option>
            ))}
          </select>
        ) : (
          <Input
            id={id}
            data-testid={testId}
            type={field.kind === "number" ? "number" : "text"}
            value={String(value ?? "")}
            onChange={(e) =>
              setParam(
                field.name,
                field.kind === "number" ? Number(e.target.value) : e.target.value,
              )
            }
          />
        )}
      </div>
    );
  };

  return (
    <section
      data-testid={selectors.editor.card}
      aria-label={t(strings.editor.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.editor.title)}</h2>

      {opsQuery.isLoading && (
        <p data-testid={selectors.editor.loading} className="mt-2 text-slate-200">
          {t(strings.editor.loading)}
        </p>
      )}
      {opsQuery.error && (
        <p data-testid={selectors.editor.error} className="mt-2 text-red-400">
          {t(strings.editor.error)}
        </p>
      )}

      {opsQuery.data && (
        <form
          data-testid={selectors.editor.paramsForm}
          onSubmit={onSubmit}
          className="mt-3 flex flex-col gap-4"
        >
          <div>
            <button
              type="button"
              data-testid={selectors.editor.dropzone}
              onClick={() => fileInputRef.current?.click()}
              onDrop={onDrop}
              onDragOver={(e) => e.preventDefault()}
              className="flex w-full flex-col items-start rounded-lg border border-dashed border-white/20 p-4 text-left text-sm text-slate-300 hover:border-white/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
            >
              <span className="font-medium text-slate-200">{t(strings.editor.uploadLabel)}</span>
              <span className="mt-1 text-xs text-slate-400">
                {file ? file.name : t(strings.editor.uploadHint)}
              </span>
            </button>
            <input
              id={selectors.editor.fileInput}
              data-testid={selectors.editor.fileInput}
              ref={fileInputRef}
              type="file"
              accept="image/*"
              capture="environment"
              aria-label={t(strings.editor.uploadLabel)}
              onChange={(e) => onSelectFile(e.target.files?.[0] ?? null)}
              className="sr-only"
            />
          </div>

          <div className="flex flex-col gap-1">
            <label htmlFor={selectors.editor.operationSelect} className="text-xs text-slate-400">
              {t(strings.editor.operationLabel)}
            </label>
            <select
              id={selectors.editor.operationSelect}
              data-testid={selectors.editor.operationSelect}
              value={operation}
              onChange={(e) => onSelectOperation(e.target.value)}
              className="h-10 rounded-md border border-white/20 bg-white/5 px-3 text-sm text-white focus:outline-none focus:ring-2 focus:ring-white/40"
            >
              {operations.map((op) => (
                <option key={op.name} value={op.name} className="bg-slate-900">
                  {op.name}
                </option>
              ))}
            </select>
          </div>

          {spec && spec.fields.length > 0 && (
            <div className="grid grid-cols-2 gap-3">{spec.fields.map(renderField)}</div>
          )}

          {spec?.acceptsOverlay && (
            <div className="flex flex-col gap-1">
              <label htmlFor={selectors.editor.overlayInput} className="text-xs text-slate-400">
                {t(strings.editor.overlayLabel)}
              </label>
              <input
                id={selectors.editor.overlayInput}
                data-testid={selectors.editor.overlayInput}
                type="file"
                accept="image/*"
                onChange={(e) => setOverlay(e.target.files?.[0] ?? null)}
                className="block w-full text-xs text-slate-300 file:mr-3 file:rounded-control file:border-0 file:bg-app-primary file:px-3 file:py-2 file:text-app-primary-foreground"
              />
            </div>
          )}

          <Button
            data-testid={selectors.editor.runButton}
            type="submit"
            disabled={!file || !operation || runMutation.isPending}
          >
            {runMutation.isPending ? t(strings.editor.running) : t(strings.editor.run)}
          </Button>
        </form>
      )}

      {runMutation.error && (
        <p data-testid={selectors.editor.runError} className="mt-2 text-red-400">
          {errorMessage(runMutation.error, t)}
        </p>
      )}

      {!result && !runMutation.isPending && opsQuery.data && (
        <p data-testid={selectors.editor.empty} className="mt-4 text-slate-400">
          {t(strings.editor.empty)}
        </p>
      )}

      {(originalUrl || result) && (
        <div className="mt-4 grid grid-cols-1 gap-4 sm:grid-cols-2">
          {originalUrl && (
            <figure data-testid={selectors.editor.original} className="flex flex-col gap-2">
              <figcaption className="text-xs text-slate-400">
                {t(strings.editor.originalLabel)}
              </figcaption>
              <img
                src={originalUrl}
                alt={t(strings.editor.originalLabel)}
                className="max-h-64 w-full rounded-lg border border-white/10 object-contain"
              />
            </figure>
          )}

          {result?.kind === "image" && (
            <figure data-testid={selectors.editor.result} className="flex flex-col gap-2">
              <figcaption className="text-xs text-slate-400">
                {t(strings.editor.resultLabel)}
              </figcaption>
              <img
                data-testid={selectors.editor.resultImage}
                src={result.url}
                alt={t(strings.editor.resultLabel)}
                className="max-h-64 w-full rounded-lg border border-white/10 object-contain"
              />
              <p data-testid={selectors.editor.resultMeta} className="text-xs text-slate-500">
                {t(strings.editor.resultMeta, {
                  width: result.width,
                  height: result.height,
                  format: result.format,
                })}
              </p>
              <a
                data-testid={selectors.editor.downloadLink}
                href={result.url}
                download={`result.${result.format || "png"}`}
                className="text-sm text-app-primary underline"
              >
                {t(strings.editor.download)}
              </a>
            </figure>
          )}

          {result?.kind === "metadata" && (
            <figure data-testid={selectors.editor.result} className="flex flex-col gap-2">
              <figcaption className="text-xs text-slate-400">
                {t(strings.editor.metadataLabel)}
              </figcaption>
              <pre
                data-testid={selectors.editor.metadataOutput}
                className="max-h-64 overflow-auto rounded-lg border border-white/10 bg-black/40 p-3 text-xs text-slate-200"
              >
                {result.json}
              </pre>
            </figure>
          )}
        </div>
      )}
    </section>
  );
}
