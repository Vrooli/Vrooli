import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

vi.mock("../../api/domains", () => ({
  domainsClient: {
    convergenceReport: vi.fn(),
  },
}));

import { domainsClient } from "../../api/domains";
import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { ConvergenceReport } from "./ConvergenceReport";
import { ConvergenceSeverity, DomainSource } from "@vrooli/proto-types/architecture-cartographer/v1/domains/domains_pb";

type ReportResult = Awaited<ReturnType<typeof domainsClient.convergenceReport>>;

afterEach(() => {
  cleanup();
  vi.mocked(domainsClient.convergenceReport).mockReset();
});

describe("ConvergenceReport", () => {
  it("renders the converged state when there are no findings", async () => {
    vi.mocked(domainsClient.convergenceReport).mockResolvedValue({
      scenario: "demo",
      authority: DomainSource.DOMAINS_DOC,
      findings: [],
    } as unknown as ReportResult);

    renderWithProviders(<ConvergenceReport scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.domains.convergence.converged)).toBeInTheDocument(),
    );
  });

  it("renders one row per finding with a text severity label, domain, kind, and message", async () => {
    vi.mocked(domainsClient.convergenceReport).mockResolvedValue({
      scenario: "demo",
      authority: DomainSource.DOMAINS_DOC,
      findings: [
        {
          kind: "missing_implementation",
          domain: "graph",
          severity: ConvergenceSeverity.WARN,
          message: "Declared in DOMAINS.md but no api folder.",
          sources: [DomainSource.DOMAINS_DOC],
        },
        {
          kind: "ui_feature_no_domain",
          domain: "settings",
          severity: ConvergenceSeverity.INFO,
          message: "UI feature folder with no declared domain.",
          sources: [DomainSource.UI_FEATURES],
        },
      ],
    } as unknown as ReportResult);

    renderWithProviders(<ConvergenceReport scenario="demo" />);

    await waitFor(() =>
      expect(
        screen.getByTestId(selectors.features.domains.convergence.finding({ index: 0 })),
      ).toBeInTheDocument(),
    );
    const warnRow = screen.getByTestId(
      selectors.features.domains.convergence.finding({ index: 0 }),
    );
    const infoRow = screen.getByTestId(
      selectors.features.domains.convergence.finding({ index: 1 }),
    );
    // Severity is not color-only: the text label (cimode key path) is present.
    expect(screen.getByText(strings.pages.targetDomains.convergence.severityWarn)).toBeInTheDocument();
    expect(screen.getByText(strings.pages.targetDomains.convergence.severityInfo)).toBeInTheDocument();
    // Finding data (domain, kind, message) is raw API content rendered in the row.
    expect(warnRow).toHaveTextContent("graph");
    expect(warnRow).toHaveTextContent("missing_implementation");
    expect(warnRow).toHaveTextContent("Declared in DOMAINS.md but no api folder.");
    expect(infoRow).toHaveTextContent("settings");
    expect(infoRow).toHaveTextContent("ui_feature_no_domain");
  });

  it("shows the error state when the report query fails", async () => {
    vi.mocked(domainsClient.convergenceReport).mockRejectedValue(new Error("boom"));

    renderWithProviders(<ConvergenceReport scenario="demo" />);

    await waitFor(() =>
      expect(screen.getByTestId(selectors.features.domains.convergence.error)).toBeInTheDocument(),
    );
  });
});
