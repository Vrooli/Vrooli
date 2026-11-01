package com.vrooli.portal.tv;

import java.util.ArrayDeque;

/** Bounded routing policy; no Android objects, screen text or remote input. */
final class HomeOverridePolicy {
    private static final String GOOGLE_HOME = "com.google.android.apps.tv.launcherx/com.google.android.apps.tv.launcherx.home.HomeActivity";
    private static final String GOOGLE_PROFILE_CHOOSER = "com.google.android.apps.tv.launcherx/com.google.android.apps.tv.launcherx.profile.chooser.ProfileChooserActivity";
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
        String actualComponent = eventPackage + "/" + actualClass;
        // Google TV can route Home straight to its profile chooser without
        // showing the resolved Home activity. Match only this known pair.
        stockHomeVisible = expectedComponent.equals(actualComponent)
                || (GOOGLE_HOME.equals(expectedComponent) && GOOGLE_PROFILE_CHOOSER.equals(actualComponent));
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
