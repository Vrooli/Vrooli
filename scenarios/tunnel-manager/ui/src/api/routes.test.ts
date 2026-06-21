import { describe, expect, it, vi } from "vitest";

import { createExternalRoute, deleteRoute, listRoutes, RouteSource, routesClient } from "./routes";
import { makeExternalRoute, makeRoute } from "../test-utils/mocks/routes";

describe("routes API helpers", () => {
  it("returns the route list from the generated client", async () => {
    const routes = [makeRoute(), makeExternalRoute()];
    const spy = vi.spyOn(routesClient, "listRoutes").mockResolvedValueOnce({ routes } as never);

    await expect(listRoutes()).resolves.toBe(routes);
    expect(spy).toHaveBeenCalledWith({ tier: 0 });
  });

  it("creates an external route with EXTERNAL source and a service target", async () => {
    const route = makeExternalRoute();
    const spy = vi.spyOn(routesClient, "createRoute").mockResolvedValueOnce({ route } as never);

    await createExternalRoute({ subdomain: "billing", serviceTarget: "http://127.0.0.1:7000" });
    expect(spy).toHaveBeenCalledWith({
      subdomain: "billing",
      serviceTarget: "http://127.0.0.1:7000",
      domain: "",
      source: RouteSource.EXTERNAL,
    });
  });

  it("passes an explicit domain through when given", async () => {
    const route = makeExternalRoute();
    const spy = vi.spyOn(routesClient, "createRoute").mockResolvedValueOnce({ route } as never);

    await createExternalRoute({ subdomain: "billing", serviceTarget: "http://127.0.0.1:7000", domain: "example.com" });
    expect(spy).toHaveBeenCalledWith({
      subdomain: "billing",
      serviceTarget: "http://127.0.0.1:7000",
      domain: "example.com",
      source: RouteSource.EXTERNAL,
    });
  });

  it("deletes a route by id", async () => {
    const spy = vi.spyOn(routesClient, "deleteRoute").mockResolvedValueOnce({ deleted: true } as never);

    await deleteRoute("route-1");
    expect(spy).toHaveBeenCalledWith({ id: "route-1" });
  });
});
