import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  ValidateScenarioResponseSchema,
  ValidationStatus,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";
import {
  AssessmentFindingSchema,
  MaturityAssessmentSchema,
} from "@vrooli/proto-types/common/v1/maturity_pb";

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
    status: ValidationStatus.FAILED,
    assessment: create(MaturityAssessmentSchema, {
      scenario: "security-health",
      provider: "security-health",
      phase: "security",
      version: "test",
      findingsBySeverity: { SEVERITY_ERROR: 1, SEVERITY_WARNING: 1 },
      findings: [
        create(AssessmentFindingSchema, {
          code: "gitleaks.generic-api-key",
          severity: Severity.ERROR,
          title: "Hardcoded API key",
          message: "A credential is committed.",
          remediation: "Rotate the key and move it to vault.",
          location: "api/config.go:12",
        }),
        create(AssessmentFindingSchema, {
          code: "gosec.G404",
          severity: Severity.WARNING,
          title: "Weak RNG",
          message: "math/rand used.",
          remediation: "Use crypto/rand.",
          location: "api/util.go:3",
        }),
      ],
    }),
  });

const cleanResponse = () =>
  create(ValidateScenarioResponseSchema, {
    scenario: "security-health",
    status: ValidationStatus.PASSED,
    assessment: create(MaturityAssessmentSchema, {
      scenario: "security-health",
      provider: "security-health",
      phase: "security",
      version: "test",
    }),
  });

describe("PostureCard", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a failed verdict, summary and findings", async () => {
    mockValidate.mockResolvedValue(failedResponse());
    renderWithProviders(<PostureCard />);

    await waitFor(() => expect(screen.getByTestId(selectors.posture.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.posture.status)).toHaveAttribute("data-passed", "false");

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
