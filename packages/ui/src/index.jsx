import { ApolloProvider } from "@apollo/client";
import { Router } from "@local/shared";
import { initializeApollo } from "api/utils/initialize";
import { ErrorBoundary } from "components/ErrorBoundary/ErrorBoundary";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import "./i18n"; // Must import for translations to work
import reportWebVitals from "./reportWebVitals";
import * as serviceWorkerRegistration from "./serviceWorkerRegistration";
import { getDeviceInfo } from "./utils/display/device";
import { PubSub } from "./utils/pubsub";

const client = initializeApollo();

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
    <Router>
        <ApolloProvider client={client}>
            <ErrorBoundary>
                <App />
            </ErrorBoundary>
        </ApolloProvider>
    </Router>,
);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://cra.link/PWA
serviceWorkerRegistration.register({
    onUpdate: (registration) => {
        if (registration && registration.waiting) {
            registration.waiting.postMessage({ type: "SKIP_WAITING" });
            if (confirm("New version available! The site will now update. Press \"Cancel\" if you need to save any unsaved data.")) {
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


// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();
