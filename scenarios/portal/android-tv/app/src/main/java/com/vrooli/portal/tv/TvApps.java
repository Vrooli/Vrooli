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
    private volatile List<ResolveInfo> snapshot = Collections.emptyList();
    private volatile Map<String, String> labels = Collections.emptyMap();
    private volatile Map<String, android.graphics.drawable.Drawable.ConstantState> artwork = Collections.emptyMap();
    private String revision = "";
    TvApps(PackageManager packages, String ownPackage) { this.packages = packages; this.ownPackage = ownPackage; }
    List<ResolveInfo> installed() { return new ArrayList<>(snapshot); }
    String label(ResolveInfo app) { return labels.getOrDefault(app.activityInfo.packageName, app.activityInfo.packageName); }
    android.graphics.drawable.Drawable artwork(ResolveInfo app, android.content.res.Resources resources) {
        android.graphics.drawable.Drawable.ConstantState value = artwork.get(app.activityInfo.packageName);
        return value == null ? null : value.newDrawable(resources);
    }
    boolean refresh() {
        List<ResolveInfo> values = new ArrayList<>(packages.queryIntentActivities(new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LEANBACK_LAUNCHER), 0));
        values.removeIf(app -> ownPackage.equals(app.activityInfo.packageName) || "com.vrooli.portal.tv".equals(app.activityInfo.packageName) || "com.vrooli.portal.tv.home".equals(app.activityInfo.packageName));
        Map<String, String> names = new HashMap<>();
        for (ResolveInfo app : values) names.put(app.activityInfo.packageName, app.loadLabel(packages).toString());
        values.sort((a, b) -> String.CASE_INSENSITIVE_ORDER.compare(names.get(a.activityInfo.packageName), names.get(b.activityInfo.packageName)));
        Set<String> seen = new HashSet<>(); values.removeIf(app -> !seen.add(app.activityInfo.packageName));
        StringBuilder signature = new StringBuilder();
        for (ResolveInfo value : values) signature.append(value.activityInfo.packageName).append('/').append(value.activityInfo.name).append(':').append(names.get(value.activityInfo.packageName)).append(':').append(value.activityInfo.applicationInfo.sourceDir).append('\n');
        boolean changed = !signature.toString().equals(revision);
        if (!changed) return false;
        Map<String, android.graphics.drawable.Drawable.ConstantState> images = new HashMap<>();
        for (ResolveInfo app : values) {
            android.graphics.drawable.Drawable image = app.activityInfo.loadBanner(packages);
            if (image == null) image = app.loadIcon(packages);
            if (image != null && image.getConstantState() != null) images.put(app.activityInfo.packageName, image.getConstantState());
        }
        revision = signature.toString(); labels = names; artwork = images; snapshot = values; return true;
    }
    Intent launch(ResolveInfo app) {
        return new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_LEANBACK_LAUNCHER)
            .setComponent(new ComponentName(app.activityInfo.packageName, app.activityInfo.name))
            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_RESET_TASK_IF_NEEDED);
    }
}
