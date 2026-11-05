import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { renderWithProviders } from "../../test-utils";
import { makeAuditQualityResponse, makeExplainFindingResponse, makeFixConfigResponse } from "./testFactories";
import { ScenarioAuditWorkbench } from "./ScenarioAuditWorkbench";

const mocks = vi.hoisted(() => ({
  auditQuality: vi.fn(),
  explainFinding: vi.fn(),
  previewFixConfig: vi.fn(),
}));

vi.mock("../../api/audit", () => ({
  auditClient: mocks,
}));

describe("ScenarioAuditWorkbench", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders audit overview, surfaces, grouped findings, and contract detail", async () => {
    mocks.auditQuality.mockResolvedValue(makeAuditQualityResponse());
    mocks.explainFinding.mockResolvedValue(makeExplainFindingResponse());

    renderWithProviders(<ScenarioAuditWorkbench />);

    expect(await screen.findByTestId(selectors.qualityWorkbench.status)).toHaveTextContent("failed");
    expect(screen.getByTestId(selectors.qualityWorkbench.maturity)).toHaveTextContent("R2");
    expect(screen.getByTestId(selectors.qualityWorkbench.surfaceCard({ id: "ui" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.qualityWorkbench.findingRow({ id: "finding-tsconfig" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.qualityWorkbench.contractDetail)).toHaveTextContent("TS_CONFIG_STRICT");
    expect(screen.getByTestId(selectors.qualityWorkbench.autofixCandidate({ ruleId: "TS_CONFIG_STRICT" }))).toBeInTheDocument();

    expect(mocks.auditQuality).toHaveBeenCalledWith({
      scenario: "quality-health",
      includeCommandExecution: true,
      includeAutofixPreview: true,
      useCache: true,
    });
  });

  it("submits a selected scenario and filters findings", async () => {
    const user = userEvent.setup();
    mocks.auditQuality.mockResolvedValue(makeAuditQualityResponse());
    mocks.explainFinding.mockResolvedValue(makeExplainFindingResponse());

    renderWithProviders(<ScenarioAuditWorkbench />);

    const input = await screen.findByTestId(selectors.qualityWorkbench.scenarioInput);
    await user.clear(input);
    await user.type(input, "web-console");
    await user.click(screen.getByTestId(selectors.qualityWorkbench.runButton));

    await waitFor(() => {
      expect(mocks.auditQuality).toHaveBeenLastCalledWith({
        scenario: "web-console",
        includeCommandExecution: true,
        includeAutofixPreview: true,
        useCache: true,
      });
    });

    await user.selectOptions(screen.getByLabelText(strings.quality.severityFilter), "warning");
    expect(screen.queryByTestId(selectors.qualityWorkbench.findingRow({ id: "finding-tsconfig" }))).not.toBeInTheDocument();
    expect(screen.getByTestId(selectors.qualityWorkbench.findingRow({ id: "finding-pattern" }))).toBeInTheDocument();
  });

  it("previews autofix candidates explicitly", async () => {
    const user = userEvent.setup();
    mocks.auditQuality.mockResolvedValue(makeAuditQualityResponse());
    mocks.explainFinding.mockResolvedValue(makeExplainFindingResponse());
    mocks.previewFixConfig.mockResolvedValue(makeFixConfigResponse());

    renderWithProviders(<ScenarioAuditWorkbench />);

    await screen.findByTestId(selectors.qualityWorkbench.previewFixButton);
    await user.click(screen.getByTestId(selectors.qualityWorkbench.previewFixButton));

    await waitFor(() => {
      expect(mocks.previewFixConfig).toHaveBeenCalledWith({
        scenario: "quality-health",
        ruleIds: ["TS_CONFIG_STRICT"],
        apply: false,
      });
    });
    expect(screen.getByTestId(selectors.qualityWorkbench.autofixPreview)).toHaveTextContent("tsconfig.json");
  });
});
