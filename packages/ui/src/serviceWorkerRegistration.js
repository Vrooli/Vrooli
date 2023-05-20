// This optional code is used to register a service worker.
// register() is not called by default.

// This lets the app load faster on subsequent visits in production, and gives
// it offline capabilities. However, it also means that developers (and users)
// will only see deployed updates on subsequent visits to a page, after all the
// existing tabs open on the page have been closed, since previously cached
// resources are updated in the background.

// To learn more about the benefits of this model and instructions on how to
// opt-in, read https://cra.link/PWA

function urlBase64ToUint8Array(base64String) {
    const padding = "=".repeat((4 - base64String.length % 4) % 4);
    const base64 = (base64String + padding)
        .replace(/\-/g, "+")
        .replace(/_/g, "/");

    const rawData = window.atob(base64);
    const outputArray = new Uint8Array(rawData.length);

    for (let i = 0; i < rawData.length; ++i) {
        outputArray[i] = rawData.charCodeAt(i);
    }
    return outputArray;
}

export async function requestNotificationPermission() {
    const permission = await Notification.requestPermission();
    if (permission === "granted") {
        console.info("Notification permission granted!");
    } else {
        console.error("Notification permission denied");
    }
    return permission;
}

export async function subscribeUserToPush() {
    if (!("PushManager" in window)) {
        console.warn("Push notifications are not supported in this browser. This could be because the browser is too old or because it is running in a non-secure context (http instead of https).");
        return null;
    }
    const permission = await requestNotificationPermission();
    if (permission !== "granted") {
        console.warn("Push permissions not granted.");
        return null;
    }
    try {
        const registration = await navigator.serviceWorker.register(
            `${import.meta.env.BASE_URL}service-worker.js`,
            { scope: import.meta.env.BASE_URL },
        );
        console.log("getting subscribeoptions", import.meta.env);
        const subscribeOptions = {
            userVisibleOnly: true,
            applicationServerKey: urlBase64ToUint8Array(
                import.meta.env.VITE_VAPID_PUBLIC_KEY,
            ),
        };
        console.log("push notification subscribeOptions: ", subscribeOptions);
        const pushSubscription = await registration.pushManager.subscribe(subscribeOptions);
        console.log(
            "Received PushSubscription: ",
            JSON.stringify(pushSubscription),
        );
        return pushSubscription;
    } catch (error) {
        console.error("Error subscribing to push notifications:", error);
        return null;
    }
}

const isLocalhost = Boolean(
    window.location.hostname === "localhost" ||
    // [::1] is the IPv6 localhost address.
    window.location.hostname === "[::1]" ||
    // 127.0.0.0/8 are considered localhost for IPv4.
    window.location.hostname.match(/^127(?:\.(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)){3}$/),
);

export function register(config) {
    console.log("register 1", config, "serviceWorker" in navigator);
    if (import.meta.env.PROD && "serviceWorker" in navigator) {
        console.log("register 2");
        // The URL constructor is available in all browsers that support SW.
        const publicUrl = new URL(import.meta.env.BASE_URL, window.location.href);
        console.log("register 3", publicUrl, "window.location.origin: ", window.location.origin);
        if (publicUrl.origin !== window.location.origin) {
            console.log("register 4 - bad origin");
            // Our service worker won't work if PUBLIC_URL/BASE_URL is on a different origin
            // from what our page is served on. This might happen if a CDN is used to
            // serve assets; see https://github.com/facebook/create-react-app/issues/2374
            return;
        }

        console.log("register 5");
        // Function for checking registration of service worker
        const checkRegister = () => {
            const swUrl = `${window.location.origin}/service-worker.js`;
            console.log("register 6 - checking register...", swUrl);
            if (isLocalhost) {
                console.log("register 7 is localhost");
                // This is running on localhost. Let's check if a service worker still exists or not.
                checkValidServiceWorker(swUrl, config);

                // Add some additional logging to localhost, pointing developers to the
                // service worker/PWA documentation.
                navigator.serviceWorker.ready.then(() => {
                    console.log(
                        "This web app is being served cache-first by a service " +
                        "worker. To learn more, visit https://cra.link/PWA",
                    );
                });
            } else {
                console.log("register 8 is not localhost");
                // Is not localhost. Just register service worker
                registerValidSW(swUrl, config);
            }
        };

        console.log("register 9");
        // Check for registration on load and visibility change
        window.addEventListener("load", checkRegister);
        // document.addEventListener('visibilitychange', () => {
        //     if (document.visibilityState === 'visible') {
        //         checkRegister();
        //     }
        // });
    }
}

function registerValidSW(swUrl, config) {
    navigator.serviceWorker
        .register(swUrl)
        .then((registration) => {
            registration.onupdatefound = () => {
                const installingWorker = registration.installing;
                if (installingWorker == null) {
                    return;
                }
                installingWorker.onstatechange = () => {
                    if (installingWorker.state === "installed") {
                        if (navigator.serviceWorker.controller) {
                            // At this point, the updated precached content has been fetched,
                            // but the previous service worker will still serve the older
                            // content until all client tabs are closed.
                            console.log(
                                "New content is available and will be used when all " +
                                "tabs for this page are closed. See https://cra.link/PWA.",
                            );

                            // Execute callback
                            if (config && config.onUpdate) {
                                config.onUpdate(registration);
                            }
                        } else {
                            // At this point, everything has been precached.
                            // It's the perfect time to display a
                            // "Content is cached for offline use." message.
                            console.log("Content is cached for offline use.");

                            // Execute callback
                            if (config && config.onSuccess) {
                                config.onSuccess(registration);
                            }
                        }
                        // If running in standalone, request notification permission
                        if (window.matchMedia("(display-mode: standalone)").matches) {
                            Notification.requestPermission();
                        }
                    }
                };
            };
        })
        .catch((error) => {
            console.error("Error during service worker registration:", error);
        });
}

function checkValidServiceWorker(swUrl, config) {
    // Check if the service worker can be found. If it can't reload the page.
    fetch(swUrl, {
        headers: { "Service-Worker": "script" },
    })
        .then((response) => {
            // Ensure service worker exists, and that we really are getting a JS file.
            const contentType = response.headers.get("content-type");
            if (
                response.status === 404 ||
                (contentType != null && contentType.indexOf("javascript") === -1)
            ) {
                // No service worker found. Probably a different app. Reload the page.
                navigator.serviceWorker.ready.then((registration) => {
                    registration.unregister().then(() => {
                        window.location.reload();
                    });
                });
            } else {
                // Service worker found. Proceed as normal.
                registerValidSW(swUrl, config);
            }
        })
        .catch(() => {
            console.log("No internet connection found. App is running in offline mode.");
        });
}

export function unregister() {
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.ready
            .then((registration) => {
                registration.unregister();
            })
            .catch((error) => {
                console.error(error.message);
            });
    }
}
