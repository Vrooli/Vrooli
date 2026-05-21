import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { BusyPill } from "./BusyPill";
import type { PlaybooksClaim } from "../../lib/api";
import * as api from "../../lib/api";

function makeClaim(overrides: Partial<PlaybooksClaim> = {}): PlaybooksClaim {
  const now = new Date();
  return {
    scenario_name: "demo",
    run_id: "run-1",
    mode: "routed",
    started_by: "tester",
    acquired_at: now.toISOString(),
    heartbeat_at: now.toISOString(),
    expires_at: new Date(now.getTime() + 120_000).toISOString(),
    alive: true,
    ...overrides,
  };
}

describe("BusyPill", () => {
  it("renders an active claim with running label", () => {
    render(<BusyPill claim={makeClaim()} />);
    expect(screen.getByText(/Playbooks running/i)).toBeInTheDocument();
    expect(screen.getByText(/run run-1/i)).toBeInTheDocument();
  });

  it("renders a stale claim with stale label", () => {
    render(<BusyPill claim={makeClaim({ alive: false })} />);
    expect(screen.getByText(/Stale playbooks claim/i)).toBeInTheDocument();
  });

  it("releases the claim when confirmed", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(true);
    const released = makeClaim();
    const spy = vi.spyOn(api, "releasePlaybooksClaim").mockResolvedValue(released);
    const onReleased = vi.fn();
    render(<BusyPill claim={released} onReleased={onReleased} />);

    fireEvent.click(screen.getByTestId("playbooks-busy-pill-release"));
    await waitFor(() => expect(spy).toHaveBeenCalledWith("demo"));
    await waitFor(() => expect(onReleased).toHaveBeenCalled());
  });

  it("does not call API when force-release is cancelled", () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const spy = vi.spyOn(api, "releasePlaybooksClaim");
    render(<BusyPill claim={makeClaim()} />);
    fireEvent.click(screen.getByTestId("playbooks-busy-pill-release"));
    expect(spy).not.toHaveBeenCalled();
  });
});
