import { cleanup, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { renderWithProviders as render } from "../test-utils";
import { API_BASE } from "../api/base";
import { getActiveAudioToolsClient, setActiveAudioToolsClientForTesting, useAudioToolsClient } from "../audio-integration";
import AudioIntegrationRoute from "./AudioIntegrationRoute";

function ClientLabel() {
  const client = useAudioToolsClient();
  return <output data-testid="client-base-url">{client.baseUrl}</output>;
}

afterEach(() => {
  cleanup();
  setActiveAudioToolsClientForTesting(null);
});

describe("AudioIntegrationRoute", () => {
  it("provides its configured client to route content and registers that client for API helpers", () => {
    render(
      <AudioIntegrationRoute>
        <ClientLabel />
      </AudioIntegrationRoute>,
    );

    expect(screen.getByTestId("client-base-url").textContent).toBe(API_BASE);
    expect(getActiveAudioToolsClient().baseUrl).toBe(API_BASE);
  });

  it("keeps the route client registered when its child content updates", () => {
    const view = render(
      <AudioIntegrationRoute>
        <ClientLabel />
      </AudioIntegrationRoute>,
    );
    const registeredClient = getActiveAudioToolsClient();

    view.rerender(
      <AudioIntegrationRoute>
        <><ClientLabel /><span>updated route content</span></>
      </AudioIntegrationRoute>,
    );

    expect(screen.getByText(/updated route content/)).toBeTruthy();
    expect(getActiveAudioToolsClient()).toBe(registeredClient);
  });
});
