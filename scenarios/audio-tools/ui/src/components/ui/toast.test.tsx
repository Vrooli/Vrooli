import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, act, fireEvent } from "@testing-library/react";

import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { Toaster, pushToast, dismissToast } from "./toast";
import { strings } from "../../consts/strings";

function renderToaster() {
  return render(<Toaster />);
}

// Test-specific data constants (not UI copy — avoids the no-restricted-syntax rule)
const TOAST_TITLE = "Hello toast";
const TOAST_BODY = "Body text here";
const TOAST_LINK_LABEL = "Click me";
const TOAST_LINK_URL = "https://example.com";
const TOAST_DISMISS_TITLE = "Dismissible";
const TOAST_AUTO_DISMISS_TITLE = "Auto-dismiss";
const TOAST_TARGETED_TITLE = "Targeted";
const TOAST_CUSTOM_ID_TITLE = "Custom ID";
const TOAST_MULTI_FIRST = "First";
const TOAST_MULTI_SECOND = "Second";

beforeEach(() => {
  vi.useFakeTimers();
});

afterEach(() => {
  // auto-dismiss all pending toasts before cleanup
  act(() => {
    vi.advanceTimersByTime(8001);
  });
  cleanup();
  vi.useRealTimers();
});

describe("Toaster / pushToast / dismissToast", () => {
  it("renders a toast after pushToast is called", () => {
    renderToaster();
    act(() => {
      pushToast({ title: TOAST_TITLE });
    });
    expect(screen.getByText(TOAST_TITLE)).toBeInTheDocument();
  });

  it("renders the toast body when provided", () => {
    renderToaster();
    act(() => {
      pushToast({ title: TOAST_TITLE, body: TOAST_BODY });
    });
    expect(screen.getByText(TOAST_BODY)).toBeInTheDocument();
  });

  it("renders a link when href is provided", () => {
    renderToaster();
    act(() => {
      pushToast({ title: TOAST_TITLE, href: TOAST_LINK_URL, hrefLabel: TOAST_LINK_LABEL });
    });
    const link = screen.getByRole("link", { name: new RegExp(TOAST_LINK_LABEL) });
    expect(link).toHaveAttribute("href", TOAST_LINK_URL);
  });

  it("uses default hrefLabel from strings when hrefLabel is omitted", () => {
    renderToaster();
    act(() => {
      pushToast({ title: TOAST_TITLE, href: TOAST_LINK_URL });
    });
    // In cimode t('toast.viewDefault') returns the key itself
    expect(screen.getByRole("link")).toHaveTextContent(strings.toast.viewDefault);
  });

  it("does not render body element when body is omitted", () => {
    renderToaster();
    act(() => {
      pushToast({ title: TOAST_TITLE });
    });
    expect(screen.queryByText(TOAST_BODY)).not.toBeInTheDocument();
  });

  it("dismisses a toast when the dismiss button is clicked", () => {
    renderToaster();
    act(() => {
      pushToast({ id: "dismiss-test", title: TOAST_DISMISS_TITLE });
    });
    expect(screen.getByText(TOAST_DISMISS_TITLE)).toBeInTheDocument();
    const btn = screen.getByRole("button", { name: strings.toast.dismiss });
    act(() => { fireEvent.click(btn); });
    expect(screen.queryByText(TOAST_DISMISS_TITLE)).not.toBeInTheDocument();
  });

  it("auto-dismisses after 8 seconds", () => {
    renderToaster();
    act(() => {
      pushToast({ id: "auto-dismiss-test", title: TOAST_AUTO_DISMISS_TITLE });
    });
    expect(screen.getByText(TOAST_AUTO_DISMISS_TITLE)).toBeInTheDocument();
    act(() => {
      vi.advanceTimersByTime(8000);
    });
    expect(screen.queryByText(TOAST_AUTO_DISMISS_TITLE)).not.toBeInTheDocument();
  });

  it("dismissToast removes a specific toast by id", () => {
    renderToaster();
    act(() => {
      pushToast({ id: "targeted-id", title: TOAST_TARGETED_TITLE });
    });
    expect(screen.getByText(TOAST_TARGETED_TITLE)).toBeInTheDocument();
    act(() => {
      dismissToast("targeted-id");
    });
    expect(screen.queryByText(TOAST_TARGETED_TITLE)).not.toBeInTheDocument();
  });

  it("uses provided id when given", () => {
    renderToaster();
    act(() => {
      pushToast({ id: "my-custom-id", title: TOAST_CUSTOM_ID_TITLE });
    });
    expect(screen.getByText(TOAST_CUSTOM_ID_TITLE)).toBeInTheDocument();
    act(() => {
      dismissToast("my-custom-id");
    });
    expect(screen.queryByText(TOAST_CUSTOM_ID_TITLE)).not.toBeInTheDocument();
  });

  it("renders multiple toasts simultaneously", () => {
    renderToaster();
    act(() => {
      pushToast({ id: "multi-1", title: TOAST_MULTI_FIRST });
      pushToast({ id: "multi-2", title: TOAST_MULTI_SECOND });
    });
    expect(screen.getByText(TOAST_MULTI_FIRST)).toBeInTheDocument();
    expect(screen.getByText(TOAST_MULTI_SECOND)).toBeInTheDocument();
  });

  it("live region has role=region and aria-live=polite", () => {
    renderToaster();
    const region = screen.getByRole("region");
    expect(region).toHaveAttribute("aria-live", "polite");
  });
});
