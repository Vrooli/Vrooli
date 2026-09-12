import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { ErrorBoundary } from "./ErrorBoundary";

// Throw is the canonical fixture: a component that synchronously throws
// during render when `when` is true. Sharing one fixture across cases
// keeps the surface narrow — the boundary's contract is "if a child
// throws during render, swap to the fallback" and that's what each
// test exercises.
function Throw({ when, message = "boom" }: { when: boolean; message?: string }) {
  if (when) {
    throw new Error(message);
  }
  return <div data-testid="ok">ok</div>;
}

// Suppress React's intentional console.error for boundary-caught
// throws across every test in this file. Without the silence, the
// runner output drowns the actual failure messages whenever a test
// regresses. Restored after each case so other suites still see the
// noise if they trigger it unexpectedly.
let consoleError: ReturnType<typeof vi.spyOn>;

beforeEach(() => {
  consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
});

afterEach(() => {
  consoleError.mockRestore();
});

describe("ErrorBoundary", () => {
  it("renders children when no error is thrown", () => {
    renderWithProviders(
      <ErrorBoundary>
        <Throw when={false} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId("ok")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.errorBoundary.root)).not.toBeInTheDocument();
  });

  it("renders the default fallback when a child throws", () => {
    renderWithProviders(
      <ErrorBoundary>
        <Throw when={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId(selectors.errorBoundary.root)).toBeInTheDocument();
    // cimode (the test-setup default) returns the key path verbatim;
    // asserting on the key proves the fallback consulted the registry
    // rather than hard-coding English copy.
    expect(screen.getByText(strings.errorBoundary.title)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.errorBoundary.retryButton)).toBeInTheDocument();
  });

  it("invokes onError with the thrown error", () => {
    const onError = vi.fn();
    renderWithProviders(
      <ErrorBoundary onError={onError}>
        <Throw when={true} message="boundary-test" />
      </ErrorBoundary>,
    );
    expect(onError).toHaveBeenCalledTimes(1);
    const call = onError.mock.calls[0];
    expect(call).toBeDefined();
    const [err, info] = call!;
    expect(err).toBeInstanceOf(Error);
    if (!(err instanceof Error)) {
      throw new Error("onError first argument must be an Error");
    }
    expect(err.message).toBe("boundary-test");
    expect(info).toHaveProperty("componentStack");
  });

  it("renders a custom fallback when provided", () => {
    renderWithProviders(
      <ErrorBoundary fallback={<div data-testid="custom-fallback">custom</div>}>
        <Throw when={true} />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId("custom-fallback")).toBeInTheDocument();
    // Default fallback must be absent when a custom one is supplied.
    expect(screen.queryByTestId(selectors.errorBoundary.root)).not.toBeInTheDocument();
  });

  it("recovers when retry is clicked and children stop throwing", async () => {
    const user = userEvent.setup();

    // The control object lets the test flip the throw state across
    // re-renders without changing component identity. React's
    // boundary semantics reset only when the boundary itself
    // re-renders, which happens when handleRetry calls setState — the
    // boundary then re-renders its children, and Recoverable picks up
    // the new control value.
    const control = { value: true };
    function Recoverable() {
      return <Throw when={control.value} />;
    }

    renderWithProviders(
      <ErrorBoundary>
        <Recoverable />
      </ErrorBoundary>,
    );
    expect(screen.getByTestId(selectors.errorBoundary.root)).toBeInTheDocument();

    control.value = false;
    await user.click(screen.getByTestId(selectors.errorBoundary.retryButton));

    expect(screen.getByTestId("ok")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.errorBoundary.root)).not.toBeInTheDocument();
  });
});
