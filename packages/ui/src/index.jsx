import { ErrorBoundary } from "components/ErrorBoundary/ErrorBoundary";
import ReactDOM from "react-dom/client";
import { Router } from "route";
import { App } from "./App";
import "./i18n"; // Must import for translations to work
import * as serviceWorkerRegistration from "./serviceWorkerRegistration";
import { getDeviceInfo } from "./utils/display/device";
import { PubSub } from "./utils/pubsub";

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
    <Router>
        <ErrorBoundary>
            <App />
        </ErrorBoundary>
    </Router>,
);

// Enable service worker in production for PWA and offline support
if (import.meta.env.PROD) {
    serviceWorkerRegistration.register({
        onUpdate: (registration) => {
            if (registration && registration.waiting) {
                registration.waiting.postMessage({ type: "SKIP_WAITING" });
                if (window.confirm("New version available! The site will now update. Press \"Cancel\" if you need to save any unsaved data.")) {
                    window.location.reload();
                }
            }
        },
    });
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.ready.then((registration) => {
            // Add listener to detect new updates, so we can show a message to the user
            registration.addEventListener("updatefound", () => {
                PubSub.get().publishSnack({ message: "Downloading new updates...", autoHideDuration: "persist" });
            });
            // Send message to service worker to let it know if this is a standalone (i.e. downloaded) PWA. 
            // Standalone PWAs come with more assets, like splash screens.
            //TODO not used yet
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
// ReactGA.initialize(import.meta.env.VITE_GOOGLE_TRACKING_ID);
// const sendToAnalytics = ({ name, delta, id }) => {
//     console.log("sendToAnalytics", { name, delta, id }, import.meta.env.VITE_GOOGLE_TRACKING_ID);
//     ReactGA.event({
//         category: "Web Vitals",
//         action: name,
//         value: Math.round(name === "CLS" ? delta * 1000 : delta), // CLS is reported as a fraction, so multiply by 1000 to make it more readable
//         label: id,
//         nonInteraction: true,
//     });
// };
// reportWebVitals(sendToAnalytics);
