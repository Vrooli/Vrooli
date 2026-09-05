import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/signals", () => ({
  signalsClient: {
    boundaryHealth: vi.fn(),
  },
}));

import { signalsClient } from "../../api/signals";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { BoundaryHealth } from "./BoundaryHealth";
import { CouplingSeverity } from "@vrooli/proto-types/architecture-cartographer/v1/signals/signals_pb";

type ReportResult = Awaited<ReturnType<typeof signalsClient.boundaryHealth>>;

afterEach(() => {
  cleanup();
  vi.mocked(signalsClient.boundaryHealth).mockReset();
});

describe("BoundaryHealth", () => {
  it("renders the empty state when there are no domains", async () => {
    vi.mocked(signalsClient.boundaryHealth).mockResolvedValue({
      scenario: "demo",
      totalDomains: 0,
      domains: [],
    } as unknown as ReportResult);

    renderWithProviders(<BoundaryHealth scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.domains.boundaries.empty)).toBeInTheDocument(),
    );
  });

  it("renders one row per domain with score, metrics, stable-kernel tag, and smells", async () => {
    vi.mocked(signalsClient.boundaryHealth).mockResolvedValue({
      scenario: "demo",
      totalDomains: 2,
      domains: [
        {
          domain: "graph",
          archetype: "service",
          efferent: 3,
          afferent: 1,
          instability: 0.75,
          fanOut: 0.5,
          dependsOn: ["conflicts", "signals", "apply"],
          dependedBy: ["overview"],
          stableKernel: false,
          healthScore: 0.42,
          smells: [
            {
              kind: "unstable_dependency",
              severity: CouplingSeverity.WARN,
              message: "Depends on a less-stable domain.",
            },
          ],
        },
        {
          domain: "errors",
          archetype: "kernel",
          efferent: 0,
          afferent: 4,
          instability: 0,
          fanOut: 0,
          dependsOn: [],
          dependedBy: ["graph", "conflicts", "apply", "signals"],
          stableKernel: true,
          healthScore: 0.95,
          smells: [],
        },
      ],
    } as unknown as ReportResult);

    renderWithProviders(<BoundaryHealth scenario="demo" />);

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.domains.boundaries.row({ domain: "graph" })),
      ).toBeInTheDocument(),
    );
    const graphRow = screen.getByTestId(selectors.features.domains.boundaries.row({ domain: "graph" }));
    const errorsRow = screen.getByTestId(selectors.features.domains.boundaries.row({ domain: "errors" }));

    // Score is rendered as text (with a label), never color-only.
    expect(graphRow).toHaveTextContent("0.42");
    expect(errorsRow).toHaveTextContent("0.95");
    // Metrics present.
    expect(graphRow).toHaveTextContent("0.75");
    expect(graphRow).toHaveTextContent("unstable_dependency");
    expect(graphRow).toHaveTextContent("Depends on a less-stable domain.");
    // Smell severity uses the text label (cimode key path).
    expect(screen.getByText(strings.pages.targetDomains.boundaries.severityWarn)).toBeInTheDocument();
    // Stable-kernel tag only on the kernel domain.
    expect(errorsRow).toHaveTextContent(strings.pages.targetDomains.boundaries.stableKernel);
    expect(graphRow).not.toHaveTextContent(strings.pages.targetDomains.boundaries.stableKernel);
  });

  it("shows a friendly error state when the query fails (no snapshot yet)", async () => {
    vi.mocked(signalsClient.boundaryHealth).mockRejectedValue(new Error("not found"));

    renderWithProviders(<BoundaryHealth scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.domains.boundaries.error)).toBeInTheDocument(),
    );
  });
});
