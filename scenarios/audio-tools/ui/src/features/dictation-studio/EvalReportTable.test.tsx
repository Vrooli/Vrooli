import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { EvalReportTable } from "./EvalReportTable";

const report = {
  qualityMeasured: true,
  latencyMeasured: true,
  latencyHonesty: "Intra-run timing only.",
  normalizationPolicy: { werPolicy: "normalized", overlapAgreementPolicy: "strict" },
  warnings: [{ code: "fixture", severity: "warning", message: "fixture warning" }],
  promotionVerdicts: [{ engineId: "kyutai", modelId: "model", strategy: "stream", policyProfile: "strict", stable: false, reasons: ["evidence missing"] }],
  summary: { winnerStrategy: "stream", winnerLabel: "stream", recommendation: "Use stream", confidence: "low", reasons: ["faster"], confidenceNotes: ["small sample"] },
  perStrategy: [{
    strategy: "stream",
    label: "Streaming",
    wer: 0.2,
    substitutions: 1,
    insertions: 1,
    deletions: 1,
    refWords: 10,
    whisperCalls: 2,
    whisperAudioSeconds: 4,
    rtf: 0.5,
    finalizationLatencyP50Ms: 10,
    finalizationLatencyP95Ms: 25,
    partialRevisions: 2,
    werDeltaVsWinner: 0,
    p95DeltaMsVsWinner: 0,
    callMultiplierVsWinner: 1,
    verdict: "recommended-despite-safety-failure",
    reasons: ["fast"],
    warnings: [{ code: "safety", severity: "warning", message: "unsafe" }],
    safety: { passed: false, retractionFree: false, droppedSpanFree: false, maxDroppedSpanWords: 3, droppedSpanThresholdWords: 2, retractionEvents: [{ atMs: 1 }], reasons: ["dropped"] },
    stageAttribution: { ingressLostWords: 1, strategyLostWords: 2, egressLostWords: 3, egressRejectEvents: 1, notes: [] },
    lengthCurves: [{ bucket: "1m", minDurationMs: 1000, maxDurationMs: 2000, clipCount: 1, wer: 0.2, finalizationLatencyP95Ms: 25, meanTimeToFirstCommitMs: 8, maxDroppedSpanWords: 3 }],
    scaling: { confidence: "low", points: [{}], latencyClassification: "linear", computeClassification: "flat", latencyFit: { metric: "latency", model: "linear", rSquared: 0.9, slopePerSecond: 1, unit: "ms", exponent: 1 }, computeFit: null, metricFits: [], reasons: ["limited"], warnings: [{ code: "sample", severity: "warning", message: "small" }] },
    commitCount: 1,
    speakerRejectionCount: 0,
    perClip: [{ clipId: "clip-1", wer: 0.3, substitutions: 1, insertions: 1, deletions: 0, whisperCalls: 2, partialRevisions: 1, finalizationLatencyP95Ms: 30, reference: "reference", hypothesis: "hypothesis", normalizedReference: "reference", normalizedHypothesis: "hypothesis", error: "provider timeout", editOperations: [{ kind: "substitution", referenceToken: "a", hypothesisToken: "b" }] }],
  }],
} as never;

describe("EvalReportTable", () => {
  it("renders safety, promotion, scaling, curve, and worst-clip evidence", () => {
    render(<EvalReportTable report={report} />);

    expect(screen.getByText("Use stream")).toBeInTheDocument();
    expect(screen.getByText("evidence missing")).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "dictationStudio.lengthCurveTitle" })).toBeInTheDocument();
  });

  it("renders an intentionally sparse measurement without unsupported evidence", () => {
    render(<EvalReportTable report={{
      qualityMeasured: false,
      latencyMeasured: false,
      warnings: [],
      perStrategy: [{
        strategy: "batch",
        label: "Batch",
        wer: 0,
        substitutions: 0,
        insertions: 0,
        deletions: 0,
        whisperCalls: 0,
        whisperAudioSeconds: 0,
        rtf: 0,
        finalizationLatencyP50Ms: 0,
        finalizationLatencyP95Ms: 0,
        partialRevisions: 0,
        werDeltaVsWinner: 0,
        p95DeltaMsVsWinner: 0,
        reasons: [],
        warnings: [],
        perClip: [],
      }],
    } as never} />);

    expect(screen.getByText("dictationStudio.qualityNotMeasured")).toBeInTheDocument();
    expect(screen.getByText("dictationStudio.latencyNotMeasured")).toBeInTheDocument();
    expect(screen.queryByTestId("dictation-promotion-verdicts")).not.toBeInTheDocument();
  });

  it("renders safe, zero-valued evidence and omits unavailable scaling fits", () => {
    render(<EvalReportTable report={{
      qualityMeasured: true,
      latencyMeasured: false,
      warnings: [],
      promotionVerdicts: [{ engineId: "local", modelId: "", strategy: "batch", policyProfile: "", stable: true, reasons: [] }],
      summary: { winnerStrategy: "batch", recommendation: "Use batch", confidence: "high", reasons: [], confidenceNotes: [] },
      perStrategy: [{
        strategy: "batch", label: "Batch", wer: 0, substitutions: 0, insertions: 0, deletions: 0,
        whisperCalls: 0, whisperAudioSeconds: 0, rtf: 0, finalizationLatencyP50Ms: 0,
        finalizationLatencyP95Ms: 0, partialRevisions: 0, werDeltaVsWinner: 0, p95DeltaMsVsWinner: 0,
        reasons: [], warnings: [], verdict: "", safety: { passed: true, retractionFree: true, maxDroppedSpanWords: 0, droppedSpanThresholdWords: 0, retractionEvents: [] },
        lengthCurves: [{ bucket: "short", minDurationMs: 0, maxDurationMs: 0, clipCount: 0, wer: 0, finalizationLatencyP95Ms: 0, meanTimeToFirstCommitMs: 0 }],
        scaling: { confidence: "", points: [], latencyClassification: "", computeClassification: "", latencyFit: { metric: "latency", model: "linear", rSquared: 0, slopePerSecond: 0, unit: "", exponent: 0 }, computeFit: { metric: "compute", model: "none", rSquared: 0, slopePerSecond: 0, exponent: 0 }, metricFits: [{ metric: "detail", model: "linear", rSquared: 0, slopePerSecond: 0, unit: "", exponent: 0 }, { metric: "ignored", model: "none", rSquared: 0, slopePerSecond: 0, exponent: 0 }], reasons: [], warnings: [] },
        perClip: [{ clipId: "clean", wer: 0.1, substitutions: 0, insertions: 0, deletions: 0, whisperCalls: 0, partialRevisions: 0, finalizationLatencyP95Ms: 0, reference: "a", hypothesis: "", normalizedReference: "", normalizedHypothesis: "", editOperations: [{ kind: "match", referenceToken: "a", hypothesisToken: "a" }] }],
      }],
    } as never} />);

    expect(screen.getByText("Use batch")).toBeInTheDocument();
    expect(screen.getByText("dictationStudio.promotionStable")).toBeInTheDocument();
  });
});
