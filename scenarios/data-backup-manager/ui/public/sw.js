const CACHE_NAME = "data-backup-manager-shell-v1";
const APP_SHELL = ["/", "/index.html", "/site.webmanifest"];

self.addEventListener("install", (event) => {
	event.waitUntil(caches.open(CACHE_NAME).then((cache) => cache.addAll(APP_SHELL)));
});

self.addEventListener("activate", (event) => {
	event.waitUntil(
		caches.keys().then((keys) => Promise.all(
			keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)),
		)),
	);
});

self.addEventListener("fetch", (event) => {
	if (event.request.method !== "GET") return;

	event.respondWith(
		fetch(event.request)
			.then((response) => {
				if (response.ok && new URL(event.request.url).origin === self.location.origin) {
					const copy = response.clone();
					void caches.open(CACHE_NAME).then((cache) => cache.put(event.request, copy));
				}
				return response;
			})
			.catch(() => caches.match(event.request).then((cached) => cached || caches.match("/index.html"))),
	);
});
