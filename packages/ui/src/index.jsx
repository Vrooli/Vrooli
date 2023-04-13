import { ApolloProvider } from '@apollo/client';
import { Router } from '@shared/route';
import { initializeApollo } from 'api/utils/initialize';
import { ErrorBoundary } from 'components/ErrorBoundary/ErrorBoundary';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import './i18n'; // Must import for translations to work
import reportWebVitals from './reportWebVitals';
import * as serviceWorkerRegistration from './serviceWorkerRegistration';
import { PubSub } from './utils/pubsub';

const client = initializeApollo();

const root = ReactDOM.createRoot(document.getElementById("root"));
root.render(
    <Router>
        <ApolloProvider client={client}>
            <ErrorBoundary>
                <App />
            </ErrorBoundary>
        </ApolloProvider>
    </Router>
);

// If you want your app to work offline and load faster, you can change
// unregister() to register() below. Note this comes with some pitfalls.
// Learn more about service workers: https://cra.link/PWA
serviceWorkerRegistration.register({
    onUpdate: (registration) => {
        if (registration && registration.waiting) {
            registration.waiting.postMessage({ type: 'SKIP_WAITING' });
            window.location.reload();
        }
    },
});

// Listen for service worker state changes
// if ('serviceWorker' in navigator) {
//     navigator.serviceWorker.addEventListener('message', (event) => {
//         console.log('SERVICE WORKER MESSAGE', event)
//         if (event.data && event.data.type === 'SW_UPDATE_START') {
//             console.log('SW_UPDATE_START', event)
//             // Display a notification when the service worker starts updating
//             PubSub.get().publishSnack({ message: 'Downloading new updates...' });
//         }
//     });

//     navigator.serviceWorker.ready.then((registration) => {
//         registration.installing?.addEventListener('statechange', (event) => {
//             if (event.target.state === 'activated') {
//                 alert('New version available! The site will now update.');
//             }
//         });
//     });
// }

if ('serviceWorker' in navigator) {
    // Listen for updatefound event on the service worker registration
    navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data && event.data.type === 'SW_UPDATE_START') {
            // Display a notification when the service worker starts updating
            PubSub.get().publishSnack({ message: 'Downloading new updates 1...' });
        }
    });

    navigator.serviceWorker.ready.then((registration) => {
        registration.addEventListener('updatefound', () => {
            PubSub.get().publishSnack({ message: 'Downloading new updates 2...' });
        });

        registration.installing?.addEventListener('statechange', (event) => {
            if (event.target.state === 'activated') {
                alert('New version available! The site will now update.');
            }
        });
    });
}


// Shows message, but too late
// if ('serviceWorker' in navigator) {
//     window.addEventListener('load', () => {
//         navigator.serviceWorker.register('/service-worker.js').then((registration) => {
//             console.log('Service worker registered with scope:', registration.scope);

//             registration.addEventListener('updatefound', () => {
//                 const newWorker = registration.installing;
//                 console.log('New service worker found:', newWorker);

//                 newWorker.addEventListener('statechange', () => {
//                     if (newWorker.state === 'installed') {
//                         newWorker.postMessage({ type: 'SW_UPDATE_CHECK' });
//                     }
//                 });
//             });
//         });

//         navigator.serviceWorker.addEventListener('message', (event) => {
//             if (event.data && event.data.type === 'SW_UPDATE_START') {
//                 console.log('service worker received message from client', event)
//                 PubSub.get().publishSnack({ message: 'Downloading new updates...' });
//             }
//         });
//     });
// }


// If you want to start measuring performance in your app, pass a function
// to log results (for example: reportWebVitals(console.log))
// or send to an analytics endpoint. Learn more: https://bit.ly/CRA-vitals
reportWebVitals();