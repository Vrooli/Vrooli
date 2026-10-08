const CACHE_NAME = "notification-hub-app-shell-v1";
const APP_SHELL_URLS = ["./", "./site.webmanifest"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL_URLS))
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name)))
      )
      .then(() => self.clients.claim())
  );
});

self.addEventListener("push", (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = { body: event.data ? event.data.text() : "Notification available" };
  }
  event.waitUntil(self.registration.showNotification(payload.title || "Notification Hub", {
    body: payload.body || "Notification available",
    tag: payload.id || undefined,
    data: { id: payload.id || "", url: payload.url || "" },
  }));
});

// A tap opens the page the hub named (an ask opens /asks/<id>), resolved
// against this worker's scope so a proxied or path-prefixed origin still
// works. iOS home-screen apps get no notification action buttons, so the
// tap-then-choose page is the answer path there.
function notificationTarget(data) {
  const path = data && typeof data.url === "string" && data.url ? data.url : "./";
  try {
    const target = new URL(path, self.registration.scope);
    return target.origin === new URL(self.registration.scope).origin ? target.href : self.registration.scope;
  } catch {
    return self.registration.scope;
  }
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = notificationTarget(event.notification.data);
  event.waitUntil(clients.matchAll({ type: "window", includeUncontrolled: true }).then(async (windows) => {
    const existing = windows[0];
    if (existing && typeof existing.navigate === "function") {
      try {
        const navigated = await existing.navigate(target);
        return (navigated || existing).focus();
      } catch {
        // An uncontrolled window cannot be navigated by the worker.
      }
    }
    return clients.openWindow(target);
  }));
});

self.addEventListener("pushsubscriptionchange", (event) => {
  event.waitUntil((async () => {
    // Reuse the old subscription's options so the VAPID applicationServerKey
    // carries over; subscribing without it fails.
    const options = event.oldSubscription && event.oldSubscription.options ? event.oldSubscription.options : { userVisibleOnly: true };
    const replacement = await self.registration.pushManager.subscribe(options);
    await registerWithHub(replacement.toJSON());
  })());
});

// Tell the hub about a replaced subscription. The owner session cookie rides
// along (same origin), so this works with no window open; without it the hub
// would keep the dead endpoint and the operator would silently stop hearing.
function registerWithHub(subscription) {
  const keys = subscription.keys || {};
  const url = new URL("vrooli.notification_hub.v1.recipients.RecipientsService/RegisterPushSubscription", self.registration.scope);
  return fetch(url.href, {
    method: "POST",
    credentials: "same-origin",
    headers: { "Content-Type": "application/json", "Connect-Protocol-Version": "1", "X-Vrooli-Browser-Session": "1" },
    body: JSON.stringify({ endpoint: subscription.endpoint, p256dh: keys.p256dh, auth: keys.auth, origin: new URL(self.registration.scope).origin }),
  });
}

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(() => caches.match("./").then((response) => response || Response.error()))
    );
    return;
  }

  event.respondWith(
    caches.match(request).then((cached) => cached || fetch(request))
  );
});
