import { Mic, MicOff } from "lucide-react";
import { useCallback, useRef } from "react";
import { useVoiceInput } from "@vrooli/react-component-library/useVoiceInput/3";

import { Button } from "../../components/ui/button";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { portalVoice } from "./portalVoice";

interface VoiceComposerButtonProps {
  onTranscript: (text: string) => void;
  disabled?: boolean;
}

export type FinalTranscriptState = { normalized: string; at: number };

/**
 * Accepts a finalized transcript once per short callback window. Audio Tools
 * can deliver the same final event from both its streaming and batch paths;
 * callers use the returned state only after acceptance.
 */
export function acceptFinalTranscript(
  text: string,
  previous: FinalTranscriptState | undefined,
  now: number,
  windowMs = 2_000,
): { value: string; state: FinalTranscriptState } | undefined {
  const value = text.trim();
  if (!value) return undefined;
  const normalized = value.replace(/\s+/g, " ").toLocaleLowerCase();
  if (previous && previous.normalized === normalized && now - previous.at <= windowMs) return undefined;
  return { value, state: { normalized, at: now } };
}

/** Optional Audio Tools voice control. Text entry remains independent. */
export function VoiceComposerButton({ onTranscript, disabled = false }: VoiceComposerButtonProps) {
  const { t } = useTranslation();
  const lastFinal = useRef<FinalTranscriptState>();
  const handleTranscript = useCallback((text: string) => {
    const accepted = acceptFinalTranscript(text, lastFinal.current, Date.now());
    if (!accepted) return;
    lastFinal.current = accepted.state;
    onTranscript(accepted.value);
  }, [onTranscript]);
  const voice = useVoiceInput({
    services: portalVoice.services,
    voiceEnabled: portalVoice.enabled,
    voiceLanguage: "en",
    vadSilenceTimeoutMs: 700,
    persistentMode: false,
    wakeWordEnabled: false,
    segmentSilenceMs: 700,
    capabilityCheck: portalVoice.capabilityCheck,
    onTranscript: handleTranscript,
  });
  const active = voice.isActive || voice.isPreparing || voice.isTranscribing;
  const unavailable = !voice.supported;
  const label = unavailable
    ? t(strings.chat.composer.voiceUnavailable)
    : active
      ? t(strings.chat.composer.voiceStop)
      : t(strings.chat.composer.voiceStart);

  return (
    <Button
      type="button"
      variant="outline"
      aria-label={label}
      title={label}
      disabled={disabled || unavailable || voice.isTranscribing}
      onClick={() => {
        if (active) voice.stopRecording();
        else void voice.startRecording();
      }}
    >
      {active ? <MicOff aria-hidden className="mr-2 h-4 w-4" /> : <Mic aria-hidden className="mr-2 h-4 w-4" />}
      {label}
    </Button>
  );
}
