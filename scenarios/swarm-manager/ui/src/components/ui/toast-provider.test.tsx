import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "./toast-provider";
import { useToast, resetToastProviderWarning, type ToastInput } from "../../hooks/useToast";

/** Renders a button per toast so a test can fire them in order. */
function Harness({ toasts }: { toasts: ToastInput[] }) {
  const { notify, dismissAll } = useToast();
  return (
    <div>
      {toasts.map((toast, index) => (
        <button key={index} type="button" onClick={() => notify(toast)}>
          send-{index}
        </button>
      ))}
      <button type="button" onClick={dismissAll}>dismiss-all</button>
    </div>
  );
}

function renderToasts(toasts: ToastInput[]) {
  return render(<ToastProvider><Harness toasts={toasts} /></ToastProvider>);
}

describe("ToastProvider", () => {
  it("shows a toast when notified", async () => {
    renderToasts([{ kind: "success", message: "Goal archived" }]);
    await userEvent.click(screen.getByText("send-0"));
    expect(await screen.findByText("Goal archived")).toBeInTheDocument();
  });

  it("announces errors assertively and everything else politely", async () => {
    renderToasts([
      { kind: "error", message: "It broke" },
      { kind: "success", message: "It worked" },
    ]);

    await userEvent.click(screen.getByText("send-0"));
    await userEvent.click(screen.getByText("send-1"));

    expect(await screen.findByRole("alert")).toHaveTextContent("It broke");
    expect(screen.getByRole("status")).toHaveTextContent("It worked");
  });

  it("renders the description under the headline", async () => {
    renderToasts([{ kind: "error", message: "Couldn't start review", description: "milestone has no criteria" }]);
    await userEvent.click(screen.getByText("send-0"));
    const toast = await screen.findByTestId("toast");
    expect(toast).toHaveTextContent("Couldn't start review");
    expect(toast).toHaveTextContent("milestone has no criteria");
  });

  it("dismisses on the close button", async () => {
    renderToasts([{ kind: "info", message: "Heads up" }]);
    await userEvent.click(screen.getByText("send-0"));
    await screen.findByText("Heads up");

    await userEvent.click(screen.getByTestId("toast-dismiss"));
    await waitFor(() => expect(screen.queryByText("Heads up")).toBeNull());
  });

  it("runs the action and dismisses the toast", async () => {
    const onClick = vi.fn();
    renderToasts([{ kind: "error", message: "Failed", action: { label: "Retry", onClick } }]);
    await userEvent.click(screen.getByText("send-0"));

    await userEvent.click(await screen.findByTestId("toast-action"));

    expect(onClick).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(screen.queryByText("Failed")).toBeNull());
  });

  it("replaces rather than stacks toasts sharing a key", async () => {
    renderToasts([
      { kind: "error", message: "Attempt failed", key: "save" },
      { kind: "error", message: "Attempt failed again", key: "save" },
    ]);

    await userEvent.click(screen.getByText("send-0"));
    await userEvent.click(screen.getByText("send-1"));

    await waitFor(() => expect(screen.getAllByTestId("toast")).toHaveLength(1));
    expect(screen.getByTestId("toast")).toHaveTextContent("Attempt failed again");
  });

  it("stacks toasts with different keys", async () => {
    renderToasts([
      { kind: "error", message: "First", key: "a" },
      { kind: "error", message: "Second", key: "b" },
    ]);
    await userEvent.click(screen.getByText("send-0"));
    await userEvent.click(screen.getByText("send-1"));
    expect(await screen.findAllByTestId("toast")).toHaveLength(2);
  });

  it("dismissAll clears the viewport", async () => {
    renderToasts([{ kind: "info", message: "One", key: "a" }, { kind: "info", message: "Two", key: "b" }]);
    await userEvent.click(screen.getByText("send-0"));
    await userEvent.click(screen.getByText("send-1"));
    await screen.findByText("Two");

    await userEvent.click(screen.getByText("dismiss-all"));
    await waitFor(() => expect(screen.queryAllByTestId("toast")).toHaveLength(0));
  });

  describe("lifetimes", () => {
    beforeEach(() => vi.useFakeTimers());
    afterEach(() => vi.useRealTimers());

    it("auto-dismisses a success but keeps an error until acknowledged", async () => {
      render(
        <ToastProvider>
          <Harness toasts={[
            { kind: "success", message: "Saved" },
            { kind: "error", message: "Broke" },
          ]} />
        </ToastProvider>,
      );

      act(() => { screen.getByText("send-0").click(); });
      act(() => { screen.getByText("send-1").click(); });
      expect(screen.getByText("Saved")).toBeInTheDocument();

      // Well past the 5s success lifetime.
      act(() => { vi.advanceTimersByTime(30_000); });

      expect(screen.queryByText("Saved")).toBeNull();
      // An operator who looked away must still be able to find out what failed.
      expect(screen.getByText("Broke")).toBeInTheDocument();
    });

    it("caps the viewport, evicting successes before errors", async () => {
      const many: ToastInput[] = [
        { kind: "error", message: "E1", key: "e1" },
        { kind: "success", message: "S1", key: "s1" },
        { kind: "success", message: "S2", key: "s2" },
        { kind: "success", message: "S3", key: "s3" },
        { kind: "success", message: "S4", key: "s4" },
        { kind: "success", message: "S5", key: "s5" },
      ];
      render(<ToastProvider><Harness toasts={many} /></ToastProvider>);

      for (let i = 0; i < many.length; i += 1) {
        act(() => { screen.getByText(`send-${i}`).click(); });
      }

      expect(screen.getAllByTestId("toast")).toHaveLength(4);
      expect(screen.getByText("E1")).toBeInTheDocument();
    });
  });

  it("degrades to a no-op outside a provider instead of throwing", async () => {
    resetToastProviderWarning();
    const warn = vi.spyOn(console, "warn").mockImplementation(() => undefined);

    // No <ToastProvider> — a page that forgot the wiring must still render.
    render(<Harness toasts={[{ kind: "error", message: "Nowhere to go" }]} />);
    await userEvent.click(screen.getByText("send-0"));

    expect(screen.queryByTestId("toast")).toBeNull();
    expect(warn).toHaveBeenCalledWith(expect.stringContaining("outside <ToastProvider>"));
    warn.mockRestore();
  });
});
