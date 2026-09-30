/**
 * Self-test for scenario-to-cloud's local renderWithProviders.
 *
 * The helper provides a QueryClient and nothing else. Note that most hook
 * tests in this scenario build their own inline `createWrapper()` rather than
 * using this helper, so its behaviour was previously unexercised despite the
 * scenario making heavy use of React Query.
 */
import { describe, expect, it } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { useQuery } from "@tanstack/react-query";

import { renderWithProviders } from "./renderWithProviders";

// Retry-with-backoff would exceed this; retry-off resolves within a tick.
const PROMPT = { timeout: 800 };

describe("scenario-to-cloud renderWithProviders", () => {
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

  it("resolves a successful query through the provided client", async () => {
    const Probe = () => {
      const q = useQuery({ queryKey: ["ok"], queryFn: () => Promise.resolve("value") });
      return <span data-testid="data">{String(q.data)}</span>;
    };

    renderWithProviders(<Probe />);
    await waitFor(() => expect(screen.getByTestId("data").textContent).toBe("value"));
  });
});
