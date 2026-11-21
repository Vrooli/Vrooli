/**
 * Routing smoke — for each canonical path (`/`, `/notes`, `/settings`) the
 * matching page selector is in the document. Page-internal behaviour is
 * exercised in per-page tests; this file's job is to assert the router config.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { TestAppRouter } from "./routes";

describe("AppRouter", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("renders the dashboard at /", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.dashboard)).toBeInTheDocument();
  });

  it("renders the settings page at /settings", () => {
    renderWithProviders(<TestAppRouter initialEntries={["/settings"]} />, { withoutRouter: true });
    expect(screen.getByTestId(selectors.pages.settings)).toBeInTheDocument();
  });

  it("renders target availability from the Android target inventory", async () => {
    const fetchSpy = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      targets: [
        { id: "emulator", label: "Android emulator", available: true, device_kind: "emulator", transport: { kind: "adb" } },
        { id: "phone", label: "Physical phone", available: false, reason: "offline", next_action: "Connect the phone" },
        { id: "sparse", available: false },
      ],
    }), { status: 200 }));
    vi.stubGlobal("fetch", fetchSpy);

    renderWithProviders(<TestAppRouter initialEntries={["/targets"]} />, { withoutRouter: true });

    expect(await screen.findByText("Android emulator")).toBeInTheDocument();
    expect(screen.getByText("Physical phone")).toBeInTheDocument();
    expect(screen.getByText("sparse")).toBeInTheDocument();
    expect(screen.getByText("offline")).toBeInTheDocument();
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("renders a validation run and its evidence links", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify([
      {
        run_id: "run-1",
        state: "completed",
        gate: { passed: true },
        cells: [{ cell: { journey_id: "launch", target_id: "emulator", disposition: "passed", evidence: [{ uri: "evidence://launch" }] } }],
      },
      { run_id: "run-2", state: "partial", gate: { passed: false, reason: "needs evidence" }, cells: [{ state: "pending" }] },
    ]), { status: 200 })));

    renderWithProviders(<TestAppRouter initialEntries={["/runs"]} />, { withoutRouter: true });

    expect(await screen.findByText("run-1")).toBeInTheDocument();
    expect(screen.getByText(/evidence:\/\/launch/)).toBeInTheDocument();
    expect(screen.getByText(/needs evidence/)).toBeInTheDocument();
    expect(screen.getByText(/pages\.runs\.none/)).toBeInTheDocument();
  });

  it("renders readiness blockers and obligations", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({
      rungs: [
        { id: "build", title: "Build", state: "blocked", owner: "release", next_action: "Install the signing capability", obligation: "No local signing key" },
        { id: "unknown", state: "ready" },
      ],
    }), { status: 200 })));

    renderWithProviders(<TestAppRouter initialEntries={["/readiness"]} />, { withoutRouter: true });

    expect(await screen.findByText("Build")).toBeInTheDocument();
    expect(screen.getByText(/Install the signing capability/)).toBeInTheDocument();
    expect(screen.getByText(/No local signing key/)).toBeInTheDocument();
  });

  it("shows an honest alert when a target inventory request fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("target service unavailable")));

    renderWithProviders(<TestAppRouter initialEntries={["/targets"]} />, { withoutRouter: true });

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("target service unavailable"));
  });
});
