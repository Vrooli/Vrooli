import { screen } from "@testing-library/react";
import { renderWithProviders as render } from "@vrooli/api-base/testing";
import { describe, expect, it } from "vitest";
import { StreamingDegradationNotice } from "./StreamingDegradationNotice";

describe("StreamingDegradationNotice", () => {
  it("renders a turn-scoped status rather than a provider-down pill", () => {
    render(<StreamingDegradationNotice notice="Streaming degraded — buffered mode is active." />);
    expect(screen.getByTestId("streaming-degradation-notice")).toHaveTextContent("buffered mode");
  });

  it("does not render after recovery", () => {
    render(<StreamingDegradationNotice notice={null} />);
    expect(screen.queryByTestId("streaming-degradation-notice")).toBeNull();
  });
});
