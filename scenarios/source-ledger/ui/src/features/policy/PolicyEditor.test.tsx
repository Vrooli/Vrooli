import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { PolicyEditor } from "./PolicyEditor";

const effective = {
  frontierTarget: 32,
  wakeBudgetLines: 96,
  wakeBudgetChars: 12000,
  maxEntryLines: 2,
  maxEntryChars: 200,
  frontierTargetOrigin: "scope-override",
  wakeBudgetLinesOrigin: "file-default",
  wakeBudgetCharsOrigin: "file-default",
  maxEntryLinesOrigin: "file-default",
  maxEntryCharsOrigin: "file-default",
};
const defaults = { ...effective, frontierTarget: 16, frontierTargetOrigin: "file-default" };

function policyResponse(withLiveness = true) {
  return {
    effective,
    defaults,
    ...(withLiveness ? { liveness: { unsummarizedLeafCount: 2, oldestUnsummarizedLeafAt: "2026-04-26T00:00:00Z", lastSummaryAt: "2026-04-25T00:00:00Z" } } : {}),
  };
}

describe("PolicyEditor", () => {
  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it("loads effective values, origins, defaults, and liveness", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValue(new Response(JSON.stringify(policyResponse()), { status: 200 }));

    renderWithProviders(<PolicyEditor scope="agent-memory" />);

    await waitFor(() => expect(screen.getByLabelText("Frontier target")).toHaveValue(32));
    expect(screen.getByText(/origin: scope-override · default: 16/)).toBeInTheDocument();
    expect(screen.getByText(/Unsummarized leaves: 2/)).toBeInTheDocument();
  });

  it("keeps a safe loading surface when the initial policy read fails", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValue(new Error("ledger unavailable"));
    renderWithProviders(<PolicyEditor scope="agent-memory" />);
    await waitFor(() => expect(screen.getByText("Loading policy…")).toBeInTheDocument());
    expect(screen.queryByRole("button", { name: "Save overrides" })).not.toBeInTheDocument();
  });

  it("saves only edited values", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async () => new Response(JSON.stringify(policyResponse()), { status: 200 }));
    renderWithProviders(<PolicyEditor scope="agent-memory" />);

    const input = await screen.findByLabelText("Whole-view characters");
    fireEvent.change(input, { target: { value: "8000" } });
    fireEvent.click(screen.getByRole("button", { name: "Save overrides" }));

    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Policy saved"));
    const saveRequest = fetchSpy.mock.calls.at(-1)?.[1];
    expect(JSON.parse(String(saveRequest?.body))).toEqual({ scope: "agent-memory", wakeBudgetChars: 8000 });
  });

  it("resets a scope and reports mutation failures", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementation(async (input) => {
      const url = String(input);
      if (url.includes("ResetPolicy")) {
        return new Response(JSON.stringify(policyResponse(false)), { status: 200 });
      }
      if (url.includes("SetPolicy")) return new Response("no", { status: 503 });
      return new Response(JSON.stringify(policyResponse()), { status: 200 });
    });
    renderWithProviders(<PolicyEditor scope="agent-memory" />);

    await screen.findByLabelText("Frontier target");
    fireEvent.click(screen.getByRole("button", { name: "Reset to defaults" }));
    await waitFor(() => expect(screen.getByRole("status")).toHaveTextContent("Policy reset"));
    expect(screen.queryByText(/Unsummarized leaves/)).not.toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Frontier target"), { target: { value: "64" } });
    fireEvent.click(screen.getByRole("button", { name: "Save overrides" }));
    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("503"));
    expect(fetchSpy).toHaveBeenCalled();
  });
});
