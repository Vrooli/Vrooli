import { useLayoutEffect } from "react";
import { AudioLines, Loader2, TextCursorInput, WandSparkles } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Popover, PopoverContent, usePopover } from "@vrooli/react-component-library/Popover/1";
import { Button } from "./ui/button";
import { strings } from "../consts/strings";

export type VoiceInputMode = "auto" | "agent" | "dictation";

/** A custom trigger adapter for the mic rendered by any workspace surface. */
function MicAnchor({ anchor }: { anchor: HTMLElement | null }) {
  const { triggerRef } = usePopover();
  useLayoutEffect(() => {
    triggerRef.current = anchor;
    return () => { if (triggerRef.current === anchor) triggerRef.current = null; };
  }, [anchor, triggerRef]);
  return null;
}

export default function VoiceModeMenu({ anchor, nativeAvailable, nativeChecking = false, onChoose, onClose, onReconnect }: {
  anchor: HTMLElement | null;
  nativeAvailable: boolean;
  nativeChecking?: boolean;
  onChoose: (mode: VoiceInputMode) => void;
  onClose: () => void;
  onReconnect?: () => void;
}) {
  const { t } = useTranslation();
  const choices = [
    { mode: "auto", label: strings.agentVoice.auto, hint: strings.agentVoice.autoHint, Icon: WandSparkles },
    { mode: "agent", label: strings.agentVoice.native, hint: nativeAvailable ? strings.agentVoice.nativeHint : nativeChecking ? strings.agentVoice.checkingMode : strings.agentVoice.unavailableMode, Icon: nativeChecking ? Loader2 : AudioLines },
    { mode: "dictation", label: strings.agentVoice.dictation, hint: strings.agentVoice.dictationHint, Icon: TextCursorInput },
  ] as const;
  return <Popover open={anchor !== null} onOpenChange={(open) => { if (!open) onClose(); }} placement="top-end" responsive="none">
    <MicAnchor anchor={anchor} />
    <PopoverContent responsive="none" initialFocus="first" role="dialog" aria-label={t(strings.agentVoice.route)} aria-describedby="voice-input-mode-hint" data-testid="voice-input-mode-menu" className="wc-stable-theme w-72 max-w-[calc(100vw-2rem)] p-2">
      <p className="px-2 pb-1 pt-1 text-sm font-medium">{t(strings.agentVoice.route)}</p>
      {choices.map(({ mode, label, hint, Icon }) => <Button key={mode} type="button" variant="ghost" size="lg" disabled={mode === "agent" && !nativeAvailable} aria-busy={mode === "agent" && nativeChecking ? true : undefined} data-testid={`voice-input-mode-${mode}`} shape="square" className="h-auto min-h-11 w-full justify-start px-2 py-2 text-start" onClick={() => { onChoose(mode); }}>
        <span className="flex w-full items-start gap-3">
          <Icon className={`mt-0.5 h-4 w-4 shrink-0${mode === "agent" && nativeChecking ? " motion-safe:animate-spin" : ""}`} aria-hidden />
          <span className="min-w-0 space-y-0.5 whitespace-normal leading-snug">
            <span className="block text-sm">{t(label)}</span>
            <span className="block text-xs font-normal text-wc-text-muted">{t(hint)}</span>
          </span>
        </span>
      </Button>)}
      <p id="voice-input-mode-hint" className="px-2 pb-1 pt-2 text-xs text-wc-text-muted">{t(strings.agentVoice.firstUseHint)}</p>
      {onReconnect && <Button type="button" variant="ghost" className="mt-1 w-full" onClick={onReconnect}>{t(strings.agentVoice.reconnect)}</Button>}
    </PopoverContent>
  </Popover>;
}
