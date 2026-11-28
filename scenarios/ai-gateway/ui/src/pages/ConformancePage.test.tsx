import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create } from "@bufbuild/protobuf";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { scanScenario } from "../api/gateway";
import {
  ConformanceFindingSchema,
  ScanScenarioResponseSchema,
} from "@vrooli/proto-types/ai-gateway/v1/conformance/conformance_pb";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { renderWithProviders } from "../test-utils";
import { ConformancePage } from "./ConformancePage";

vi.mock("../api/gateway", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/gateway")>();
  const { makeGatewayApiMocks } = await import("../test-utils/mocks/gateway");
  return { ...actual, ...makeGatewayApiMocks() };
});

describe("ConformancePage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("[REQ:AIGW-UI-DASHBOARD] runs a scenario scan and renders grouped findings", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ConformancePage />);

    await user.clear(screen.getByTestId(selectors.conformance.scenarioInput));
    await user.type(screen.getByTestId(selectors.conformance.scenarioInput), "portal");
    await user.click(screen.getByTestId(selectors.conformance.submit));

    expect(scanScenario).toHaveBeenCalledWith("portal");
    expect(await screen.findByTestId(selectors.conformance.result)).toBeInTheDocument();
  });

  it("renders clean scans and scan errors", async () => {
    vi.mocked(scanScenario).mockResolvedValueOnce(create(ScanScenarioResponseSchema, {
      scenario: "ai-gateway",
      maturityLevel: "L4",
      findings: [],
      recommendations: [],
    }));
    const user = userEvent.setup();
    renderWithProviders(<ConformancePage />);

    await user.click(screen.getByTestId(selectors.conformance.submit));
    expect(await screen.findByText(strings.pages.conformance.empty)).toBeInTheDocument();

    vi.mocked(scanScenario).mockRejectedValueOnce(new Error("scan failed"));
    await user.click(screen.getByTestId(selectors.conformance.submit));
    expect(await screen.findByTestId(selectors.conformance.error)).toHaveTextContent("scan failed");
  });

  it("renders low, high, and neutral finding severities", async () => {
    vi.mocked(scanScenario).mockResolvedValueOnce(create(ScanScenarioResponseSchema, {
      scenario: "portal",
      maturityLevel: "L2",
      findings: [
        create(ConformanceFindingSchema, { ruleId: "LOW", severity: "low", path: "a.go", message: "low", remediation: "fix" }),
        create(ConformanceFindingSchema, { ruleId: "HIGH", severity: "high", path: "b.go", message: "high", remediation: "fix" }),
        create(ConformanceFindingSchema, { ruleId: "NOTE", severity: "note", path: "c.go", message: "note", remediation: "fix" }),
      ],
      recommendations: [],
    }));
    const user = userEvent.setup();
    renderWithProviders(<ConformancePage />);

    await user.click(screen.getByTestId(selectors.conformance.submit));

    const result = await screen.findByTestId(selectors.conformance.result);
    expect(result).toHaveTextContent("LOW");
    expect(result).toHaveTextContent("HIGH");
    expect(result).toHaveTextContent("NOTE");
  });
});
