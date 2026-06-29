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

type Mode = "free" | "scripted";

export function DictationStudioPage() {
  const { t } = useTranslation();
  const qc = useQueryClient();

  const [mode, setMode] = useState<Mode>("free");
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
                    variant={mode === "free" ? "default" : "outline"}
                    aria-pressed={mode === "free"}
                    onClick={() => setMode("free")}
                  >
                    {t(strings.dictationStudio.modeFree)}
                  </Button>
                  <Button
                    type="button"
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
