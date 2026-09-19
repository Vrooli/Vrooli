/**
 * FindingsPage tests — scenario picker → validate → grouped findings → fix
 * preview/apply. Both connect clients are mocked at the module boundary so
 * tests assert component behavior against fixture proto messages.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Code, ConnectError } from "@connectrpc/connect";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { setLocale } from "../../i18n";

vi.mock("../../api/contract", () => ({
  contractClient: {
    validateScenario: vi.fn(),
    getMatrix: vi.fn(),
    getDrift: vi.fn(),
    logManualValidation: vi.fn(),
  },
}));

vi.mock("../../api/validation", () => ({
  validationClient: {
    previewFix: vi.fn(),
    applyFix: vi.fn(),
    validateScenario: vi.fn(),
  },
}));

import { FindingsPage } from "./FindingsPage";
import { contractClient } from "../../api/contract";
import { validationClient } from "../../api/validation";
import {
  makeBusinessContractReport,
  makeCapabilityRollup,
  makeContractFinding,
  makeFixResponse,
  makeValidateScenarioResponse,
} from "./mocks/factories";

const loadScenario = async (slug = "business-health") => {
  const user = userEvent.setup();
  await user.type(screen.getByTestId(selectors.scenarioPicker.input), slug);
  await user.click(screen.getByTestId(selectors.scenarioPicker.submit));
  return user;
};

describe("FindingsPage", () => {
  beforeEach(() => {
    vi.mocked(contractClient.validateScenario).mockReset();
    vi.mocked(validationClient.previewFix).mockReset();
    vi.mocked(validationClient.applyFix).mockReset();
  });
  afterEach(async () => {
    cleanup();
    await setLocale("en");
  });

  it("starts on the choose-scenario empty state without validating", () => {
    renderWithProviders(<FindingsPage />);
    expect(screen.getByText(strings.common.chooseScenario)).toBeInTheDocument();
    expect(contractClient.validateScenario).not.toHaveBeenCalled();
  });

  it("shows a loading state while validation is in flight", async () => {
    vi.mocked(contractClient.validateScenario).mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<FindingsPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.findings.loading)).toBeInTheDocument();
  });

  it("renders grouped findings on success", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse(),
    );
    renderWithProviders(<FindingsPage />);
    await loadScenario();

    expect(await screen.findByTestId(selectors.findings.list)).toBeInTheDocument();
    expect(
      screen.getByTestId(selectors.findings.item({ code: "intent.ot_orphan" })),
    ).toBeInTheDocument();
  });

  it("shows the empty state for a clean scenario", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        status: "PASSED",
        report: makeBusinessContractReport({ findings: [], capabilities: [] }),
      }),
    );
    renderWithProviders(<FindingsPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.findings.empty)).toBeInTheDocument();
  });

  it("renders an error alert when validation fails", async () => {
    vi.mocked(contractClient.validateScenario).mockRejectedValue(new Error("boom"));
    renderWithProviders(<FindingsPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.findings.error)).toBeInTheDocument();
  });

  it("strips a :CLAIM-ID suffix from the code in the remediation doc link", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          capabilities: [makeCapabilityRollup({ capabilityId: "intent_linkage" })],
          findings: [makeContractFinding({ code: "intent.ot_orphan:CLAIM-1" })],
        }),
      }),
    );
    renderWithProviders(<FindingsPage />);
    await loadScenario();

    const link = await screen.findByTestId(selectors.findings.docLink);
    expect(link).toHaveAttribute("href", "docs/findings/intent.ot_orphan.md");
  });

  it("previews a fix and renders diffs, then applies on explicit confirm", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          findings: [makeContractFinding({ code: "intent.ot_orphan", autofixAvailable: true })],
        }),
      }),
    );
    vi.mocked(validationClient.previewFix).mockResolvedValue(makeFixResponse());
    vi.mocked(validationClient.applyFix).mockResolvedValue(
      makeFixResponse({ applied: true }),
    );

    renderWithProviders(<FindingsPage />);
    const user = await loadScenario();

    await user.click(await screen.findByTestId(selectors.findings.previewFix));

    expect(await screen.findByTestId(selectors.findings.fixDiff)).toBeInTheDocument();
    expect(validationClient.previewFix).toHaveBeenCalledWith(
      expect.objectContaining({ scenario: "business-health", ruleIds: ["intent.ot_orphan"] }),
    );

    await user.click(screen.getByTestId(selectors.findings.applyFix));
    await waitFor(() => {
      expect(validationClient.applyFix).toHaveBeenCalledWith(
        expect.objectContaining({ scenario: "business-health", ruleIds: ["intent.ot_orphan"] }),
      );
    });
    expect(await screen.findByText(strings.findings.fixApplied)).toBeInTheDocument();
  });

  it("treats an Unimplemented preview as an empty-fix notice, not an error", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          findings: [makeContractFinding({ code: "intent.ot_orphan", autofixAvailable: true })],
        }),
      }),
    );
    vi.mocked(validationClient.previewFix).mockRejectedValue(
      new ConnectError("no fixer", Code.Unimplemented),
    );

    renderWithProviders(<FindingsPage />);
    const user = await loadScenario();

    await user.click(await screen.findByTestId(selectors.findings.previewFix));
    const messages = await screen.findByTestId(selectors.findings.fixMessages);
    expect(messages).toHaveTextContent(strings.findings.fixEmpty);
  });

  it("shows the no-fixable notice when preview returns zero candidates", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          findings: [makeContractFinding({ code: "intent.ot_orphan", autofixAvailable: true })],
        }),
      }),
    );
    vi.mocked(validationClient.previewFix).mockResolvedValue(
      makeFixResponse({ candidates: [], messages: [] }),
    );
    renderWithProviders(<FindingsPage />);
    const user = await loadScenario();

    await user.click(await screen.findByTestId(selectors.findings.previewFix));
    expect(await screen.findByText(strings.findings.noFixable)).toBeInTheDocument();
  });

  it("renders fix messages alongside the diff", async () => {
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse({
        report: makeBusinessContractReport({
          findings: [makeContractFinding({ code: "intent.ot_orphan", autofixAvailable: true })],
        }),
      }),
    );
    vi.mocked(validationClient.previewFix).mockResolvedValue(
      makeFixResponse({ messages: ["patched cleanly"] }),
    );
    renderWithProviders(<FindingsPage />);
    const user = await loadScenario();

    await user.click(await screen.findByTestId(selectors.findings.previewFix));
    expect(await screen.findByTestId(selectors.findings.fixDiff)).toBeInTheDocument();
    const messages = await screen.findByTestId(selectors.findings.fixMessages);
    expect(messages).toHaveTextContent("patched cleanly");
  });

  it("renders under a real locale", async () => {
    await setLocale("ja");
    vi.mocked(contractClient.validateScenario).mockResolvedValue(
      makeValidateScenarioResponse(),
    );
    renderWithProviders(<FindingsPage />);
    await loadScenario();
    expect(await screen.findByTestId(selectors.findings.list)).toBeInTheDocument();
  });
});
