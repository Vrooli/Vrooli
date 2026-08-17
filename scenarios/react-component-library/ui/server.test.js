import { describe, expect, it } from "vitest";

import {
  documentForAssetDetail,
  documentForAssetPreview,
  isAssetDetailRoute,
  isAssetPreviewRoute,
  shouldProxyToApi,
} from "./server.js";

describe("ui server route proxying", () => {
  it("forwards preview harness and runtime routes to the API before SPA fallback", () => {
    expect(shouldProxyToApi("/preview/cmp-7/harness.html")).toBe(true);
    expect(shouldProxyToApi("/preview/runtime/react@18.3.1/index.js")).toBe(true);
    expect(shouldProxyToApi("/preview/runtime/npm/lucide-react@0.424.0/index.js")).toBe(true);
  });

  it("keeps Connect-RPC forwarding without proxying normal SPA routes", () => {
    expect(
      shouldProxyToApi(
        "/vrooli.react_component_library.v1.components.ComponentsService/ListComponents",
      ),
    ).toBe(true);
    expect(shouldProxyToApi("/components")).toBe(false);
    expect(shouldProxyToApi("/previewing")).toBe(false);
    expect(shouldProxyToApi("/assets/index.js")).toBe(false);
  });

  it("claims catalog details without intercepting emitted static files", () => {
    expect(isAssetDetailRoute("/assets/cmp-7")).toBe(true);
    expect(isAssetDetailRoute("/assets/react-component-library%3AuseFocusTrap")).toBe(true);
    expect(isAssetDetailRoute("/assets/index.js")).toBe(false);
    expect(isAssetDetailRoute("/assets/cmp-7/files/source.ts")).toBe(false);
  });

  it("claims the nested preview popout route", () => {
    expect(isAssetPreviewRoute("/assets/cmp-7/preview")).toBe(true);
    expect(isAssetPreviewRoute("/assets/cmp-7")).toBe(false);
    expect(isAssetPreviewRoute("/assets/cmp-7/preview/index.js")).toBe(false);
  });

  it("makes relative build assets resolve at the application root for a direct detail load", () => {
    expect(documentForAssetDetail("<html><head></head><body></body></html>")).toContain(
      '<head><base href="../">',
    );
    const directRoute = "http://localhost:21242/assets/component-id?tab=tests";
    expect(new URL("./assets/app.js", directRoute).pathname).toBe("/assets/assets/app.js");
    expect(new URL("./assets/app.js", new URL("../", directRoute)).pathname).toBe("/assets/app.js");
    expect(documentForAssetPreview("<html><head></head><body></body></html>")).toContain(
      '<head><base href="../../">',
    );
  });
});
