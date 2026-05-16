export function EnableAudioBanner(props: { onEnable: () => void }): JSX.Element {
  return (
    <div role="status" className="audio-tools-embed-enable-audio-banner">
      <span>Audio is muted. Click to enable.</span>
      <button type="button" onClick={props.onEnable}>
        Enable audio
      </button>
    </div>
  );
}
