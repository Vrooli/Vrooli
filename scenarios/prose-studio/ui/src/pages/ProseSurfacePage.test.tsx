import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { ProseSurfacePage, VariationBoard } from "./ProseSurfacePage";
import { renderWithProviders } from "../test-utils";

const renderSurface = (surface: "variation" | "styles" | "document" | "declarations") =>
  renderWithProviders(<ProseSurfacePage surface={surface} />);

describe("Prose Studio surfaces", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("keeps the variation board set-level and score-free while empty", () => {
    renderSurface("variation");
    expect(screen.getByRole("heading", { name: "Variation Board" })).toBeInTheDocument();
    expect(screen.getByText("Diversity basis")).toBeInTheDocument();
    expect(screen.queryByText(/quality score|candidate verdict/i)).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reroll/i })).toBeDisabled();
  });

  it("renders the style and document empty states with navigable actions", () => {
    renderSurface("styles");
    expect(screen.getByText("No declared styles registered")).toBeInTheDocument();
    cleanup();
    renderSurface("document");
    expect(screen.getByText("No document selected")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Open variation board" })).toHaveAttribute("href", "/variation");
  });

  it("renders candidate text and enables a negative reroll action without introducing a per-card quality ordering", async () => {
    renderWithProviders(<VariationBoard candidates={["A measured paragraph"]} />);
    expect(screen.getByLabelText("Candidate 1")).toHaveTextContent("A measured paragraph");
    const reroll = screen.getByRole("button", { name: /reroll/i });
    expect(reroll).toBeEnabled();
    await reroll.click();
    expect(screen.getByRole("status")).toHaveTextContent("Reroll requested");
  });

  it("shows measured diversity and transient generation states", () => {
    renderWithProviders(<VariationBoard candidates={["A measured paragraph"]} diversity={0.812} />);
    expect(screen.getByLabelText("Set diversity 0.812")).toHaveTextContent("0.81");
    cleanup();

    renderWithProviders(<VariationBoard loading />);
    expect(screen.getByRole("status")).toHaveTextContent("Generating a measured candidate set");
    cleanup();

    renderWithProviders(<VariationBoard error="Gateway unavailable" />);
    expect(screen.getByRole("alert")).toHaveTextContent("Gateway unavailable");
  });

  it("reports declaration validation failure and permits retry", async () => {
    const fetchMock = vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("gateway unavailable"));
    renderSurface("declarations");
    expect(await screen.findByRole("alert")).toHaveTextContent("Declaration validation failed");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    fetchMock.mockResolvedValue(new Response("[]", { status: 200, headers: { "Content-Type": "application/json" } }));
    await screen.getByRole("button", { name: "Retry validation" }).click();
    await waitFor(() => expect(screen.getByText("Declaration scan complete")).toBeInTheDocument());
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
