package com.vrooli.portal.tv;

import android.content.Context;
import android.net.nsd.NsdManager;
import android.net.nsd.NsdServiceInfo;
import android.net.wifi.WifiManager;
import android.os.Handler;
import android.os.Looper;
import java.nio.charset.StandardCharsets;
import java.util.*;

/** One bounded NSD browse. Android 11 permits only one outstanding resolution. */
final class PortalDiscovery {
    interface Result { void complete(List<String> origins, String status); }
    private final NsdManager manager;
    private final Handler main = new Handler(Looper.getMainLooper());
    private final WifiManager.MulticastLock multicast;
    private NsdManager.DiscoveryListener listener;
    private int generation;
    private boolean resolving, plaintext;
    private final ArrayDeque<NsdServiceInfo> queue = new ArrayDeque<>();
    private final Set<String> seen = new HashSet<>();
    private final List<String> origins = new ArrayList<>();
    PortalDiscovery(Context context) {
        manager = (NsdManager) context.getSystemService(Context.NSD_SERVICE);
        WifiManager wifi = (WifiManager) context.getApplicationContext().getSystemService(Context.WIFI_SERVICE);
        multicast = wifi == null ? null : wifi.createMulticastLock("portal-discovery");
        if (multicast != null) multicast.setReferenceCounted(false);
    }
    void start(Result result) {
        stop(); final int request = generation;
        queue.clear(); seen.clear(); origins.clear(); resolving = false; plaintext = false;
        if (manager == null) { result.complete(Collections.emptyList(), "Local discovery is unavailable. Enter a Portal address in Settings."); return; }
        listener = new NsdManager.DiscoveryListener() {
            public void onDiscoveryStarted(String type) {}
            public void onDiscoveryStopped(String type) {}
            public void onStopDiscoveryFailed(String type, int code) {}
            public void onStartDiscoveryFailed(String type, int code) { main.post(() -> { if (request == generation) finish(result, "Local discovery is unavailable. You can still enter a Portal address."); }); }
            public void onServiceLost(NsdServiceInfo service) {}
            public void onServiceFound(NsdServiceInfo service) { main.post(() -> {
                if (request != generation || seen.size() >= 8 || !seen.add(service.getServiceName())) return;
                queue.add(service); resolve(request);
            }); }
        };
        try {
            if (multicast != null) multicast.acquire();
            manager.discoverServices("_vrooli-portal._tcp.", NsdManager.PROTOCOL_DNS_SD, listener);
            main.postDelayed(() -> { if (request == generation) finish(result, plaintext ? "Portal found on your home network. Choose a secure connection or the allowed LAN fallback in Settings." : origins.isEmpty() ? "No local Portal found. Check that Portal is running, or enter its address in Settings." : "Local Portal found. Choose it in Settings to connect."); }, 8000);
        } catch (RuntimeException ignored) { finish(result, "Local discovery is unavailable. Enter a Portal address in Settings."); }
    }
    private void resolve(int request) {
        if (request != generation || resolving || queue.isEmpty()) return;
        resolving = true;
        try { manager.resolveService(queue.remove(), new NsdManager.ResolveListener() {
            public void onResolveFailed(NsdServiceInfo service, int code) { main.post(() -> { if (request == generation) { resolving = false; resolve(request); } }); }
            public void onServiceResolved(NsdServiceInfo service) { main.post(() -> {
                if (request != generation) return;
                try {
                    Map<String, byte[]> attributes = service.getAttributes();
                    String protocol = value(attributes.get("protocol")), url = value(attributes.get("url"));
                    if ("portal-catalog-v1".equals(protocol) && url.startsWith("http://")) plaintext = true;
                    String origin = PortalOrigin.discovered(url, protocol);
                    if (!origins.contains(origin)) origins.add(origin);
                } catch (Exception ignored) { /* Untrusted/unsupported hints never replace a route. */ }
                resolving = false; resolve(request);
            }); }
        }); } catch (RuntimeException ignored) { resolving = false; resolve(request); }
    }
    private static String value(byte[] data) { return data == null || data.length > 1024 ? "" : new String(data, StandardCharsets.UTF_8); }
    private void finish(Result result, String status) {
        List<String> found = new ArrayList<>(origins); stop(); result.complete(found, status);
    }
    void stop() {
        ++generation;
        if (listener != null && manager != null) { try { manager.stopServiceDiscovery(listener); } catch (RuntimeException ignored) {} }
        listener = null;
        if (multicast != null && multicast.isHeld()) multicast.release();
    }
}
