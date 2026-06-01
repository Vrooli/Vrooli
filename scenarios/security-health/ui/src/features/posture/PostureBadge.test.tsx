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
        passed: false,
        summary: create(SummarySchema, { errors: 1, warnings: 0, infos: 2 }),
        findings: [
          create(FindingSchema, { ruleId: "gitleaks.x", severity: Severity.ERROR, title: "Leaked secret", scanner: "gitleaks" }),
          create(FindingSchema, { ruleId: "info.y", severity: Severity.INFO, title: "Informational", scanner: "osv-scanner" }),
        ],
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
        passed: true,
        summary: create(SummarySchema, { errors: 0, warnings: 0, infos: 0 }),
        findings: [],
      }),
    );
    renderWithProviders(<PostureBadge scenario="demo" />);

    await waitFor(() => expect(screen.getByTestId(selectors.widget.status)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.widget.status)).toHaveAttribute("data-passed", "true");
    expect(screen.queryByTestId(selectors.widget.topFindings)).not.toBeInTheDocument();
  });
});
