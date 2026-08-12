import { act, cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { LiveAnnouncer, useLiveAnnouncer } from "./LiveAnnouncer/versions/1.0.0/LiveAnnouncer";
import { renderWithProviders } from "../test-utils";

function AnnouncerProbe() {
  const { announce, clear } = useLiveAnnouncer();
  return (
    <>
      <button onClick={() => announce("Saved", { priority: "assertive", durationMs: 900 })}>
        announce
      </button>
      <button onClick={clear}>clear</button>
    </>
  );
}

describe("LiveAnnouncer", () => {
  afterEach(() => {
    cleanup();
    vi.useRealTimers();
  });

  it("queues announcements, exposes priority, and clears the queue", async () => {
    vi.useFakeTimers();
    renderWithProviders(
      <LiveAnnouncer>
        <AnnouncerProbe />
      </LiveAnnouncer>,
    );
    const live = screen.getByRole("status");

    act(() => screen.getByRole("button", { name: "announce" }).click());
    expect(live).toHaveTextContent("Saved");
    expect(live).toHaveAttribute("aria-live", "assertive");
    act(() => screen.getByRole("button", { name: "clear" }).click());
    expect(live).toHaveTextContent("");
  });

  it("uses the document fallback when no provider is mounted", () => {
    function FallbackProbe() {
      const { announce, clear } = useLiveAnnouncer();
      return (
        <>
          <button onClick={() => announce("Copied")}>copy</button>
          <button onClick={clear}>clear fallback</button>
        </>
      );
    }
    renderWithProviders(<FallbackProbe />);
    screen.getByRole("button", { name: "copy" }).click();
    const fallback = document.querySelector("[data-vrooli-live-announcer]");
    expect(fallback).toHaveTextContent("Copied");
    screen.getByRole("button", { name: "clear fallback" }).click();
    expect(fallback).toHaveTextContent("");
  });
});
