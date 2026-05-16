export interface VoiceCommandSuggestionProps {
  suggestion: string;
  onAccept: () => void;
  onDismiss: () => void;
}

export function VoiceCommandSuggestion(props: VoiceCommandSuggestionProps): JSX.Element {
  return (
    <div role="dialog" aria-label="Voice command suggestion" className="audio-tools-embed-voice-command-suggestion">
      <span>{props.suggestion}</span>
      <button type="button" onClick={props.onAccept}>Accept</button>
      <button type="button" onClick={props.onDismiss}>Dismiss</button>
    </div>
  );
}
