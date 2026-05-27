import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Loader2, Mic, Square } from "lucide-react";

import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { LoadingRows } from "../../components/composites/LoadingRows";
import { pushToast } from "../../components/ui/toast";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import {
  deleteWakeWordTemplate,
  getWakeWordConfig,
  saveWakeWordTemplate,
  type WakeWordSample,
} from "../../services/wakeWord";
import { recordWakeWordSample, type RecordHandle } from "./wakeWordRecorder";

const MIN_SAMPLES = 3;
const MAX_SAMPLES = 5;
const MIN_THRESHOLD = 0.1;
const MAX_THRESHOLD = 0.95;
const THRESHOLD_STEP = 0.05;
const DEFAULT_THRESHOLD = 0.6;

export function WakeWordPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();
  const config = useQuery({ queryKey: ["wakeWord", "config"], queryFn: getWakeWordConfig });

  const [label, setLabel] = useState("");
  const [threshold, setThreshold] = useState(DEFAULT_THRESHOLD);
  const [samples, setSamples] = useState<WakeWordSample[]>([]);
  const [recordHandle, setRecordHandle] = useState<RecordHandle | null>(null);
  const [recordError, setRecordError] = useState<string | null>(null);
  const [hydrated, setHydrated] = useState(false);

  // Hydrate the form from the persisted template on first successful load.
  // (Audio bytes are not echoed back by the server, so we only restore the
  // label + threshold and treat the sample set as a fresh re-enrollment.)
  if (config.data && !hydrated) {
    setHydrated(true);
    const tpl = config.data.template;
    if (tpl) {
      setLabel(tpl.label);
      if (tpl.threshold > 0) setThreshold(tpl.threshold);
    }
  }

  const saveMut = useMutation({
    mutationFn: () => saveWakeWordTemplate({ label, threshold, samples }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["wakeWord", "config"] });
      pushToast({ title: t(strings.wakeWordAdmin.saveSuccess) });
    },
  });

  const deleteMut = useMutation({
    mutationFn: deleteWakeWordTemplate,
    onSuccess: () => {
      setSamples([]);
      void qc.invalidateQueries({ queryKey: ["wakeWord", "config"] });
      pushToast({ title: t(strings.wakeWordAdmin.deleteSuccess) });
    },
  });

  const startRecording = async () => {
    if (samples.length >= MAX_SAMPLES || recordHandle) return;
    setRecordError(null);
    try {
      const handle = await recordWakeWordSample();
      setRecordHandle(handle);
      const sample = await handle.done;
      setSamples((prev) => (prev.length >= MAX_SAMPLES ? prev : [...prev, sample]));
      setRecordHandle(null);
    } catch {
      setRecordError(t(strings.wakeWordAdmin.micDenied));
      setRecordHandle(null);
    }
  };

  const removeSample = (index: number) => {
    setSamples((prev) => prev.filter((_, i) => i !== index));
  };

  if (config.isPending) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.wakeWordAdmin.pageTitle)}
          description={t(strings.wakeWordAdmin.pageDescription)}
        />
        <LoadingRows />
      </div>
    );
  }

  if (config.isError) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader
          title={t(strings.wakeWordAdmin.pageTitle)}
          description={t(strings.wakeWordAdmin.pageDescription)}
        />
        <ApiErrorState error={config.error} onRetry={() => void config.refetch()} />
      </div>
    );
  }

  const cfg = config.data;
  const canSave = samples.length >= MIN_SAMPLES && samples.length <= MAX_SAMPLES && label.trim() !== "";
  const atMax = samples.length >= MAX_SAMPLES;
  const recording = recordHandle !== null;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t(strings.wakeWordAdmin.pageTitle)}
        description={t(strings.wakeWordAdmin.pageDescription)}
      />

      <Panel title={t(strings.wakeWordAdmin.statusTitle)}>
        <dl className="grid grid-cols-2 gap-2 px-4 py-3 text-xs">
          <dt className="text-app-muted-foreground">{t(strings.wakeWordAdmin.statusConfigured)}</dt>
          <dd>{String(cfg.configured)}</dd>
          <dt className="text-app-muted-foreground">{t(strings.wakeWordAdmin.statusSampleCount)}</dt>
          <dd>{cfg.template?.samples.length ?? 0}</dd>
        </dl>
        {!cfg.configured ? (
          <p className="px-4 pb-3 text-sm text-app-muted-foreground">
            {t(strings.wakeWordAdmin.noTemplate)}
          </p>
        ) : null}
      </Panel>

      <Panel title={t(strings.wakeWordAdmin.enrollTitle)}>
        <div className="flex flex-col gap-4 px-4 py-3">
          <label className="flex flex-col gap-1 text-sm">
            {t(strings.wakeWordAdmin.labelField)}
            <Input
              value={label}
              onChange={(e) => setLabel(e.target.value)}
              placeholder={t(strings.wakeWordAdmin.labelPlaceholder)}
              data-testid={selectors.wakeWord.label}
            />
          </label>

          <label className="flex flex-col gap-1 text-sm">
            <span className="flex items-center justify-between">
              {t(strings.wakeWordAdmin.thresholdField)}
              <span className="font-mono text-xs text-app-muted-foreground">{threshold.toFixed(2)}</span>
            </span>
            <input
              type="range"
              min={MIN_THRESHOLD}
              max={MAX_THRESHOLD}
              step={THRESHOLD_STEP}
              value={threshold}
              onChange={(e) => setThreshold(parseFloat(e.target.value))}
              data-testid={selectors.wakeWord.threshold}
            />
            <span className="text-xs text-app-muted-foreground">{t(strings.wakeWordAdmin.thresholdHelp)}</span>
          </label>

          <div className="flex flex-col gap-2">
            <span className="text-sm font-medium">{t(strings.wakeWordAdmin.samplesTitle)}</span>
            <p className="text-xs text-app-muted-foreground">{t(strings.wakeWordAdmin.samplesHelp)}</p>
            <p
              className="text-xs text-app-muted-foreground"
              data-testid={selectors.wakeWord.sampleCount}
              data-count={samples.length}
              data-max={MAX_SAMPLES}
            >
              {t(strings.wakeWordAdmin.sampleCountStatus, { count: samples.length, max: MAX_SAMPLES })}
            </p>
            <ul className="flex flex-col gap-1">
              {samples.map((_, index) => (
                <li
                  key={index}
                  data-testid={selectors.wakeWord.sampleRow({ index: String(index) })}
                  className="flex items-center justify-between rounded-control border border-app-border px-3 py-1.5 text-sm"
                >
                  <span>{t(strings.wakeWordAdmin.sampleRow, { index: index + 1 })}</span>
                  <Button type="button" variant="ghost" size="sm" onClick={() => removeSample(index)}>
                    {t(strings.wakeWordAdmin.removeSample)}
                  </Button>
                </li>
              ))}
            </ul>
            <div>
              <Button
                type="button"
                variant={recording ? "ghost" : "default"}
                disabled={atMax && !recording}
                aria-pressed={recording}
                data-testid={selectors.wakeWord.record}
                onClick={() => (recordHandle ? recordHandle.stop() : void startRecording())}
              >
                {recording ? (
                  <>
                    <Square className="h-4 w-4" aria-hidden="true" />
                    {t(strings.wakeWordAdmin.recordingButton)}
                  </>
                ) : (
                  <>
                    <Mic className="h-4 w-4" aria-hidden="true" />
                    {t(strings.wakeWordAdmin.recordButton)}
                  </>
                )}
              </Button>
            </div>
            {atMax ? (
              <p className="text-xs text-app-warning">
                {t(strings.wakeWordAdmin.maxSamplesReached, { max: MAX_SAMPLES })}
              </p>
            ) : null}
            {samples.length < MIN_SAMPLES ? (
              <p className="text-xs text-app-muted-foreground">
                {t(strings.wakeWordAdmin.needMinSamples, { min: MIN_SAMPLES })}
              </p>
            ) : null}
            {recordError ? <p className="text-xs text-app-danger">{recordError}</p> : null}
          </div>

          <div className="flex gap-2">
            <Button
              type="button"
              disabled={!canSave || saveMut.isPending}
              data-testid={selectors.wakeWord.save}
              onClick={() => saveMut.mutate()}
            >
              {saveMut.isPending ? (
                <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              ) : null}
              {t(strings.wakeWordAdmin.saveButton)}
            </Button>
            <Button
              type="button"
              variant="ghost"
              disabled={!cfg.configured || deleteMut.isPending}
              data-testid={selectors.wakeWord.delete}
              onClick={() => deleteMut.mutate()}
            >
              {t(strings.wakeWordAdmin.deleteButton)}
            </Button>
          </div>
        </div>
      </Panel>
    </div>
  );
}
