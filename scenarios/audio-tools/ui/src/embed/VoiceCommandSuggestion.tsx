import type { JSX } from "react";

export interface VoiceCommandSuggestionProps {
  suggestion: string;
  onAccept: () => void;
  onDismiss: () => void;
  ariaLabel?: string;
  acceptLabel?: string;
  dismissLabel?: string;
}

export function VoiceCommandSuggestion(props: VoiceCommandSuggestionProps): JSX.Element {
  const ariaLabel = props.ariaLabel ?? "Voice command suggestion";
  const acceptLabel = props.acceptLabel ?? "Accept";
  const dismissLabel = props.dismissLabel ?? "Dismiss";
  return (
    <div role="dialog" aria-label={ariaLabel} className="audio-tools-embed-voice-command-suggestion">
      <span>{props.suggestion}</span>
      <button type="button" onClick={props.onAccept}>{acceptLabel}</button>
      <button type="button" onClick={props.onDismiss}>{dismissLabel}</button>
    </div>
  );
}
