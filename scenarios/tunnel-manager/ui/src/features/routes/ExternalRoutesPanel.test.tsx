/**
 * ExternalRoutesPanel tests — lists only external routes, adds one (subdomain +
 * target URL → RouteSource.EXTERNAL), and deletes one. Scenario routes are
 * filtered out; the source badge distinguishes the two.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RouteSource } from "@vrooli/proto-types/tunnel-manager/v1/routes/routes_pb";

import { renderWithProviders } from "../../test-utils";
import { makeRoutesMocks, makeRoute, makeExternalRoute } from "../../test-utils/mocks/routes";

vi.mock("../../api/routes", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/routes")>();
  return { ...actual, ...makeRoutesMocks() };
});

import { ExternalRoutesPanel } from "./ExternalRoutesPanel";
import { selectors } from "../../consts/selectors";
import { setLocale } from "../../i18n";

describe("ExternalRoutesPanel", () => {
  beforeEach(async () => {
    await setLocale("en");
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it("lists only external routes with their source badge and target", async () => {
    renderWithProviders(<ExternalRoutesPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.routes.table)).toBeInTheDocument();
    });
    // Default mock has one scenario + one external route — only the external row shows.
    expect(screen.getAllByTestId(selectors.routes.row)).toHaveLength(1);
    expect(screen.getByTestId(selectors.routes.sourceBadge)).toHaveTextContent("External");
    expect(screen.getByTestId(selectors.routes.row)).toHaveTextContent("http://127.0.0.1:9000");
  });

  it("shows the empty state when there are no external routes", async () => {
    const { routesClient } = await import("../../api/routes");
    vi.mocked(routesClient.listRoutes).mockResolvedValueOnce({ routes: [makeRoute()] } as never);

    renderWithProviders(<ExternalRoutesPanel />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.queryState.empty)).toHaveTextContent("No external routes yet");
    });
  });

  it("creates an external route with EXTERNAL source from the form", async () => {
    const { routesClient } = await import("../../api/routes");
    const user = userEvent.setup();
    renderWithProviders(<ExternalRoutesPanel />);

    await user.type(await screen.findByTestId(selectors.routes.subdomainInput), "billing");
    await user.type(screen.getByTestId(selectors.routes.targetInput), "http://127.0.0.1:7000");
    await user.click(screen.getByTestId(selectors.routes.addButton));

    await waitFor(() => {
      expect(routesClient.createRoute).toHaveBeenCalledWith({
        subdomain: "billing",
        serviceTarget: "http://127.0.0.1:7000",
        domain: "",
        source: RouteSource.EXTERNAL,
      });
    });
  });

  it("keeps the add button disabled until subdomain and target are filled", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExternalRoutesPanel />);

    const addButton = await screen.findByTestId(selectors.routes.addButton);
    expect(addButton).toBeDisabled();

    await user.type(screen.getByTestId(selectors.routes.subdomainInput), "billing");
    expect(addButton).toBeDisabled();

    await user.type(screen.getByTestId(selectors.routes.targetInput), "http://127.0.0.1:7000");
    expect(addButton).toBeEnabled();
  });

  it("deletes an external route by id", async () => {
    const { routesClient } = await import("../../api/routes");
    const user = userEvent.setup();
    renderWithProviders(<ExternalRoutesPanel />);

    await waitFor(() => expect(screen.getByTestId(selectors.routes.table)).toBeInTheDocument());
    await user.click(screen.getByTestId(selectors.routes.deleteButton));

    await waitFor(() => {
      expect(routesClient.deleteRoute).toHaveBeenCalledWith({ id: makeExternalRoute().id });
    });
  });

  it("surfaces an add error when createRoute rejects", async () => {
    const { routesClient } = await import("../../api/routes");
    vi.mocked(routesClient.createRoute).mockRejectedValueOnce(new Error("nope"));
    const user = userEvent.setup();
    renderWithProviders(<ExternalRoutesPanel />);

    await user.type(await screen.findByTestId(selectors.routes.subdomainInput), "billing");
    await user.type(screen.getByTestId(selectors.routes.targetInput), "http://127.0.0.1:7000");
    await user.click(screen.getByTestId(selectors.routes.addButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.routes.addError)).toBeInTheDocument();
    });
  });
});
