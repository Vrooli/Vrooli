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
const refresh = vi.fn();
const integrations = vi.hoisted(() => ({ list: [] as unknown[] }));

vi.mock("./useIntegrationStatus", () => ({
  useIntegrationStatus: () => ({
    status: {
      override: BehaviorOverride.AUTO,
      integrations: integrations.list,
      evaluatedAt: "2026-10-29T05:00:00Z",
    },
    loading: false,
    error: "",
    refresh,
    setOverride,
  }),
}));

function renderPanel() {
  return render(
    <I18nextProvider i18n={i18n}>
      <IntegrationsPanel />
    </I18nextProvider>,
  );
}

describe("IntegrationsPanel deployment profile setup", () => {
  beforeEach(() => {
    localStorage.clear();
    setOverride.mockReset();
    integrations.list = [
      { id: "agent-manager", displayName: "agent-manager", state: IntegrationState.AVAILABLE, stats: {}, reason: "ready" },
    ];
  });

  it("shows the client profile ready and reports missing local control", () => {
    renderPanel();

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
    renderPanel();
    fireEvent.change(screen.getByTestId(selectors.integrations.deploymentEndpointInput), {
      target: { value: "https://user:secret@example.test" },
    });
    fireEvent.click(screen.getByTestId(selectors.integrations.deploymentProfileSave));
    expect(screen.getByText(strings.integrations.settings.profileInvalidEndpoint)).toBeInTheDocument();
    expect(localStorage.getItem("portal.deployment-profile.v1")).toBeNull();
  });
});

describe("IntegrationsPanel overview", () => {
  beforeEach(() => {
    setOverride.mockReset();
    refresh.mockReset();
    integrations.list = [
      { id: "openrouter", displayName: "OpenRouter", state: IntegrationState.AVAILABLE, stats: { latencyP95Ms: 410.7, sampleCount: 32n }, reason: "OpenRouter is available" },
      { id: "search-hub", displayName: "search-hub", state: IntegrationState.AVAILABLE, stats: {}, reason: "search-hub is available" },
      { id: "audio-tools", displayName: "audio-tools", state: IntegrationState.UNAVAILABLE, stats: {}, reason: "audio-tools is not running" },
      { id: "prompt-manager", displayName: "prompt-manager", state: IntegrationState.DEGRADED, stats: {}, reason: "slow" },
    ];
  });

  it("calls out what is off or failing, and counts only working parts as ready", () => {
    renderPanel();

    expect(screen.getByText("2 of 4 ready")).toBeInTheDocument();
    expect(screen.getByText(strings.integrations.settings.problemOff)).toBeInTheDocument();
    expect(screen.getByText(strings.integrations.settings.items.audioTools.problem)).toBeInTheDocument();
    expect(screen.getByText(strings.integrations.settings.problemLimited)).toBeInTheDocument();
    // Portal cannot start other apps, so no fix button is offered.
    expect(screen.queryByRole("button", { name: /start/i })).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /Check again/ }));
    expect(refresh).toHaveBeenCalledOnce();
  });

  it("keeps the search mode control on the Search detail view", () => {
    renderPanel();
    expect(screen.queryByTestId(selectors.integrations.overrideSelect)).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: new RegExp(strings.integrations.settings.items.searchHub.name) }));
    fireEvent.change(screen.getByTestId(selectors.integrations.overrideSelect), {
      target: { value: String(BehaviorOverride.FORCE_OFF) },
    });
    expect(setOverride).toHaveBeenCalledWith(BehaviorOverride.FORCE_OFF);
    // The connection settings belong to the overview, not to one integration's detail.
    expect(screen.queryByTestId(selectors.integrations.deploymentProfilePanel)).not.toBeInTheDocument();
  });

  it("shows the start command for an app that is off, but not for OpenRouter", () => {
    renderPanel();
    fireEvent.click(screen.getByRole("button", { name: new RegExp(strings.integrations.settings.items.audioTools.name) }));
    expect(screen.getByText("vrooli scenario start audio-tools")).toBeInTheDocument();
  });
});
