import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { previewRoute } from "../api/gateway";
import { selectors } from "../consts/selectors";
import { renderWithProviders } from "../test-utils";
import { RoutePreviewPage } from "./RoutePreviewPage";

vi.mock("../api/gateway", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/gateway")>();
  const { makeGatewayApiMocks } = await import("../test-utils/mocks/gateway");
  return { ...actual, ...makeGatewayApiMocks() };
});

describe("RoutePreviewPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("[REQ:AIGW-UI-DASHBOARD] submits the preview form and renders candidate routes", async () => {
    const user = userEvent.setup();
    renderWithProviders(<RoutePreviewPage />);

    await user.clear(screen.getByTestId(selectors.routePreview.scenarioInput));
    await user.type(screen.getByTestId(selectors.routePreview.scenarioInput), "prompt-injection-arena");
    await user.click(screen.getByTestId(selectors.routePreview.submit));

    expect(vi.mocked(previewRoute).mock.calls[0]?.[0]).toEqual(
      expect.objectContaining({ scenario: "prompt-injection-arena" }),
    );
    expect(await screen.findByTestId(selectors.routePreview.result)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.routePreview.candidates)).toBeInTheDocument();
  });

  it("renders preview errors and preserves the form", async () => {
    vi.mocked(previewRoute).mockRejectedValueOnce(new Error("profile blocked"));
    const user = userEvent.setup();
    renderWithProviders(<RoutePreviewPage />);

    await user.click(screen.getByTestId(selectors.routePreview.submit));

    expect(await screen.findByTestId(selectors.routePreview.error)).toHaveTextContent("profile blocked");
    expect(screen.getByTestId(selectors.routePreview.form)).toBeInTheDocument();
  });
});
