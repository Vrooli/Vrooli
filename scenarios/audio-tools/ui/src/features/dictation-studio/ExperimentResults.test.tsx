import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders as render } from "../../test-utils/renderWithProviders";
import { ExperimentResults, LiveExperimentProgress } from "./ExperimentResults";

describe("ExperimentResults", () => {
  it("offers loading a selected report and honours its pending state", () => {
    const load = vi.fn();
    render(
      <ExperimentResults
        report={null}
        selected={{ id: "exp-1", status: "queued" } as never}
        loadReportPending={false}
        onLoadReport={load}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "dictationStudio.loadReport" }));
    expect(load).toHaveBeenCalledWith("exp-1");
  });

  it("renders the empty result state when no experiment is selected", () => {
    render(<ExperimentResults report={null} selected={null} loadReportPending={false} onLoadReport={vi.fn()} />);
    expect(screen.getByText("dictationStudio.resultsEmpty")).toBeInTheDocument();
  });
});

describe("LiveExperimentProgress", () => {
  it("clamps event progress and shows a fallback explanation", () => {
    render(
      <LiveExperimentProgress
        event={{ progress: 160, message: "ignored", status: "running" } as never}
        fallbackMessage="Buffered fallback"
        status="running"
      />,
    );

    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "100");
    expect(screen.getByText("Buffered fallback")).toBeInTheDocument();
    expect(screen.getByText("dictationStudio.liveFallback")).toBeInTheDocument();
  });

  it("uses queued defaults when no event arrived", () => {
    render(<LiveExperimentProgress event={null} fallbackMessage="" status="queued" />);
    expect(screen.getByRole("progressbar")).toHaveAttribute("aria-valuenow", "0");
    expect(screen.getByText("dictationStudio.liveQueued")).toBeInTheDocument();
  });
});
