// provider-free-exception: The test uses a provider-free or feature-specific harness to isolate its boundary.
import { fireEvent, render, screen } from "@testing-library/react";
import { I18nextProvider } from "react-i18next";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BehaviorOverride, IntegrationState } from "../../api/integrations";
import { i18n } from "../../i18n";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { IntegrationsPanel } from "./IntegrationsPanel";

const setOverride = vi.fn();

vi.mock("./useIntegrationStatus", () => ({
  useIntegrationStatus: () => ({
    status: {
      override: BehaviorOverride.AUTO,
      integrations: [
        { id: "agent-manager", displayName: "agent-manager", state: IntegrationState.AVAILABLE, stats: {}, reason: "ready" },
      ],
    },
    loading: false,
    error: "",
    setOverride,
  }),
}));

describe("IntegrationsPanel deployment profile setup", () => {
  beforeEach(() => {
    localStorage.clear();
    setOverride.mockReset();
  });

  it("shows the client profile ready and reports missing local control", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IntegrationsPanel />
      </I18nextProvider>,
    );

    expect(screen.getByTestId(selectors.integrations.deploymentProfilePanel)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.integrations.deploymentProfileStatus)).toHaveTextContent(
      "integrations.settings.profileReady",
    );

    fireEvent.change(screen.getByTestId(selectors.integrations.deploymentProfileSelect), {
      target: { value: "local-control" },
    });
    expect(screen.getByTestId(selectors.integrations.deploymentProfileStatus)).toHaveTextContent(
      "integrations.settings.missingCapabilities",
    );
  });

  it("refuses an endpoint containing credentials", () => {
    render(
      <I18nextProvider i18n={i18n}>
        <IntegrationsPanel />
      </I18nextProvider>,
    );
    fireEvent.change(screen.getByTestId(selectors.integrations.deploymentEndpointInput), {
      target: { value: "https://user:secret@example.test" },
    });
    fireEvent.click(screen.getByTestId(selectors.integrations.deploymentProfileSave));
    expect(screen.getByText(strings.integrations.settings.profileInvalidEndpoint)).toBeInTheDocument();
    expect(localStorage.getItem("portal.deployment-profile.v1")).toBeNull();
  });
});
