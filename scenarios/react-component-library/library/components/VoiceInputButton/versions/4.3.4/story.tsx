import { useState } from "react";
import {
  VoiceInputButton,
  type VoiceInputButtonProps,
  type VoiceInputButtonState,
} from "./VoiceInputButton";

type LiveStoryProps = {
  args: VoiceInputButtonProps;
  environment?: { voiceInput?: string };
  log: (event: string, value?: unknown) => void;
};

/**
 * The capability fixture supplies the initial state, while the harness owns
 * the callback loop so the preview demonstrates an actual live specimen.
 */
export function Live({ args, environment, log }: LiveStoryProps) {
  const [state, setState] = useState<VoiceInputButtonState>(
    environment?.voiceInput === "recording" ? "recording" : (args.state ?? "idle"),
  );
  const nextState = state === "recording" ? "idle" : "recording";

  const setLiveState = () => {
    setState(nextState);
    log("voice-state", nextState);
  };

  const cancelTranscription = () => {
    setState("idle");
    log("voice-cancelled");
  };

  return (
    <div className="grid gap-space-xs justify-items-center">
      <VoiceInputButton
        {...args}
        state={state}
        onStart={setLiveState}
        onStop={setLiveState}
        onCancel={cancelTranscription}
      />
      <output aria-live="polite" data-testid="voice-fixture-state">
        Fixture: {environment?.voiceInput ?? "idle"} · State: {state}
      </output>
    </div>
  );
}

export function VoiceAnatomy(props: LiveStoryProps) {
  return <Live {...props} />;
}
export function VoiceStateMatrix(props: LiveStoryProps) {
  return <Live {...props} />;
}
export function VoiceModeMatrix(props: LiveStoryProps) {
  return <Live {...props} />;
}
export function VoiceSizeMatrix(props: LiveStoryProps) {
  return <Live {...props} />;
}
export function VoiceDensityMatrix(props: LiveStoryProps) {
  return <Live {...props} />;
}
