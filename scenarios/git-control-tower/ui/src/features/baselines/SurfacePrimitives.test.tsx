import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { SurfaceCaptureEmptyState } from "./SurfaceCaptureEmptyState";
import { BaselineSelector } from "./BaselineSelector";
import { SurfaceBaselineBar } from "./SurfaceBaselineBar";
import { SurfaceComparePanel } from "./SurfaceComparePanel";
import type { CompareOnDemand } from "../../lib/hooks-baselines";

// The selector + compare panel read baselines/default through hooks-baselines;
// mock that module so these primitives are tested in isolation.
const useBaselines = vi.fn();
const useDefaultBaseline = vi.fn();
const setDefaultBaseline = vi.fn();
const useCompareOnDemand = vi.fn();

vi.mock("../../lib/hooks-baselines", () => ({
  useBaselines: (...a: unknown[]) => useBaselines(...a),
  useDefaultBaseline: (...a: unknown[]) => useDefaultBaseline(...a),
  useCompareOnDemand: (...a: unknown[]) => useCompareOnDemand(...a),
}));

function compareHandle(overrides: Partial<CompareOnDemand> = {}): CompareOnDemand {
  return {
    comparing: false,
    start: vi.fn(),
    exit: vi.fn(),
    baselineName: "plan-7c3",
    diff: undefined,
    isRunning: false,
    error: null,
    ...overrides,
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  useBaselines.mockReturnValue({ data: [{ name: "plan-7c3", branch: "agi" }], isLoading: false });
  useDefaultBaseline.mockReturnValue({ defaultBaselineName: "plan-7c3", setDefaultBaseline });
});

describe("SurfaceCaptureEmptyState", () => {
  it("fires both capture intents when the service is available", () => {
    const onCaptureLoose = vi.fn();
    const onCaptureBaseline = vi.fn();
    render(
      <SurfaceCaptureEmptyState
        surface="tests"
        hasService
        onCaptureLoose={onCaptureLoose}
        onCaptureBaseline={onCaptureBaseline}
        captureLabel="Run tests"
      />,
    );

    expect(screen.getByText("No tests captured yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /run tests/i }));
    fireEvent.click(screen.getByRole("button", { name: /capture baseline/i }));
    expect(onCaptureLoose).toHaveBeenCalledOnce();
    expect(onCaptureBaseline).toHaveBeenCalledOnce();
  });

  it("disables capture and explains when the service is unavailable", () => {
    render(
      <SurfaceCaptureEmptyState
        surface="tests"
        hasService={false}
        onCaptureLoose={vi.fn()}
        onCaptureBaseline={vi.fn()}
        serviceMessage="Start test-genie to run tests"
      />,
    );
    expect(screen.getByText("Start test-genie to run tests")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /capture baseline/i })).not.toBeInTheDocument();
  });
});

describe("BaselineSelector", () => {
  it("lists baselines and sets the default on change", () => {
    useBaselines.mockReturnValue({
      data: [{ name: "plan-7c3", branch: "agi" }, { name: "pre-launch", branch: "agi" }],
      isLoading: false,
    });
    render(<BaselineSelector scenario="s" onOpenBaselines={vi.fn()} />);

    fireEvent.change(screen.getByRole("combobox"), { target: { value: "pre-launch" } });
    expect(setDefaultBaseline).toHaveBeenCalledWith("pre-launch");
  });

  it("collapses to Open Baselines when none exist", () => {
    useBaselines.mockReturnValue({ data: [], isLoading: false });
    const onOpenBaselines = vi.fn();
    render(<BaselineSelector scenario="s" onOpenBaselines={onOpenBaselines} />);

    fireEvent.click(screen.getByRole("button", { name: /open baselines/i }));
    expect(onOpenBaselines).toHaveBeenCalledOnce();
  });
});

describe("SurfaceBaselineBar", () => {
  it("starts a compare and shows Capture baseline + Open Baselines", () => {
    const compare = compareHandle();
    render(
      <SurfaceBaselineBar
        scenario="s"
        compare={compare}
        onOpenBaselines={vi.fn()}
        onCaptureBaseline={vi.fn()}
        viewingLabel="latest run"
      />,
    );

    expect(screen.getByText(/viewing: latest run/i)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /^compare$/i }));
    expect(compare.start).toHaveBeenCalledOnce();
    expect(screen.getByRole("button", { name: /capture baseline/i })).toBeInTheDocument();
  });

  it("offers Exit compare while comparing", () => {
    const compare = compareHandle({ comparing: true });
    render(<SurfaceBaselineBar scenario="s" compare={compare} onOpenBaselines={vi.fn()} />);

    fireEvent.click(screen.getByRole("button", { name: /exit compare/i }));
    expect(compare.exit).toHaveBeenCalledOnce();
  });

  it("disables Compare when no baseline is selected", () => {
    const compare = compareHandle({ baselineName: "" });
    render(<SurfaceBaselineBar scenario="s" compare={compare} onOpenBaselines={vi.fn()} />);
    expect(screen.getByRole("button", { name: /^compare$/i })).toBeDisabled();
  });
});

describe("SurfaceComparePanel", () => {
  it("renders the surface diff body once comparing", () => {
    useCompareOnDemand.mockReturnValue(
      compareHandle({
        comparing: true,
        diff: {
          surfaces: [
            {
              surfaceId: "tests",
              verdict: "regression",
              summary: "1 regression",
              regressions: ["TestFoo"],
              newFailures: [],
              preexisting: [],
              cleared: [],
            },
          ],
        } as unknown as CompareOnDemand["diff"],
      }),
    );

    render(<SurfaceComparePanel scenario="s" surface="tests" onOpenBaselines={vi.fn()} />);
    expect(screen.getByText("Regressions (1)")).toBeInTheDocument();
    expect(screen.getByText("TestFoo")).toBeInTheDocument();
  });

  it("does not render a diff body before comparing", () => {
    useCompareOnDemand.mockReturnValue(compareHandle({ comparing: false }));
    render(<SurfaceComparePanel scenario="s" surface="tests" onOpenBaselines={vi.fn()} />);
    // Bar is present; no diff frame.
    expect(screen.queryByText(/match the baseline/i)).not.toBeInTheDocument();
  });
});
