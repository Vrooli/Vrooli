import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery } from "@tanstack/react-query";

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
} from "../../api/ops";
import { errorMessage } from "../../lib/errorMessage";
import { BeforeAfter } from "./BeforeAfter";
import { OP_SPECS, opSpec, type OpField } from "./opSpecs";
import { defaultParamsFor, toRequestParams } from "./opParams";
import { useOpStack, type OpRunner } from "./useOpStack";

const OPS_QUERY_KEY = ["operations"] as const;

/**
 * The production runner: execute the op via the REST ops endpoint, then
 * materialize the resulting image as a File so the next op composes on top.
 * Metadata-read ops are rejected here — the op-stack is image-to-image only.
 */
const liveRunner: OpRunner = async (operation, params, input) => {
  const result = await runOp(operation, input, toRequestParams(params));
  if (result.kind !== "image") {
    throw new Error("op-stack expects an image result");
  }
  const blob = await fetch(result.url).then((r) => r.blob());
  const outputFile = new File([blob], `step.${result.format || "png"}`, {
    type: blob.type || "image/png",
  });
  return { result, outputFile };
};

/**
 * OpStackCard is the non-destructive editor (req 21): upload a source image,
 * apply deterministic ops one at a time onto a stack, undo/redo, and compare
 * the original against the current result with a before/after slider. The base
 * image is never mutated; every step composes API calls headlessly.
 */
export function OpStackCard() {
  const { t } = useTranslation();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const stack = useOpStack(liveRunner);

  const [operation, setOperation] = useState("");
  const [params, setParams] = useState<OpParamValues>({});

  const opsQuery = useQuery({ queryKey: OPS_QUERY_KEY, queryFn: listOperations });

  const operations: OperationInfo[] = useMemo(
    () => (opsQuery.data?.operations ?? []).filter((op) => op.name in OP_SPECS),
    [opsQuery.data],
  );

  useEffect(() => {
    if (!operation && operations.length > 0) {
      const first = operations[0]?.name ?? "";
      setOperation(first);
      setParams(defaultParamsFor(first));
    }
  }, [operation, operations]);

  const onSelectOperation = (next: string) => {
    setOperation(next);
    setParams(defaultParamsFor(next));
  };

  const setParam = useCallback(
    (name: string, value: string | number | boolean) =>
      setParams((prev) => ({ ...prev, [name]: value })),
    [],
  );

  const onApply = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (operation) {
      void stack.apply(operation, params);
    }
  };

  const spec = opSpec(operation);

  const renderField = (field: OpField) => {
    const value = params[field.name];
    const id = `stack-field-${field.name}`;
    if (field.kind === "checkbox") {
      return (
        <label key={field.name} className="flex items-center gap-2 text-sm text-slate-200">
          <input
            id={id}
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
            type={field.kind === "number" ? "number" : "text"}
            value={String(value ?? "")}
            onChange={(e) =>
              setParam(field.name, field.kind === "number" ? Number(e.target.value) : e.target.value)
            }
          />
        )}
      </div>
    );
  };

  return (
    <section
      data-testid={selectors.editor.stack.card}
      aria-label={t(strings.editor.stack.title)}
      className="mt-4 rounded-xl border border-white/10 bg-black/20 p-4"
    >
      <h2 className="text-sm font-medium text-slate-400">{t(strings.editor.stack.title)}</h2>

      <div className="mt-3">
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          className="flex w-full flex-col items-start rounded-lg border border-dashed border-white/20 p-4 text-left text-sm text-slate-300 hover:border-white/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-white/40"
        >
          <span className="font-medium text-slate-200">{t(strings.editor.uploadLabel)}</span>
          <span className="mt-1 text-xs text-slate-400">
            {stack.base ? stack.base.file.name : t(strings.editor.uploadHint)}
          </span>
        </button>
        <input
          data-testid={selectors.editor.stack.fileInput}
          ref={fileInputRef}
          type="file"
          accept="image/*"
          capture="environment"
          aria-label={t(strings.editor.uploadLabel)}
          onChange={(e) => {
            const next = e.target.files?.[0];
            if (next) {
              stack.setBase(next);
            }
          }}
          className="sr-only"
        />
      </div>

      {opsQuery.data && (
        <form onSubmit={onApply} className="mt-4 flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <label htmlFor={selectors.editor.stack.operationSelect} className="text-xs text-slate-400">
              {t(strings.editor.operationLabel)}
            </label>
            <select
              id={selectors.editor.stack.operationSelect}
              data-testid={selectors.editor.stack.operationSelect}
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

          <div className="flex flex-wrap gap-2">
            <Button
              data-testid={selectors.editor.stack.applyButton}
              type="submit"
              disabled={!stack.base || !operation || stack.applying}
            >
              {stack.applying ? t(strings.editor.stack.applying) : t(strings.editor.stack.apply)}
            </Button>
            <Button
              data-testid={selectors.editor.stack.undoButton}
              type="button"
              onClick={stack.undo}
              disabled={!stack.canUndo}
            >
              {t(strings.editor.stack.undo)}
            </Button>
            <Button
              data-testid={selectors.editor.stack.redoButton}
              type="button"
              onClick={stack.redo}
              disabled={!stack.canRedo}
            >
              {t(strings.editor.stack.redo)}
            </Button>
            <Button
              data-testid={selectors.editor.stack.resetButton}
              type="button"
              onClick={stack.reset}
              disabled={stack.entries.length === 0}
            >
              {t(strings.editor.stack.reset)}
            </Button>
          </div>
        </form>
      )}

      {stack.error != null && (
        <p data-testid={selectors.editor.stack.error} className="mt-2 text-red-400">
          {errorMessage(stack.error, t)}
        </p>
      )}

      <h3 className="mt-4 text-xs font-semibold uppercase text-slate-500">
        {t(strings.editor.stack.stepsHeading)}
      </h3>
      {stack.entries.length === 0 ? (
        <p data-testid={selectors.editor.stack.empty} className="mt-1 text-xs text-slate-400">
          {t(strings.editor.stack.empty)}
        </p>
      ) : (
        <ol data-testid={selectors.editor.stack.list} className="mt-1 space-y-1 text-xs text-slate-300">
          {stack.entries.map((entry, index) => (
            <li key={`${entry.operation}-${index}`} className="rounded border border-white/10 px-2 py-1">
              {t(strings.editor.stack.step, { index: index + 1, operation: entry.operation })}
            </li>
          ))}
        </ol>
      )}

      {stack.base && (
        <div data-testid={selectors.editor.stack.preview} className="mt-4">
          {stack.entries.length > 0 && stack.previewUrl ? (
            <BeforeAfter beforeUrl={stack.base.url} afterUrl={stack.previewUrl} />
          ) : (
            <img
              src={stack.base.url}
              alt={t(strings.editor.originalLabel)}
              className="max-h-64 w-full rounded-lg border border-white/10 object-contain"
            />
          )}
        </div>
      )}
    </section>
  );
}
