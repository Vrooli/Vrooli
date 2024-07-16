/* eslint-disable no-undef */
/* eslint-disable no-restricted-globals */
/* global importScripts workbox */
// Import the necessary Workbox scripts using importScripts
importScripts("https://storage.googleapis.com/workbox-cdn/releases/7.0.0/workbox-sw.js");

// This service worker can be customized!
// See https://developers.google.com/web/tools/workbox/modules
// for the list of available Workbox modules, or add any other
// code you'd like.
// You can also remove this file if you'd prefer not to use a
// service worker, and the Workbox build step will be skipped.
const { clientsClaim } = (workbox.core);
const { ExpirationPlugin } = (workbox.expiration);
const { createHandlerBoundToURL, precacheAndRoute } = (workbox.precaching);
const { registerRoute } = (workbox.routing);
const { CacheFirst } = (workbox.strategies);

const CACHE_NAME = "vrooli-cache";
const CURRENT_CACHE_VERSION = "2024-07-16-g"; // Change this value to force a cache update

// eslint-disable-next-line no-magic-numbers
const DAYS_30_SECONDS = 30 * 24 * 60 * 60;
const CACHE_EXPIRATION = DAYS_30_SECONDS;

clientsClaim();

// Precache all of the assets generated by your build process.
// Their URLs are injected into the manifest variable below.
// This variable must be present somewhere in your service worker file,
// even if you decide not to use precaching. See https://cra.link/PWA
const precache = self.__WB_MANIFEST ?? [];
precacheAndRoute(precache);

// Set up App Shell-style routing, so that all navigation requests
// are fulfilled with your index.html shell. Learn more at
// https://developers.google.com/web/fundamentals/architecture/app-shell
const fileExtensionRegexp = new RegExp("/[^/?]+\\.[^/]+$");
registerRoute(
    ({ request, url }) => {
        if (request.mode !== "navigate") {
            return false;
        }
        if (url.pathname.startsWith("/_")) {
            return false;
        }
        if (url.pathname.match(fileExtensionRegexp)) {
            return false;
        }
        return true;
    },
    async ({ url, event }) => {
        try {
            const handler = createHandlerBoundToURL(self.location.origin + "/index.html");
            const response = await handler({ event });

            const clonedResponse = response.clone();
            const newResponse = new Response(clonedResponse.body, {
                status: clonedResponse.status,
                statusText: clonedResponse.statusText,
                headers: clonedResponse.headers,
            });

            return newResponse;
        } catch (error) {
            console.error(`Error in custom routing function: ${error}`);
            return Response.error();
        }
    },
);

registerRoute(
    ({ url }) => {
        // Exclude /api routes from cache
        if (url.pathname.startsWith("/api")) {
            return false;
        }
        return true;
    },
    new CacheFirst({
        cacheName: CACHE_NAME,
        plugins: [
            new ExpirationPlugin({
                maxEntries: 200,
                maxAgeSeconds: CACHE_EXPIRATION,
            }),
        ],
    }),
);

// Listen to events sent from the main application (`index.jsx`)
self.addEventListener("message", (event) => {
    if (event.data && event.data.type === "SKIP_WAITING") {
        self.skipWaiting();
    }
    // if (event.data && event.data.type === "SW_UPDATE_CHECK") {
    //     console.log("post received from client: SW_UPDATE_CHECK");
    //     event.source.postMessage({ type: "SW_UPDATE_START" });
    // }
});

// Listen for the install event
self.addEventListener("install", (event) => {
    console.log("Service worker installing...", event);
    // Instruct the service worker to skip waiting and immediately become active
    event.waitUntil(self.skipWaiting());
    // NOTE: You can't send a message to the main application here, because
    // the OLD service worker is still in control of the page. That means the 
    // NEW one (i.e. this one) can't communicate with the main application yet.
});

self.addEventListener("activate", (event) => {
    event.waitUntil(
        caches.keys().then((cacheNames) => {
            return Promise.all(
                cacheNames.map((cacheName) => {
                    if (cacheName !== CURRENT_CACHE_VERSION) {
                        console.log(`Deleting old cache: ${cacheName}`);
                        return caches.delete(cacheName);
                    }
                }),
            );
        }).then(() => {
            self.clients.claim();
        }),
    );
});

// Handle push notifications
self.addEventListener("push", (event) => {
    const data = event.data.json();
    const options = {
        body: data.body,
        icon: data.icon,
        badge: data.badge,
        image: data.image,
        vibrate: data.vibrate,
        actions: data.actions,
        tag: data.tag,
        renotify: data.renotify,
        silent: data.silent,
        requireInteraction: data.requireInteraction,
        data: data.data,
    };

    event.waitUntil(self.registration.showNotification(data.title, options));
});
self.addEventListener("notificationclick", (event) => {
    event.notification.close(); // Close the notification when clicked

    // Determine what action to take based on the clicked action or the notification click itself
    if (event.action) {
        // Handle specific action clicks
        // Example: if (event.action === 'some-action') { /* handle 'some-action' */}
        console.log("Notification action clicked: ", event.action);
    } else {
        // Handle the notification click when no specific action is defined
        event.waitUntil(
            clients.matchAll({ type: "window" }).then((clientList) => {
                const urlToOpen = new URL("/", location.origin).href;

                for (let i = 0; i < clientList.length; i++) {
                    const client = clientList[i];
                    if (client.url === urlToOpen && "focus" in client) {
                        return client.focus();
                    }
                }

                if (clients.openWindow) {
                    return clients.openWindow(urlToOpen);
                }
            }),
        );
    }
});

self.addEventListener("periodicsync", (event) => {
    console.log("periodicsync", event);
    // if (event.tag === UPDATE_CHECK) {
    //     event.waitUntil(checkForUpdates());
    // }
});

self.addEventListener("message", (event) => {
    console.log("message", event);
    // if (event.data === UPDATE_CHECK) {
    //     event.waitUntil(checkForUpdates());
    // }
});
