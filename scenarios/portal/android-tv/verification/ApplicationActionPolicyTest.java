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
        System.out.println("PASS: owned-app menu actions map to explicit lifecycle calls; Portal is excluded and destructive actions are confirmed");
    }
}
