export function VoiceRejectionBanner(props: {
  reason: string;
  onDismiss?: () => void;
}): JSX.Element {
  return (
    <div role="alert" className="audio-tools-embed-voice-rejection-banner">
      <span>{props.reason}</span>
      {props.onDismiss ? (
        <button type="button" onClick={props.onDismiss}>
          Dismiss
        </button>
      ) : null}
    </div>
  );
}
