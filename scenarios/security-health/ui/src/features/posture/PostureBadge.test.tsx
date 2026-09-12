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

import { PostureBadge } from "./PostureBadge";
import { validationClient, Severity } from "../../api/validation";

const mockValidate = vi.mocked(validationClient.validateScenario);

describe("PostureBadge", () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders a failed status and the top ERROR/WARNING findings", async () => {
    mockValidate.mockResolvedValue(
      create(ValidateScenarioResponseSchema, {
        scenario: "demo",
        status: ValidationStatus.FAILED,
        assessment: create(MaturityAssessmentSchema, {
          scenario: "demo",
          provider: "security-health",
          phase: "security",
          version: "test",
          findingsBySeverity: { SEVERITY_ERROR: 1, SEVERITY_INFO: 2 },
          findings: [
            create(AssessmentFindingSchema, {
              code: "gitleaks.x",
              severity: Severity.ERROR,
              title: "Leaked secret",
            }),
            create(AssessmentFindingSchema, {
              code: "info.y",
              severity: Severity.INFO,
              title: "Informational",
            }),
          ],
        }),
      }),
    );
    renderWithProviders(<PostureBadge scenario="demo" />);

    await waitFor(() => expect(screen.getByTestId(selectors.widget.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.widget.status)).toHaveAttribute("data-passed", "false");

    const top = screen.getByTestId(selectors.widget.topFindings);
    expect(within(top).getByText("Leaked secret")).toBeInTheDocument();
    // INFO findings are not surfaced in the badge rollup.
    expect(within(top).queryByText("Informational")).not.toBeInTheDocument();
  });

  it("renders a clean status with no top-findings list", async () => {
    mockValidate.mockResolvedValue(
      create(ValidateScenarioResponseSchema, {
        scenario: "demo",
        status: ValidationStatus.PASSED,
        assessment: create(MaturityAssessmentSchema, {
          scenario: "demo",
          provider: "security-health",
          phase: "security",
          version: "test",
        }),
      }),
    );
    renderWithProviders(<PostureBadge scenario="demo" />);

    await waitFor(() => expect(screen.getByTestId(selectors.widget.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.widget.status)).toHaveAttribute("data-passed", "true");
    expect(screen.queryByTestId(selectors.widget.topFindings)).not.toBeInTheDocument();
  });
});
