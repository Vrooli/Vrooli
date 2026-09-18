import { useCallback, useEffect, useRef, useState } from "react";
import { Camera, Square, Video } from "lucide-react";
import { useTranslation } from "react-i18next";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";

import { strings } from "../../consts/strings";
import { getCameraMediaDevices } from "../../lib/attachments";

interface CameraCaptureDialogProps {
  open: boolean;
  onClose: () => void;
  /** Receives the captured photo or video clip. */
  onCapture: (file: File) => void;
  /** Fallback when getMedia is unavailable or denied. */
  onUseFilePicker: () => void;
}

type CaptureMode = "photo" | "video";

const RECORDER_MIME_CANDIDATES = [
  "video/webm;codecs=vp9",
  "video/webm;codecs=vp8",
  "video/webm",
  "video/mp4",
];

function pickRecorderMime(): string | undefined {
  if (typeof MediaRecorder === "undefined" || typeof MediaRecorder.isTypeSupported !== "function") {
    return undefined;
  }
  return RECORDER_MIME_CANDIDATES.find((mime) => MediaRecorder.isTypeSupported(mime));
}

function timestampName(extension: string): string {
  const stamp = new Date().toISOString().replace(/[:.]/g, "-");
  return `camera-${stamp}.${extension}`;
}

/**
 * CameraCaptureDialog records a photo or short video from the device camera
 * via getUserMedia + MediaRecorder. It is only used on devices where an in-app
 * camera is meaningful; mobile browsers can also use the native capture input.
 * The dialog degrades to the file picker when the camera is unavailable.
 */
export default function CameraCaptureDialog({ open, onClose, onCapture, onUseFilePicker }: CameraCaptureDialogProps) {
  const { t } = useTranslation();
  const videoRef = useRef<HTMLVideoElement>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const recorderRef = useRef<MediaRecorder | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const [mode, setMode] = useState<CaptureMode>("photo");
  const [error, setError] = useState<string | null>(null);
  const [recording, setRecording] = useState(false);
  const [elapsed, setElapsed] = useState(0);

  const stopStream = useCallback(() => {
    if (recorderRef.current?.state === "recording") {
      recorderRef.current.stop();
    }
    recorderRef.current = null;
    chunksRef.current = [];
    streamRef.current?.getTracks().forEach((track) => { track.stop(); });
    streamRef.current = null;
    if (videoRef.current) videoRef.current.srcObject = null;
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    setRecording(false);
    setElapsed(0);
  }, []);

  // Acquire the camera while the dialog is open; release it on close/unmount.
  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    setError(null);

    const start = async () => {
      const devices = getCameraMediaDevices();
      if (!devices?.getUserMedia) {
        setError(t(strings.composer.cameraUnavailable));
        return;
      }
      try {
        const stream = await devices.getUserMedia({
          video: { facingMode: "environment" },
          audio: false,
        });
        if (cancelled) {
          stream.getTracks().forEach((track) => { track.stop(); });
          return;
        }
        streamRef.current = stream;
        if (videoRef.current) {
          videoRef.current.srcObject = stream;
          await videoRef.current.play().catch(() => {});
        }
      } catch {
        if (!cancelled) setError(t(strings.composer.cameraUnavailable));
      }
    };
    void start();

    return () => {
      cancelled = true;
      stopStream();
    };
  }, [open, stopStream, t]);

  const capturePhoto = useCallback(() => {
    const video = videoRef.current;
    if (!video || video.videoWidth === 0) return;
    const canvas = document.createElement("canvas");
    canvas.width = video.videoWidth;
    canvas.height = video.videoHeight;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;
    ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
    canvas.toBlob((blob) => {
      if (!blob) return;
      onCapture(new File([blob], timestampName("png"), { type: "image/png" }));
      onClose();
    }, "image/png");
  }, [onCapture, onClose]);

  const startRecording = useCallback(() => {
    const stream = streamRef.current;
    if (!stream || typeof MediaRecorder === "undefined") {
      setError(t(strings.composer.cameraUnavailable));
      return;
    }
    const mime = pickRecorderMime();
    const recorder = new MediaRecorder(stream, mime ? { mimeType: mime } : undefined);
    chunksRef.current = [];
    recorder.ondataavailable = (event) => {
      if (event.data.size > 0) chunksRef.current.push(event.data);
    };
    recorder.onstop = () => {
      const type = recorder.mimeType || "video/webm";
      const blob = new Blob(chunksRef.current, { type });
      chunksRef.current = [];
      const extension = type.includes("mp4") ? "mp4" : "webm";
      onCapture(new File([blob], timestampName(extension), { type }));
      onClose();
    };
    recorderRef.current = recorder;
    recorder.start();
    setRecording(true);
    setElapsed(0);
    timerRef.current = setInterval(() => { setElapsed((secs) => secs + 1); }, 1000);
  }, [onCapture, onClose, t]);

  const stopRecording = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    setRecording(false);
    recorderRef.current?.stop();
  }, []);

  const handleUseFilePicker = useCallback(() => {
    onClose();
    onUseFilePicker();
  }, [onClose, onUseFilePicker]);

  return (
    <ResponsiveDialog
      open={open}
      onClose={onClose}
      title={t(strings.composer.cameraTitle)}
      closeLabel={t(strings.composer.cameraClose)}
      testId="camera-capture"
      size="md"
      contentPadding="none"
      avoidKeyboard
    >
      <div className="flex flex-col gap-3 p-3">
        {error ? (
          <div className="flex flex-col items-center gap-3 py-8 text-center" data-testid="camera-capture-error">
            <Camera className="h-8 w-8 text-wc-text-muted" aria-hidden />
            <p className="text-sm text-wc-text-primary">{error}</p>
            <p className="text-xs text-wc-text-muted">{t(strings.composer.cameraUnavailableHint)}</p>
            <button
              type="button"
              data-testid="camera-use-file-picker"
              onClick={handleUseFilePicker}
              className="rounded border border-wc-default bg-wc-surface-input px-3 py-2 text-sm text-wc-text-primary"
            >
              {t(strings.composer.cameraChooseFile)}
            </button>
          </div>
        ) : (
          <>
            <div className="relative overflow-hidden rounded-lg bg-black">
              <video
                ref={videoRef}
                data-testid="camera-capture-video"
                muted
                playsInline
                autoPlay
                className="max-h-[min(60vh,30rem)] w-full object-contain"
              />
              {recording && (
                <span
                  data-testid="camera-recording-indicator"
                  className="absolute end-2 top-2 inline-flex items-center gap-1.5 rounded-full bg-black/60 px-2 py-1 text-xs text-white"
                >
                  <span className="h-2 w-2 animate-pulse rounded-full bg-red-500" />
                  {t(strings.composer.cameraRecording, { seconds: elapsed })}
                </span>
              )}
            </div>

            <div className="flex items-center justify-center gap-2">
              <button
                type="button"
                data-testid="camera-mode-photo"
                aria-pressed={mode === "photo"}
                onClick={() => { if (!recording) setMode("photo"); }}
                className={`inline-flex items-center gap-1.5 rounded border px-3 py-2 text-sm ${
                  mode === "photo" ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary" : "border-wc-default bg-wc-surface-input text-wc-text-secondary"
                }`}
              >
                <Camera className="h-4 w-4" aria-hidden />
                {t(strings.composer.cameraPhotoMode)}
              </button>
              <button
                type="button"
                data-testid="camera-mode-video"
                aria-pressed={mode === "video"}
                onClick={() => { if (!recording) setMode("video"); }}
                className={`inline-flex items-center gap-1.5 rounded border px-3 py-2 text-sm ${
                  mode === "video" ? "border-wc-accent bg-wc-accent/20 text-wc-text-primary" : "border-wc-default bg-wc-surface-input text-wc-text-secondary"
                }`}
              >
                <Video className="h-4 w-4" aria-hidden />
                {t(strings.composer.cameraVideoMode)}
              </button>
            </div>

            <div className="flex justify-center">
              {mode === "photo" ? (
                <button
                  type="button"
                  data-testid="camera-capture-photo"
                  onClick={capturePhoto}
                  className="inline-flex items-center gap-2 rounded-full border border-wc-accent bg-wc-accent/20 px-5 py-3 text-sm font-medium text-wc-text-primary"
                >
                  <Camera className="h-4 w-4" aria-hidden />
                  {t(strings.composer.cameraCapture)}
                </button>
              ) : recording ? (
                <button
                  type="button"
                  data-testid="camera-stop-recording"
                  onClick={stopRecording}
                  className="inline-flex items-center gap-2 rounded-full border border-red-500/60 bg-red-500/20 px-5 py-3 text-sm font-medium text-wc-text-primary"
                >
                  <Square className="h-4 w-4" aria-hidden />
                  {t(strings.composer.cameraStopRecording)}
                </button>
              ) : (
                <button
                  type="button"
                  data-testid="camera-start-recording"
                  onClick={startRecording}
                  className="inline-flex items-center gap-2 rounded-full border border-wc-accent bg-wc-accent/20 px-5 py-3 text-sm font-medium text-wc-text-primary"
                >
                  <Video className="h-4 w-4" aria-hidden />
                  {t(strings.composer.cameraStartRecording)}
                </button>
              )}
            </div>
          </>
        )}
        {!error && <p className="text-center text-xs text-wc-text-muted">{t(strings.composer.cameraHint)}</p>}
      </div>
    </ResponsiveDialog>
  );
}
