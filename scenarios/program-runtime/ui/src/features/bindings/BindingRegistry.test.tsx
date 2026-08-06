import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";

import { BindingRegistry } from "./BindingRegistry";
import { renderWithProviders } from "../../test-utils";
import { fetchBindings, fetchUnbound } from "../../api/bindings";

vi.mock("../../api/bindings", () => ({
  fetchBindings: vi.fn().mockResolvedValue([{ id: "fixture/list", effect: "read", signature: "FixtureService.ListItems()" }]),
  fetchUnbound: vi.fn().mockResolvedValue([{ scenario: "legacy", command: "run", reason: "UNBOUND_REASON_NO_MANIFEST", detail: "scenario has no CLI manifest" }]),
}));

describe("BindingRegistry", () => {
  it("renders bound and unbound capabilities with reasons", async () => { // [REQ:PRT-P1-007]
    renderWithProviders(<BindingRegistry />);
    expect(await screen.findByText("fixture/list")).toBeInTheDocument();
    expect(await screen.findByText("no manifest")).toBeInTheDocument();
  });

  it("renders explicit empty states", async () => {
    vi.mocked(fetchBindings).mockResolvedValueOnce([]);
    vi.mocked(fetchUnbound).mockResolvedValueOnce([]);
    renderWithProviders(<BindingRegistry />);
    expect(await screen.findByText("No governed bindings resolved.")).toBeInTheDocument();
    expect(await screen.findByText("No unbound capabilities reported.")).toBeInTheDocument();
  });

  it("renders a structured request failure", async () => {
    vi.mocked(fetchBindings).mockRejectedValueOnce(new Error("registry unavailable"));
    vi.mocked(fetchUnbound).mockResolvedValueOnce([]);
    renderWithProviders(<BindingRegistry />);
    expect(await screen.findByRole("alert")).toHaveTextContent("registry unavailable");
    expect(await screen.findByText("No unbound capabilities reported.")).toBeInTheDocument();
  });
});
