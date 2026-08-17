import { pathToFileURL } from "node:url";
import path from "node:path";
import fs from "node:fs";
import { proxyToApi, startScenarioServer } from "@vrooli/api-base/server";

const connectRpcPath = /^\/vrooli\.react_component_library\.v1\./;
const previewPath = /^\/preview(?:\/|$)/;

export function shouldProxyToApi(path) {
  return connectRpcPath.test(path) || previewPath.test(path);
}

export function isAssetDetailRoute(routePath) {
  const match = /^\/assets\/([^/]+)$/.exec(routePath);
  return Boolean(match && !path.extname(match[1]));
}

export function isAssetPreviewRoute(routePath) {
  return /^\/assets\/[^/]+\/preview$/.test(routePath);
}

// Vite deliberately emits relative URLs (`./assets/...`) so the app also
// works inside a tunnel or an iframe sub-path. A detail page is one segment
// below the app root, however, so those URLs would otherwise resolve to
// `/assets/assets/...`. Giving only nested SPA documents a relative parent
// base preserves both deployment modes without hard-coding a host path.
export function documentForAssetDetail(indexDocument) {
  return indexDocument.replace("<head>", '<head><base href="../">');
}

export function documentForAssetPreview(indexDocument) {
  return indexDocument.replace("<head>", '<head><base href="../../">');
}

export function startReactComponentLibraryServer() {
  return startScenarioServer({
    uiPort: process.env.UI_PORT,
    apiPort: process.env.API_PORT,
    distDir: "./dist",
    serviceName: "react-component-library",
    corsOrigins: "*",
    setupRoutes(app) {
      app.get("/assets/:id/preview", (req, res, next) => {
        if (!isAssetPreviewRoute(req.path)) {
          next();
          return;
        }
        res
          .type("html")
          .send(
            documentForAssetPreview(
              fs.readFileSync(path.resolve(process.cwd(), "dist", "index.html"), "utf8"),
            ),
          );
      });
      // api-base correctly treats /assets/* as static asset requests to avoid
      // returning HTML for a missing JS/CSS file. The catalog deliberately
      // uses the same prefix for its SPA detail route, so claim only a
      // extensionless single segment before that generic safeguard runs.
      app.get("/assets/:id", (req, res, next) => {
        if (!isAssetDetailRoute(req.path)) {
          next();
          return;
        }
        res.type("html").send(
          documentForAssetDetail(
            // This is the tiny, route-specific part of the SPA document. Static
            // assets remain served by api-base, and their URLs stay relative.
            // Reading on request avoids stale HTML after lifecycle rebuilds.
            fs.readFileSync(path.resolve(process.cwd(), "dist", "index.html"), "utf8"),
          ),
        );
      });
      app.use((req, res, next) => {
        if (!shouldProxyToApi(req.path)) {
          next();
          return;
        }

        proxyToApi(req, res, req.originalUrl || req.url, {
          apiPort: process.env.API_PORT,
        }).catch(next);
      });
    },
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  startReactComponentLibraryServer();
}
