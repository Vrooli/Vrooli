// Keep the shared browser capture regression in the audio-tools scenario
// suite. The imported module owns the tests; this bridge keeps Vitest's root
// inside the scenario while still exercising the package source directly.
import "../../../../packages/audio-capture-browser/src/longSession.test";
