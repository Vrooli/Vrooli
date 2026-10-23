package com.vrooli.portal.tv;

import android.content.ComponentName;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.content.pm.ResolveInfo;
import java.util.*;

/** Installed app discovery and exact activity launch, independent of Portal. */
final class TvApps {
    private final PackageManager packages;
    private final String ownPackage;
    private List<ResolveInfo> snapshot = Collections.emptyList();
    private String revision = "";
    TvApps(PackageManager packages, String ownPackage) { this.packages = packages; this.ownPackage = ownPackage; refresh(); }
    List<ResolveInfo> installed() { return new ArrayList<>(snapshot); }
    boolean refresh() {
        List<ResolveInfo> values = new ArrayList<>(packages.queryIntentActivities(new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LEANBACK_LAUNCHER), 0));
        values.removeIf(app -> ownPackage.equals(app.activityInfo.packageName) || "com.vrooli.portal.tv".equals(app.activityInfo.packageName) || "com.vrooli.portal.tv.home".equals(app.activityInfo.packageName));
        values.sort((a, b) -> String.CASE_INSENSITIVE_ORDER.compare(a.loadLabel(packages).toString(), b.loadLabel(packages).toString()));
        Set<String> seen = new HashSet<>(); values.removeIf(app -> !seen.add(app.activityInfo.packageName));
        StringBuilder signature = new StringBuilder();
        for (ResolveInfo value : values) signature.append(value.activityInfo.packageName).append('/').append(value.activityInfo.name).append(':').append(value.loadLabel(packages)).append(':').append(value.activityInfo.applicationInfo.sourceDir).append('\n');
        boolean changed = !signature.toString().equals(revision);
        revision = signature.toString(); snapshot = values; return changed;
    }
    Intent launch(ResolveInfo app) {
        return new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LEANBACK_LAUNCHER)
            .setComponent(new ComponentName(app.activityInfo.packageName, app.activityInfo.name))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_RESET_TASK_IF_NEEDED);
    }
}
