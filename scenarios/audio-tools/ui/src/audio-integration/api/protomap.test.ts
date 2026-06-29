import { describe, it, expect } from "vitest";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

import {
  ProviderTier,
  AudioFormat,
  ResponseFormat,
} from "@vrooli/proto-types/audio-tools/v1/common/common_pb";
import {
  SpeakerMode,
  RejectBehavior,
  StreamingMode,
  StrategyPreference,
} from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";
import { SummarizeLevel } from "@vrooli/proto-types/audio-tools/v1/summarize/summarize_pb";

import {
  providerTierLabel,
  audioFormatFromString,
  responseFormatFromString,
  responseFormatLabel,
  speakerModeFromString,
  speakerModeLabel,
  rejectBehaviorFromString,
  rejectBehaviorLabel,
  streamingModeFromString,
  streamingModeLabel,
  strategyPreferenceFromString,
  strategyPreferenceLabel,
  summarizeLevelFromString,
  timestampToISO,
} from "./protomap";

// ---------------------------------------------------------------------------
// providerTierLabel
// ---------------------------------------------------------------------------

describe("providerTierLabel", () => {
  it("maps LOCAL → 'local'", () => expect(providerTierLabel(ProviderTier.LOCAL)).toBe("local"));
  it("maps BYOK → 'byok'", () => expect(providerTierLabel(ProviderTier.BYOK)).toBe("byok"));
  it("maps VROOLI → 'vrooli'", () => expect(providerTierLabel(ProviderTier.VROOLI)).toBe("vrooli"));
  it("maps UNSPECIFIED → ''", () => expect(providerTierLabel(ProviderTier.UNSPECIFIED)).toBe(""));
  it("maps undefined → ''", () => expect(providerTierLabel(undefined)).toBe(""));
});

// ---------------------------------------------------------------------------
// audioFormatFromString
// ---------------------------------------------------------------------------

describe("audioFormatFromString", () => {
  it("maps 'wav' → AudioFormat.WAV", () => expect(audioFormatFromString("wav")).toBe(AudioFormat.WAV));
  it("maps 'mp3' → AudioFormat.MP3", () => expect(audioFormatFromString("mp3")).toBe(AudioFormat.MP3));
  it("maps 'flac' → AudioFormat.FLAC", () => expect(audioFormatFromString("flac")).toBe(AudioFormat.FLAC));
  it("maps 'ogg' → AudioFormat.OGG", () => expect(audioFormatFromString("ogg")).toBe(AudioFormat.OGG));
  it("maps 'webm' → AudioFormat.WEBM", () => expect(audioFormatFromString("webm")).toBe(AudioFormat.WEBM));
  it("maps 'opus' → AudioFormat.OPUS", () => expect(audioFormatFromString("opus")).toBe(AudioFormat.OPUS));
  it("maps unknown string → AudioFormat.UNSPECIFIED", () => expect(audioFormatFromString("aac")).toBe(AudioFormat.UNSPECIFIED));
  it("maps undefined → AudioFormat.UNSPECIFIED", () => expect(audioFormatFromString(undefined)).toBe(AudioFormat.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// responseFormatFromString
// ---------------------------------------------------------------------------

describe("responseFormatFromString", () => {
  it("maps 'mp3' → ResponseFormat.MP3", () => expect(responseFormatFromString("mp3")).toBe(ResponseFormat.MP3));
  it("maps 'wav' → ResponseFormat.WAV", () => expect(responseFormatFromString("wav")).toBe(ResponseFormat.WAV));
  it("maps 'opus' → ResponseFormat.OPUS", () => expect(responseFormatFromString("opus")).toBe(ResponseFormat.OPUS));
  it("maps 'flac' → ResponseFormat.FLAC", () => expect(responseFormatFromString("flac")).toBe(ResponseFormat.FLAC));
  it("maps unknown string → ResponseFormat.UNSPECIFIED", () => expect(responseFormatFromString("ogg")).toBe(ResponseFormat.UNSPECIFIED));
  it("maps undefined → ResponseFormat.UNSPECIFIED", () => expect(responseFormatFromString(undefined)).toBe(ResponseFormat.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// responseFormatLabel
// ---------------------------------------------------------------------------

describe("responseFormatLabel", () => {
  it("maps MP3 → 'mp3'", () => expect(responseFormatLabel(ResponseFormat.MP3)).toBe("mp3"));
  it("maps WAV → 'wav'", () => expect(responseFormatLabel(ResponseFormat.WAV)).toBe("wav"));
  it("maps OPUS → 'opus'", () => expect(responseFormatLabel(ResponseFormat.OPUS)).toBe("opus"));
  it("maps FLAC → 'flac'", () => expect(responseFormatLabel(ResponseFormat.FLAC)).toBe("flac"));
  it("maps UNSPECIFIED → ''", () => expect(responseFormatLabel(ResponseFormat.UNSPECIFIED)).toBe(""));
  it("maps undefined → ''", () => expect(responseFormatLabel(undefined)).toBe(""));
});

// ---------------------------------------------------------------------------
// speakerModeFromString
// ---------------------------------------------------------------------------

describe("speakerModeFromString", () => {
  it("maps 'off' → SpeakerMode.OFF", () => expect(speakerModeFromString("off")).toBe(SpeakerMode.OFF));
  it("maps 'filter' → SpeakerMode.FILTER", () => expect(speakerModeFromString("filter")).toBe(SpeakerMode.FILTER));
  it("maps 'advisory' → SpeakerMode.ADVISORY", () => expect(speakerModeFromString("advisory")).toBe(SpeakerMode.ADVISORY));
  it("maps unknown → SpeakerMode.UNSPECIFIED", () => expect(speakerModeFromString("unknown")).toBe(SpeakerMode.UNSPECIFIED));
  it("maps undefined → SpeakerMode.UNSPECIFIED", () => expect(speakerModeFromString(undefined)).toBe(SpeakerMode.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// speakerModeLabel
// ---------------------------------------------------------------------------

describe("speakerModeLabel", () => {
  it("maps OFF → 'off'", () => expect(speakerModeLabel(SpeakerMode.OFF)).toBe("off"));
  it("maps FILTER → 'filter'", () => expect(speakerModeLabel(SpeakerMode.FILTER)).toBe("filter"));
  it("maps ADVISORY → 'advisory'", () => expect(speakerModeLabel(SpeakerMode.ADVISORY)).toBe("advisory"));
  it("maps UNSPECIFIED → 'filter' (default)", () => expect(speakerModeLabel(SpeakerMode.UNSPECIFIED)).toBe("filter"));
  it("maps undefined → 'filter' (default)", () => expect(speakerModeLabel(undefined)).toBe("filter"));
});

// ---------------------------------------------------------------------------
// rejectBehaviorFromString
// ---------------------------------------------------------------------------

describe("rejectBehaviorFromString", () => {
  it("maps 'drop' → RejectBehavior.DROP", () => expect(rejectBehaviorFromString("drop")).toBe(RejectBehavior.DROP));
  it("maps 'show-muted' → RejectBehavior.SHOW_MUTED", () => expect(rejectBehaviorFromString("show-muted")).toBe(RejectBehavior.SHOW_MUTED));
  it("maps unknown → RejectBehavior.UNSPECIFIED", () => expect(rejectBehaviorFromString("block")).toBe(RejectBehavior.UNSPECIFIED));
  it("maps undefined → RejectBehavior.UNSPECIFIED", () => expect(rejectBehaviorFromString(undefined)).toBe(RejectBehavior.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// rejectBehaviorLabel
// ---------------------------------------------------------------------------

describe("rejectBehaviorLabel", () => {
  it("maps DROP → 'drop'", () => expect(rejectBehaviorLabel(RejectBehavior.DROP)).toBe("drop"));
  it("maps SHOW_MUTED → 'show-muted'", () => expect(rejectBehaviorLabel(RejectBehavior.SHOW_MUTED)).toBe("show-muted"));
  it("maps UNSPECIFIED → 'drop' (default)", () => expect(rejectBehaviorLabel(RejectBehavior.UNSPECIFIED)).toBe("drop"));
  it("maps undefined → 'drop' (default)", () => expect(rejectBehaviorLabel(undefined)).toBe("drop"));
});

// ---------------------------------------------------------------------------
// streamingModeFromString
// ---------------------------------------------------------------------------

describe("streamingModeFromString", () => {
  it("maps 'auto' → StreamingMode.AUTO", () => expect(streamingModeFromString("auto")).toBe(StreamingMode.AUTO));
  it("maps 'off' → StreamingMode.OFF", () => expect(streamingModeFromString("off")).toBe(StreamingMode.OFF));
  it("maps unknown → StreamingMode.UNSPECIFIED", () => expect(streamingModeFromString("on")).toBe(StreamingMode.UNSPECIFIED));
  it("maps undefined → StreamingMode.UNSPECIFIED", () => expect(streamingModeFromString(undefined)).toBe(StreamingMode.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// streamingModeLabel
// ---------------------------------------------------------------------------

describe("streamingModeLabel", () => {
  it("maps AUTO → 'auto'", () => expect(streamingModeLabel(StreamingMode.AUTO)).toBe("auto"));
  it("maps OFF → 'off'", () => expect(streamingModeLabel(StreamingMode.OFF)).toBe("off"));
  it("maps UNSPECIFIED → 'unspecified'", () => expect(streamingModeLabel(StreamingMode.UNSPECIFIED)).toBe("unspecified"));
  it("maps undefined → 'unspecified'", () => expect(streamingModeLabel(undefined)).toBe("unspecified"));
});

// ---------------------------------------------------------------------------
// strategyPreferenceFromString
// ---------------------------------------------------------------------------

describe("strategyPreferenceFromString", () => {
  it("maps 'auto' → StrategyPreference.AUTO", () => expect(strategyPreferenceFromString("auto")).toBe(StrategyPreference.AUTO));
  it("maps 'vad' → StrategyPreference.VAD", () => expect(strategyPreferenceFromString("vad")).toBe(StrategyPreference.VAD));
  it("maps 'overlap' → StrategyPreference.OVERLAP", () => expect(strategyPreferenceFromString("overlap")).toBe(StrategyPreference.OVERLAP));
  it("maps 'passthrough' → StrategyPreference.PASSTHROUGH", () => expect(strategyPreferenceFromString("passthrough")).toBe(StrategyPreference.PASSTHROUGH));
  it("maps unknown → StrategyPreference.UNSPECIFIED", () => expect(strategyPreferenceFromString("batch")).toBe(StrategyPreference.UNSPECIFIED));
  it("maps undefined → StrategyPreference.UNSPECIFIED", () => expect(strategyPreferenceFromString(undefined)).toBe(StrategyPreference.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// strategyPreferenceLabel
// ---------------------------------------------------------------------------

describe("strategyPreferenceLabel", () => {
  it("maps AUTO → 'auto'", () => expect(strategyPreferenceLabel(StrategyPreference.AUTO)).toBe("auto"));
  it("maps VAD → 'vad'", () => expect(strategyPreferenceLabel(StrategyPreference.VAD)).toBe("vad"));
  it("maps OVERLAP → 'overlap'", () => expect(strategyPreferenceLabel(StrategyPreference.OVERLAP)).toBe("overlap"));
  it("maps PASSTHROUGH → 'passthrough'", () => expect(strategyPreferenceLabel(StrategyPreference.PASSTHROUGH)).toBe("passthrough"));
  it("maps UNSPECIFIED → 'unspecified'", () => expect(strategyPreferenceLabel(StrategyPreference.UNSPECIFIED)).toBe("unspecified"));
  it("maps undefined → 'unspecified'", () => expect(strategyPreferenceLabel(undefined)).toBe("unspecified"));
});

// ---------------------------------------------------------------------------
// summarizeLevelFromString
// ---------------------------------------------------------------------------

describe("summarizeLevelFromString", () => {
  it("maps 'light' → SummarizeLevel.LIGHT", () => expect(summarizeLevelFromString("light")).toBe(SummarizeLevel.LIGHT));
  it("maps 'moderate' → SummarizeLevel.MODERATE", () => expect(summarizeLevelFromString("moderate")).toBe(SummarizeLevel.MODERATE));
  it("maps 'heavy' → SummarizeLevel.HEAVY", () => expect(summarizeLevelFromString("heavy")).toBe(SummarizeLevel.HEAVY));
  it("maps unknown → SummarizeLevel.UNSPECIFIED", () => expect(summarizeLevelFromString("extreme")).toBe(SummarizeLevel.UNSPECIFIED));
  it("maps undefined → SummarizeLevel.UNSPECIFIED", () => expect(summarizeLevelFromString(undefined)).toBe(SummarizeLevel.UNSPECIFIED));
});

// ---------------------------------------------------------------------------
// timestampToISO
// ---------------------------------------------------------------------------

describe("timestampToISO", () => {
  it("returns '' for undefined", () => {
    expect(timestampToISO(undefined)).toBe("");
  });

  it("returns '' for empty string (falsy)", () => {
    expect(timestampToISO("")).toBe("");
  });

  it("passes through an existing ISO string unchanged", () => {
    const iso = "2023-11-14T17:00:00.000Z";
    expect(timestampToISO(iso)).toBe(iso);
  });

  it("converts a proto Timestamp with bigint seconds to ISO", () => {
    // Unix epoch — cast needed because Message<> brand is not constructable from plain objects
    const result = timestampToISO({ seconds: 0n, nanos: 0 } as unknown as Timestamp);
    expect(result).toBe("1970-01-01T00:00:00.000Z");
  });

  it("converts a proto Timestamp with number seconds to ISO (runtime number fallback)", () => {
    // protomap.ts handles `typeof ts.seconds !== "bigint"` at runtime; test via cast
    const result = timestampToISO({ seconds: 0 as unknown as bigint, nanos: 0 } as unknown as Timestamp);
    expect(result).toBe("1970-01-01T00:00:00.000Z");
  });

  it("includes sub-second precision from nanos", () => {
    // 500ms = 500_000_000 nanos
    const result = timestampToISO({ seconds: 0n, nanos: 500_000_000 } as unknown as Timestamp);
    expect(result).toBe("1970-01-01T00:00:00.500Z");
  });

  it("converts a realistic timestamp (2023-11-14 17:00:00 UTC)", () => {
    const epochMs = new Date("2023-11-14T17:00:00.000Z").getTime();
    const seconds = BigInt(Math.floor(epochMs / 1000));
    const result = timestampToISO({ seconds, nanos: 0 } as unknown as Timestamp);
    expect(result).toBe("2023-11-14T17:00:00.000Z");
  });
});
