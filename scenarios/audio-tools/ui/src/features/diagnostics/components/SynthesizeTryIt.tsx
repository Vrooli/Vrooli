import { useEffect, useState } from "react";
import { Loader2, Send } from "lucide-react";
import { Panel } from "../../../components/ui/panel";
import { Button } from "../../../components/ui/button";
import { Textarea } from "../../../components/ui/textarea";
import { Select } from "../../../components/ui/select";
import { ApiErrorState } from "../../../components/composites/ApiErrorState";
import { synthesize, listVoices } from "../../../services/tts";
import type { ProviderTrace } from "../../../services/diagnostics";
import type { ApiError } from "../../../api/client";
import { useTranslation } from "../../../i18n";
import { strings } from "../../../consts/strings";

export function SynthesizeTryIt({ onTrace }: { onTrace: (t: ProviderTrace) => void }) {
  const { t } = useTranslation();
  const [text, setText] = useState("");
  const [voice, setVoice] = useState("voice.feminine.warm");
  const [voices, setVoices] = useState<{ id: string; name: string }[]>([]);
  const [busy, setBusy] = useState(false);
  const [audioUrl, setAudioUrl] = useState<string>("");
  const [error, setError] = useState<ApiError | null>(null);

  useEffect(() => {
    let cancelled = false;
    void listVoices().then((r) => {
      if (cancelled) return;
      if (r.ok && r.data.length > 0) setVoices(r.data);
    });
    return () => { cancelled = true; };
  }, []);

  const run = async () => {
    if (!text.trim()) return;
    setBusy(true);
    setError(null);
    if (audioUrl) URL.revokeObjectURL(audioUrl);
    const r = await synthesize(text, voice, 1.0, "wav");
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    const buf = r.data.audio.slice().buffer;
    const blob = new Blob([buf], { type: r.data.contentType || "audio/wav" });
    setAudioUrl(URL.createObjectURL(blob));
    onTrace({ providerTier: r.data.providerTier, providerId: r.data.providerId, modelId: r.data.modelId, latencyMs: r.data.latencyMs });
  };

  return (
    <Panel title={t(strings.diagnostics.synthesizeTitle)} description={t(strings.diagnostics.synthesizeDescription)}>
      <div className="flex flex-col gap-3">
        <Textarea value={text} onChange={(e) => setText(e.currentTarget.value)} rows={4} placeholder={t(strings.diagnostics.synthesizeTextPlaceholder)} />
        <div className="flex flex-wrap items-end gap-3">
          <label htmlFor="synthesize-voice" className="flex flex-col gap-1 text-xs text-app-muted-foreground">
            {t(strings.diagnostics.synthesizeVoiceLabel)}
            <Select id="synthesize-voice" value={voice} onChange={(e) => setVoice(e.currentTarget.value)} className="w-56">
              {(voices.length ? voices : [{ id: voice, name: voice }]).map((v) => (
                <option key={v.id} value={v.id}>{v.name}</option>
              ))}
            </Select>
          </label>
          <Button onClick={() => void run()} disabled={!text.trim() || busy}>
            {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" /> : <Send className="h-4 w-4" aria-hidden="true" />}
            {t(strings.diagnostics.synthesizeAction)}
          </Button>
        </div>
        {error ? <ApiErrorState error={error} /> : null}
        {audioUrl ? (
          <audio controls src={audioUrl} className="w-full">
            <track kind="captions" />
          </audio>
        ) : null}
      </div>
    </Panel>
  );
}
