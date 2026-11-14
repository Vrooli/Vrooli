import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { renderWithProviders } from "../../test-utils";
import { StatusBadge } from "./StatusBadge";

describe("StatusBadge", () => {
  it("renders each health status and safely handles an unknown runtime value", () => {
    renderWithProviders(
      <>
        <StatusBadge status="ok" />
        <StatusBadge status="warning" />
        <StatusBadge status="critical" />
        <StatusBadge status={"unknown" as never} />
      </>,
    );
    expect(screen.getByText("OK")).toBeInTheDocument();
    expect(screen.getByText("WARNING")).toBeInTheDocument();
    expect(screen.getByText("CRITICAL")).toBeInTheDocument();
    expect(screen.getByText("UNKNOWN")).toBeInTheDocument();
  });
});
