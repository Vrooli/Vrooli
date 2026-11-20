// provider-free-exception: this test mounts RegionCorrectionProvider itself; the canonical
// wrapper would nest a second provider and fight over the same context.
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { act, fireEvent, render, screen } from "@testing-library/react";

import { REGION_CORRECTION_SCHEMA, REGION_LONG_PRESS_MS, RegionCorrectionProvider, type RegionCorrectionEvidence, type ResolvedRegionCorrection } from "./RegionCorrectionProvider";
import { strings } from "../consts/strings";

const evidence: RegionCorrectionEvidence = {
  schema: REGION_CORRECTION_SCHEMA,
  kind: "correction",
  correction_id: "correction-42",
  origin: { kind: "served-page", id: "capture-7" },
  owner_scenario: "example-owner",
  corpus: "ui-health-ex1-private",
  storage: "local",
  private: true,
  capture: { id: "capture-7", build_id: `sha256:${"a".repeat(64)}`, route: "/settings", state_id: "ready", data_id: "fixture-set-1", fidelity: "captured", viewport: { width: 390, height: 844 }, image: { path: "captures/current.png", sha256: "c".repeat(64) } },
  region: {
    id: "region-submit",
    node_id: "node-8",
    label: "Submit button",
    component: "SettingsForm",
    evidence_id: "evidence-7",
    crop: { path: "private/crop.png", sha256: "b".repeat(64) },
    bounds: { x: 0.1, y: 0.2, width: 0.3, height: 0.1 },
  },
};
const resolved: ResolvedRegionCorrection = { evidence, cropPreviewUrl: "/local/private/evidence/crop-7.png" };

const rejectedRegion = "Rejected page region";
const pageRegion = "A page region";
const scrollingRegion = "Scrollable content";
const unversionedRegion = "Unversioned content";
const unsafePathRegion = "Unsafe evidence path";
const staleRegion = "Stale page region";
const tappableRegion = "A region selected by tap";

/**
 * Renders a provider whose host reports evidence that must fail validation,
 * then long-presses the marked region. Validation is the only gate before the
 * dialog opens, so a clean rejection shows the status line and no dialog.
 */
async function expectRejected(badValue: unknown) {
  const resolveEvidence = vi.fn();
  resolveEvidence.mockResolvedValue(badValue);
  render(
    <RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={vi.fn()}>
      <section data-ux-feedback-target>{rejectedRegion}</section>
    </RegionCorrectionProvider>,
  );
  fireEvent.pointerDown(screen.getByText(rejectedRegion), {
    pointerType: "touch",
    button: 0,
    clientX: 20,
    clientY: 20,
  });
  await act(async () => {
    vi.advanceTimersByTime(REGION_LONG_PRESS_MS);
    await Promise.resolve();
  });
  expect(screen.queryByRole("dialog")).toBeNull();
  expect(screen.getByRole("status")).toHaveTextContent("No correction was sent");
}

describe("RegionCorrectionProvider", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => { vi.useRealTimers(); document.getElementById("rcl-layer-root")?.remove(); });

  it("works as a root wrapper without AppShell and submits the exact selected-region evidence", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue(resolved);
    const submitCorrection = vi.fn().mockResolvedValue(undefined);
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={submitCorrection}>
      <main><section data-ux-feedback-target data-ux-node-id="node-8" data-ux-component="SettingsForm">{pageRegion}</section></main>
    </RegionCorrectionProvider>);

    fireEvent.pointerDown(screen.getByText(pageRegion), { pointerType: "touch", button: 0, clientX: 24, clientY: 36 });
    await act(async () => { vi.advanceTimersByTime(REGION_LONG_PRESS_MS); await Promise.resolve(); });
    expect(screen.getByRole("dialog", { name: strings.regionCorrection.title })).toBeTruthy();
    fireEvent.change(screen.getByLabelText(strings.regionCorrection.messageLabel), { target: { value: "This action is hard to find." } });
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: strings.regionCorrection.submit })); await Promise.resolve(); });
    expect(submitCorrection).toHaveBeenCalledWith({ ...evidence, text: "This action is hard to find." });
    expect(screen.getByRole("status")).toHaveTextContent("Correction queued locally with its page evidence.");
  });

  it("cancels the gesture when the finger moves so ordinary scrolling remains ordinary", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue(resolved);
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={vi.fn()}>
      <section data-ux-feedback-target>{scrollingRegion}</section>
    </RegionCorrectionProvider>);
    const target = screen.getByText(scrollingRegion);
    fireEvent.pointerDown(target, { pointerType: "touch", button: 0, clientX: 20, clientY: 20 });
    fireEvent.pointerMove(target, { pointerType: "touch", clientX: 33, clientY: 20 });
    await act(async () => { vi.advanceTimersByTime(REGION_LONG_PRESS_MS); await Promise.resolve(); });
    expect(resolveEvidence).not.toHaveBeenCalled();
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("refuses incomplete evidence instead of sending a location guess", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue({ ...resolved, evidence: { ...evidence, capture: { ...evidence.capture, build_id: "unknown" } } });
    const submitCorrection = vi.fn();
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={submitCorrection}>
      <section data-ux-feedback-target>{unversionedRegion}</section>
    </RegionCorrectionProvider>);
    fireEvent.pointerDown(screen.getByText(unversionedRegion), { pointerType: "touch", button: 0, clientX: 20, clientY: 20 });
    await act(async () => { vi.advanceTimersByTime(REGION_LONG_PRESS_MS); await Promise.resolve(); });
    expect(screen.getByRole("status")).toHaveTextContent("No correction was sent");
    expect(submitCorrection).not.toHaveBeenCalled();
  });

  it("refuses evidence paths that escape or alias the local bundle", async () => {
    const unsafe = { ...evidence, capture: { ...evidence.capture, image: { ...evidence.capture.image, path: "captures/../private.png" } } };
    const resolveEvidence = vi.fn().mockResolvedValue({ ...resolved, evidence: unsafe });
    const submitCorrection = vi.fn();
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={submitCorrection}>
      <section data-ux-feedback-target>{unsafePathRegion}</section>
    </RegionCorrectionProvider>);
    fireEvent.pointerDown(screen.getByText(unsafePathRegion), { pointerType: "touch", button: 0, clientX: 20, clientY: 20 });
    await act(async () => { vi.advanceTimersByTime(REGION_LONG_PRESS_MS); await Promise.resolve(); });
    expect(screen.getByRole("status")).toHaveTextContent("No correction was sent");
    expect(submitCorrection).not.toHaveBeenCalled();
  });

  it("requires a served-page origin to match its capture identity", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue({ ...resolved, evidence: { ...evidence, origin: { kind: "served-page" as const, id: "stale-capture" } } });
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={vi.fn()}>
      <section data-ux-feedback-target>{staleRegion}</section>
    </RegionCorrectionProvider>);
    fireEvent.pointerDown(screen.getByText(staleRegion), { pointerType: "touch", button: 0, clientX: 20, clientY: 20 });
    await act(async () => { vi.advanceTimersByTime(REGION_LONG_PRESS_MS); await Promise.resolve(); });
    expect(screen.queryByRole("dialog")).toBeNull();
    expect(screen.getByRole("status")).toHaveTextContent("No correction was sent");
  });

  it("offers an accessible tap-to-select path for users who cannot long-press", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue(resolved);
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={vi.fn()}>
      <section data-ux-feedback-target>{tappableRegion}</section>
    </RegionCorrectionProvider>);
    fireEvent.click(screen.getByRole("button", { name: strings.regionCorrection.report }));
    expect(screen.getByRole("status")).toHaveTextContent(strings.regionCorrection.tapInstruction);
    await act(async () => { fireEvent.click(screen.getByText(tappableRegion)); await Promise.resolve(); });
    expect(resolveEvidence).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("dialog")).toBeTruthy();
  });

  it("rejects host evidence whose schema is not this contract", async () => {
    await expectRejected({ ...resolved, evidence: { ...evidence, schema: "ux-region-feedback/v2" } });
  });

  it("rejects host evidence whose kind is not a correction", async () => {
    await expectRejected({ ...resolved, evidence: { ...evidence, kind: "feedback" } });
  });

  it("rejects host evidence that is not marked private", async () => {
    await expectRejected({ ...resolved, evidence: { ...evidence, private: false } });
  });

  it("rejects host evidence with no origin", async () => {
    const { origin: _origin, ...withoutOrigin } = evidence;
    await expectRejected({ ...resolved, evidence: withoutOrigin });
  });

  it("rejects host evidence whose bounds fall outside the capture", async () => {
    await expectRejected({
      ...resolved,
      evidence: {
        ...evidence,
        region: { ...evidence.region, bounds: { x: 0.9, y: 0.2, width: 0.5, height: 0.1 } },
      },
    });
  });

  it("can target an explicitly marked button without triggering its normal action", async () => {
    const resolveEvidence = vi.fn().mockResolvedValue(resolved);
    const action = vi.fn();
    render(<RegionCorrectionProvider resolveEvidence={resolveEvidence} submitCorrection={vi.fn()}>
      <button data-ux-feedback-target onClick={action}>Save region</button>
    </RegionCorrectionProvider>);
    fireEvent.click(screen.getByRole("button", { name: strings.regionCorrection.report }));
    await act(async () => { fireEvent.click(screen.getByRole("button", { name: "Save region" })); await Promise.resolve(); });
    expect(action).not.toHaveBeenCalled();
    expect(resolveEvidence).toHaveBeenCalledOnce();
    expect(screen.getByRole("dialog")).toBeTruthy();
  });
});
