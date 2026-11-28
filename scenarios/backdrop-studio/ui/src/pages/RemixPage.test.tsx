import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { makeStudioMocks, makeStyle, renderWithProviders } from "../test-utils";
import { RemixPage } from "./RemixPage";

vi.mock("../api/studio", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/studio")>();
  return { ...actual, ...makeStudioMocks() };
});

async function openFork() {
  renderWithProviders(<RemixPage />, { routerEntries: ["/remix?parent=cyanotype-arcade"] });
  await screen.findByTestId("remix-comparison");
  fireEvent.change(screen.getByTestId("remix-value-select"), { target: { value: "bauhaus" } });
  fireEvent.change(screen.getByTestId("remix-id-input"), { target: { value: "bauhaus-arcade" } });
  fireEvent.click(screen.getByRole("button", { name: /renderBoth/i }));
}

describe("RemixPage", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("asks for a parent before offering to fork", async () => {
    renderWithProviders(<RemixPage />, { routerEntries: ["/remix"] });
    expect(await screen.findByText(strings.pages.remix.chooseStyle)).toBeInTheDocument();
  });

  it("renders the fork beside its parent", async () => {
    const { listStyles } = await import("../api/studio");
    vi.mocked(listStyles).mockResolvedValue([
      makeStyle(),
      makeStyle({ id: "other", lineage: "bauhaus" }),
    ]);
    await openFork();
    expect(await screen.findByTestId("remix-preview-cyanotype-arcade")).toBeInTheDocument();
    expect(screen.getByTestId("remix-preview-bauhaus-arcade")).toBeInTheDocument();
  });

  it("saves the fork with its parent recorded as lineage", async () => {
    const { listStyles, createStyle } = await import("../api/studio");
    vi.mocked(listStyles).mockResolvedValue([
      makeStyle(),
      makeStyle({ id: "other", lineage: "bauhaus" }),
    ]);
    await openFork();
    fireEvent.click(screen.getByRole("button", { name: /remix\.save/i }));
    await waitFor(() =>
      expect(vi.mocked(createStyle)).toHaveBeenCalledWith(
        expect.objectContaining({
          id: "bauhaus-arcade",
          lineage: "bauhaus",
          parentId: "cyanotype-arcade",
        }),
      ),
    );
  });

  it("reports a refused save with the server's own message", async () => {
    const { listStyles, createStyle } = await import("../api/studio");
    vi.mocked(listStyles).mockResolvedValue([
      makeStyle(),
      makeStyle({ id: "other", lineage: "bauhaus" }),
    ]);
    vi.mocked(createStyle).mockRejectedValue(new Error("invalid lineage value"));
    await openFork();
    fireEvent.click(screen.getByRole("button", { name: /remix\.save/i }));
    expect(await screen.findByRole("alert")).toHaveTextContent("invalid lineage value");
  });

  it("announces loading while the catalog is in flight", () => {
    renderWithProviders(<RemixPage />, { routerEntries: ["/remix?parent=cyanotype-arcade"] });
    expect(screen.getByTestId(selectors.pages.remix)).toHaveAttribute(
      "data-experience-state",
      "loading",
    );
  });
});
