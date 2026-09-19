/**
 * App tests — the auth gate.
 *
 * `App` shows the first-run OnboardingScreen until this browser holds a device
 * token, then the routed shell. We render `<App />` directly (it owns its own providers); the
 * gate reads the session that `SessionProvider` initialises from localStorage,
 * so seeding localStorage before mount picks the branch. `<App>` mounts
 * `createBrowserRouter`, which is fine in jsdom for a smoke assertion.
 */
import { afterEach, describe, expect, it } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

import App from "./App";
import { makeDevice, seedSession } from "./test-utils";
import { selectors } from "./consts/selectors";
import { TrustState } from "@vrooli/proto-types/device-sync-hub/v1/devices/devices_pb";

// App provides theme/session/realtime but (like production) relies on the
// QueryClient mounted above it in main.tsx. Supply one here so the realtime
// hook and the transfer query have a client; retry:false keeps a doomed
// network read from retrying in jsdom.
const renderApp = () => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <App />
    </QueryClientProvider>,
  );
};

describe("App auth gate", () => {
  afterEach(() => {
    cleanup();
  });

  it("shows the first-run onboarding screen when this browser is not paired", () => {
    renderApp();
    expect(screen.getByTestId(selectors.onboarding.screen)).toBeInTheDocument();
  });

  it("shows the paired shell once a device token is present", () => {
    seedSession();
    renderApp();
    expect(screen.getByTestId(selectors.app.title)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.transfer)).toBeInTheDocument();
  });

  it("keeps a pending device in onboarding instead of attempting trusted transfers", () => {
    seedSession({
      deviceToken: "pending-device-token",
      device: makeDevice({ trustState: TrustState.PENDING }),
      ownerToken: "owner-jwt",
    });

    renderApp();

    expect(screen.getByTestId(selectors.onboarding.screen)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.join.waiting)).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.pages.transfer)).not.toBeInTheDocument();
  });
});
