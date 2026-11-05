package com.vrooli.portal.tv;

import android.app.Activity;
import android.app.role.RoleManager;
import android.content.ComponentName;
import android.content.Intent;
import android.content.Context;
import android.content.SharedPreferences;
import android.content.pm.ResolveInfo;
import android.provider.Settings;
import java.util.List;

/** Android owns Home selection. Keep the observed previous Home as a native escape. */
final class HomeIntegration {
    private final Activity activity;
    private final SharedPreferences preferences;
    HomeIntegration(Activity activity, SharedPreferences preferences) { this.activity = activity; this.preferences = preferences; }
    private Intent homeIntent() { return new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_HOME); }
    boolean selected() {
        ResolveInfo home = activity.getPackageManager().resolveActivity(homeIntent(), 0);
        return home != null && home.activityInfo != null && activity.getPackageName().equals(home.activityInfo.packageName);
    }
    void rememberPrevious() {
        ResolveInfo home = activity.getPackageManager().resolveActivity(homeIntent(), 0);
        if (home == null || home.activityInfo == null || activity.getPackageName().equals(home.activityInfo.packageName)) return;
        ComponentName observed = new ComponentName(home.activityInfo.packageName, home.activityInfo.name);
        List<ResolveInfo> choices = activity.getPackageManager().queryIntentActivities(homeIntent(), 0);
        for (ResolveInfo choice : choices) {
            ComponentName candidate = new ComponentName(choice.activityInfo.packageName, choice.activityInfo.name);
            if (candidate.equals(observed)) { preferences.edit().putString("previous-home", observed.flattenToString()).apply(); return; }
        }
    }
    void chooseHome() {
        rememberPrevious();
        if (android.os.Build.VERSION.SDK_INT >= 29) {
            RoleManager roles = activity.getSystemService(RoleManager.class);
            if (roles != null && roles.isRoleAvailable(RoleManager.ROLE_HOME) && !roles.isRoleHeld(RoleManager.ROLE_HOME)) {
                activity.startActivityForResult(roles.createRequestRoleIntent(RoleManager.ROLE_HOME), 1201); return;
            }
        }
        activity.startActivity(new Intent(Settings.ACTION_HOME_SETTINGS));
    }
    boolean previousAvailable() { return previous() != null; }
    boolean overrideRequested() { return preferences.getBoolean(HomeOverrideService.ENABLED, false); }
    boolean overrideConnected() { return HomeOverrideService.connected(); }
    static boolean overrideAllowed(Context context) {
        String enabled = Settings.Secure.getString(context.getContentResolver(), Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES);
        if (enabled == null) return false;
        ComponentName own = new ComponentName(context.getPackageName(), HomeOverrideService.class.getName());
        for (String value : enabled.split(":")) if (own.equals(ComponentName.unflattenFromString(value))) return true;
        return false;
    }
    boolean overrideAllowed() { return overrideAllowed(activity); }
    void enableOverride() {
        rememberPrevious();
        if (previous() == null) throw new IllegalStateException("previous_home_unavailable");
        if (!preferences.edit().putBoolean(HomeOverrideService.ENABLED, true).commit()) throw new IllegalStateException("preference_unavailable");
        if (!overrideConnected()) openAccessibilitySettings();
    }
    void openAccessibilitySettings() {
        // Some Google TV forwarders accept the generic action then display a
        // dead-end toast. Prefer the installed TV Settings handler explicitly.
        for (Intent intent : new Intent[]{new Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS).setPackage("com.android.tv.settings"),
                new Intent(Settings.ACTION_SETTINGS).setPackage("com.android.tv.settings"), new Intent(Settings.ACTION_SETTINGS)}) {
            ResolveInfo target = activity.getPackageManager().resolveActivity(intent, 0);
            if (target == null || target.activityInfo == null) continue;
            try {
                activity.startActivity(intent.setComponent(new ComponentName(target.activityInfo.packageName, target.activityInfo.name)));
                return;
            } catch (android.content.ActivityNotFoundException | SecurityException unavailable) { /* Try the next supported Settings route. */ }
        }
        throw new IllegalStateException("settings_unavailable");
    }
    void openPrevious() {
        ComponentName previous = previous();
        if (previous == null) throw new IllegalStateException("previous_home_unavailable");
        if (!preferences.edit().putBoolean(HomeOverrideService.ENABLED, false).commit()) throw new IllegalStateException("preference_unavailable");
        activity.startActivity(new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_HOME).setComponent(previous).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
    }
    private ComponentName previous() {
        ComponentName value = ComponentName.unflattenFromString(preferences.getString("previous-home", ""));
        if (value == null || value.getPackageName().equals(activity.getPackageName())) return null;
        try { activity.getPackageManager().getActivityInfo(value, 0); return value; }
        catch (Exception ignored) { return null; }
    }
}
