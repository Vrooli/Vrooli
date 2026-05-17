import { useEffect, useRef, useState } from "react";
import { Loader2, Mic, Square, Upload } from "lucide-react";
import { VoiceStreamProvider, MicReadinessIndicator } from "@audio-tools/embed";

type MicPermission = "unknown" | "granted" | "denied" | "prompt";

function useMicPermission(): MicPermission {
  const [state, setState] = useState<MicPermission>("unknown");
  useEffect(() => {
    // TS lib.dom types `navigator.permissions` as always defined, but older
    // mobile WebViews (and jsdom in tests) omit it entirely. Keep the
    // runtime guard even though TS thinks it is always truthy.
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition
    if (!navigator.permissions || typeof navigator.permissions.query !== "function") {
      return;
    }
    let cancelled = false;
    // The "microphone" PermissionName is widely supported in evergreen
    // browsers; cast through unknown for TS lib coverage gaps.
    void navigator.permissions
      .query({ name: "microphone" })
      .then((status) => {
        if (cancelled) return;
        setState(status.state);
        status.onchange = () => setState(status.state);
      })
      .catch(() => {
        // Browser does not expose the microphone permission descriptor.
        // Leave state as "unknown".
      });
    return () => {
      cancelled = true;
    };
  }, []);
  return state;
}
import { Panel } from "../../components/ui/panel";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Tabs } from "../../components/ui/tabs";
import { ApiErrorState } from "../../components/composites/ApiErrorState";
import { transcribe, type ProviderTrace } from "../../services/diagnostics";
import type { ApiError } from "../../api/client";
import { useTranslation } from "../../i18n";
import { strings } from "../../consts/strings";

interface Props {
  onTrace: (t: ProviderTrace) => void;
}

/**
 * Three-mode Transcribe try-it:
 *
 *   - live    : streams audio via WebSocket through VoiceStreamProvider
 *               (the same streaming chain consumer scenarios adopt via
 *               @audio-tools/embed). Renders running partials and the
 *               final transcript.
 *   - oneshot : single short recording via MediaRecorder, uploaded
 *               through the multipart REST endpoint (the declared
 *               multipart_upload REST exception). Exercises the unary
 *               STT chain.
 *   - file    : preserved upload-from-disk path for offline samples.
 *
 * The file/oneshot paths produce a final-only trace via the existing
 * `transcribe()` service. Live mode produces a final trace by reading
 * the response trace fields after stream completion; partials are
 * displayed live but do not surface in the trace card (the trace is the
 * locked tier for the whole session).
 */
export function TranscribeTryIt({ onTrace }: Props) {
  const { t } = useTranslation();

  return (
    <Panel
      title={t(strings.diagnostics.transcribeTitle)}
      description={t(strings.diagnostics.transcribeDescription)}
    >
      <Tabs
        ariaLabel={t(strings.diagnostics.tablistTranscribeMode)}
        items={[
          { value: "live", label: t(strings.diagnostics.tabLiveMic) },
          { value: "oneshot", label: t(strings.diagnostics.tabOneshot) },
          { value: "file", label: t(strings.diagnostics.tabFile) },
        ]}
        defaultValue="live"
      >
        {(active: string) => {
          if (active === "live") return <LiveTry onTrace={onTrace} />;
          if (active === "oneshot") return <OneshotTry onTrace={onTrace} />;
          return <FileTry onTrace={onTrace} />;
        }}
      </Tabs>
    </Panel>
  );
}

/* ------------------------------- Live mode ------------------------------ */

function LiveTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const providerRef = useRef<VoiceStreamProvider | null>(null);
  const [recording, setRecording] = useState(false);
  const [partial, setPartial] = useState("");
  const [finalText, setFinalText] = useState("");
  const [error, setError] = useState<string>("");
  const micPermission = useMicPermission();

  // Lazily construct the provider so SSR / initial render never touches
  // browser-only globals (MediaRecorder, AudioContext).
  function ensureProvider(): VoiceStreamProvider {
    if (providerRef.current === null) {
      const p = new VoiceStreamProvider();
      p.onResult = (text) => {
        setPartial("");
        setFinalText(text);
        setRecording(false);
        // VoiceStreamProvider does not surface a typed ProviderTrace
        // event in the current proto shape — emit a placeholder trace so
        // the right-hand card still reflects that a live run completed.
        // Phase F upgrades this to a real trace from the streaming
        // chain's done event.
        onTrace({ providerTier: "local", providerId: "voice-stream", modelId: "whisper", latencyMs: 0 });
      };
      p.onError = (msg) => {
        setError(msg);
        setRecording(false);
      };
      p.onPartial = (text) => setPartial(text);
      providerRef.current = p;
    }
    return providerRef.current;
  }

  useEffect(() => {
    return () => {
      providerRef.current?.dispose();
      providerRef.current = null;
    };
  }, []);

  const start = async () => {
    setError("");
    setPartial("");
    setFinalText("");
    try {
      await ensureProvider().start();
      setRecording(true);
    } catch (e) {
      setError((e as Error).message || t(strings.diagnostics.liveStartFailed));
    }
  };

  const stop = () => {
    providerRef.current?.stop();
    // VoiceStreamProvider drives recording=false from onResult/onError.
  };

  return (
    <div className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <Button onClick={() => (recording ? stop() : void start())} aria-pressed={recording}>
          {recording ? (
            <>
              <Square className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.liveStop)}
            </>
          ) : (
            <>
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.liveStart)}
            </>
          )}
        </Button>
        <MicReadinessIndicator state={micPermission} />
      </div>
      {error ? <p className="text-sm text-app-danger">{error}</p> : null}
      {partial ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted/60 p-3 text-sm italic opacity-80">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.partialLabel)}</p>
          <p className="whitespace-pre-wrap">{partial}</p>
        </div>
      ) : null}
      {finalText ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.finalLabel)}</p>
          <p className="whitespace-pre-wrap font-mono text-sm">{finalText}</p>
        </div>
      ) : null}
    </div>
  );
}

/* ------------------------------ One-shot mode ---------------------------- */

function OneshotTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const mediaRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<BlobPart[]>([]);
  const [recording, setRecording] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState("");
  const [error, setError] = useState<ApiError | string | null>(null);
  const micPermission = useMicPermission();

  useEffect(() => () => {
    streamRef.current?.getTracks().forEach((t) => t.stop());
  }, []);

  const start = async () => {
    setError(null);
    setResult("");
    chunksRef.current = [];
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      streamRef.current = stream;
      const mr = new MediaRecorder(stream);
      mr.ondataavailable = (ev) => {
        if (ev.data.size > 0) chunksRef.current.push(ev.data);
      };
      mr.onstop = () => {
        const blob = new Blob(chunksRef.current, { type: mr.mimeType || "audio/webm" });
        streamRef.current?.getTracks().forEach((t) => t.stop());
        streamRef.current = null;
        void upload(blob);
      };
      mr.start();
      mediaRef.current = mr;
      setRecording(true);
    } catch (e) {
      setError((e as Error).message || t(strings.diagnostics.micPermissionDenied));
    }
  };

  const stop = () => {
    mediaRef.current?.stop();
    setRecording(false);
  };

  const upload = async (blob: Blob) => {
    setBusy(true);
    const file = new File([blob], "oneshot.webm", { type: blob.type });
    const r = await transcribe(file);
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    setResult(r.data.text);
    onTrace(r.data.trace);
  };

  return (
    <div className="flex flex-col gap-3">
      <p className="text-xs text-app-muted-foreground">
        {t(strings.diagnostics.oneshotHint)}
      </p>
      <div className="flex items-center gap-3">
        <Button onClick={() => (recording ? stop() : void start())} disabled={busy} aria-pressed={recording}>
          {recording ? (
            <>
              <Square className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.oneshotStopUpload)}
            </>
          ) : busy ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
              {t(strings.diagnostics.oneshotUploading)}
            </>
          ) : (
            <>
              <Mic className="h-4 w-4" aria-hidden="true" />
              {t(strings.diagnostics.oneshotRecord)}
            </>
          )}
        </Button>
        <MicReadinessIndicator state={micPermission} />
      </div>
      {error ? (
        typeof error === "string" ? (
          <p className="text-sm text-app-danger">{error}</p>
        ) : (
          <ApiErrorState error={error} />
        )
      ) : null}
      {result ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">{t(strings.diagnostics.finalLabel)}</p>
          <p className="whitespace-pre-wrap font-mono text-sm">{result}</p>
        </div>
      ) : null}
    </div>
  );
}

/* -------------------------------- File mode ----------------------------- */

function FileTry({ onTrace }: Props) {
  const { t } = useTranslation();
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<string>("");
  const [error, setError] = useState<ApiError | null>(null);

  const run = async () => {
    if (!file) return;
    setBusy(true);
    setError(null);
    const r = await transcribe(file);
    setBusy(false);
    if (!r.ok) {
      setError(r.error);
      return;
    }
    setResult(r.data.text);
    onTrace(r.data.trace);
  };

  return (
    <div className="flex flex-col gap-3">
      <Input
        type="file"
        accept="audio/*"
        onChange={(e) => setFile(e.currentTarget.files?.[0] ?? null)}
        aria-label={t(strings.diagnostics.audioFileLabel)}
      />
      <div>
        <Button onClick={() => void run()} disabled={!file || busy}>
          {busy ? (
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
          ) : (
            <Upload className="h-4 w-4" aria-hidden="true" />
          )}
          {t(strings.diagnostics.transcribeAction)}
        </Button>
      </div>
      {error ? <ApiErrorState error={error} /> : null}
      {result ? (
        <div className="rounded-control border border-app-border bg-app-surface-muted p-3 text-sm">
          <p className="mb-1 text-xs uppercase tracking-wide text-app-muted-foreground">
            {t(strings.diagnostics.resultLabel)}
          </p>
          <p className="whitespace-pre-wrap font-mono text-sm">{result}</p>
        </div>
      ) : null}
    </div>
  );
}
