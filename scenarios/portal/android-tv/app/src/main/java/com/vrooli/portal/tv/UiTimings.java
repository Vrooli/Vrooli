package com.vrooli.portal.tv;

import java.util.LinkedHashMap;
import java.util.Map;

/** Bounded numeric native-handler measurements, readable only by shell/system. */
final class UiTimings {
    private static final Map<String, long[]> switches = new LinkedHashMap<>();
    private static long hits, builds;
    private static String page = "home";
    private static boolean browser, loading;
    static synchronized void page(boolean cached) { if (cached) hits++; else builds++; }
    static synchronized void surface(String selected, boolean embedded, boolean opening) {
        page = selected.toLowerCase(java.util.Locale.ROOT).replace(' ', '_'); browser = embedded; loading = opening;
    }
    static synchronized void record(String page, long nanos) {
        String key = "ui.switch." + page.toLowerCase(java.util.Locale.ROOT).replace(' ', '_');
        long[] sample = switches.computeIfAbsent(key, ignored -> new long[4]);
        long micros = Math.max(0, nanos / 1000);
        sample[0]++; sample[1] = micros; sample[2] += micros; sample[3] = Math.max(sample[3], micros);
    }
    static synchronized Map<String, String> snapshot() {
        Map<String, String> result = new LinkedHashMap<>();
        for (Map.Entry<String, long[]> item : switches.entrySet()) {
            long[] s = item.getValue();
            result.put(item.getKey(), s[0] + "/" + s[1] + "/" + s[2] / s[0] + "/" + s[3]);
        }
        result.put("ui.page", page); result.put("ui.browser", Boolean.toString(browser)); result.put("ui.loading", Boolean.toString(loading));
        result.put("ui.cache_hits", Long.toString(hits)); result.put("ui.page_builds", Long.toString(builds));
        return result;
    }
}
