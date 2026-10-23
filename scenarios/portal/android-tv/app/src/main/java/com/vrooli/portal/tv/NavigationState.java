package com.vrooli.portal.tv;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/** Stable identities survive view recreation and asynchronous catalogue refreshes. */
final class NavigationState {
    static final int HOME_APP_LIMIT = 12;
    String page = "Home";
    String libraryReturn = "Home";
    String settingsReturn = "Home";
    String filter = "";
    String libraryMode = "All apps";
    final Map<String, String> focused = new HashMap<>();
    final Map<String, Integer> scroll = new HashMap<>();

    void systemHome() {
        page = "Home";
        focused.remove("Home");
        libraryReturn = "Home";
        settingsReturn = "Home";
    }

    static List<String> homeApps(List<String> installed, List<String> recent, List<String> pinned) {
        List<String> result = new ArrayList<>();
        List<String> recentFavorites = new ArrayList<>(), recentOthers = new ArrayList<>(), otherInstalled = new ArrayList<>();
        for (String identity : recent) (pinned.contains(identity) ? recentFavorites : recentOthers).add(identity);
        for (String identity : installed) if (!pinned.contains(identity)) otherInstalled.add(identity);
        for (List<String> group : java.util.Arrays.asList(recentFavorites, pinned, recentOthers, otherInstalled)) {
            for (String identity : group) if (installed.contains(identity) && !result.contains(identity)) {
                result.add(identity);
                if (result.size() == HOME_APP_LIMIT) return result;
            }
        }
        return result;
    }

    boolean exitsOnBack(boolean homeEnabled) {
        return "Home".equals(page) && !homeEnabled;
    }

    void navigate(String destination) {
        if ("Vrooli".equals(destination) && !"Vrooli".equals(page)) {
            String previous = "Settings".equals(page) ? settingsReturn : page;
            if (!"Vrooli".equals(previous)) libraryReturn = previous;
        }
        if ("Settings".equals(destination) && !"Settings".equals(page)) settingsReturn = page;
        page = destination;
    }
    String back() {
        if ("Vrooli".equals(page)) return "Vrooli".equals(libraryReturn) ? "Home" : libraryReturn;
        if ("Settings".equals(page)) return "Settings".equals(settingsReturn) ? "Home" : settingsReturn;
        return "Home";
    }
    static List<String> favorites(List<String> installed, List<String> saved, int limit) {
        List<String> result = new ArrayList<>();
        for (String identity : saved) {
            if (result.size() >= limit) break;
            if (installed.contains(identity) && !result.contains(identity)) result.add(identity);
        }
        return result;
    }
    static boolean move(List<String> values, String identity, int delta) {
        int from = values.indexOf(identity), to = from + delta;
        if (from < 0 || to < 0 || to >= values.size()) return false;
        values.remove(from);
        values.add(to, identity);
        return true;
    }
}
