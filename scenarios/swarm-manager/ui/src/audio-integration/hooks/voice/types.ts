// DOC: docs/internal/SEAMS.md#voice-input-provider-seam
//
// Shared substrate types live in @vrooli/audio-capture-browser. Keeping this
// path as a shim prevents swarm-manager from silently growing a second voice
// state machine; host-specific API/proto adapters remain beside it.
export * from "@vrooli/audio-capture-browser";
