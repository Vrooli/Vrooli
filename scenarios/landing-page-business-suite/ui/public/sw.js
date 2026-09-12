const CACHE_NAME = 'silent-founder-os-shell-v1';
const NAVIGATION_FALLBACK = '/';

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then(async (cache) => {
      try {
        await cache.add(NAVIGATION_FALLBACK);
      } catch {
        // Installation remains usable when the first network request is offline.
      }
    }),
  );
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((keys) =>
      Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))),
    ),
  );
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
  if (event.request.method !== 'GET' || event.request.mode !== 'navigate') {
    return;
  }

  event.respondWith(
    fetch(event.request)
      .then((response) => {
        const copy = response.clone();
        void caches.open(CACHE_NAME).then((cache) => cache.put(NAVIGATION_FALLBACK, copy));
        return response;
      })
      .catch(async () => (await caches.match(NAVIGATION_FALLBACK)) || Response.error()),
  );
});
