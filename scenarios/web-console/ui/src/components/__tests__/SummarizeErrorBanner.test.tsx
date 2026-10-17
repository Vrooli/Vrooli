import { renderWithProviders as render } from "../../test-utils";
import { describe, it, expect, vi } from "vitest";
import { screen, fireEvent } from "@testing-library/react";
import { BannerHarness } from "../banners/__tests__/harness";
import { summarizeErrorBanner } from "../banners/descriptors";
import type { SummarizeErrorState } from "../../types/summarize";
import { strings } from "../../consts/strings";

const RETRY = "summarize-error-banner-retry";
const DISMISS = "summarize-error-banner-dismiss";

function makeState(overrides?: Partial<SummarizeErrorState>): SummarizeErrorState {
  return {
    sessionId: "sess-1",
    eventId: "ev-1",
    message: "Ollama timed out",
    source: "auto",
    status: "failed",
    ...overrides,
  };
}

function renderBanner(
  state: SummarizeErrorState,
  handlers: { onRetry?: () => void; onDismiss?: () => void } = {},
) {
  return render(
    <BannerHarness
      build={(t) =>
        summarizeErrorBanner(t, state, {
          onRetry: handlers.onRetry ?? vi.fn(),
          onDismiss: handlers.onDismiss ?? vi.fn(),
        })
      }
    />,
  );
}

describe("summarize error banner", () => {
  it("renders the message and source label for auto failures", () => {
    renderBanner(makeState({ source: "auto" }));
    expect(screen.getByTestId("summarize-error-banner")).toBeInTheDocument();
    expect(screen.getByText(strings.summarizeError.autoFailed)).toBeInTheDocument();
    expect(screen.getByText("Ollama timed out")).toBeInTheDocument();
  });

  it("renders a different source label for on-demand failures", () => {
    renderBanner(makeState({ source: "on-demand" }));
    expect(screen.getByText(strings.summarizeError.failed)).toBeInTheDocument();
  });

  it("clicking retry fires onRetry", () => {
    const onRetry = vi.fn();
    renderBanner(makeState(), { onRetry });
    fireEvent.click(screen.getByTestId(RETRY));
    expect(onRetry).toHaveBeenCalledTimes(1);
  });

  it("clicking dismiss fires onDismiss", () => {
    const onDismiss = vi.fn();
    renderBanner(makeState(), { onDismiss });
    fireEvent.click(screen.getByTestId(DISMISS));
    expect(onDismiss).toHaveBeenCalledTimes(1);
  });

  it("disables retry while retrying and shows a spinner", () => {
    renderBanner(makeState({ status: "retrying" }));
    expect(screen.getByTestId(RETRY)).toBeDisabled();
    expect(screen.getByText(strings.summarizeError.retrying)).toBeInTheDocument();
  });

  it("banner carries data attributes for source and status", () => {
    renderBanner(makeState({ source: "on-demand", status: "retrying" }));
    const banner = screen.getByTestId("summarize-error-banner");
    expect(banner.getAttribute("data-source")).toBe("on-demand");
    expect(banner.getAttribute("data-status")).toBe("retrying");
  });

  it("is a danger-tone notice on the shared base, and announces assertively", () => {
    renderBanner(makeState());
    const banner = screen.getByTestId("summarize-error-banner");
    expect(banner).toHaveAttribute("data-rcl-banner");
    expect(banner).toHaveAttribute("data-tone", "danger");
    expect(banner.getAttribute("role")).toBe("alert");
    expect(banner.getAttribute("aria-live")).toBe("assertive");
  });
});
