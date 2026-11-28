import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { strings } from "../../consts/strings";
import { EvalReportTable } from "./EvalReportTable";

const winnerRecommendation = "Use stream";
const batchRecommendation = "Use batch";
const promotionReason = "evidence missing";
const messageOnly = "message only";
const codeOnly = "code only — coded";
const severityOnly = "info: severity only";
const addedToken = "added";
const removedToken = "removed";
const replacementToken = "new";
const matchedToken = "same";
const hiddenClip = "clip-9";
const unknownScaling = "unknown · 1 dictationStudio.scalingPoints";
const scalingLatencyLabel = "dictationStudio.scalingLatency";
const latencyDetail = "latency-detail";
const emptyUnit = "empty-unit";
const firstReason = "first reason";
const hiddenReason = "hidden reason";
const infoWarning = "info warning";

const report = {
  qualityMeasured: true,
  latencyMeasured: true,
  latencyHonesty: "Intra-run timing only.",
  normalizationPolicy: { werPolicy: "normalized", overlapAgreementPolicy: "strict" },
  warnings: [{ code: "fixture", severity: "warning", message: "fixture warning" }],
  promotionVerdicts: [{ engineId: "kyutai", modelId: "model", strategy: "stream", policyProfile: "strict", stable: false, reasons: [promotionReason] }],
  summary: { winnerStrategy: "stream", winnerLabel: "stream", recommendation: winnerRecommendation, confidence: "low", reasons: ["faster"], confidenceNotes: ["small sample"] },
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

    expect(screen.getByText(winnerRecommendation)).toBeInTheDocument();
    expect(screen.getByText(promotionReason)).toBeInTheDocument();
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

    expect(screen.getByText(strings.dictationStudio.qualityNotMeasured)).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.latencyNotMeasured)).toBeInTheDocument();
    expect(screen.queryByTestId("dictation-promotion-verdicts")).not.toBeInTheDocument();
  });

  it("renders safe, zero-valued evidence and omits unavailable scaling fits", () => {
    render(<EvalReportTable report={{
      qualityMeasured: true,
      latencyMeasured: false,
      warnings: [],
      promotionVerdicts: [{ engineId: "local", modelId: "", strategy: "batch", policyProfile: "", stable: true, reasons: [] }],
      summary: { winnerStrategy: "batch", recommendation: batchRecommendation, confidence: "high", reasons: [], confidenceNotes: [] },
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

    expect(screen.getByText(batchRecommendation)).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.promotionStable)).toBeInTheDocument();
  });

  it("renders optional warnings, stage attribution, and scaling fit variants", () => {
    render(<EvalReportTable report={{
      qualityMeasured: true,
      latencyMeasured: true,
      warnings: [
        { message: messageOnly },
        { code: "code only", message: "coded" },
        { severity: "info", message: severityOnly.slice(6) },
      ],
      perStrategy: [{
        strategy: "batch", label: "Batch", wer: 0, substitutions: 0, insertions: 0, deletions: 0,
        whisperCalls: 0, whisperAudioSeconds: 0, rtf: 0, finalizationLatencyP50Ms: 0,
        finalizationLatencyP95Ms: 0, partialRevisions: 0, werDeltaVsWinner: 0, p95DeltaMsVsWinner: 0,
        reasons: [], warnings: [], perClip: [], stageAttribution: {
          ingressLostWords: 0, strategyLostWords: 1, egressLostWords: 2, egressRejectEvents: 0, notes: [],
        },
        scaling: {
          confidence: "", points: [{}], latencyClassification: "", computeClassification: "flat",
          latencyFit: null,
          computeFit: { metric: "compute", model: "power", rSquared: 0.5, slopePerSecond: 2, unit: "units", exponent: 1.25 },
          metricFits: [
            { metric: "latency-detail", model: "linear", rSquared: 0.75, slopePerSecond: 3, unit: "ms", exponent: 0 },
            { metric: "empty-unit", model: "linear", rSquared: 0.2, slopePerSecond: 0, unit: "", exponent: 0 },
            { metric: "no-fit", model: "none", rSquared: 0, slopePerSecond: 0, exponent: 0 },
          ],
          reasons: ["first reason", "second reason", "hidden reason"],
          warnings: [{ code: "notice", severity: "info", message: "info warning" }],
        },
      }],
    } as never} />);

    expect(screen.getByText(messageOnly)).toBeInTheDocument();
    expect(screen.getByText(codeOnly)).toBeInTheDocument();
    expect(screen.getByText(severityOnly)).toBeInTheDocument();
    expect(screen.getByText(/0 · .*1 · .*2/)).toBeInTheDocument();
    expect(screen.getByText(unknownScaling)).toBeInTheDocument();
    expect(screen.getByText(scalingLatencyLabel).parentElement).toHaveTextContent("—");
    expect(screen.getByText(strings.dictationStudio.scalingComputeFit).parentElement).toHaveTextContent("power");
    expect(screen.getByText(latencyDetail)).toBeInTheDocument();
    expect(screen.getByText(emptyUnit).parentElement).toHaveTextContent("/s");
    expect(screen.getByText(firstReason)).toBeInTheDocument();
    expect(screen.queryByText(hiddenReason)).not.toBeInTheDocument();
    expect(screen.getByText(infoWarning)).toBeInTheDocument();
  });

  it("shows non-error clip edits, limits the drilldown, and handles empty edit tokens", () => {
    const clips = Array.from({ length: 10 }, (_, index) => ({
      clipId: `clip-${index}`,
      wer: index === 9 ? 0 : index / 10,
      substitutions: 1, insertions: 1, deletions: 1, whisperCalls: 1, partialRevisions: 0,
      finalizationLatencyP95Ms: 0, reference: "ref", hypothesis: "hyp", normalizedReference: "",
      normalizedHypothesis: "", editOperations: [
        { kind: "insertion", referenceToken: "", hypothesisToken: "added" },
        { kind: "deletion", referenceToken: "removed", hypothesisToken: "" },
        { kind: "substitution", referenceToken: "old", hypothesisToken: "new" },
        { kind: "match", referenceToken: "same", hypothesisToken: "same" },
        { kind: "match", referenceToken: "", hypothesisToken: "" },
      ],
      ...(index === 0 ? { error: "failure" } : {}),
    }));
    render(<EvalReportTable report={{
      qualityMeasured: true, latencyMeasured: false, warnings: [], perStrategy: [{
        strategy: "batch", label: "Batch", wer: 0, substitutions: 0, insertions: 0, deletions: 0,
        whisperCalls: 0, whisperAudioSeconds: 0, rtf: 0, finalizationLatencyP50Ms: 0,
        finalizationLatencyP95Ms: 0, partialRevisions: 0, werDeltaVsWinner: 0, p95DeltaMsVsWinner: 0,
        reasons: [], warnings: [], perClip: clips,
      }],
    } as never} />);

    expect(document.querySelectorAll('details[data-testid^="dictation-eval-clip-"]')).toHaveLength(8);
    expect(screen.getAllByText(addedToken)).toHaveLength(8);
    expect(screen.getAllByText(removedToken)).toHaveLength(8);
    expect(screen.getAllByText(replacementToken)).toHaveLength(8);
    expect(screen.getAllByText(matchedToken)).toHaveLength(8);
    expect(document.querySelectorAll('span[title="match"]')).toHaveLength(16);
    expect(screen.getAllByText(/dictationStudio.colP95: —/)).toHaveLength(8);
    expect(screen.queryByText(hiddenClip)).not.toBeInTheDocument();
  });
});
