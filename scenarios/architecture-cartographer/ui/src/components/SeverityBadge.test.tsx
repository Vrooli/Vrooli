import { describe, expect, it } from "vitest";
import { render, screen } from "@testing-library/react";

import { selectors } from "../consts/selectors";
import { SeverityBadge } from "./SeverityBadge";

describe("SeverityBadge", () => {
  it("renders a label alongside the icon (no color-only signal)", () => {
    render(<SeverityBadge level="high" label="HIGH" />);
    const el = screen.getByTestId(selectors.shared.severityBadge.root({ level: "high" }));
    expect(el).toHaveTextContent("HIGH");
  });

  it("emits a level-specific testid", () => {
    render(<SeverityBadge level="critical" label="C" />);
    expect(
      screen.getByTestId(selectors.shared.severityBadge.root({ level: "critical" })),
    ).toBeInTheDocument();
  });

  it("renders all five severity levels without error", () => {
    const levels = ["info", "low", "medium", "high", "critical"] as const;
    for (const lvl of levels) {
      render(<SeverityBadge level={lvl} label={lvl} />);
      expect(
        screen.getByTestId(selectors.shared.severityBadge.root({ level: lvl })),
      ).toBeInTheDocument();
    }
  });
});
