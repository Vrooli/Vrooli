import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";

import { renderWithProviders } from "../../test-utils";
import { makeGolden, makeGoldenMocks } from "./mocks";
import { ListGoldensResponseSchema } from "@vrooli/proto-types/development-toolchain-validator/v1/golden/golden_pb";

vi.mock("../../api/golden", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/golden")>();
  return { ...actual, ...makeGoldenMocks() };
});

import { GoldensCard } from "./GoldensCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("GoldensCard", () => {
  beforeEach(async () => {
    await setLocale("en");
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    vi.restoreAllMocks();
  });

  it("renders the empty state when there are no goldens", async () => {
    renderWithProviders(<GoldensCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.empty)).toBeInTheDocument();
    });
  });

  it("renders the list when goldens are returned", async () => {
    const { goldenClient } = await import("../../api/golden");
    vi.mocked(goldenClient.listGoldens).mockResolvedValueOnce(
      create(ListGoldensResponseSchema, {
        goldens: [
          makeGolden({ slug: "alpha", templateId: "react-vite", templateVersionPinned: "1.0.1" }),
          makeGolden({ slug: "bravo", templateId: "react-vite", templateVersionPinned: "1.0.1" }),
        ],
      }),
    );

    renderWithProviders(<GoldensCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.list)).toBeInTheDocument();
    });
    expect(screen.getByText("alpha")).toBeInTheDocument();
    expect(screen.getByText("bravo")).toBeInTheDocument();
  });

  it("submits register form to the API", async () => {
    const { goldenClient } = await import("../../api/golden");
    renderWithProviders(<GoldensCard />);

    await userEvent.type(screen.getByTestId(selectors.goldens.registerSlug), "alpha");
    await userEvent.type(screen.getByTestId(selectors.goldens.registerTemplate), "react-vite");
    await userEvent.type(screen.getByTestId(selectors.goldens.registerVersion), "1.0.1");
    await userEvent.type(screen.getByTestId(selectors.goldens.registerPath), "scenarios/alpha");

    await userEvent.click(screen.getByTestId(selectors.goldens.registerSubmit));

    await waitFor(() => {
      expect(vi.mocked(goldenClient.registerGolden)).toHaveBeenCalledWith({
        slug: "alpha",
        template: "react-vite",
        version: "1.0.1",
        path: "scenarios/alpha",
      });
    });
  });

  it("opens detail panel on row click and triggers regenerate", async () => {
    const { goldenClient } = await import("../../api/golden");
    vi.mocked(goldenClient.listGoldens).mockResolvedValueOnce(
      create(ListGoldensResponseSchema, { goldens: [makeGolden({ slug: "alpha" })] }),
    );
    renderWithProviders(<GoldensCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.list)).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId(selectors.goldens.row));
    expect(screen.getByTestId(selectors.goldens.detail)).toBeInTheDocument();

    await userEvent.click(screen.getByTestId(selectors.goldens.detailRegenerate));
    await waitFor(() => {
      expect(vi.mocked(goldenClient.regenerateGolden)).toHaveBeenCalledWith({ slug: "alpha" });
    });
  });

  it("triggers delete from detail panel", async () => {
    const { goldenClient } = await import("../../api/golden");
    vi.mocked(goldenClient.listGoldens).mockResolvedValueOnce(
      create(ListGoldensResponseSchema, { goldens: [makeGolden({ slug: "alpha" })] }),
    );
    renderWithProviders(<GoldensCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.goldens.list)).toBeInTheDocument();
    });

    await userEvent.click(screen.getByTestId(selectors.goldens.row));
    await userEvent.click(screen.getByTestId(selectors.goldens.detailDelete));
    await waitFor(() => {
      expect(vi.mocked(goldenClient.deleteGolden)).toHaveBeenCalledWith({ slug: "alpha" });
    });
  });
});
