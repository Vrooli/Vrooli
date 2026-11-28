import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import {
  ValidateTargetResponseSchema,
  ValidationStatus,
} from "@vrooli/proto-types/scenario-validation/v1/validation_pb";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { setLocale } from "../i18n";

vi.mock("../api/validation", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/validation")>();
  return { ...actual, validationClient: { validateTarget: vi.fn() } };
});

import { validationClient } from "../api/validation";
import { ValidationPage } from "./ValidationPage";

describe("ValidationPage", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("validates a selected target kind and renders the shared result", async () => {
    vi.mocked(validationClient.validateTarget).mockResolvedValue(
      create(ValidateTargetResponseSchema, { status: ValidationStatus.PASSED }),
    );
    const user = userEvent.setup();
    renderWithProviders(<ValidationPage />);

    await user.type(screen.getByTestId(selectors.validation.id), "structure-health");
    await user.click(screen.getByTestId(selectors.validation.submit));

    await waitFor(() => expect(screen.getByTestId(selectors.validation.result)).toBeInTheDocument());
    expect(validationClient.validateTarget).toHaveBeenCalledWith(
      expect.objectContaining({ id: "structure-health" }),
    );
  });
});
