import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  FindingSchema,
  SummarySchema,
  ValidateScenarioResponseSchema,
} from "@vrooli/proto-types/security-health/v1/validation/validation_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";

vi.mock("../../api/validation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/validation")>();
  return { ...actual, validationClient: { validateScenario: vi.fn() } };
});

import { PostureCard } from "./PostureCard";
import { validationClient, Severity } from "../../api/validation";

const mockValidate = vi.mocked(validationClient.validateScenario);

const failedResponse = () =>
  create(ValidateScenarioResponseSchema, {
    scenario: "security-health",
    passed: false,
    summary: create(SummarySchema, { errors: 1, warnings: 1, infos: 0 }),
    skippedScanners: ["osv-scanner"],
    findings: [
      create(FindingSchema, {
        ruleId: "gitleaks.generic-api-key",
        severity: Severity.ERROR,
        title: "Hardcoded API key",
        description: "A credential is committed.",
        remediation: "Rotate the key and move it to vault.",
        filePath: "api/config.go:12",
        scanner: "gitleaks",
      }),
      create(FindingSchema, {
        ruleId: "gosec.G404",
        severity: Severity.WARNING,
        title: "Weak RNG",
        description: "math/rand used.",
        remediation: "Use crypto/rand.",
        filePath: "api/util.go:3",
        scanner: "gosec",
      }),
    ],
  });

const cleanResponse = () =>
  create(ValidateScenarioResponseSchema, {
    scenario: "security-health",
    passed: true,
    summary: create(SummarySchema, { errors: 0, warnings: 0, infos: 0 }),
    skippedScanners: [],
    findings: [],
  });

describe("PostureCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a failed verdict, summary, skipped scanners and findings", async () => {
    mockValidate.mockResolvedValue(failedResponse());
    renderWithProviders(<PostureCard />);

    await waitFor(() => expect(screen.getByTestId(selectors.posture.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.posture.status)).toHaveAttribute("data-passed", "false");
    // Interpolated values aren't rendered under i18n cimode; assert the line is present.
    expect(screen.getByTestId(selectors.posture.skipped)).toBeInTheDocument();

    const list = screen.getByTestId(selectors.posture.findings);
    expect(within(list).getByText("Hardcoded API key")).toBeInTheDocument();
    expect(within(list).getByText("Weak RNG")).toBeInTheDocument();
  });

  it("shows the clean empty state when nothing is found", async () => {
    mockValidate.mockResolvedValue(cleanResponse());
    renderWithProviders(<PostureCard />);

    await waitFor(() => expect(screen.getByTestId(selectors.posture.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.posture.status)).toHaveAttribute("data-passed", "true");
    expect(screen.getByTestId(selectors.posture.empty)).toBeInTheDocument();
  });

  it("filters to a single scanner when scannerFilter is set (Secrets view)", async () => {
    mockValidate.mockResolvedValue(failedResponse());
    renderWithProviders(<PostureCard scannerFilter="gitleaks" />);

    await waitFor(() => expect(screen.getByTestId(selectors.posture.findings)).toBeInTheDocument());
    const list = screen.getByTestId(selectors.posture.findings);
    expect(within(list).getByText("Hardcoded API key")).toBeInTheDocument();
    expect(within(list).queryByText("Weak RNG")).not.toBeInTheDocument();
  });
});
