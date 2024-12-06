/* eslint-disable no-undef */
import { ErrorBoundary } from "components/ErrorBoundary/ErrorBoundary";
import ReactDOM from "react-dom/client";
import { Router } from "route";
import { App } from "./App";
import "./i18n"; // Must import for translations to work
import * as serviceWorkerRegistration from "./serviceWorkerRegistration";
import { getDeviceInfo } from "./utils/display/device";
import { PubSub } from "./utils/pubsub";

/**
 * Used for finding excessive component re-renders. 
 * See https://react-scan.million.dev/ for more information.
 */
const USE_REACT_SCAN = false && process.env.NODE_ENV === "development";

// eslint-disable-next-line no-magic-numbers
const HOURS_1_MS = 60 * 60 * 1000;

const root = ReactDOM.createRoot(document.getElementById("root"));

if (USE_REACT_SCAN) {
    const script = document.createElement("script");
    script.src = "https://unpkg.com/react-scan/dist/auto.global.js";
    document.head.appendChild(script);
}

root.render(
    <Router>
        <ErrorBoundary>
            <App />
        </ErrorBoundary>
    </Router>,
);

// Enable service worker in production for PWA and offline support
if (process.env.PROD) {
    serviceWorkerRegistration.register({
        onUpdate: (registration) => {
            console.log("onUpdate", registration);
            if (registration && registration.waiting) {
                registration.waiting.postMessage({ type: "SKIP_WAITING" });
                PubSub.get().publish("snack", {
                    autoHideDuration: "persist",
                    id: "pwa-update",
                    message: "New version available!",
                    buttonKey: "Reload",
                    buttonClicked: function updateVersionButtonClicked() {
                        window.location.reload();
                    },
                });
            }
        },
    });
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.ready.then((registration) => {
            // Check for updates periodically
            setInterval(() => { registration.update(); }, HOURS_1_MS);

            // Listen for updatefound event
            registration.addEventListener("updatefound", () => {
                console.log("New service worker found", registration.installing, registration);
                const newWorker = registration.installing;

                function handleUpdateState() {
                    console.log("in handleUpdateState", newWorker.state);
                    if (newWorker.state === "installing") {
                        PubSub.get().publish("snack", {
                            autoHideDuration: "persist",
                            id: "pwa-update",
                            message: "Downloading updates...",
                        });
                    } else if (newWorker.state === "activated") {
                        PubSub.get().publish("snack", {
                            autoHideDuration: "persist",
                            id: "pwa-update",
                            message: "New version available!",
                            buttonKey: "Reload",
                            buttonClicked: function updateVersionButtonClicked() {
                                window.location.reload();
                            },
                        });
                    }
                }

                newWorker.addEventListener("statechange", () => {
                    console.log("in statechange", newWorker.state);
                    handleUpdateState();
                });
                handleUpdateState();
            });

            // Listen for controlling change
            navigator.serviceWorker.addEventListener("controllerchange", () => {
                if (!refreshing) {
                    refreshing = true;
                    window.location.reload();
                }
            });

            // Send message about standalone status
            registration.active.postMessage({
                type: "IS_STANDALONE",
                isStandalone: getDeviceInfo().isStandalone,
            });
        });
    }
} else {
    serviceWorkerRegistration.unregister();
}

// // Measure performance with Google Analytics. 
// // See results at https://analytics.google.com/
// ReactGA.initialize(process.env.VITE_GOOGLE_TRACKING_ID);
// const sendToAnalytics = ({ name, delta, id }) => {
//     console.log("sendToAnalytics", { name, delta, id }, process.env.VITE_GOOGLE_TRACKING_ID);
//     ReactGA.event({
//         category: "Web Vitals",
//         action: name,
//         value: Math.round(name === "CLS" ? delta * 1000 : delta), // CLS is reported as a fraction, so multiply by 1000 to make it more readable
//         label: id,
//         nonInteraction: true,
//     });
// };
// reportWebVitals(sendToAnalytics);
