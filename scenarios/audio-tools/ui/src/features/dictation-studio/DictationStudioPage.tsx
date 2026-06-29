import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";

import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Tabs } from "../../components/ui/tabs";
import { PageHeader } from "../../components/composites/PageHeader";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { pushToast } from "../../components/ui/toast";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";
import { selectors } from "../../consts/selectors";
import { ClipSource, createClip } from "../../services/corpus";
import { DictationRecorder, type CapturedClip } from "./DictationRecorder";
import { TranscriptEditor } from "./TranscriptEditor";
import { TagInput } from "./TagInput";
import { PromptPlayback } from "./PromptPlayback";
import { CorpusListView } from "./CorpusListView";
import { EvalReportView } from "./EvalReportView";
import { CUSTOM_SCRIPT_ID, DICTATION_SCRIPTS, findDictationScript } from "./scripts";

type Mode = "free" | "scripted";

export function DictationStudioPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();

  const [mode, setMode] = useState<Mode>("free");
  const [selectedScriptId, setSelectedScriptId] = useState(CUSTOM_SCRIPT_ID);
  const [scriptedPrompt, setScriptedPrompt] = useState("");
  const [captured, setCaptured] = useState<CapturedClip | null>(null);
  const [transcript, setTranscript] = useState("");
  const [tags, setTags] = useState<string[]>([]);

  const save = useMutation({
    mutationFn: () => {
      if (!captured) throw new Error("no captured audio");
      return createClip({
        audio: captured.audio,
        referenceText: transcript.trim(),
        tags,
        durationMs: captured.durationMs,
        sampleRateHz: captured.sampleRateHz,
        format: "pcm_s16le",
        source: mode === "scripted" ? ClipSource.SCRIPTED : ClipSource.FREE_FORM,
      });
    },
    onSuccess: () => {
      pushToast({ title: t(strings.dictationStudio.saveSuccess) });
      setCaptured(null);
      setTranscript("");
      setTags([]);
      void qc.invalidateQueries({ queryKey: ["corpus", "clips"] });
    },
  });

  const onCaptured = (clip: CapturedClip) => {
    setCaptured(clip);
    setTranscript(mode === "scripted" ? scriptedPrompt : clip.transcript);
  };

  const onScriptChange = (scriptId: string) => {
    setSelectedScriptId(scriptId);
    const script = findDictationScript(scriptId);
    if (!script) return;
    setScriptedPrompt(script.text);
    setTags(script.tags);
  };

  const selectedScript = findDictationScript(selectedScriptId);

  const canSave = captured !== null && transcript.trim() !== "";

  const tabs = [
    { value: "record", label: t(strings.dictationStudio.tabRecord) },
    { value: "corpus", label: t(strings.dictationStudio.tabCorpus) },
    { value: "report", label: t(strings.dictationStudio.tabReport) },
  ];

  return (
    <div className="flex max-w-5xl flex-col gap-4">
      <PageHeader
        title={t(strings.dictationStudio.pageTitle)}
        description={t(strings.dictationStudio.pageDescription)}
      />

      <Tabs items={tabs} defaultValue="record" ariaLabel={t(strings.dictationStudio.tabsLabel)}>
        {(active) =>
          active === "record" ? (
            <div className="flex flex-col gap-4">
              <Panel title={t(strings.dictationStudio.modeTitle)}>
                <div className="flex flex-wrap gap-2">
                  <Button
                    type="button"
                    data-testid={selectors.dictationStudio.modeFree}
                    variant={mode === "free" ? "default" : "outline"}
                    aria-pressed={mode === "free"}
                    onClick={() => setMode("free")}
                  >
                    {t(strings.dictationStudio.modeFree)}
                  </Button>
                  <Button
                    type="button"
                    data-testid={selectors.dictationStudio.modeScripted}
                    variant={mode === "scripted" ? "default" : "outline"}
                    aria-pressed={mode === "scripted"}
                    onClick={() => setMode("scripted")}
                  >
                    {t(strings.dictationStudio.modeScripted)}
                  </Button>
                </div>
              </Panel>

              {mode === "scripted" ? (
                <Panel title={t(strings.dictationStudio.scriptedPromptTitle)}>
                  <div className="mb-3 grid gap-2 md:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)]">
                    <label className="flex flex-col gap-1 text-sm font-medium text-app-foreground">
                      {t(strings.dictationStudio.scriptPickerLabel)}
                      <select
                        className="rounded-control border border-app-border bg-app-surface px-3 py-2 text-sm"
                        value={selectedScriptId}
                        onChange={(event) => onScriptChange(event.currentTarget.value)}
                        data-testid={selectors.dictationStudio.scriptPicker}
                      >
                        <option value={CUSTOM_SCRIPT_ID}>{t(strings.dictationStudio.customScript)}</option>
                        {DICTATION_SCRIPTS.map((script) => (
                          <option key={script.id} value={script.id}>
                            {script.title}
                          </option>
                        ))}
                      </select>
                    </label>
                    <div
                      className="rounded-control border border-app-border bg-app-surface-muted/60 p-3 text-sm"
                      data-testid={selectors.dictationStudio.scriptDetails}
                    >
                      <p className="font-medium text-app-foreground">
                        {selectedScript?.title ?? t(strings.dictationStudio.customScript)}
                      </p>
                      <p className="mt-1 text-app-muted-foreground">
                        {selectedScript?.purpose ?? t(strings.dictationStudio.customScriptHint)}
                      </p>
                      {selectedScript ? (
                        <p className="mt-2 text-xs text-app-muted-foreground">
                          {selectedScript.tags.join(", ")}
                        </p>
                      ) : null}
                    </div>
                  </div>
                  <PromptPlayback prompt={scriptedPrompt} onChange={setScriptedPrompt} />
                </Panel>
              ) : null}

              <Panel title={t(strings.dictationStudio.recordTitle)}>
                <DictationRecorder onCaptured={onCaptured} />
              </Panel>

              <Panel title={t(strings.dictationStudio.transcriptTitle)}>
                <TranscriptEditor value={transcript} onChange={setTranscript} />
              </Panel>

              <Panel title={t(strings.dictationStudio.tagsTitle)} description={t(strings.dictationStudio.tagsHint)}>
                <TagInput tags={tags} onChange={setTags} />
              </Panel>

              <div className="flex flex-col gap-2">
                <div>
                  <Button
                    type="button"
                    data-testid={selectors.dictationStudio.saveClip}
                    disabled={!canSave || save.isPending}
                    onClick={() => save.mutate()}
                  >
                    {save.isPending ? t(strings.dictationStudio.saving) : t(strings.dictationStudio.saveClip)}
                  </Button>
                </div>
                {!canSave ? (
                  <p className="text-xs text-app-muted-foreground">{t(strings.dictationStudio.saveDisabledHint)}</p>
                ) : null}
                {save.isError ? <ApiErrorState error={save.error} onRetry={() => save.mutate()} /> : null}
              </div>
            </div>
          ) : active === "corpus" ? (
            <Panel title={t(strings.dictationStudio.corpusTitle)}>
              <CorpusListView />
            </Panel>
          ) : (
            <Panel title={t(strings.dictationStudio.reportTitle)}>
              <EvalReportView />
            </Panel>
          )
        }
      </Tabs>
    </div>
  );
}
