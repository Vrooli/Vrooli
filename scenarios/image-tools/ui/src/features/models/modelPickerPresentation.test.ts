/**
 * Unit tests for `modelPickerPresentation` — the PURE derivation layer that
 * turns a host-annotated `CandidateModel` into the chips, action, and host line
 * the picker renders. Asserts on stable key paths + tones (never translated
 * copy) so locale edits can't break it, and table-tests every `fitClass` and
 * `readyState` the contract defines.
 */
import { describe, expect, it } from "vitest";

import { strings } from "../../consts/strings";
import {
  makeBackendReadiness,
  makeCandidateModel,
  makeHostSummary,
  makeModelFit,
} from "./mocks/factories";
import {
  PICKER_TONE_CLASS,
  actionFor,
  alsoNeedsBackend,
  fitChip,
  hostSummaryLine,
  present,
  statusChip,
  supportChip,
  type PickerTone,
} from "./modelPickerPresentation";

const gpuHost = makeHostSummary({ hasGpu: true });
const cpuHost = makeHostSummary({ hasGpu: false, gpuName: "", vramKnown: false });

describe("fitChip", () => {
  it("reads a GPU-viable model as a positive 'runs on your GPU' badge", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "gpu" }) });
    expect(fitChip(candidate, gpuHost)).toEqual({
      key: strings.models.picker.fit.gpu,
      tone: "positive",
    });
  });

  it("reads a CPU-capable model on a CPU-only host as a positive CPU badge", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "cpu" }) });
    expect(fitChip(candidate, cpuHost)).toEqual({
      key: strings.models.picker.fit.cpu,
      tone: "positive",
    });
  });

  it("reads a CPU model on a GPU host as a neutral CPU fallback (not a failure)", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "cpu" }) });
    expect(fitChip(candidate, gpuHost)).toEqual({
      key: strings.models.picker.fit.cpuFallback,
      tone: "neutral",
    });
  });

  it("surfaces the VRAM shortfall for an insufficient_vram fit (defaulting to ≥1 GB)", () => {
    const short = makeCandidateModel({
      fit: makeModelFit({ fitClass: "insufficient_vram", vramShortfallGb: 6 }),
    });
    expect(fitChip(short, gpuHost)).toEqual({
      key: strings.models.picker.fit.insufficientVram,
      values: { gb: 6 },
      tone: "caution",
    });

    const noShortfall = makeCandidateModel({
      fit: makeModelFit({ fitClass: "insufficient_vram", vramShortfallGb: 0 }),
    });
    expect(fitChip(noShortfall, gpuHost).values).toEqual({ gb: 1 });
  });

  it("reads a no_gpu fit as a caution chip", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "no_gpu" }) });
    expect(fitChip(candidate, gpuHost)).toEqual({
      key: strings.models.picker.fit.noGpu,
      tone: "caution",
    });
  });

  it("reads an unsupported_os fit as a muted chip carrying the host os/arch", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "unsupported_os" }) });
    const host = makeHostSummary({ os: "darwin", arch: "arm64" });
    expect(fitChip(candidate, host)).toEqual({
      key: strings.models.picker.fit.unsupportedOs,
      values: { os: "darwin", arch: "arm64" },
      tone: "muted",
    });
  });

  it("falls back to a neutral CPU chip for an unknown/empty fit class", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "" }) });
    expect(fitChip(candidate, gpuHost)).toEqual({
      key: strings.models.picker.fit.cpu,
      tone: "neutral",
    });
    // Also covers an entirely missing fit message.
    const noFit = makeCandidateModel({ fit: undefined });
    expect(fitChip(noFit, gpuHost).tone).toBe("neutral");
  });

  it("treats unsupported_os with no host as empty interpolation (no crash)", () => {
    const candidate = makeCandidateModel({ fit: makeModelFit({ fitClass: "unsupported_os" }) });
    expect(fitChip(candidate, undefined).values).toEqual({ os: "", arch: "" });
  });
});

describe("statusChip", () => {
  const cases: Array<[string, string, PickerTone]> = [
    ["ready", strings.models.picker.state.ready, "positive"],
    ["needs_model_install", strings.models.picker.state.needsModel, "info"],
    ["needs_backend", strings.models.picker.state.needsBackend, "info"],
    ["needs_backend_manual", strings.models.picker.state.needsBackendManual, "caution"],
    ["needs_both", strings.models.picker.state.needsBoth, "info"],
    ["disabled", strings.models.picker.state.disabled, "neutral"],
    ["insufficient", strings.models.picker.state.insufficient, "muted"],
    ["unsupported", strings.models.picker.state.unsupported, "muted"],
  ];

  it.each(cases)("maps ready_state '%s' to its status chip", (readyState, key, tone) => {
    const chip = statusChip(makeCandidateModel({ readyState }));
    expect(chip).toEqual({ key, tone });
  });

  it("falls back to a neutral ready chip for an unknown ready_state", () => {
    expect(statusChip(makeCandidateModel({ readyState: "???" }))).toEqual({
      key: strings.models.picker.state.ready,
      tone: "neutral",
    });
  });
});

describe("actionFor", () => {
  const cases: Array<[string, ReturnType<typeof actionFor>]> = [
    ["ready", "select"],
    ["needs_model_install", "install-model"],
    ["needs_both", "install-model"],
    ["needs_backend", "install-backend"],
    ["needs_backend_manual", "manual"],
    ["disabled", "enable"],
    ["insufficient", "none"],
    ["unsupported", "none"],
  ];

  it.each(cases)("ready_state '%s' offers the '%s' action", (readyState, action) => {
    expect(actionFor(makeCandidateModel({ readyState }))).toBe(action);
  });
});

describe("present", () => {
  it("marks a ready candidate selectable and not dimmed", () => {
    const view = present(makeCandidateModel({ readyState: "ready" }), gpuHost);
    expect(view.selectable).toBe(true);
    expect(view.dimmed).toBe(false);
    expect(view.action).toBe("select");
    expect(view.fit.key).toBe(strings.models.picker.fit.gpu);
    expect(view.status.key).toBe(strings.models.picker.state.ready);
  });

  it("dims an insufficient and an unsupported candidate", () => {
    expect(present(makeCandidateModel({ readyState: "insufficient" }), gpuHost).dimmed).toBe(true);
    expect(present(makeCandidateModel({ readyState: "unsupported" }), gpuHost).dimmed).toBe(true);
  });

  it("a needs_model_install candidate is neither selectable nor dimmed", () => {
    const view = present(makeCandidateModel({ readyState: "needs_model_install" }), gpuHost);
    expect(view.selectable).toBe(false);
    expect(view.dimmed).toBe(false);
  });

  it("surfaces the support chip + caveat for a derived candidate and dims it when unproven", () => {
    const view = present(
      makeCandidateModel({
        readyState: "derived_pipeline_unproven",
        support: "derived",
        technique: "diffusers-inpaint",
        caveat: "derived: a base checkpoint inpaints via the standard pipeline",
      }),
      gpuHost,
    );
    expect(view.support?.key).toBe(strings.models.picker.support.viaWorkflow);
    expect(view.caveat).toContain("standard pipeline");
    expect(view.selectable).toBe(false);
    expect(view.dimmed).toBe(true);
    expect(view.action).toBe("none");
  });

  it("a native candidate carries no support chip and no caveat", () => {
    const view = present(makeCandidateModel({ readyState: "ready", support: "native" }), gpuHost);
    expect(view.support).toBeUndefined();
    expect(view.caveat).toBeUndefined();
  });
});

describe("supportChip", () => {
  it("returns a via-workflow caution chip for a derived candidate", () => {
    const chip = supportChip(makeCandidateModel({ support: "derived" }));
    expect(chip).toEqual({ key: strings.models.picker.support.viaWorkflow, tone: "caution" });
  });

  it("returns undefined for a native candidate", () => {
    expect(supportChip(makeCandidateModel({ support: "native" }))).toBeUndefined();
    expect(supportChip(makeCandidateModel({ support: "" }))).toBeUndefined();
  });
});

describe("alsoNeedsBackend", () => {
  it("is true only for needs_both with an auto-installable backend", () => {
    const both = makeCandidateModel({
      readyState: "needs_both",
      backend: makeBackendReadiness({ installTier: "auto" }),
    });
    expect(alsoNeedsBackend(both)).toBe(true);
  });

  it("is false for needs_both when the backend is manual", () => {
    const both = makeCandidateModel({
      readyState: "needs_both",
      backend: makeBackendReadiness({ installTier: "manual" }),
    });
    expect(alsoNeedsBackend(both)).toBe(false);
  });

  it("is false for any non-needs_both ready_state", () => {
    expect(alsoNeedsBackend(makeCandidateModel({ readyState: "needs_model_install" }))).toBe(false);
  });
});

describe("hostSummaryLine", () => {
  it("reports no-GPU for a missing host", () => {
    expect(hostSummaryLine(undefined)).toEqual({
      key: strings.models.picker.host.noGpu,
      tone: "neutral",
    });
  });

  it("reports no-GPU for a CPU-only host", () => {
    expect(hostSummaryLine(makeHostSummary({ hasGpu: false }))).toEqual({
      key: strings.models.picker.host.noGpu,
      tone: "neutral",
    });
  });

  it("reports a GPU with unknown VRAM by name", () => {
    const host = makeHostSummary({ hasGpu: true, gpuName: "RTX 3060", vramKnown: false });
    expect(hostSummaryLine(host)).toEqual({
      key: strings.models.picker.host.gpuUnknown,
      values: { name: "RTX 3060" },
      tone: "neutral",
    });
  });

  it("reports a GPU with known VRAM totals", () => {
    const host = makeHostSummary({
      hasGpu: true,
      gpuName: "RTX 4090",
      vramKnown: true,
      vramTotalGb: 24,
      vramFreeGb: 20,
    });
    expect(hostSummaryLine(host)).toEqual({
      key: strings.models.picker.host.gpu,
      values: { name: "RTX 4090", total: 24, free: 20 },
      tone: "neutral",
    });
  });
});

describe("PICKER_TONE_CLASS", () => {
  it("maps every tone to a paired background + text class (never color-only)", () => {
    const tones: PickerTone[] = ["positive", "info", "caution", "muted", "neutral"];
    for (const tone of tones) {
      const cls = PICKER_TONE_CLASS[tone];
      expect(cls).toMatch(/bg-/);
      expect(cls).toMatch(/text-/);
    }
  });
});
