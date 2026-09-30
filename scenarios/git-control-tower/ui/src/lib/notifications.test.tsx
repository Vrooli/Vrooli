import { act, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Toast } from "@vrooli/react-component-library/Toast/1";
import { ToastManagerProvider } from "@vrooli/react-component-library/ToastManager/1";
import {
  STICKY,
  SYNC_TOAST_ID,
  TRANSIENT_MS,
  useMutationErrorToasts,
  useNotifications
} from "./notifications";

function Harness({ children }: { children: React.ReactNode }) {
  return (
    <ToastManagerProvider maxVisible={4}>
      {children}
      <Toast />
    </ToastManagerProvider>
  );
}

describe("notifications", () => {
  it("keeps the sync lifecycle in one notice instead of stacking three", () => {
    function Driver() {
      const { notifySync } = useNotifications();
      return (
        <>
          <button
            type="button"
            onClick={() =>
              notifySync({ tone: "info", title: "Pushing to remote", message: "Pushing 4s", durationMs: STICKY })
            }
          >
            progress
          </button>
          <button
            type="button"
            onClick={() =>
              notifySync({ tone: "success", title: "Pushed to origin/agi", durationMs: TRANSIENT_MS })
            }
          >
            done
          </button>
        </>
      );
    }

    render(
      <Harness>
        <Driver />
      </Harness>
    );

    act(() => screen.getByText("progress").click());
    expect(screen.getByText("Pushing to remote")).toBeInTheDocument();

    act(() => screen.getByText("done").click());
    expect(screen.queryByText("Pushing to remote")).not.toBeInTheDocument();
    expect(screen.getByText("Pushed to origin/agi")).toBeInTheDocument();
  });

  it("re-pushes rather than updating, so a progress timer cannot expire the result", () => {
    // ToastManager only clears an existing timer on push. An update leaves the original
    // schedule running, which would dismiss a sticky failure on the progress toast's clock.
    vi.useFakeTimers();
    try {
      function Driver() {
        const { notifySync } = useNotifications();
        return (
          <>
            <button
              type="button"
              onClick={() => notifySync({ tone: "info", title: "Pushing", durationMs: 2000 })}
            >
              progress
            </button>
            <button
              type="button"
              onClick={() => notifySync({ tone: "error", title: "Push failed", durationMs: STICKY })}
            >
              fail
            </button>
          </>
        );
      }

      render(
        <Harness>
          <Driver />
        </Harness>
      );

      act(() => screen.getByText("progress").click());
      act(() => screen.getByText("fail").click());
      act(() => vi.advanceTimersByTime(10_000));

      // The manager re-announces a patched notice, so these words also land in the live
      // region. Assert against the toast's own title rather than any matching text.
      expect(document.querySelector("[data-rcl-toast-title]")).toHaveTextContent("Push failed");
    } finally {
      vi.useRealTimers();
    }
  });

  it("exposes a stable id so the sync notice is addressable", () => {
    expect(SYNC_TOAST_ID).toBe("gct.sync");
  });

  it("reports a mutation failure and clears the mutation so a retry is not blocked", () => {
    const reset = vi.fn();

    function Driver({ error }: { error: Error | null }) {
      useMutationErrorToasts([{ label: "Stage files", error, reset }]);
      return null;
    }

    const { rerender } = render(
      <Harness>
        <Driver error={null} />
      </Harness>
    );
    expect(reset).not.toHaveBeenCalled();

    rerender(
      <Harness>
        <Driver error={new Error("pathspec did not match any file")} />
      </Harness>
    );

    expect(screen.getByText("Stage files failed")).toBeInTheDocument();
    expect(screen.getByText("pathspec did not match any file")).toBeInTheDocument();
    expect(reset).toHaveBeenCalled();
  });

  it("collapses repeated failures of one mutation into a single notice", () => {
    function Driver({ error }: { error: Error | null }) {
      useMutationErrorToasts([{ label: "Stage files", error, reset: vi.fn() }]);
      return null;
    }

    const { rerender } = render(
      <Harness>
        <Driver error={new Error("first")} />
      </Harness>
    );
    rerender(
      <Harness>
        <Driver error={new Error("second")} />
      </Harness>
    );

    // Exact matcher: the live-region announcement carries the same words in a longer
    // string, and matching loosely would count it as a second notice.
    expect(screen.getAllByText("Stage files failed")).toHaveLength(1);
    expect(screen.getByText("second")).toBeInTheDocument();
  });
});
