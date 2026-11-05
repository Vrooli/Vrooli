import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { useEffect } from "react";
import { describe, expect, it, vi } from "vitest";

import {
  ToastManagerProvider,
  useToastActions,
} from "@vrooli/react-component-library/ToastManager/1";
import { Toast } from "@vrooli/react-component-library/Toast/1.0.14";

function Harness({
  tone = "info",
  title = "Sending",
}: {
  tone?: "info" | "success";
  title?: string;
}) {
  const actions = useToastActions();
  useEffect(() => {
    actions.push({ id: "sync", tone, title, durationMs: 0 });
  }, [actions, title, tone]);
  return null;
}

function renderToast(props?: { tone?: "info" | "success"; title?: string }) {
  return render(
    <ToastManagerProvider>
      <Harness {...props} />
      <Toast />
    </ToastManagerProvider>,
  );
}

describe("Toast current release contract", () => {
  it("updates one mounted notice and morphs its tone icon", async () => {
    const view = renderToast({ title: "Sending" });
    await waitFor(() =>
      expect(screen.getByTestId("feedback.toast")).toBeInTheDocument(),
    );
    const firstToast = screen.getByTestId("feedback.toast");
    const presence = screen.getByTestId("motion.presence");
    await waitFor(() =>
      expect(presence).toHaveAttribute("data-presence-phase", "entered"),
    );

    await act(async () => {
      view.rerender(
        <ToastManagerProvider>
          <Harness tone="success" title="Sent" />
          <Toast />
        </ToastManagerProvider>,
      );
    });

    expect(screen.getByTestId("feedback.toast")).toBe(firstToast);
    expect(firstToast).toHaveAttribute("data-tone", "success");
    expect(firstToast).toHaveTextContent("Sent");
    expect(
      firstToast.querySelector("[data-rcl-morphing-icon]"),
    ).toBeInTheDocument();
    expect(presence).toHaveAttribute("data-presence-phase", "entered");
  });

  it("dismisses when the notice is swiped toward the inline start", async () => {
    vi.useFakeTimers();
    try {
      renderToast({ title: "Sending" });
      const toast = screen.getByTestId("feedback.toast");

      act(() => {
        fireEvent.pointerDown(toast, {
          button: 0,
          clientX: 300,
          clientY: 200,
          pointerId: 7,
          pointerType: "touch",
        });
        fireEvent.pointerMove(window, {
          clientX: 180,
          clientY: 200,
          pointerId: 7,
          timeStamp: 40,
        });
        fireEvent.pointerUp(window, {
          clientX: 140,
          clientY: 200,
          pointerId: 7,
          timeStamp: 80,
        });
      });

      expect(screen.getByTestId("motion.presence")).toHaveAttribute(
        "data-presence-phase",
        "exiting",
      );
      act(() => vi.advanceTimersByTime(250));
      expect(screen.queryByTestId("feedback.toast")).not.toBeInTheDocument();
    } finally {
      vi.useRealTimers();
    }
  });
});
