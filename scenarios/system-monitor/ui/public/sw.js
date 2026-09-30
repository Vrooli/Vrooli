const CACHE_NAME = 'system-monitor-shell-v1';
const APP_SHELL = ['./', './index.html', './favicon.svg', './site.webmanifest'];

self.addEventListener('install', event => {
  event.waitUntil(caches.open(CACHE_NAME).then(cache => cache.addAll(APP_SHELL)));
  self.skipWaiting();
});

self.addEventListener('activate', event => {
  event.waitUntil(
    caches.keys().then(keys => Promise.all(
      keys.filter(key => key !== CACHE_NAME).map(key => caches.delete(key)),
    )),
  );
  self.clients.claim();
});

self.addEventListener('fetch', event => {
  if (event.request.method !== 'GET') return;
  const request = event.request;
  event.respondWith(
    fetch(request).catch(() => caches.match(request).then(response => response ?? caches.match('./index.html'))),
  );
});
