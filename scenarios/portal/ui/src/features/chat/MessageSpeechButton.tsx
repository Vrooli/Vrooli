import { Volume2, VolumeX } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { Button } from "../../components/ui/button";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { portalSpeech } from "./portalVoice";

type Props = { text: string };
type State = "idle" | "checking" | "speaking" | "unavailable" | "failed";

/** Optional assistant-message speech with explicit cancellation. */
export function MessageSpeechButton({ text }: Props) {
  const { t } = useTranslation();
  const [state, setState] = useState<State>("idle");
  const abort = useRef<AbortController | null>(null);
  const audio = useRef<HTMLAudioElement | null>(null);
  const objectURL = useRef<string | null>(null);

  useEffect(() => () => {
    abort.current?.abort();
    audio.current?.pause();
    if (objectURL.current) URL.revokeObjectURL(objectURL.current);
  }, []);

  const stop = () => {
    const hadSpeech = abort.current !== null || audio.current !== null || state === "speaking";
    abort.current?.abort();
    abort.current = null;
    audio.current?.pause();
    audio.current = null;
    if (objectURL.current) URL.revokeObjectURL(objectURL.current);
    objectURL.current = null;
    if (hadSpeech && "speechSynthesis" in window) window.speechSynthesis.cancel();
    setState("idle");
  };

  const speak = async () => {
    stop();
    const controller = new AbortController();
    abort.current = controller;
    setState("checking");
    try {
      if (portalSpeech.enabled && await portalSpeech.capabilityCheck()) {
        const blob = await portalSpeech.synthesize(text, controller.signal);
        if (controller.signal.aborted) return;
        const url = URL.createObjectURL(blob);
        objectURL.current = url;
        const player = new Audio(url);
        audio.current = player;
        player.onended = () => { if (objectURL.current === url) stop(); };
        setState("speaking");
        await player.play();
        return;
      }
      if (typeof window === "undefined" || !("speechSynthesis" in window)) {
        setState("unavailable");
        return;
      }
      const utterance = new SpeechSynthesisUtterance(text);
      utterance.onend = () => { if (!controller.signal.aborted) setState("idle"); };
      utterance.onerror = (event) => { if (event.error !== "canceled" && !controller.signal.aborted) setState("failed"); };
      setState("speaking");
      window.speechSynthesis.speak(utterance);
    } catch (error) {
      if (controller.signal.aborted || (error instanceof Error && error.name === "AbortError")) return;
      setState(portalSpeech.enabled ? "failed" : "unavailable");
    }
  };

  const active = state === "checking" || state === "speaking";
  const label = state === "unavailable" ? t(strings.chat.message.speechUnavailable) : state === "failed" ? t(strings.chat.message.speechFailed) : active ? t(strings.chat.message.stopSpeaking) : t(strings.chat.message.speak);
  return (
    <Button type="button" size="sm" variant="outline" aria-label={label} title={label} disabled={state === "unavailable"} onClick={() => active ? stop() : void speak()}>
      {active ? <VolumeX aria-hidden className="h-4 w-4" /> : <Volume2 aria-hidden className="h-4 w-4" />}
    </Button>
  );
}
