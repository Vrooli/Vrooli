import { useEffect, useId, useMemo, useState } from "react";
import { Loader2, Play } from "lucide-react";
import { Panel } from "../../../components/ui/panel";
import { Button } from "../../../components/ui/button";
import { Input } from "@vrooli/react-component-library/Input/1.3.6";
import { Select } from "../../../components/ui/select";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import type { ApiError } from "../../../api/client";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";
import {
  audioOps,
  type AudioOpId,
  type AudioOpOptionLabelKey,
  type AudioOpParam,
  type AudioOpParamName,
  type AudioOpSpec,
} from "../audioOps.generated";
import {
  runAudioOp,
  type AudioOpValue,
  type AudioOpValues,
  type AudioOpOutput,
} from "../../../services/audioOps";

const FIELD_LABELS = {
  audio: strings.diagnostics.audioOps.fields.audio,
  output_format: strings.diagnostics.audioOps.fields.output_format,
  input_format: strings.diagnostics.audioOps.fields.input_format,
  sample_rate: strings.diagnostics.audioOps.fields.sample_rate,
  channels: strings.diagnostics.audioOps.fields.channels,
  bitrate: strings.diagnostics.audioOps.fields.bitrate,
  format: strings.diagnostics.audioOps.fields.format,
  start_seconds: strings.diagnostics.audioOps.fields.start_seconds,
  end_seconds: strings.diagnostics.audioOps.fields.end_seconds,
  sources: strings.diagnostics.audioOps.fields.sources,
  crossfade_seconds: strings.diagnostics.audioOps.fields.crossfade_seconds,
  chunk_seconds: strings.diagnostics.audioOps.fields.chunk_seconds,
  boundaries_seconds: strings.diagnostics.audioOps.fields.boundaries_seconds,
  fade_in_seconds: strings.diagnostics.audioOps.fields.fade_in_seconds,
  fade_out_seconds: strings.diagnostics.audioOps.fields.fade_out_seconds,
  gain_db: strings.diagnostics.audioOps.fields.gain_db,
  method: strings.diagnostics.audioOps.fields.method,
  target_lufs: strings.diagnostics.audioOps.fields.target_lufs,
} as const satisfies Record<AudioOpParamName, string>;

// textValue is the only safe stringification for a form value: File/File[]
// must never fall through to Object's "[object Object]" representation.
function textValue(value: AudioOpValue): string {
  if (typeof value === "string") return value;
  if (typeof value === "boolean") return value ? "true" : "";
  return "";
}

function defaultValues(op: AudioOpSpec): AudioOpValues {
  const values: AudioOpValues = {};
  for (const param of op.params) {
    if (param.control === "file" || param.control === "files") {
      values[param.name] = param.control === "files" ? [] : "";
    } else if (param.control === "checkbox") {
      values[param.name] = param.defaultValue === "true";
    } else {
      values[param.name] = param.defaultValue;
    }
  }
  return values;
}

function hasAllRequired(op: AudioOpSpec, values: AudioOpValues): boolean {
  const fieldsComplete = op.params.every((param) => {
    if (!param.required) return true;
    const value = values[param.name];
    switch (param.control) {
      case "file":
        return value instanceof File;
      case "files":
        return Array.isArray(value) && value.some((item) => item instanceof File);
      case "checkbox":
        return value === true;
      default:
        return textValue(value).trim() !== "";
    }
  });
  if (!fieldsComplete) return false;
  if (op.id === "split") {
    const chunkSeconds = Number(textValue(values.chunk_seconds).trim());
    const boundaries = textValue(values.boundaries_seconds)
      .split(",")
      .map((part) => part.trim())
      .filter((part) => part !== "")
      .map(Number);
    return (Number.isFinite(chunkSeconds) && chunkSeconds > 0) || boundaries.length > 0;
  }
  return true;
}

interface AudioOpsPanelProps {
  initialOperationId?: AudioOpId;
}

export function AudioOpsPanel({ initialOperationId = "transcode" }: AudioOpsPanelProps) {
  const [opId, setOpId] = useState<string>(initialOperationId);
  const op = audioOps.find((candidate) => candidate.id === opId) ?? audioOps[0];
  if (!op) return null;
  return <AudioOpsPanelBody key={op.id} op={op} onSelect={setOpId} />;
}

function AudioOpsPanelBody({ op, onSelect }: { op: AudioOpSpec; onSelect: (id: string) => void }) {
  const { t } = useTranslation();
  const [values, setValues] = useState<AudioOpValues>(() => defaultValues(op));
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<AudioOpOutput | null>(null);
  const [error, setError] = useState<ApiError | null>(null);

  const setValue = (name: string, value: AudioOpValue) =>
    setValues((prev) => ({ ...prev, [name]: value }));

  const downloads = useMemo(() => {
    if (!result) return [];
    if (result.kind === "audio") {
      return [makeDownload(result.bytes, result.contentType, result.filename)];
    }
    if (result.kind === "chunks") {
      return result.chunks.map((chunk) =>
        makeDownload(chunk.bytes, chunk.contentType, chunk.filename),
      );
    }
    return [];
  }, [result]);

  useEffect(() => {
    return () => {
      for (const download of downloads) URL.revokeObjectURL(download.url);
    };
  }, [downloads]);

  const run = async () => {
    if (!hasAllRequired(op, values) || busy) return;
    setBusy(true);
    setError(null);
    setResult(null);
    const response = await runAudioOp({ id: op.id, method: op.method, values });
    setBusy(false);
    if (!response.ok) {
      setError(response.error);
      return;
    }
    setResult(response.data);
  };

  const basic = op.params.filter((param) => !param.advanced);
  const advanced = op.params.filter((param) => param.advanced);
  const metadataJSON =
    result?.kind === "metadata" ? JSON.stringify(result.metadata, null, 2) : "";

  return (
    <Panel title={t(op.labelKey)} description={t(op.descriptionKey)}>
      <div className="flex flex-col gap-3">
        <label htmlFor="audio-op-select" className="flex flex-col gap-1 text-xs text-app-muted-foreground">
          {t(strings.diagnostics.audioOps.operationLabel)}
          <Select
            id="audio-op-select"
            value={op.id}
            onChange={(e) => onSelect(e.currentTarget.value)}
          >
            {audioOps.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {t(candidate.labelKey)}
              </option>
            ))}
          </Select>
        </label>

        {basic.map((param) => (
          <Field
            key={param.name}
            param={param}
            value={values[param.name]}
            onChange={(value) => setValue(param.name, value)}
          />
        ))}

        {advanced.length > 0 ? (
          <details className="rounded-control border border-app-border bg-app-surface-muted/40 px-3 py-2">
            <summary className="cursor-pointer text-xs font-medium text-app-muted-foreground">
              {t(strings.diagnostics.audioOps.advanced)}
            </summary>
            <div className="mt-3 flex flex-col gap-3">
              {advanced.map((param) => (
                <Field
                  key={param.name}
                  param={param}
                  value={values[param.name]}
                  onChange={(value) => setValue(param.name, value)}
                />
              ))}
            </div>
          </details>
        ) : null}

        <div>
          <Button onClick={() => void run()} disabled={!hasAllRequired(op, values) || busy}>
            {busy ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : (
              <Play className="h-4 w-4" aria-hidden="true" />
            )}
            {busy
              ? t(strings.diagnostics.audioOps.runningAction)
              : t(strings.diagnostics.audioOps.runAction)}
          </Button>
        </div>

        {error ? <ApiErrorState error={error} /> : null}

        {downloads.length > 0 ? (
          <div className="flex flex-col gap-1">
            <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
              {result?.kind === "chunks"
                ? t(strings.diagnostics.audioOps.chunksLabel)
                : t(strings.diagnostics.audioOps.resultLabel)}
            </p>
            {downloads.map((download) => (
              <a
                key={download.filename}
                href={download.url}
                download={download.filename}
                className="text-sm text-app-primary underline"
              >
                {t(strings.diagnostics.audioOps.download)} {download.filename}
              </a>
            ))}
          </div>
        ) : null}

        {metadataJSON ? (
          <div className="flex flex-col gap-1">
            <p className="text-xs uppercase tracking-wide text-app-muted-foreground">
              {t(strings.diagnostics.audioOps.metadataLabel)}
            </p>
            <pre className="overflow-auto rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
              {metadataJSON}
            </pre>
          </div>
        ) : null}
      </div>
    </Panel>
  );
}

function Field({
  param,
  value,
  onChange,
}: {
  param: AudioOpParam;
  value: AudioOpValue;
  onChange: (value: AudioOpValue) => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const label = FIELD_LABELS[param.name];
  const labelText = param.unit ? `${t(label)} (${param.unit})` : t(label);
  const translate = (key: AudioOpOptionLabelKey) => t(key);

  if (param.control === "checkbox") {
    return (
      <label htmlFor={id} className="flex items-center gap-2 text-sm text-app-foreground">
        <input
          id={id}
          type="checkbox"
          className="h-4 w-4 rounded border-app-border text-app-primary focus-visible:ring-2 focus-visible:ring-app-focus/50"
          checked={value === true}
          onChange={(e) => onChange(e.currentTarget.checked)}
        />
        {labelText}
      </label>
    );
  }

  return (
    <label htmlFor={id} className="flex flex-col gap-1 text-xs text-app-muted-foreground">
      {labelText}
      {renderControl(id, param, value, onChange, translate)}
    </label>
  );
}

function renderControl(
  id: string,
  param: AudioOpParam,
  value: AudioOpValue,
  onChange: (value: AudioOpValue) => void,
  t: (key: AudioOpOptionLabelKey) => string,
) {
  switch (param.control) {
    case "file":
      return (
        <Input
          id={id}
          type="file"
          accept="audio/*"
          required={param.required}
          onChange={(e) => onChange(e.currentTarget.files?.[0] ?? "")}
        />
      );
    case "files":
      return (
        <Input
          id={id}
          type="file"
          accept="audio/*"
          multiple
          required={param.required}
          onChange={(e) => onChange(Array.from(e.currentTarget.files ?? []))}
        />
      );
    case "number":
      return (
        <Input
          id={id}
          type="number"
          inputMode="decimal"
          min={param.min}
          max={param.max}
          step={param.step}
          required={param.required}
          value={textValue(value)}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      );
    case "select":
      return (
        <Select
          id={id}
          value={textValue(value)}
          required={param.required}
          onChange={(e) => onChange(e.currentTarget.value)}
        >
          {(param.options ?? []).map((option) => (
            <option key={option.value} value={option.value}>
              {t(option.labelKey)}
            </option>
          ))}
        </Select>
      );
    case "text":
      return (
        <Input
          id={id}
          type="text"
          required={param.required}
          value={textValue(value)}
          onChange={(e) => onChange(e.currentTarget.value)}
        />
      );
  }
}

function makeDownload(bytes: Uint8Array, contentType: string, filename: string) {
  // Copy into a plain ArrayBuffer so the Blob constructor accepts the bytes
  // under TypeScript's Uint8Array<ArrayBufferLike> type.
  const buffer = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(buffer).set(bytes);
  return {
    filename,
    url: URL.createObjectURL(new Blob([buffer], { type: contentType })),
  };
}
