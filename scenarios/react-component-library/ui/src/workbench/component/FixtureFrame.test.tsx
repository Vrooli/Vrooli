import { act, fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { setLocale } from "../../i18n";
import { FIXTURE_TIMEOUT_MS, FixtureFrame } from "./FixtureFrame";

function renderFrame(onStatus = vi.fn(), onRetry?: () => void) {
  const view = renderWithProviders(
    <FixtureFrame
      assetId="asset-1"
      storyId="default"
      version="1.0.0"
      title="Specimen"
      onStatus={onStatus}
      onRetry={onRetry}
    />,
  );
  return { onStatus, view };
}

function jsonResponse(body: string, status: number): Response {
  return new Response(body, { status });
}

async function flushProbe() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

beforeEach(async () => {
  // The suite runs in cimode (keys, not copy). This file asserts the failure
  // sentence a person would actually read, so it opts into English.
  await setLocale("en");
});

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("FixtureFrame failure path", () => {
  it("reports an HTTP failure instead of spinning", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("not found", 404)));
    const onStatus = renderFrame().onStatus;

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("HTTP 404");
    expect(onStatus).toHaveBeenCalledWith("error", expect.stringContaining("404"));
  });

  it("reports an unreachable harness", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("connection refused")));
    renderFrame();

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("connection refused");
  });

  it("fails when the harness loads but never answers", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const onStatus = renderFrame().onStatus;

    await flushProbe();

    act(() => {
      vi.advanceTimersByTime(FIXTURE_TIMEOUT_MS + 50);
    });

    expect(screen.getByRole("alert")).toHaveTextContent(
      `within ${FIXTURE_TIMEOUT_MS / 1000} seconds`,
    );
    expect(onStatus).toHaveBeenCalledWith("error", expect.stringContaining("did not answer"));
    expect(onStatus).not.toHaveBeenCalledWith("ready", undefined);
  });

  it("reports an iframe error event and offers a retry", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const onRetry = vi.fn();
    const { onStatus } = renderFrame(vi.fn(), onRetry);

    const frame = await screen.findByTestId("fixture-iframe");
    fireEvent.error(frame);

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("could not be mounted");
    expect(onStatus).toHaveBeenCalledWith("error", expect.stringContaining("could not be mounted"));

    fireEvent.click(screen.getByTestId("fixture-retry"));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("becomes ready when the harness posts preview-ready", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const { onStatus } = renderFrame();

    const frame = await screen.findByTestId<HTMLIFrameElement>("fixture-iframe");
    const source = frame.contentWindow;
    expect(source).not.toBeNull();

    act(() => {
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { type: "preview-ready", id: "asset-1", story: "default", version: "1.0.0" },
          source,
        }),
      );
    });

    expect(onStatus).toHaveBeenCalledWith("ready", undefined);
    expect(frame.style.visibility).toBe("visible");
    expect(screen.queryByRole("status")).toBeNull();
  });

  it("ignores a ready message that came from another frame", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const { onStatus } = renderFrame();

    await screen.findByTestId("fixture-iframe");
    act(() => {
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { type: "preview-ready", id: "asset-1" },
          source: window,
        }),
      );
    });

    expect(onStatus).not.toHaveBeenCalledWith("ready", undefined);
  });

  it("clears the readiness timeout on unmount", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const onStatus = vi.fn();
    const { view } = renderFrame(onStatus);

    await flushProbe();
    expect(screen.getByTestId("fixture-iframe")).toBeInTheDocument();

    view.unmount();
    act(() => {
      vi.advanceTimersByTime(FIXTURE_TIMEOUT_MS * 2);
    });

    expect(onStatus).not.toHaveBeenCalledWith("error", expect.anything());
    expect(onStatus).not.toHaveBeenCalledWith("ready", expect.anything());
  });

  it("clears the readiness timeout once the harness is ready", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse("<html></html>", 200)));
    const onStatus = vi.fn();
    renderFrame(onStatus);

    await flushProbe();
    const frame = screen.getByTestId<HTMLIFrameElement>("fixture-iframe");
    act(() => {
      window.dispatchEvent(
        new MessageEvent("message", {
          data: { type: "preview-ready", id: "asset-1" },
          source: frame.contentWindow,
        }),
      );
    });
    act(() => {
      vi.advanceTimersByTime(FIXTURE_TIMEOUT_MS * 2);
    });

    expect(onStatus).not.toHaveBeenCalledWith("error", expect.anything());
  });
});
