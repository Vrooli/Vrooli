# Voice Input Manual Validation

This is an RCL-only handoff, not an authorization to adopt the capability in
web-console or connect a real transcription backend.

1. Open the Voice Input Button catalog specimens and confirm idle, preparing,
   always-on recording, timeout recording, recovering, unavailable, error, and
   speaker-rejected states render without a microphone permission prompt.
2. Confirm each state has an honest accessible name; unavailable is disabled;
   the rejection specimen exposes the keyboard-operable **Transcribe anyway**
   action.
3. In the fake lifecycle harness, start/stop repeatedly and verify one start
   cue and at most one stop cue per capture, with no retained timer, listener,
   or capture after each terminal path.
4. Verify always-on mode accepts multiple settled segments across silence;
   verify timeout mode alone stops on its countdown; verify device end and
   adapter failure report their distinct terminal reasons.
5. Record explicit user approval before planning any adopter migration. The
   next plan must separately cover web-console integration, audio-tools durable
   streaming/recovery, provider/GPU simplification, and copied-code deletion.
