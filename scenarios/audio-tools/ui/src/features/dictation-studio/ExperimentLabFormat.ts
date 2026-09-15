import { type ExperimentRow, type StartExperimentInput } from "../../services/experiment";

// Passthrough is the executable Kyutai provider cell; the three batch rows
// remain Whisper strategy comparisons.
export const strategyOptions = ["batch", "vad_segment", "overlap_agree", "passthrough"] as const;

export function pct(value: number): string {
  return `${(value * 100).toFixed(1)}%`;
}

export function ms(value: number): string {
  return String(Math.round(value));
}

export function isTerminal(status: ExperimentRow["status"]): boolean {
  return status === "succeeded" || status === "failed" || status === "canceled";
}

export function hasSweepDurations(input: StartExperimentInput): boolean {
  return input.sweepDurationsCsv
    .split(",")
    .map((part) => part.trim())
    .some(Boolean);
}

export function defaultInput(): StartExperimentInput {
  return {
    name: "Dictation experiment",
    clipIds: [],
    engineIds: ["whisper-local", "kyutai"],
    strategies: [...strategyOptions],
    realtimeRepeats: 0,
    latencyTailSeconds: 8,
    chunkMs: 100,
    seed: 42,
    longForm: true,
    targetDurationSeconds: 180,
    gapMs: 5000,
    tagContains: "",
    sweepDurationsCsv: "",
    overlapMaxStallRejects: -1,
    overlapWindowMs: 0,
    overlapCommitRuns: 0,
    overlapMaxWindowMs: 25000,
    vadSilenceMs: 0,
    noiseTypesCsv: "",
    snrDbCsv: "12",
    competingVoicesCsv: "",
    competingText: "",
    speakerTargetProfileId: "",
    speakerExtraction: false,
    speakerVerification: false,
    speakerMode: "filter",
    speakerThreshold: 0.5,
    speakerFallback: true,
    speakerAblation: false,
    droppedSpanThresholdWords: 4,
  };
}
