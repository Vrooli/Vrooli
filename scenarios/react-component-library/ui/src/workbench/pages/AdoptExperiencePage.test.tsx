import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import type { UseQueryResult } from "@tanstack/react-query";
import { Route, Routes, useLocation } from "react-router-dom";

import { createWorkspace, listWorkspaces } from "../../api/workspaces";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import * as data from "../data";
import { AdoptExperiencePage } from "./AdoptExperiencePage";

vi.mock("../data", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../data")>();
  return { ...actual, useDesignPages: vi.fn(), adoptExperiencePage: vi.fn() };
});

vi.mock("../../api/workspaces", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/workspaces")>();
  return { ...actual, createWorkspace: vi.fn(), listWorkspaces: vi.fn() };
});

function queryResult<T>(
  value: T | undefined,
  options: { isLoading?: boolean; isError?: boolean } = {},
) {
  return {
    data: value,
    error: options.isError ? new Error("unavailable") : null,
    isError: options.isError ?? false,
    isLoading: options.isLoading ?? false,
    isFetching: false,
    isSuccess: !options.isError && value !== undefined,
    refetch: vi.fn(),
  } as unknown as UseQueryResult<T>;
}

function Location() {
  const location = useLocation();
  return <span data-testid="location">{location.pathname + location.search}</span>;
}

function renderRoute() {
  return renderWithProviders(
    <Routes>
      <Route
        path="/design/:scenario/:page"
        element={
          <>
            <AdoptExperiencePage />
            <Location />
          </>
        }
      />
      <Route path="/design" element={<Location />} />
    </Routes>,
    { initialEntries: ["/design/harbor/welcome"] },
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("AdoptExperiencePage", () => {
  it("keeps the experience page as intent and creates one bound workspace composition", async () => {
    vi.mocked(data.useDesignPages).mockReturnValue(
      queryResult([
        {
          page: "welcome",
          title: "Welcome",
          route: "/onboarding/welcome",
          status: "region-declared",
          contentHash: "hash",
          regionCount: 3,
          registered: true,
          routes: [],
        },
      ]),
    );
    vi.mocked(createWorkspace).mockResolvedValue({ id: "ws_welcome" } as never);
    renderRoute();

    expect(await screen.findByText(strings.workbench.adopt.lede)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("adopt-create-workspace"));

    await waitFor(() => expect(createWorkspace).toHaveBeenCalledOnce());
    const call = vi.mocked(createWorkspace).mock.calls[0];
    expect(call).toBeDefined();
    if (!call) throw new Error("workspace creation request was not recorded");
    const request = call[0];
    expect(request.bindingScenario).toBe("harbor");
    expect(request.bindingPage).toBe("welcome");
    const firstPage = request.pages?.[0];
    expect(firstPage).toBeDefined();
    if (!firstPage) throw new Error("adoption must create a workspace page");
    expect(firstPage.name).toBe("Welcome");
    expect(firstPage.route).toBe("/onboarding/welcome");
    expect(await screen.findByTestId("location")).toHaveTextContent("/design?workspace=ws_welcome");
  });

  it("recovers a simultaneous adoption by opening the winning binding", async () => {
    vi.mocked(data.useDesignPages).mockReturnValue(
      queryResult([
        {
          page: "welcome",
          title: "Welcome",
          route: "/welcome",
          status: "region-declared",
          contentHash: "hash",
          regionCount: 1,
          registered: true,
          routes: [],
        },
      ]),
    );
    vi.mocked(createWorkspace).mockRejectedValue(new Error("experience page is already bound"));
    vi.mocked(listWorkspaces).mockResolvedValue([
      { bindingScenario: "harbor", bindingPage: "welcome", id: "ws_winner" } as never,
    ]);
    renderRoute();

    fireEvent.click(await screen.findByTestId("adopt-create-workspace"));
    await waitFor(() =>
      expect(screen.getByTestId("location")).toHaveTextContent("/design?workspace=ws_winner"),
    );
  });

  it("imports a composition, exposes its durable recovery evidence, and opens the workspace", async () => {
    vi.mocked(data.useDesignPages).mockReturnValue(
      queryResult([
        {
          page: "conversations",
          title: "Conversations",
          route: "/conversations",
          status: "composition-present",
          contentHash: "hash",
          regionCount: 10,
          registered: true,
          routes: ["/conversations"],
        },
      ]),
    );
    vi.mocked(createWorkspace).mockClear();
    vi.mocked(data.adoptExperiencePage).mockResolvedValue({
      workspace: { id: "ws_conversations", name: "Conversations" },
      receiptId: "ir_123",
      sourceHash: "source-sha256",
      targetHash: "target-sha256",
      alreadyAdopted: false,
      receiptJson: JSON.stringify({ renderParity: false, renderGap: "Legacy bindings still run through the retained source adapter." }),
    });
    renderWithProviders(
      <Routes>
        <Route
          path="/design/:scenario/:page"
          element={
            <>
              <AdoptExperiencePage />
              <Location />
            </>
          }
        />
        <Route path="/design" element={<Location />} />
      </Routes>,
      { initialEntries: ["/design/switchboard/conversations"] },
    );

    expect(screen.getByText(strings.workbench.adopt.compositionPresentBody)).toBeInTheDocument();
    expect(screen.queryByTestId("adopt-create-workspace")).not.toBeInTheDocument();
    expect(createWorkspace).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("adopt-import-experience"));
    expect(await screen.findByTestId("adopt-receipt-id")).toHaveTextContent("ir_123");
    expect(screen.getByTestId("adopt-source-hash")).toHaveTextContent("source-sha256");
    expect(screen.getByTestId("adopt-target-hash")).toHaveTextContent("target-sha256");
    expect(screen.getByText(strings.workbench.adopt.parityPartial)).toBeInTheDocument();
    expect(screen.getByTestId("adopt-render-gap")).toHaveTextContent("Legacy bindings still run through the retained source adapter.");
    expect(screen.getByText(strings.workbench.adopt.sourceIntact, { exact: false })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: strings.workbench.adopt.downloadReceipt })).toHaveAttribute("download", "switchboard-conversations-recovery-receipt.json");
    fireEvent.click(screen.getByRole("button", { name: strings.workbench.adopt.openWorkspace }));
    await waitFor(() => expect(screen.getByTestId("location")).toHaveTextContent("/design?workspace=ws_conversations"));
    expect(data.adoptExperiencePage).toHaveBeenCalledWith("switchboard", "conversations");
  });
});
