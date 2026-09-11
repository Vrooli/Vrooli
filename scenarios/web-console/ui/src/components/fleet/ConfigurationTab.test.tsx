import { fireEvent, waitFor, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders as render } from "../../test-utils";
import type { Machine } from "../../api/machines";

const apiMocks = vi.hoisted(() => ({
  getConfiguration: vi.fn(),
  listCredentialGrants: vi.fn(),
}));

vi.mock("../../api/machines", async () => {
  const actual = await vi.importActual<typeof import("../../api/machines")>("../../api/machines");
  return {
    ...actual,
    getConfiguration: apiMocks.getConfiguration,
    listCredentialGrants: apiMocks.listCredentialGrants,
  };
});

import { ConfigurationTab } from "./ConfigurationTab";

function machine(): Machine {
  return {
    target: {
      id: "minimouse",
      kind: "bridge-node",
      label: "minimouse",
      available: true,
    },
    grant: { summary: "Read terminal", effects: ["read"], appCount: 1, coversAllApps: false, scopes: [], preset: "read" },
    heartbeatAgeSeconds: 4,
    manageable: true,
    drift: [],
  };
}

describe("ConfigurationTab", () => {
  it("explains that a backend route failure may not be repaired by re-apply", async () => {
    apiMocks.getConfiguration.mockRejectedValueOnce(new Error("502 Bad Gateway: ListOperatorInputs returned 404"));
    apiMocks.listCredentialGrants.mockResolvedValueOnce({ grants: [] });

    render(<ConfigurationTab machine={machine()} />);

    await waitFor(() => {
      expect(screen.getByText(/The Bridge path could not retrieve configuration questions/)).toBeInTheDocument();
    });
    expect(screen.getByText(/re-apply may not fix a missing backend route/)).toBeInTheDocument();
  });

  it("labels a missing target procedure as an onboarding API incompatibility", async () => {
    apiMocks.getConfiguration.mockRejectedValueOnce(new Error("target_onboarding_incompatible: the target onboarding API is missing a required procedure"));
    apiMocks.listCredentialGrants.mockResolvedValueOnce({ grants: [] });

    render(<ConfigurationTab machine={machine()} />);

    await waitFor(() => {
      expect(screen.getByText("machines.configIncompatibleTitle")).toBeInTheDocument();
    });
    expect(screen.getByText("machines.configIncompatibleBody")).toBeInTheDocument();
  });

  it("offers a safe refresh before the consequential re-apply action", async () => {
    apiMocks.getConfiguration
      .mockRejectedValueOnce(new Error("502 Bad Gateway: ListOperatorInputs returned 404"))
      .mockResolvedValueOnce({ questions: [], readiness: null, detail: null });
    apiMocks.listCredentialGrants.mockResolvedValue({ grants: [] });

    render(<ConfigurationTab machine={machine()} />);

    await waitFor(() => {
      expect(screen.getByTestId("machine-configuration-refresh")).toBeInTheDocument();
    });
    apiMocks.getConfiguration.mockClear();
    fireEvent.click(screen.getByTestId("machine-configuration-refresh"));

    await waitFor(() => {
      expect(screen.queryByTestId("machine-configuration-refresh")).not.toBeInTheDocument();
    });
    expect(apiMocks.getConfiguration).toHaveBeenCalledTimes(1);
  });
});
