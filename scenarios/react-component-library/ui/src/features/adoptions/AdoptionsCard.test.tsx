import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Route, Routes, useLocation } from "react-router-dom";

import { renderWithProviders } from "../../test-utils";
import {
  LibraryVersionStatus,
  LocalStatus,
  makeAdoption,
  makeListAdoptionsResponse,
  makeRefreshAdoptionsResponse,
  makeSuggestAdoptionsResponse,
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
          makeAdoption({
            id: "a",
            scenario: "swarm-manager",
            libraryVersionStatus: LibraryVersionStatus.CURRENT,
            localStatus: LocalStatus.CLEAN,
          }),
          makeAdoption({
            id: "b",
            scenario: "flow-verifier",
            libraryVersionStatus: LibraryVersionStatus.BEHIND,
            localStatus: LocalStatus.CLEAN,
            statusDetail: "library at 1.1.0",
          }),
          makeAdoption({
            id: "c",
            scenario: "system-monitor",
            libraryVersionStatus: LibraryVersionStatus.CURRENT,
            localStatus: LocalStatus.MODIFIED,
          }),
          makeAdoption({
            id: "d",
            scenario: "drift-smoke",
            libraryVersionStatus: LibraryVersionStatus.UNKNOWN,
            localStatus: LocalStatus.UNKNOWN,
          }),
        ],
      }),
    );

    renderWithProviders(<AdoptionsCard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.adoptions.list)).toBeInTheDocument();
    });

    const statuses = screen
      .getAllByTestId(selectors.adoptions.itemStatus)
      .map((n) => n.textContent);
    expect(statuses).toHaveLength(4);
    expect(statuses).toEqual(
      expect.arrayContaining([
        "Current / Clean",
        "Behind / Clean",
        "Current / Modified",
        "Unknown / Unknown",
      ]),
    );
    expect(screen.getByTestId(selectors.adoptions.summary).textContent).toContain("current: 2");
    expect(screen.getByTestId(selectors.adoptions.itemStatusDetail)).toHaveTextContent(
      "library at 1.1.0",
    );
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
      makeRefreshAdoptionsResponse({ libraryCurrent: 1, libraryBehind: 2 }),
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

  it("labels discovery suggestions as heuristic candidates", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.suggestAdoptions).mockResolvedValueOnce(
      makeSuggestAdoptionsResponse({
        suggestions: [
          {
            componentId: "hook-focus-trap",
            displayName: "useFocusTrap",
            scenario: "web-console",
            reasons: ["matching import inventory"],
            classification: 1,
          },
        ],
      }),
    );

    renderWithProviders(<AdoptionsCard />);

    expect(
      await screen.findByText("Heuristic candidate — review before adopting"),
    ).toBeInTheDocument();
    expect(screen.getByText("matching import inventory")).toBeInTheDocument();
  });

  it("routes suggestion adoption through the prefilled guided launcher flow", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.suggestAdoptions).mockResolvedValueOnce(
      makeSuggestAdoptionsResponse({
        suggestions: [
          {
            componentId: "asset-1",
            displayName: "Button",
            scenario: "demo",
            reasons: [],
            classification: 1,
          },
        ],
      }),
    );
    const LocationProbe = () => <output data-testid="location">{useLocation().search}</output>;
    const user = userEvent.setup();

    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={
            <>
              <AdoptionsCard suggestionsOnly />
              <LocationProbe />
            </>
          }
        />
      </Routes>,
    );
    await user.click(await screen.findByRole("button", { name: "Adopt" }));
    expect(screen.getByTestId("location")).toHaveTextContent(
      "?action=adopt&assetId=asset-1&targetScenario=demo",
    );
  });

  it("routes standard adoption entry points through the guided launcher and keeps local re-link explicit", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValue(makeListAdoptionsResponse());
    const LocationProbe = () => <output data-testid="location">{useLocation().search}</output>;
    const user = userEvent.setup();
    renderWithProviders(
      <Routes>
        <Route
          path="/"
          element={
            <>
              <AdoptionsCard />
              <LocationProbe />
            </>
          }
        />
      </Routes>,
    );

    await user.click(await screen.findByTestId(selectors.adoptions.createButton));
    expect(screen.getByTestId("location")).toHaveTextContent("?action=adopt");
    await user.click(screen.getByText("Advanced local re-link"));
    await user.click(screen.getByRole("button", { name: "Open local re-link" }));
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("filters the adoption table by current, behind, and modified status", async () => {
    const { adoptionsClient } = await import("../../api/adoptions");
    vi.mocked(adoptionsClient.listAdoptions).mockResolvedValue(
      makeListAdoptionsResponse({
        adoptions: [
          makeAdoption({
            id: "current",
            scenario: "current",
            libraryVersionStatus: LibraryVersionStatus.CURRENT,
            localStatus: LocalStatus.CLEAN,
          }),
          makeAdoption({
            id: "behind",
            scenario: "behind",
            libraryVersionStatus: LibraryVersionStatus.BEHIND,
            localStatus: LocalStatus.CLEAN,
          }),
          makeAdoption({
            id: "modified",
            scenario: "modified",
            libraryVersionStatus: LibraryVersionStatus.CURRENT,
            localStatus: LocalStatus.MODIFIED,
          }),
        ],
      }),
    );
    const user = userEvent.setup();
    renderWithProviders(<AdoptionsCard />);
    await screen.findByTestId(selectors.adoptions.list);

    await user.click(screen.getByRole("button", { name: "Behind" }));
    expect(screen.getAllByTestId(selectors.adoptions.itemScenario)).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Modified" }));
    expect(screen.getAllByTestId(selectors.adoptions.itemScenario)).toHaveLength(1);
    await user.click(screen.getByRole("button", { name: "Current" }));
    expect(screen.getAllByTestId(selectors.adoptions.itemScenario)).toHaveLength(2);
  });
});
