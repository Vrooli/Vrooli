import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils/renderWithProviders";
import { useBrand } from "./useBrand";

function BrandProbe() {
  const query = useBrand();
  return <output data-testid="brand-data">{JSON.stringify(query.data ?? null)}</output>;
}

describe("useBrand", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("uses the neutral fallback when Brand Manager has no assignment", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("", { status: 503 })));
    renderWithProviders(<BrandProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("brand-data")).toHaveTextContent(
        JSON.stringify({
          assigned: false,
          name: "React Component Library",
          mark: null,
          accent: null,
          wordmark: null,
        }),
      ),
    );
    expect(fetch).toHaveBeenCalledWith(expect.stringMatching(/\/api\/brand$/), {
      headers: { accept: "application/json" },
    });
  });

  it("normalizes an assigned brand and preserves its supplied identity assets", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            assigned: true,
            name: "  Acme  ",
            mark: "/mark.svg",
            accent: "#123456",
            wordmark: "data:image/svg+xml,wordmark",
          }),
          { status: 200, headers: { "content-type": "application/json" } },
        ),
      ),
    );
    renderWithProviders(<BrandProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("brand-data")).toHaveTextContent(
        JSON.stringify({
          assigned: true,
          name: "Acme",
          mark: "/mark.svg",
          accent: "#123456",
          wordmark: "data:image/svg+xml,wordmark",
        }),
      ),
    );
  });

  it.each([
    [null, false, "React Component Library"],
    [{ assigned: true, name: "   " }, false, "React Component Library"],
    [
      { assigned: false, name: "Unassigned Brand", mark: "", accent: "", wordmark: "" },
      false,
      "Unassigned Brand",
    ],
    [
      { assigned: true, name: "Assigned", mark: null, accent: null, wordmark: null },
      true,
      "Assigned",
    ],
  ])("handles incomplete assignment data (%j)", async (body, assigned, name) => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { "content-type": "application/json" },
        }),
      ),
    );
    renderWithProviders(<BrandProbe />);

    await waitFor(() =>
      expect(screen.getByTestId("brand-data")).toHaveTextContent(
        JSON.stringify({
          assigned,
          name,
          mark: body && "mark" in body ? (body.mark ?? null) : null,
          accent: body && "accent" in body ? (body.accent ?? null) : null,
          wordmark: body && "wordmark" in body ? (body.wordmark ?? null) : null,
        }),
      ),
    );
  });
});
