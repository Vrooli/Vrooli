import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { renderWithProviders } from "../test-utils";
import { ComplianceOverview } from "./ComplianceOverview";

describe("ComplianceOverview", () => {
  afterEach(cleanup);

  it("displays current compliance and vulnerability metrics with help", () => {
    renderWithProviders(
      <ComplianceOverview
        overallScore={91}
        configuredComponents={8}
        securityScore={88}
        vaultHealth={94}
        vulnerabilitySummary={{ critical: 1, high: 2, medium: 3, low: 4 }}
        isComplianceLoading={false}
      />
    );
    expect(screen.getByText("91%")).toBeInTheDocument();
    expect(screen.getByText("Configured Components")).toBeInTheDocument();
    expect(screen.getByText("8")).toBeInTheDocument();
    expect(screen.getByText("4")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Help" }));
    expect(screen.getByText("Compliance Metrics")).toBeInTheDocument();
    expect(screen.getByText("Vulnerability Severities:")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Close help dialog" }));
    expect(screen.queryByText("Compliance Metrics")).not.toBeInTheDocument();
  });

  it("uses loading placeholders instead of stale metric values", () => {
    renderWithProviders(
      <ComplianceOverview
        vulnerabilitySummary={{ critical: 0, high: 0, medium: 0, low: 0 }}
        isComplianceLoading
      />
    );
    expect(screen.getByText("Vulnerability Mix")).toBeInTheDocument();
    expect(screen.queryByText("Overall Score")).not.toBeInTheDocument();
    expect(screen.getByText("Scan Inputs")).toBeInTheDocument();
  });
});
