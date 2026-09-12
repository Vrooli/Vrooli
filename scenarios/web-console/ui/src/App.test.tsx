import { fireEvent, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "./test-utils";
import App from "./App";
import { useWorkspaceStore } from "./stores/useWorkspaceStore";
import type { BannerAction, BannerDescriptor, MaybeBanner } from "./components/banners/types";

const { fetchHealth, useCapabilities } = vi.hoisted(() => ({
  fetchHealth: vi.fn(),
  useCapabilities: vi.fn(),
}));

vi.mock("./api/health", () => ({ fetchHealth }));
vi.mock("./hooks/useCapabilities", () => ({ useCapabilities }));
vi.mock("./consts/config", async (importActual) => ({
  ...(await importActual<typeof import("./consts/config")>()),
  HEALTH_RETRY_COUNT: 0,
  HEALTH_RETRY_DELAY_MS: 0,
}));
vi.mock("./components/Workspace", () => ({
  default: ({ appBanners }: { appBanners: MaybeBanner[] }) => (
    <div data-testid="workspace-mock">
      {appBanners.filter((banner): banner is BannerDescriptor => Boolean(banner)).map((banner) => (
        <div key={banner.id} data-testid={banner.testId}>
          <span>{banner.title}</span>
          {banner.actions?.map((action: BannerAction) => (
            <button key={action.id} onClick={action.onSelect}>{action.label}</button>
          ))}
          {banner.onDismiss && <button onClick={banner.onDismiss}>dismiss</button>}
        </div>
      ))}
    </div>
  ),
}));

describe("App bootstrap and top-level notices", () => {
  beforeEach(() => {
    fetchHealth.mockReset();
    useCapabilities.mockReset();
    useCapabilities.mockReturnValue({ data: { capabilities: [] }, error: null });
    // Intent is session-scoped; a leaked latch would make the
    // "stays quiet" case pass for the wrong reason.
    useWorkspaceStore.setState({ audioIntent: false });
  });

  it("shows the loading boundary before mounting the workspace", async () => {
    fetchHealth.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<App />);
    expect(screen.getByText("app.loading")).toBeInTheDocument();
  });

  it("stays quiet about unavailable audio until something asks for it", async () => {
    // The reader has not touched audio in this session. An optional
    // side-feature being down is not news on load — and because the
    // condition outlives a dismissal, a notice here could not be got
    // rid of, only replaced by the next notice for the same fact.
    fetchHealth.mockResolvedValue({ status: "ok" });
    useCapabilities.mockReturnValue({
      data: { capabilities: [{ id: "audio-tools", status: "unavailable", reasonCode: "scenario_stopped" }] },
      error: null,
    });

    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId("workspace-mock")).toBeInTheDocument());
    expect(screen.queryByTestId("audio-unavailable-banner")).not.toBeInTheDocument();
  });

  it("raises audio-unavailable once the session has reached for audio", async () => {
    fetchHealth.mockResolvedValue({ status: "ok" });
    useCapabilities.mockReturnValue({
      data: { capabilities: [{ id: "audio-tools", status: "unavailable", reasonCode: "scenario_stopped" }] },
      error: null,
    });
    // What the mic button, a speak action, or the voice settings tabs latch.
    useWorkspaceStore.setState({ audioIntent: true });

    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId("audio-unavailable-banner")).toBeInTheDocument());
    expect(screen.getByTestId("workspace-mock")).toBeInTheDocument();
  });

  it("offers retry and dismiss actions when the health query fails", async () => {
    const error = new Error("API unavailable");
    fetchHealth.mockRejectedValue(error);
    useCapabilities.mockReturnValue({ data: { capabilities: [] }, error: error });
    // The subject here is that dismissing one notice leaves the
    // others standing, so this case needs a second notice to
    // survive — hence the intent latch.
    useWorkspaceStore.setState({ audioIntent: true });

    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId("connection-banner")).toBeInTheDocument(), { timeout: 5000 });
    fireEvent.click(screen.getByRole("button", { name: "dismiss" }));
    expect(screen.queryByTestId("connection-banner")).not.toBeInTheDocument();
    expect(screen.getByTestId("audio-unavailable-banner")).toBeInTheDocument();
  });

  it("retries the health query from the connection banner", async () => {
    const error = new Error("temporary outage");
    fetchHealth.mockRejectedValueOnce(error).mockResolvedValueOnce({ status: "ok" });
    renderWithProviders(<App />);
    await waitFor(() => expect(screen.getByTestId("connection-banner")).toBeInTheDocument(), { timeout: 5000 });
    fireEvent.click(screen.getByRole("button", { name: "app.connectionBanner.retry" }));
    await waitFor(() => expect(fetchHealth).toHaveBeenCalledTimes(2), { timeout: 5000 });
  });
});
