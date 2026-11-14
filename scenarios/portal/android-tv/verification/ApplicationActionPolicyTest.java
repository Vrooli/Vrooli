package com.vrooli.portal.tv;

import java.util.Arrays;

public final class ApplicationActionPolicyTest {
    private static void expect(Object expected, Object actual) {
        if (!expected.equals(actual)) throw new AssertionError("expected " + expected + ", got " + actual);
    }

    public static void main(String[] args) {
        expect(Arrays.asList("Add to favorites", "Start app", "Restart app", "Stop app"),
                Arrays.asList(ApplicationActionPolicy.options("web-console", false)));
        expect(Arrays.asList("Remove from favorites", "Start app", "Restart app", "Stop app"),
                Arrays.asList(ApplicationActionPolicy.options("web-console", true)));
        expect(Arrays.asList("Add to favorites"), Arrays.asList(ApplicationActionPolicy.options("portal", false)));
        if (!ApplicationActionPolicy.isFavoriteSelection(0) || ApplicationActionPolicy.isFavoriteSelection(1)) throw new AssertionError("only the first menu item toggles favorite state");
        expect("", ApplicationActionPolicy.actionForSelection("portal", 1));
        expect("", ApplicationActionPolicy.actionForSelection("web-console", 0));
        expect("start", ApplicationActionPolicy.actionForSelection("web-console", 1));
        expect("restart", ApplicationActionPolicy.actionForSelection("web-console", 2));
        expect("stop", ApplicationActionPolicy.actionForSelection("web-console", 3));
        expect("", ApplicationActionPolicy.actionForSelection("web-console", 4));
        if (ApplicationActionPolicy.requiresConfirmation("start")) throw new AssertionError("starting an app should not require destructive confirmation");
        if (!ApplicationActionPolicy.requiresConfirmation("restart") || !ApplicationActionPolicy.requiresConfirmation("stop")) throw new AssertionError("restart and stop must require confirmation");
        expect(ApplicationActionPolicy.LifecycleResult.CONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("start", "running", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.CONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("restart", "running", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.CONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("stop", "stopped", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.CONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("stop", "available", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("start", "stopped", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("restart", "starting", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("stop", "running", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("start", "running", "unverified"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("start", "RUNNING", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("start", "", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse("unknown", "running", "completed"));
        expect(ApplicationActionPolicy.LifecycleResult.UNCONFIRMED,
                ApplicationActionPolicy.interpretLifecycleResponse(null, null, null));
        expect("Set a Portal address. App management requires trusted HTTPS.",
                ApplicationActionPolicy.managementGuidance(false, false, false));
        expect("HTTP is browse-only. App management requires trusted HTTPS.",
                ApplicationActionPolicy.managementGuidance(true, false, false));
        expect("HTTPS address set. Verify certificate trust, then sign in to manage apps.",
                ApplicationActionPolicy.managementGuidance(true, true, false));
        expect("Local sign-in token present. Actions may need renewed sign-in and lifecycle permission.",
                ApplicationActionPolicy.managementGuidance(true, true, true));
        System.out.println("PASS: app action guidance preserves trusted HTTPS, sign-in, and lifecycle permission requirements");
    }
}
