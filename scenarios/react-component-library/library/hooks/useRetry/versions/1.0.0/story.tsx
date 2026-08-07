import { useRef } from "react";
import { useRetry } from "./useRetry";

export function Default() {
  const callCount = useRef(0);
  const retry = useRetry<string>({ maxAttempts: 3, backoff: 80 });
  const run = () =>
    void retry
      .run(() => {
        const next = callCount.current + 1;
        callCount.current = next;
        if (next < 2) throw new Error("temporary");
        return Promise.resolve("Connected");
      })
      .catch(() => undefined);
  return (
    <div style={{ display: "grid", gap: "var(--space-sm, 12px)" }}>
      <button type="button" onClick={run}>
        Try connection
      </button>
      <span role="status">
        {retry.status === "running"
          ? `Attempt ${retry.attempt}…`
          : (retry.value ?? retry.status)}
      </span>
    </div>
  );
}

export function NonRetryable() {
  const retry = useRetry({ maxAttempts: 4, isRetryable: () => false });
  return (
    <button
      type="button"
      onClick={() =>
        void retry
          .run(() => Promise.reject(new Error("not retryable")))
          .catch(() => undefined)
      }
    >
      Run classified failure
    </button>
  );
}
