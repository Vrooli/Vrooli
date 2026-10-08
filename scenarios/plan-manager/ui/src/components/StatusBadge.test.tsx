import { CircleCheck } from "lucide-react";
import { describe, expect, it } from "vitest";

import { renderWithProviders } from "../test-utils";
import { strings } from "../consts/strings";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders the icon branch and custom classes", () => {
    const { getByTestId } = renderWithProviders(
      <StatusBadge
        data-testid="badge"
        className="extra-class"
        descriptor={{ tone: "success", labelKey: strings.verdict.pass }}
        icon={<CircleCheck data-testid="badge-icon" />}
      />,
    );

    expect(getByTestId("badge")).toHaveClass("extra-class");
    expect(getByTestId("badge-icon")).toBeInTheDocument();
  });
});
