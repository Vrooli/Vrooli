import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";
import { ValidationStatus } from "../../api/storage";
import { makeFinding, makeValidateResponse } from "../storage/mocks/factories";

const { validateScenario } = vi.hoisted(() => ({ validateScenario: vi.fn() }));

vi.mock("../../api/storage", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/storage")>();
  return { ...actual, storageClient: { ...actual.storageClient, validateScenario } };
});

import { ValidateView } from "./ValidateView";

describe("ValidateView accessibility", () => {
  beforeEach(async () => {
    await setLocale("en");
  });
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("renders the findings result without axe violations", async () => {
    validateScenario.mockResolvedValue(
      makeValidateResponse({
        status: ValidationStatus.FAILED,
        findings: [makeFinding({ autofixAvailable: true })],
        findingsBySeverity: { SEVERITY_ERROR: 1 },
      }),
    );
    const { container } = renderWithProviders(<ValidateView />, {
      routerEntries: ["/validate?scenario=demo"],
    });
    await waitFor(() => expect(screen.getByTestId(selectors.validate.findingsList)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });

  it("renders the clean state without axe violations", async () => {
    validateScenario.mockResolvedValue(makeValidateResponse({ status: ValidationStatus.PASSED }));
    const { container } = renderWithProviders(<ValidateView />, {
      routerEntries: ["/validate?scenario=ok"],
    });
    await waitFor(() => expect(screen.getByTestId(selectors.validate.clean)).toBeInTheDocument());
    await expectNoA11yViolations(container);
  });
});
