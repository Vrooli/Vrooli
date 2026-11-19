import { screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { IntegrationCard } from "../../../library/components/IntegrationCard/versions/0.1.5/IntegrationCard";
import { renderWithProviders } from "../test-utils/renderWithProviders";

// Default copy owned by the published IntegrationCard draft.
const connectionLabel = "Connection";
const providerLabel = "Provider";

describe("IntegrationCard draft", () => {
  it("retains visible defaults when identity and status props are omitted", () => {
    renderWithProviders(<IntegrationCard />);

    expect(screen.getByText(connectionLabel)).toBeInTheDocument();
    expect(screen.getByText(providerLabel)).toBeInTheDocument();
    expect(screen.getByRole("status", { name: "Connection status: Unknown" })).toBeInTheDocument();
  });
});
