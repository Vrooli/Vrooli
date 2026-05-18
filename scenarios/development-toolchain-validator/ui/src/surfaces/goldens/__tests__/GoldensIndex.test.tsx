/**
 * GoldensIndex surface — loading / empty / data / error states + register-new
 * sheet wiring.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { create } from "@bufbuild/protobuf";
import {
  GoldenSchema,
  ListGoldensResponseSchema,
} from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";

import { selectors } from "../../../consts/selectors";
import { renderWithProviders } from "../../../test-utils";

vi.mock("../../../api/golden", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../../api/golden")>();
  return {
    ...actual,
    goldenClient: {
      listGoldens: vi.fn().mockResolvedValue({ goldens: [] }),
      getGolden: vi.fn(),
      registerGolden: vi.fn(),
      regenerateGolden: vi.fn(),
      deleteGolden: vi.fn(),
    },
  };
});

import { GoldensIndex } from "../GoldensIndex";
import { goldenClient } from "../../../api/golden";

const makeGolden = (slug: string) =>
  create(GoldenSchema, {
    slug,
    templateId: "react-vite",
    templateVersionPinned: "1.0.0",
    path: `fixtures/${slug}`,
  });

const emptyList = () => create(ListGoldensResponseSchema, { goldens: [] });

beforeEach(() => {
  vi.mocked(goldenClient.listGoldens).mockResolvedValue(emptyList());
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("GoldensIndex", () => {
  it("renders the loading skeleton while the query is pending", () => {
    vi.mocked(goldenClient.listGoldens).mockReturnValueOnce(new Promise(() => undefined));
    renderWithProviders(<GoldensIndex />);
    expect(screen.getByTestId(selectors.goldens.loading)).toBeInTheDocument();
  });

  it("renders the empty state when no goldens are registered", async () => {
    renderWithProviders(<GoldensIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.empty)).toBeInTheDocument();
    });
  });

  it("renders one row per golden when data is present", async () => {
    vi.mocked(goldenClient.listGoldens).mockResolvedValueOnce(
      create(ListGoldensResponseSchema, {
        goldens: [makeGolden("fixture-slug"), makeGolden("another")],
      }),
    );
    renderWithProviders(<GoldensIndex />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.goldens.row).length).toBe(2);
    });
  });

  it("opens the register-new sheet when the action is clicked", async () => {
    const user = userEvent.setup();
    renderWithProviders(<GoldensIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.registerOpen)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.goldens.registerOpen));
    expect(screen.getByTestId(selectors.goldens.registerForm)).toBeInTheDocument();
  });

  it("renders an error message when the query fails", async () => {
    vi.mocked(goldenClient.listGoldens).mockRejectedValueOnce(new Error("boom"));
    renderWithProviders(<GoldensIndex />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.error)).toBeInTheDocument();
    });
  });
});
