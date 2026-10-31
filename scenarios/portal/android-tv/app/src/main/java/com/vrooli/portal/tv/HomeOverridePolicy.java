package com.vrooli.portal.tv;

import java.util.ArrayDeque;

/** Bounded routing policy; no Android objects, screen text or remote input. */
final class HomeOverridePolicy {
    enum Decision { IGNORE, OPEN_HOME, PAUSE }
    private final ArrayDeque<Long> redirects = new ArrayDeque<>();
    private long lastRedirect = -1;
    private boolean stockHomeVisible;

    static boolean startForBoot(int current, int previous) { return current >= 0 && current != previous; }

    void homeArrived() { redirects.clear(); stockHomeVisible = false; }

    Decision window(boolean enabled, boolean interactive, boolean unlocked,
                    String expectedComponent, String eventPackage, String eventClass, long now) {
        if (expectedComponent == null || eventPackage == null || eventClass == null || now < 0) { stockHomeVisible = false; return Decision.IGNORE; }
        String actualClass = eventClass.startsWith(".") ? eventPackage + eventClass : eventClass;
        stockHomeVisible = expectedComponent.equals(eventPackage + "/" + actualClass);
        return wake(enabled, interactive, unlocked, now);
    }

    Decision wake(boolean enabled, boolean interactive, boolean unlocked, long now) {
        if (!enabled || !interactive || !unlocked || !stockHomeVisible || now < 0) return Decision.IGNORE;
        if (lastRedirect >= 0 && now - lastRedirect < 300) return Decision.IGNORE;
        while (!redirects.isEmpty() && now - redirects.peekFirst() >= 5000) redirects.removeFirst();
        if (redirects.size() >= 4) return Decision.PAUSE;
        redirects.addLast(now); lastRedirect = now;
        return Decision.OPEN_HOME;
    }
}
