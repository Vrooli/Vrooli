/**
 * AssetsCard tests — focused on the assets-card surface only. Renders
 * <AssetsCard /> directly so failures point at assets-feature behaviour, not
 * shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { makeAsset, makeListAssetsResponse } from "./mocks/factories";
import { makeAssetsMocks } from "./mocks/assets";

vi.mock("../../api/assets", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/assets")>();
  return { ...actual, ...makeAssetsMocks() };
});

import { AssetsCard } from "./AssetsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AssetsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listAssets resolves with no assets", async () => {
    const { assetsClient } = await import("../../api/assets");
    vi.mocked(assetsClient.listAssets).mockResolvedValueOnce(makeListAssetsResponse());

    renderWithProviders(<AssetsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.assets.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.assets.list)).not.toBeInTheDocument();
  });

  it("renders the list with filename + size when listAssets returns items", async () => {
    const { assetsClient } = await import("../../api/assets");
    vi.mocked(assetsClient.listAssets).mockResolvedValueOnce(
      makeListAssetsResponse({
        assets: [
          makeAsset({ id: "a", filename: "logo.png", size: 2048n, brandId: "b1" }),
          makeAsset({ id: "b", filename: "icon.svg", mimeType: "image/svg+xml", size: 512n }),
        ],
      }),
    );

    renderWithProviders(<AssetsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.assets.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.assets.list);
    expect(list.textContent).toContain("logo.png");
    expect(list.textContent).toContain("icon.svg");
    expect(screen.getAllByTestId(selectors.assets.size)[0]?.textContent).toContain("2.0 KB");
    expect(screen.getAllByTestId(selectors.assets.mimeType)[1]?.textContent).toContain("image/svg+xml");
  });
});
