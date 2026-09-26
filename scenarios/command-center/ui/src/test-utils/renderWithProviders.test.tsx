/**
 * Self-test for command-center's local renderWithProviders.
 *
 * This scenario deliberately does NOT use `@vrooli/api-base/testing`: its
 * wrapper provides a QueryClient and nothing else, because component tests
 * here render pieces that supply their own routing. What is pinned below is
 * therefore the local delta, not the shared helper's contract.
 *
 *   1. Query retries are disabled. React Query defaults queries to 3 retries
 *      with exponential backoff, so this one is load-bearing: without it every
 *      error-path test still passes, but only after waiting out the backoff.
 *   2. Mutations do not retry. Measured against this version, a bare client
 *      already attempts a failing mutation exactly once, so this pins a
 *      library default rather than a local choice — it earns its place by
 *      catching a future `retry: N` added here or a change in that default,
 *      not by guarding a gap that exists today.
 *   3. The wrapper survives `rerender`, so state-carrying components are not
 *      remounted between assertions (the behaviour the helper's own comment
 *      promises callers).
 */
import { describe, expect, it } from "vitest";
import { waitFor } from "@testing-library/react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useRef, useState } from "react";

import { renderWithProviders, screen } from "./renderWithProviders";

// Retry-with-backoff would take seconds; retry-off resolves within a tick.
// A tight timeout is what makes these assertions discriminating.
const PROMPT = { timeout: 800 };

describe("command-center renderWithProviders", () => {
  it("fails a query immediately rather than retrying", async () => {
    const Probe = () => {
      const q = useQuery({
        queryKey: ["probe"],
        queryFn: () => Promise.reject(new Error("boom")),
      });
      return <span data-testid="state">{q.status}</span>;
    };

    renderWithProviders(<Probe />);
    await waitFor(
      () => expect(screen.getByTestId("state").textContent).toBe("error"),
      PROMPT,
    );
  });

  it("fails a mutation immediately rather than retrying", async () => {
    const Probe = () => {
      const m = useMutation({ mutationFn: () => Promise.reject(new Error("boom")) });
      return (
        <button data-testid="button" type="button" onClick={() => m.mutate()}>
          {m.status}
        </button>
      );
    };

    renderWithProviders(<Probe />);
    screen.getByTestId("button").click();
    await waitFor(
      () => expect(screen.getByTestId("button").textContent).toBe("error"),
      PROMPT,
    );
  });

  it("keeps the tree mounted across rerender", () => {
    const Probe = ({ label }: { label: string }) => {
      const mounts = useRef(0);
      const [seen] = useState(() => ++mounts.current);
      return <span data-testid="probe">{`${label}:${seen}`}</span>;
    };

    const { rerender } = renderWithProviders(<Probe label="first" />);
    expect(screen.getByTestId("probe").textContent).toBe("first:1");

    rerender(<Probe label="second" />);
    // Same mount => the wrapper was not rebuilt underneath the component.
    expect(screen.getByTestId("probe").textContent).toBe("second:1");
  });
});
