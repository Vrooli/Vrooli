import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { expectNoA11yViolations } from "../test-utils/a11y";
import { authoredSample } from "../test-utils/readings";
import { ladderFixture, ladderReading } from "../test-utils/ladder";
import { NextRungReadout } from "./NextRungReadout";

describe("NextRungReadout", () => {
  it("names the next release, what it opens, and the enabling work in its way", async () => { // [REQ:CC-P1-016]
    const { container } = renderWithProviders(<NextRungReadout reading={ladderReading()} />);
    expect(screen.getByLabelText("Rank 1")).toHaveTextContent("1");
    expect(screen.getByText("web-console")).toBeInTheDocument();
    expect(screen.getByText("Trigger met")).toBeInTheDocument();
    expect(screen.getByText("desktop")).toHaveAttribute("data-kind", "ramp");
    expect(screen.getByText("voice_minutes")).toHaveAttribute("data-kind", "stream");
    expect(screen.getByText("2 open")).toBeInTheDocument();
    expect(screen.getByText("Idea · direct enabler")).toBeInTheDocument();
    expect(screen.getByText("Idea · due by rank 1")).toBeInTheDocument();
    expect(screen.getByText("Offer Desk release-ladder contract")).toBeInTheDocument();
    expect(screen.getByText("not reported")).toHaveAttribute("data-gap", "true");
    expect(container.querySelector(".cc-next-rung__then")).toHaveTextContent("Then 2 system-monitor · 3 git-control-tower · 4 agent-manager · +1 more");
    expect(screen.getByText(/Unscheduled: treasury/)).toBeInTheDocument();
    expect(container.firstElementChild).toHaveAttribute("data-provenance", "measured");
    await expectNoA11yViolations(container);
  });
  it("marks an authored schedule as illustrative rather than measured", () => {
    const { container } = renderWithProviders(<NextRungReadout reading={ladderReading({ coverage: "IN-REACH", trust: "UNAVAILABLE", value: null, observedAt: null, ladder: undefined, sample: { ...authoredSample(5), ladder: ladderFixture() } })} />);
    expect(container.firstElementChild).toHaveAttribute("data-provenance", "sample");
    expect(screen.getByText("illustrative")).toBeInTheDocument();
  });
  it("says so when every scheduled release has shipped", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ ladder: ladderFixture({ nextRank: 0 }) })} />);
    expect(screen.getByText("Every scheduled release has shipped.")).toBeInTheDocument();
  });
  it("shows the untrusted reason instead of passing the schedule off as fact", () => {
    renderWithProviders(<NextRungReadout reading={ladderReading({ trust: "UNTRUSTED", trustReason: "producer did not supply observation time" })} />);
    expect(screen.getByText(/producer did not supply observation time/)).toBeInTheDocument();
  });
});
