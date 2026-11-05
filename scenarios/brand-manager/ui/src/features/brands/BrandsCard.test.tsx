/**
 * BrandsCard tests — focused on the brands-card surface only. Renders
 * <BrandsCard /> directly so failures point at brands-feature behaviour, not
 * shell composition. Follows the canonical mock-builder pattern.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { makeBrand, makeColors, makeCreateBrandResponse, makeListBrandsResponse } from "./mocks/factories";
import { makeBrandsMocks } from "./mocks/brands";

vi.mock("../../api/brands", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/brands")>();
  return { ...actual, ...makeBrandsMocks() };
});

import { BrandsCard } from "./BrandsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("BrandsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the empty state when listBrands resolves with no brands", async () => {
    const { brandsClient } = await import("../../api/brands");
    vi.mocked(brandsClient.listBrands).mockResolvedValueOnce(makeListBrandsResponse());

    renderWithProviders(<BrandsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.brands.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.brands.list)).not.toBeInTheDocument();
  });

  it("renders the list with name + version when listBrands returns items", async () => {
    const { brandsClient } = await import("../../api/brands");
    vi.mocked(brandsClient.listBrands).mockResolvedValueOnce(
      makeListBrandsResponse({
        brands: [
          makeBrand({ id: "a", name: "Acme", version: 3, colors: makeColors({ primary: "#112233" }) }),
          makeBrand({ id: "b", name: "Globex", version: 1 }),
        ],
      }),
    );

    renderWithProviders(<BrandsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.brands.list)).toBeInTheDocument();
    });
    const list = screen.getByTestId(selectors.brands.list);
    expect(list.textContent).toContain("Acme");
    expect(list.textContent).toContain("Globex");
    expect(screen.getAllByTestId(selectors.brands.version)[0]?.textContent).toContain("3");
  });

  it("invokes createBrand when the create button is clicked", async () => {
    const { brandsClient } = await import("../../api/brands");
    vi.mocked(brandsClient.listBrands).mockResolvedValue(makeListBrandsResponse());
    vi.mocked(brandsClient.createBrand).mockResolvedValueOnce(
      makeCreateBrandResponse({ brand: makeBrand({ id: "new" }) }),
    );

    const user = userEvent.setup();
    renderWithProviders(<BrandsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.brands.createButton)).toBeInTheDocument();
    });

    await user.click(screen.getByTestId(selectors.brands.createButton));

    await waitFor(() => {
      expect(brandsClient.createBrand).toHaveBeenCalledTimes(1);
    });
    expect(vi.mocked(brandsClient.createBrand).mock.calls[0]?.[0]).toMatchObject({ name: expect.any(String) });
  });
});
