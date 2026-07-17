const CACHE_NAME = "react-component-library-app-shell-v2";
const APP_SHELL_URLS = ["./", "./site.webmanifest"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL_URLS)),
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(names.filter((name) => name !== CACHE_NAME).map((name) => caches.delete(name))),
      )
      .then(() => self.clients.claim()),
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  // RPC calls and preview frames must retain normal browser failure semantics.
  // Intercepting them made a rejected network request surface as an unhandled
  // FetchEvent promise in DevTools, then obscured the actual API error.
  if (request.method !== "GET" || new URL(request.url).origin !== self.location.origin) {
    return;
  }
  if (request.mode === "navigate") {
    event.respondWith(
      fetch(request).catch(() => caches.match("./").then((response) => response || new Response("Offline", { status: 503, statusText: "Service Unavailable" }))),
    );
    return;
  }

	 event.respondWith(caches.match(request).then((cached) => cached || fetch(request).catch(() => new Response("Offline", { status: 503, statusText: "Service Unavailable" }))));
});
