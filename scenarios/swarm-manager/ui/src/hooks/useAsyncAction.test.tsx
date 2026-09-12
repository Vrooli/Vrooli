import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { ToastProvider } from "../components/ui/toast-provider";
import { ApiError } from "../lib/api-client";
import { useAsyncAction, type AsyncActionOptions } from "./useAsyncAction";

function Harness({
  action,
  options,
  errorMessage = "Couldn't complete this action",
  successMessage,
  onResult,
}: {
  action: () => Promise<unknown>;
  options?: AsyncActionOptions;
  errorMessage?: string;
  successMessage?: string;
  onResult?: (ok: boolean) => void;
}) {
  const { pending, error, run, fail, reset } = useAsyncAction(options);
  return (
    <div>
      <button type="button" onClick={() => void run(action, { errorMessage, successMessage }).then((ok) => onResult?.(ok))}>go</button>
      <button type="button" onClick={() => fail("Pick a target first")}>fail</button>
      <button type="button" onClick={reset}>reset</button>
      <span data-testid="pending">{pending ? "pending" : "idle"}</span>
      <span data-testid="error">{error ?? ""}</span>
    </div>
  );
}

function renderAction(props: Parameters<typeof Harness>[0]) {
  return render(<ToastProvider><Harness {...props} /></ToastProvider>);
}

describe("useAsyncAction", () => {
  it("exposes the server's reason rather than a generic fallback", async () => {
    // The hand-rolled pattern this replaces used `cause instanceof Error ?
    // cause.message : fallback`, which threw away ApiError's structured body.
    renderAction({
      action: () => Promise.reject(new ApiError("http", "milestone not found: goal/gone", { status: 404 })),
    });

    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("milestone not found: goal/gone"));
  });

  it("resolves true on success and false on failure", async () => {
    const onResult = vi.fn();
    renderAction({ action: () => Promise.resolve("ok"), onResult });
    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(onResult).toHaveBeenCalledWith(true));

    const onFailure = vi.fn();
    renderAction({ action: () => Promise.reject(new Error("no")), onResult: onFailure, options: { toastOnError: false } });
    await userEvent.click(screen.getAllByText("go")[1] as HTMLElement);
    await waitFor(() => expect(onFailure).toHaveBeenCalledWith(false));
  });

  it("ignores a second click while the first is still in flight", async () => {
    // `pending` state lags a render behind, so a double-tap would otherwise
    // fire the request twice.
    const action = vi.fn(() => new Promise<string>((resolve) => setTimeout(() => resolve("ok"), 50)));
    renderAction({ action });

    const button = screen.getByText("go");
    await userEvent.click(button);
    await userEvent.click(button);

    await waitFor(() => expect(screen.getByTestId("pending")).toHaveTextContent("idle"));
    expect(action).toHaveBeenCalledTimes(1);
  });

  it("toasts the failure by default so an unrendered error cannot vanish", async () => {
    renderAction({ action: () => Promise.reject(new Error("boom")), errorMessage: "Couldn't save follow-up" });
    await userEvent.click(screen.getByText("go"));
    expect(await screen.findByRole("alert")).toHaveTextContent("Couldn't save follow-up");
  });

  it("suppresses the toast when the caller renders the error inline", async () => {
    renderAction({
      action: () => Promise.reject(new Error("boom")),
      options: { toastOnError: false },
    });

    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("boom"));
    expect(screen.queryByTestId("toast")).toBeNull();
  });

  it("clears a previous error when the action is retried", async () => {
    let shouldFail = true;
    renderAction({
      action: () => shouldFail ? Promise.reject(new Error("first failure")) : Promise.resolve("ok"),
      options: { toastOnError: false },
    });

    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("first failure"));

    shouldFail = false;
    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent(""));
  });

  it("fail() reports a precondition through the same channel", async () => {
    renderAction({ action: () => Promise.resolve("ok"), options: { toastOnError: false } });
    await userEvent.click(screen.getByText("fail"));
    expect(screen.getByTestId("error")).toHaveTextContent("Pick a target first");
  });

  it("reset() clears the error", async () => {
    renderAction({ action: () => Promise.reject(new Error("boom")), options: { toastOnError: false } });
    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("error")).toHaveTextContent("boom"));

    await userEvent.click(screen.getByText("reset"));
    expect(screen.getByTestId("error")).toHaveTextContent("");
  });

  it("confirms success when asked", async () => {
    renderAction({ action: () => Promise.resolve("ok"), successMessage: "Follow-up saved" });
    await userEvent.click(screen.getByText("go"));
    expect(await screen.findByText("Follow-up saved")).toBeInTheDocument();
  });

  it("does not write state after the component unmounts", async () => {
    const errors: unknown[] = [];
    const original = console.error;
    console.error = (...args: unknown[]) => errors.push(args);

    let reject: (reason: Error) => void = () => undefined;
    const { unmount } = renderAction({
      action: () => new Promise<string>((_resolve, rej) => { reject = rej; }),
      options: { toastOnError: false },
    });

    await userEvent.click(screen.getByText("go"));
    unmount();
    reject(new Error("late failure"));
    await new Promise((resolve) => setTimeout(resolve, 10));

    console.error = original;
    const warnings = errors.flat().filter((entry) => typeof entry === "string" && entry.includes("unmounted"));
    expect(warnings).toHaveLength(0);
  });
});
