import { describe, expect, it } from "vitest";
import { StatusIcon } from "./StatusIcon";
import type { HealthStatus } from "../../lib/api";
import { renderWithProviders } from "../../test-utils";

describe("StatusIcon", () => {
  it("uses the neutral activity icon for an unknown status", () => {
    const { container } = renderIcon("unknown");
    expect(container.querySelector(".lucide-activity")).toBeInTheDocument();
  });
});

function renderIcon(status: "ok" | "warning" | "critical" | "unknown") {
  // Keep this helper local so the test remains a user-visible icon assertion.
  return renderWithProviders(<StatusIcon status={status as HealthStatus} />);
}
