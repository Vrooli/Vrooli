package com.vrooli.portal.tv;

import android.graphics.Bitmap;
import android.os.Handler;
import android.os.Looper;
import android.util.LruCache;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.function.Consumer;

/** Small artwork cache; no page engines, speculative previews or unbounded request queue. */
final class ArtworkLoader {
    private final ExecutorService workers = Executors.newFixedThreadPool(2);
    private final Handler main = new Handler(Looper.getMainLooper());
    private final Map<String, Consumer<Bitmap>> pending = new HashMap<>();
    private final LruCache<String, Bitmap> cache = new LruCache<String, Bitmap>(4 * 1024 * 1024) {
        @Override protected int sizeOf(String key, Bitmap bitmap) { return bitmap.getAllocationByteCount(); }
    };
    private boolean closed;
    void load(PortalClient client, String path, Consumer<Bitmap> receive) {
        String key = client.origin + path;
        Bitmap existing = cache.get(key);
        if (existing != null) { receive.accept(existing); return; }
        if (closed) return;
        if (pending.containsKey(key)) { pending.put(key, receive); return; }
        if (pending.size() >= 18) return;
        pending.put(key, receive);
        workers.execute(() -> {
            Bitmap value = null;
            try { value = client.artwork(path); } catch (Exception ignored) { /* Keep honest letter fallback. */ }
            final Bitmap result = value;
            main.post(() -> {
                Consumer<Bitmap> latest = pending.remove(key);
                if (closed) return;
                if (result != null) { cache.put(key, result); if (latest != null) latest.accept(result); }
            });
        });
    }
    void close() { closed = true; workers.shutdownNow(); cache.evictAll(); }
}
