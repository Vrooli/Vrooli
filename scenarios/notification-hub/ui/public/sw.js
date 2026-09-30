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
    data: { id: payload.id || "" },
  }));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  event.waitUntil(clients.matchAll({ type: "window", includeUncontrolled: true }).then((windows) => {
    const existing = windows[0];
    return existing ? existing.focus() : clients.openWindow("./");
  }));
});

self.addEventListener("pushsubscriptionchange", (event) => {
  event.waitUntil((async () => {
    const replacement = await self.registration.pushManager.subscribe({ userVisibleOnly: true });
    const openClients = await clients.matchAll({ type: "window", includeUncontrolled: true });
    openClients.forEach((client) => client.postMessage({ type: "push-subscription-changed", subscription: replacement.toJSON() }));
  })());
});

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
