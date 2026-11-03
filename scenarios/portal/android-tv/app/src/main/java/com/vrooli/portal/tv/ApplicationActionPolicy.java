package com.vrooli.portal.tv;

/** Keeps the owned-app menu vocabulary and selection mapping deterministic. */
final class ApplicationActionPolicy {
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
}
