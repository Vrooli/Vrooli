package com.vrooli.portal.tv;

import android.accessibilityservice.AccessibilityService;
import android.accessibilityservice.AccessibilityServiceInfo;
import android.app.KeyguardManager;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.ComponentName;
import android.content.Context;
import android.content.Intent;
import android.content.IntentFilter;
import android.content.SharedPreferences;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.os.PowerManager;
import android.os.SystemClock;
import android.provider.Settings;
import android.view.accessibility.AccessibilityEvent;

/** Opt-in device-local Home routing. Android Settings owns the binding permission. */
public final class HomeOverrideService extends AccessibilityService {
    static final String ENABLED = "home-override-enabled";
    static final String SETTINGS_ACTION = "com.vrooli.portal.tv.HOME_SETTINGS";
    private static final String CHANNEL = "vrooli-home-override";
    private static final int NOTIFICATION = 41;
    private static volatile HomeOverrideService live;
    private final HomeOverridePolicy policy = new HomeOverridePolicy();
    private final Handler handler = new Handler(Looper.getMainLooper());
    private SharedPreferences preferences;
    private boolean connected, receiverRegistered, bootPending;
    private boolean foreground;
    private String lastStockClass = "unknown";
    private int boot = -1, wakeGeneration;
    private final BroadcastReceiver wake = new BroadcastReceiver() {
        @Override public void onReceive(Context context, Intent intent) { deferWake(); }
    };
    private final SharedPreferences.OnSharedPreferenceChangeListener changed = (prefs, key) -> {
        if ("previous-home".equals(key) && connected) configure();
        if (ENABLED.equals(key)) {
            if (connected) configure();
            if (requested()) startKeepAlive(); else { bootPending = false; stopKeepAlive(); }
        }
    };

    static boolean connected() { HomeOverrideService service = live; return service != null && service.connected; }
    static boolean foreground() { HomeOverrideService service = live; return service != null && service.foreground; }
    static String lastStockClass() { HomeOverrideService service = live; return service == null ? "unknown" : service.lastStockClass; }
    static void homeArrived() { HomeOverrideService service = live; if (service != null) service.policy.homeArrived(); }
    private boolean requested() { return BuildConfig.HOME_ENABLED && preferences != null && preferences.getBoolean(ENABLED, false); }

    @Override public void onCreate() {
        super.onCreate();
        live = this;
        preferences = getSharedPreferences("portal-tv", MODE_PRIVATE);
        preferences.registerOnSharedPreferenceChangeListener(changed);
    }

    @Override protected void onServiceConnected() {
        if (!BuildConfig.HOME_ENABLED) { disableSelf(); return; }
        live = this; connected = true;
        try {
            configure();
            if (!receiverRegistered) {
                IntentFilter filter = new IntentFilter(Intent.ACTION_SCREEN_ON); filter.addAction(Intent.ACTION_USER_PRESENT);
                if (android.os.Build.VERSION.SDK_INT >= 33) registerReceiver(wake, filter, Context.RECEIVER_NOT_EXPORTED);
                else registerReceiver(wake, filter);
                receiverRegistered = true;
            }
            boot = Settings.Global.getInt(getContentResolver(), Settings.Global.BOOT_COUNT, -1);
            bootPending = HomeOverridePolicy.startForBoot(boot, preferences.getInt("home-override-boot", -1));
            if (requested()) { startKeepAlive(); considerWake(); deferWake(); }
        } catch (RuntimeException unavailable) { pause("connect-" + unavailable.getClass().getSimpleName()); }
    }

    private void configure() {
        ComponentName previous = ComponentName.unflattenFromString(preferences.getString("previous-home", ""));
        AccessibilityServiceInfo info = getServiceInfo();
        if (info == null) info = new AccessibilityServiceInfo();
        info.eventTypes = requested() && previous != null ? AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED : 0;
        info.feedbackType = AccessibilityServiceInfo.FEEDBACK_GENERIC;
        info.notificationTimeout = 0; info.flags = 0;
        // Other window metadata only clears one in-memory stock-Home boolean.
        // Never retrieve nodes, screen text, gestures or key streams.
        // Android 11 adds filters cumulatively; even a paused own-package filter
        // would remain after resuming. Gate delivery with eventTypes instead.
        info.packageNames = null;
        setServiceInfo(info);
    }

    private void startKeepAlive() {
        if (!connected || !requested() || !HomeIntegration.overrideAllowed(this)) return;
        try { startForegroundService(new Intent(this, HomeOverrideService.class)); }
        catch (RuntimeException unavailable) { pause("start-" + unavailable.getClass().getSimpleName()); }
    }

    @Override public int onStartCommand(Intent intent, int flags, int startId) {
        if (!requested() || !HomeIntegration.overrideAllowed(this)) { stopKeepAlive(); return START_NOT_STICKY; }
        try {
            NotificationManager manager = getSystemService(NotificationManager.class);
            NotificationChannel channel = new NotificationChannel(CHANNEL, "TV Home override", NotificationManager.IMPORTANCE_LOW);
            channel.setSound(null, null); channel.enableVibration(false); manager.createNotificationChannel(channel);
            Intent settings = new Intent(this, MainActivity.class).setAction(SETTINGS_ACTION).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP);
            PendingIntent open = PendingIntent.getActivity(this, NOTIFICATION, settings, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
            Notification notification = new Notification.Builder(this, CHANNEL).setSmallIcon(R.drawable.ic_settings)
                    .setContentTitle("Vrooli TV").setContentText(connected ? "Home override is active · open Settings to pause" : "Home override is reconnecting · open Settings to pause")
                    .setContentIntent(open).setOngoing(true).setShowWhen(false).setCategory(Notification.CATEGORY_SERVICE).build();
            startForeground(NOTIFICATION, notification);
            foreground = true;
            return START_STICKY;
        } catch (RuntimeException unavailable) { pause("foreground-" + unavailable.getClass().getSimpleName()); return START_NOT_STICKY; }
    }

    private boolean interactive() { PowerManager power = getSystemService(PowerManager.class); return power != null && power.isInteractive(); }
    private boolean unlocked() { KeyguardManager keyguard = getSystemService(KeyguardManager.class); return keyguard != null && !keyguard.isKeyguardLocked(); }

    @Override public void onAccessibilityEvent(AccessibilityEvent event) {
        if (!connected || event == null || event.getEventType() != AccessibilityEvent.TYPE_WINDOW_STATE_CHANGED) return;
        ComponentName stock = ComponentName.unflattenFromString(preferences.getString("previous-home", ""));
        if (stock != null && event.getPackageName() != null && stock.getPackageName().contentEquals(event.getPackageName()) && event.getClassName() != null) {
            String value = event.getClassName().toString();
            if (value.matches("[A-Za-z0-9_.$]{1,256}")) lastStockClass = value;
        }
        HomeOverridePolicy.Decision decision = policy.window(requested(), interactive(), unlocked(), preferences.getString("previous-home", ""),
                event.getPackageName() == null ? null : event.getPackageName().toString(),
                event.getClassName() == null ? null : event.getClassName().toString(), SystemClock.elapsedRealtime());
        handle(decision);
        if (decision == HomeOverridePolicy.Decision.IGNORE && (!interactive() || !unlocked())) deferWake();
    }

    private void deferWake() {
        if (!connected || !requested()) return;
        final int generation = ++wakeGeneration;
        for (long delay : new long[]{750, 2500}) handler.postDelayed(() -> { if (generation == wakeGeneration) considerWake(); }, delay);
    }
    private void considerWake() {
        if (!connected || !requested() || !interactive() || !unlocked()) return;
        if (bootPending) {
            bootPending = false;
            preferences.edit().putInt("home-override-boot", boot).apply();
            openHome();
        } else handle(policy.wake(true, true, true, SystemClock.elapsedRealtime()));
    }
    private void handle(HomeOverridePolicy.Decision decision) {
        if (decision == HomeOverridePolicy.Decision.PAUSE) pause("routing-burst");
        else if (decision == HomeOverridePolicy.Decision.OPEN_HOME) openHome();
    }
    private void openHome() {
        ++wakeGeneration;
        try {
            startActivity(new Intent(Intent.ACTION_MAIN).addCategory(Intent.CATEGORY_HOME)
                    .setComponent(new ComponentName(getPackageName(), "com.vrooli.portal.tv.HomeActivity"))
                    .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_CLEAR_TOP));
        } catch (RuntimeException unavailable) { pause("launch-" + unavailable.getClass().getSimpleName()); }
    }
    private void pause(String reason) { if (preferences != null) preferences.edit().putBoolean(ENABLED, false).putString("home-override-pause", reason).commit(); stopKeepAlive(); }
    private void stopKeepAlive() { foreground = false; ++wakeGeneration; stopForeground(STOP_FOREGROUND_REMOVE); stopSelf(); }
    @Override public void onInterrupt() { /* No speech, key stream or gesture work. */ }
    @Override public boolean onUnbind(Intent intent) { connected = false; stopKeepAlive(); return false; }
    @Override public void onDestroy() {
        connected = false; if (live == this) live = null;
        handler.removeCallbacksAndMessages(null);
        if (receiverRegistered) unregisterReceiver(wake);
        if (preferences != null) preferences.unregisterOnSharedPreferenceChangeListener(changed);
        super.onDestroy();
    }
}
