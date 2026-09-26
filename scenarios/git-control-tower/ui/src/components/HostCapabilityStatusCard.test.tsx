import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { HostCapabilityStatusCard } from "./HostCapabilityStatusCard";
import { renderWithProviders } from "../test-utils/renderWithProviders";

describe("HostCapabilityStatusCard", () => {
  it("renders capability-specific standing", () => {
    renderWithProviders(<HostCapabilityStatusCard hostLabel="GitHub · unavailable" capabilities={[{ capability: "reviews", standing: "unsupported" }, { capability: "checks", standing: "available" }]} />);
    expect(screen.getByText("unsupported")).toBeInTheDocument();
    expect(screen.getByText("available")).toBeInTheDocument();
  });
});
