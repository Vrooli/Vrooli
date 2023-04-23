import { jsx as _jsx } from "react/jsx-runtime";
import { ApolloProvider } from "@apollo/client";
import { initializeApollo } from "api/utils/initialize";
import { ErrorBoundary } from "components/ErrorBoundary/ErrorBoundary";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import "./i18n";
import reportWebVitals from "./reportWebVitals";
import * as serviceWorkerRegistration from "./serviceWorkerRegistration";
import { getDeviceInfo } from "./utils/display/device";
import { PubSub } from "./utils/pubsub";
import { Router } from "./utils/route";
export function renderApp(element) {
    const client = initializeApollo();
    const rootElement = element || document.getElementById("root");
    const root = ReactDOM.createRoot(rootElement);
    root.render(_jsx(Router, { children: _jsx(ApolloProvider, { client: client, children: _jsx(ErrorBoundary, { children: _jsx(App, {}) }) }) }));
}
if (typeof window !== "undefined") {
    renderApp();
    serviceWorkerRegistration.register({
        onUpdate: (registration) => {
            if (registration && registration.waiting) {
                registration.waiting.postMessage({ type: "SKIP_WAITING" });
                PubSub.get().publishAlertDialog({
                    message: "NewVersionAvailable",
                    buttons: [
                        { labelKey: "Reload", onClick: () => { window.location.reload(); } },
                        { labelKey: "Cancel", onClick: () => { } },
                    ]
                });
            }
        },
    });
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.ready.then((registration) => {
            registration.addEventListener("updatefound", () => {
                PubSub.get().publishSnack({ message: "Downloading new updates...", autoHideDuration: "persist" });
            });
            registration.active.postMessage({
                type: "IS_STANDALONE",
                isStandalone: getDeviceInfo().isStandalone,
            });
        });
    }
    reportWebVitals();
}
//# sourceMappingURL=index.js.map