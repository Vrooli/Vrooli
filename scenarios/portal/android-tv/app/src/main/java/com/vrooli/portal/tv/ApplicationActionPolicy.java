package com.vrooli.portal.tv;

/** Keeps the owned-app menu vocabulary and selection mapping deterministic. */
final class ApplicationActionPolicy {
    enum LifecycleResult { CONFIRMED, UNCONFIRMED }

    private ApplicationActionPolicy() {}

    static String[] options(String scenario, boolean pinned) {
        String favorite = pinned ? "Remove from favorites" : "Add to favorites";
        if ("portal".equals(scenario)) return new String[]{favorite};
        return new String[]{favorite, "Start app", "Restart app", "Stop app"};
    }

    static boolean isFavoriteSelection(int selection) {
        return selection == 0;
    }

    static String actionForSelection(String scenario, int selection) {
        if (selection <= 0 || "portal".equals(scenario)) return "";
        return switch (selection) {
            case 1 -> "start";
            case 2 -> "restart";
            case 3 -> "stop";
            default -> "";
        };
    }

    static boolean requiresConfirmation(String action) {
        return "restart".equals(action) || "stop".equals(action);
    }

    /** Interpret only explicit, recognized owner responses; every other shape is uncertain. */
    static LifecycleResult interpretLifecycleResponse(String action, String runtimeState, String outcome) {
        if (!"completed".equals(outcome)) return LifecycleResult.UNCONFIRMED;
        boolean terminalStateMatches = switch (action == null ? "" : action) {
            case "start", "restart" -> "running".equals(runtimeState);
            case "stop" -> "stopped".equals(runtimeState) || "available".equals(runtimeState);
            default -> false;
        };
        return terminalStateMatches ? LifecycleResult.CONFIRMED : LifecycleResult.UNCONFIRMED;
    }

    static String managementGuidance(boolean configured, boolean secure, boolean signedIn) {
        if (!configured) return "Set a Portal address. App management requires trusted HTTPS.";
        if (!secure) return "HTTP is browse-only. App management requires trusted HTTPS.";
        if (!signedIn) return "HTTPS address set. Verify certificate trust, then sign in to manage apps.";
        return "Local sign-in token present. Actions may need renewed sign-in and lifecycle permission.";
    }
}
