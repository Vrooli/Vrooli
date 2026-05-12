import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import {
  AdoptionStatus,
  makeAdoption,
  makeListAdoptionsResponse,
  makeRefreshAdoptionsResponse,
} from "./mocks/factories";
import { makeAdoptionsMocks } from "./mocks/adoptions";

vi.mock("../../api/adoptions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/adoptions")>();
  return { ...actual, ...makeAdoptionsMocks() };
});

import { AdoptionsCard } from "./AdoptionsCard";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("AdoptionsCard", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders empty state when there are no adoptions", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValueOnce(makeListAdoptionsResponse());

    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.empty)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.adoptions.list)).not.toBeInTheDocument();
  });

  it("renders one row per status with a non-color status label", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValueOnce(
      makeListAdoptionsResponse({
        adoptions: [
          makeAdoption({ id: "a", scenario: "swarm-manager", status: AdoptionStatus.CURRENT }),
          makeAdoption({ id: "b", scenario: "flow-verifier", status: AdoptionStatus.BEHIND, statusDetail: "library at 1.1.0" }),
          makeAdoption({ id: "c", scenario: "system-monitor", status: AdoptionStatus.MODIFIED }),
          makeAdoption({ id: "d", scenario: "drift-smoke", status: AdoptionStatus.UNKNOWN }),
        ],
      }),
    );

    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.list)).toBeInTheDocument();
    });

    const statuses = screen.getAllByTestId(selectors.adoptions.itemStatus).map((n) => n.textContent);
    expect(statuses).toEqual(["Current", "Behind", "Modified", "Unknown"]);
    expect(screen.getByTestId(selectors.adoptions.summary).textContent).toContain("current: 1");
    expect(screen.getByText("library at 1.1.0")).toBeInTheDocument();
  });

  it("forwards the scenario filter to listAdoptions", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValue(makeListAdoptionsResponse());

    const user = userEvent.setup();
    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.scenarioFilter)).toBeInTheDocument();
    });
    await user.type(screen.getByTestId(selectors.adoptions.scenarioFilter), "swarm-manager");

    await waitFor(() => {
      const calls = vi.mocked(adoptionsClient.listAdoptions).mock.calls;
      const last = calls[calls.length - 1]?.[0] ?? {};
      expect(last).toMatchObject({ scenario: "swarm-manager" });
    });
  });

  it("invokes refreshAdoptions when the refresh button is clicked", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValue(makeListAdoptionsResponse());
    vi.mocked(adoptionsClient.refreshAdoptions).mockResolvedValueOnce(
      makeRefreshAdoptionsResponse({ current: 1, behind: 2 }),
    );

    const user = userEvent.setup();
    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.refreshButton)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.adoptions.refreshButton));

    await waitFor(() => {
      expect(adoptionsClient.refreshAdoptions).toHaveBeenCalledTimes(1);
    });
  });

  it("invokes deleteAdoption with the row id", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValue(
      makeListAdoptionsResponse({ adoptions: [makeAdoption({ id: "ad-99" })] }),
    );

    const user = userEvent.setup();
    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.itemDeleteButton)).toBeInTheDocument();
    });
    await user.click(screen.getByTestId(selectors.adoptions.itemDeleteButton));

    await waitFor(() => {
      expect(adoptionsClient.deleteAdoption).toHaveBeenCalledWith({ id: "ad-99" });
    });
  });
});
