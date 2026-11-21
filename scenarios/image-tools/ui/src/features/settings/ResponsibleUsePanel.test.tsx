/**
 * ResponsibleUsePanel tests — the read-only Responsible-Use settings panel.
 * The SafetyService policy fetch is mocked, so the panel's loading/error
 * states, tier badge, enforced-control rows, op-weight table, and summary are
 * exercised in isolation.
 */
import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  ConsentWeight,
  DeploymentTier,
  SafetyPolicySchema,
  type SafetyPolicy,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const mocks = vi.hoisted(() => ({ getPolicy: vi.fn() }));
vi.mock("../../api/safety", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/safety")>();
  return { ...actual, getPolicy: mocks.getPolicy };
});

import { ResponsibleUsePanel } from "./ResponsibleUsePanel";

const makePolicy = (
  overrides: MessageInitShape<typeof SafetyPolicySchema> = {},
): SafetyPolicy =>
  create(SafetyPolicySchema, {
    tier: DeploymentTier.PUBLIC,
    requireConsent: true,
    forceNsfwScan: true,
    requireProvenance: false,
    rateLimitPerMin: 30,
    summary: "Public deployment — consent gated.",
    opWeights: [
      { operation: "text_to_image", weight: ConsentWeight.NONE },
      { operation: "edit_instruct", weight: ConsentWeight.HIGH },
    ],
    ...overrides,
  });

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("ResponsibleUsePanel", () => {
  it("shows a loading state before the policy resolves", () => {
    mocks.getPolicy.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<ResponsibleUsePanel />);
    expect(screen.getByTestId(selectors.responsibleUse.loading)).toBeInTheDocument();
  });

  it("renders the tier, controls, op-weight table, and summary on success", async () => {
    mocks.getPolicy.mockResolvedValue(makePolicy());
    renderWithProviders(<ResponsibleUsePanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.responsibleUse.tier)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.responsibleUse.tier)).toHaveTextContent(
      strings.settings.responsibleUse.tier.public,
    );

    const table = screen.getByTestId(selectors.responsibleUse.weights);
    expect(table).toHaveTextContent("edit_instruct");
    expect(table).toHaveTextContent("text_to_image");

    expect(screen.getByTestId(selectors.responsibleUse.summary)).toHaveTextContent(
      "Public deployment — consent gated.",
    );
  });

  it("falls back to the empty message when no op weights are configured", async () => {
    mocks.getPolicy.mockResolvedValue(makePolicy({ opWeights: [] }));
    renderWithProviders(<ResponsibleUsePanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.responsibleUse.tier)).toBeInTheDocument();
    });
    expect(screen.queryByTestId(selectors.responsibleUse.weights)).not.toBeInTheDocument();
    expect(screen.getByText(strings.settings.responsibleUse.empty)).toBeInTheDocument();
  });

  it("shows the local tier with consent off", async () => {
    mocks.getPolicy.mockResolvedValue(
      makePolicy({ tier: DeploymentTier.LOCAL, requireConsent: false, rateLimitPerMin: 0 }),
    );
    renderWithProviders(<ResponsibleUsePanel />);

    await waitFor(() => {
      expect(screen.getByTestId(selectors.responsibleUse.tier)).toHaveTextContent(
        strings.settings.responsibleUse.tier.local,
      );
    });
    expect(screen.getByText(strings.settings.responsibleUse.rateLimitNone)).toBeInTheDocument();
  });

  it("renders an error state when the fetch fails", async () => {
    mocks.getPolicy.mockRejectedValue(new Error("boom"));
    renderWithProviders(<ResponsibleUsePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.responsibleUse.error)).toBeInTheDocument();
    });
  });

  it("has no accessibility violations", async () => {
    mocks.getPolicy.mockResolvedValue(makePolicy());
    const { container } = renderWithProviders(<ResponsibleUsePanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.responsibleUse.tier)).toBeInTheDocument();
    });
    await expectNoA11yViolations(container);
  });
});
