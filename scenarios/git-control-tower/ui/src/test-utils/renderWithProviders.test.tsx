/**
 * Self-test for git-control-tower's local renderWithProviders.
 *
 * This helper is a thin projection over `renderWithQueryClient` in ./render —
 * it exists so the scenario test policy has a stable canonical path and so
 * feature tests do not invent query-provider variants. The contract worth
 * pinning is therefore the delegation itself plus the client defaults it
 * inherits, neither of which had a test.
 */
import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { renderWithProviders } from "./renderWithProviders";
import { createTestQueryClient } from "./render";

// Retry-with-backoff would exceed this; retry-off resolves within a tick.
const PROMPT = { timeout: 800 };

describe("git-control-tower renderWithProviders", () => {
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

  it("honors a caller-supplied queryClient", async () => {
    const seeded = createTestQueryClient();
    seeded.setQueryData(["seeded"], "hello");

    const Probe = () => {
      const q = useQuery({
        queryKey: ["seeded"],
        queryFn: () => Promise.resolve("network"),
      });
      return <span data-testid="seeded">{String(q.data)}</span>;
    };

    renderWithProviders(<Probe />, { queryClient: seeded });
    await waitFor(() => expect(screen.getByTestId("seeded").textContent).toBe("hello"));
  });
});
