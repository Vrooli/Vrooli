/**
 * useActionMutation is the app's promise that no operator-triggered action
 * can fail silently. These tests hold that promise to the fire.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ToastProvider } from "../components/ui/toast-provider";
import { ApiError } from "../lib/api-client";
import { useActionMutation, type ActionMutationOptions } from "./useActionMutation";

function createClient() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
}

function Wrapper({ children, client }: { children: ReactNode; client: QueryClient }) {
  return (
    <QueryClientProvider client={client}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}

function Harness<TData>({ options }: { options: ActionMutationOptions<TData, void> }) {
  const mutation = useActionMutation(options);
  return (
    <div>
      <button type="button" onClick={() => mutation.run()}>go</button>
      <span data-testid="pending">{mutation.isPending ? "pending" : "idle"}</span>
      <span data-testid="inline">{mutation.errorDescription?.message ?? ""}</span>
    </div>
  );
}

function renderAction<TData>(options: ActionMutationOptions<TData, void>, client = createClient()) {
  render(<Wrapper client={client}><Harness options={options} /></Wrapper>);
  return client;
}

describe("useActionMutation", () => {
  it("surfaces a failure that would otherwise be silent", async () => {
    renderAction({
      mutationFn: () => Promise.reject(new ApiError("http", "milestone has no acceptance criteria", { status: 400 })),
      errorMessage: "Couldn't start the milestone review",
    });

    await userEvent.click(screen.getByText("go"));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Couldn't start the milestone review");
    expect(alert).toHaveTextContent("milestone has no acceptance criteria");
  });

  it("stays quiet on success when no successMessage is given", async () => {
    renderAction({
      mutationFn: () => Promise.resolve("ok"),
      errorMessage: "Couldn't save",
    });

    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("pending")).toHaveTextContent("idle"));
    expect(screen.queryByTestId("toast")).toBeNull();
  });

  it("confirms success when a successMessage is given", async () => {
    renderAction({
      mutationFn: () => Promise.resolve("ok"),
      errorMessage: "Couldn't save",
      successMessage: "Saved",
    });

    await userEvent.click(screen.getByText("go"));
    expect(await screen.findByText("Saved")).toBeInTheDocument();
  });

  it("derives the success message from the result", async () => {
    renderAction({
      mutationFn: () => Promise.resolve({ name: "release-1" }),
      errorMessage: "Couldn't create",
      successMessage: (data) => `Created ${data.name}`,
    });

    await userEvent.click(screen.getByText("go"));
    expect(await screen.findByText("Created release-1")).toBeInTheDocument();
  });

  it("offers Retry on a retryable failure and re-runs the same call", async () => {
    const mutationFn = vi.fn()
      .mockRejectedValueOnce(new ApiError("network", "Network request failed"))
      .mockResolvedValueOnce("ok");

    renderAction({
      mutationFn: mutationFn as () => Promise<string>,
      errorMessage: "Couldn't save",
      successMessage: "Saved",
    });

    await userEvent.click(screen.getByText("go"));
    await userEvent.click(await screen.findByTestId("toast-action"));

    await waitFor(() => expect(mutationFn).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Saved")).toBeInTheDocument();
  });

  it("does not offer Retry when repeating the request cannot help", async () => {
    renderAction({
      mutationFn: () => Promise.reject(new ApiError("http", "name already taken", { status: 409 })),
      errorMessage: "Couldn't create",
    });

    await screen.findByText("go");
    await userEvent.click(screen.getByText("go"));

    await screen.findByRole("alert");
    expect(screen.queryByTestId("toast-action")).toBeNull();
  });

  it("respects allowRetry: false for non-idempotent actions", async () => {
    renderAction({
      mutationFn: () => Promise.reject(new ApiError("network", "Network request failed")),
      errorMessage: "Couldn't start the run",
      allowRetry: false,
    });

    await userEvent.click(screen.getByText("go"));
    await screen.findByRole("alert");
    expect(screen.queryByTestId("toast-action")).toBeNull();
  });

  it("silentError keeps the reason inline without a toast", async () => {
    renderAction({
      mutationFn: () => Promise.reject(new ApiError("http", "title is required", { status: 422 })),
      errorMessage: "Couldn't save this goal",
      silentError: true,
    });

    await userEvent.click(screen.getByText("go"));

    await waitFor(() => expect(screen.getByTestId("inline")).toHaveTextContent("title is required"));
    expect(screen.queryByTestId("toast")).toBeNull();
  });

  it("invalidates the declared query keys on success", async () => {
    const client = createClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");

    renderAction({
      mutationFn: () => Promise.resolve("ok"),
      errorMessage: "Couldn't save",
      invalidateKeys: [["goal", "workspace"], ["goals"]],
    }, client);

    await userEvent.click(screen.getByText("go"));

    await waitFor(() => expect(invalidate).toHaveBeenCalledWith({ queryKey: ["goal", "workspace"] }));
    expect(invalidate).toHaveBeenCalledWith({ queryKey: ["goals"] });
  });

  it("does not invalidate when the action failed", async () => {
    const client = createClient();
    const invalidate = vi.spyOn(client, "invalidateQueries");

    renderAction({
      mutationFn: () => Promise.reject(new Error("nope")),
      errorMessage: "Couldn't save",
      invalidateKeys: [["goals"]],
    }, client);

    await userEvent.click(screen.getByText("go"));
    await screen.findByRole("alert");

    expect(invalidate).not.toHaveBeenCalled();
  });

  it("reports pending while in flight", async () => {
    let release: (value: string) => void = () => undefined;
    renderAction({
      mutationFn: () => new Promise<string>((resolve) => { release = resolve; }),
      errorMessage: "Couldn't save",
    });

    await userEvent.click(screen.getByText("go"));
    await waitFor(() => expect(screen.getByTestId("pending")).toHaveTextContent("pending"));

    release("done");
    await waitFor(() => expect(screen.getByTestId("pending")).toHaveTextContent("idle"));
  });

  it("run() returns nothing to reject, so onClick needs no catch", async () => {
    // The reason `run` exists alongside `mutateAsync`: handing a rejecting
    // promise to onClick produces an unhandled rejection and a floating-promise
    // lint error at every call site.
    const results: unknown[] = [];
    function Probe() {
      const mutation = useActionMutation({
        mutationFn: () => Promise.reject(new Error("boom")),
        errorMessage: "Couldn't save",
      });
      return <button type="button" onClick={() => results.push(mutation.run())}>probe</button>;
    }

    render(<Wrapper client={createClient()}><Probe /></Wrapper>);
    await userEvent.click(screen.getByText("probe"));
    await screen.findByRole("alert");

    expect(results).toEqual([undefined]);
  });
});
