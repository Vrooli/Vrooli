import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { InlineExecutionHeader } from "./InlineExecutionHeader";
import { selectors } from "@constants/selectors";
import { renderWithProviders } from "@/test-utils";

describe("InlineExecutionHeader", () => {
  it("renders an accessible liveness indicator alongside a terminal status", () => {
    renderWithProviders(
      <InlineExecutionHeader
        status="completed"
        onClose={vi.fn()}
        heartbeatLabel="Heartbeat just now • Final heartbeat snapshot"
      />,
    );

    expect(screen.getByTestId(selectors.executions.viewer.statusCompleted)).toHaveTextContent("Completed");
    expect(screen.getByTestId(selectors.heartbeat.indicator)).toHaveTextContent("Final heartbeat snapshot");
  });
});
