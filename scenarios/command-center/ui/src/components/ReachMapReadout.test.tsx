import { describe, expect, it } from "vitest";
import { renderWithProviders, screen } from "../test-utils/renderWithProviders";
import { expectNoA11yViolations } from "../test-utils/a11y";
import { ladderReading } from "../test-utils/ladder";
import { ReachMapReadout } from "./ReachMapReadout";

describe("ReachMapReadout", () => {
  it("lights each ramp, stream and audience at the rung that opens it", async () => { // [REQ:CC-P1-016]
    const { container } = renderWithProviders(<ReachMapReadout reading={ladderReading()} />);
    const aiCredits = screen.getByText("ai_credits").closest("li");
    expect(aiCredits).toHaveAttribute("data-kind", "stream");
    expect(aiCredits).toHaveTextContent("opens at 4 · agent-manager");
    expect(aiCredits?.querySelector(".cc-reach__bar")).toHaveStyle({ gridColumn: "5 / -1" });
    expect(screen.getByText("mobile").closest("li")).toHaveAttribute("data-unopened", "true");
    expect(screen.getByText("not scheduled")).toBeInTheDocument();
    expect(container.querySelector(".cc-reach__col[data-next]")).toHaveAttribute("title", "1 web-console · Trigger met");
    expect(container.querySelector(".cc-reach__summary")).toHaveTextContent("By rank 5: 2/3 ramps · 2/2 streams · 3/3 audiences");
    expect(screen.getByText("2 goals follow these ranks")).toBeInTheDocument();
    await expectNoA11yViolations(container);
  });
});
