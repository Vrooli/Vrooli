package com.vrooli.portal.tv;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Intent;
import android.content.SharedPreferences;
import android.content.res.ColorStateList;
import android.content.pm.ResolveInfo;
import android.graphics.Typeface;
import android.graphics.drawable.Drawable;
import android.graphics.drawable.GradientDrawable;
import android.net.Uri;
import android.os.Bundle;
import android.text.TextUtils;
import android.view.Gravity;
import android.view.View;
import android.view.ViewGroup;
import android.view.inputmethod.EditorInfo;
import android.webkit.*;
import android.widget.*;
import org.json.*;
import java.net.URI;
import java.util.*;
import java.util.concurrent.*;

/** Native TV Home with optional Portal experiences. Android owns Home selection. */
public final class MainActivity extends Activity {
    private static final String LIBRARY = "Vrooli";
    private final NavigationState state = new NavigationState();
    private final ExecutorService network = Executors.newSingleThreadExecutor();
    private final ArtworkLoader artwork = new ArtworkLoader();
    private Map<String, View> targets = new LinkedHashMap<>();
    private static final class PageView {
        final FrameLayout host;
        final Map<String, View> targets = new LinkedHashMap<>();
        ScrollView scroll;
        PageView(FrameLayout host) { this.host = host; }
    }
    private final Map<String, PageView> pages = new HashMap<>();
    private final Set<String> dirtyPages = new HashSet<>();
    private final ExecutorService packageWorker = Executors.newSingleThreadExecutor();
    private boolean packagesRefreshing, catalogueRefreshing;
    private long catalogueCheckedAt;
    private String cataloguePayload = "";
    private FrameLayout pageHost;
    private TextView librarySummary, openingCaption, openingCancel;
    private LinearLayout openingPanel;
    private final Map<String, View> tabs = new HashMap<>();
    private final List<JSONObject> catalogue = new ArrayList<>();
    private SharedPreferences preferences;
    private TvApps tvApplications;
    private HomeIntegration homeIntegration;
    private PortalClient client;
    private PortalDiscovery discovery;
    private final List<String> discoveredOrigins = new ArrayList<>();
    private String discoveryStatus = "Looking for Portal on your home network…";
    private boolean discovering;
    private FrameLayout root, content, browser;
    private LinearLayout shell;
    private View libraryShade;
    private WebView web;
    private ScrollView currentScroll;
    private Future<?> pending;
    private volatile int requestVersion;
    private boolean destroyed, catalogueLoaded, staleCatalogue, webIsSignIn;
    private boolean nativeRefreshPending;
    private boolean retainTabFocus;
    private String catalogueError;
    private String server = "";
    private String connectionDraft;
    private List<String> favoriteApps;
    private List<String> recentApps;
    private Set<String> favoriteExperiences;
    private String settingsSection = BuildConfig.HOME_ENABLED ? "Home" : "Portal";
    private FrameLayout settingsPane;
    private final Map<String, TextView> settingsCategories = new LinkedHashMap<>();

    @Override public void onCreate(Bundle saved) {
        super.onCreate(saved);
        preferences = getSharedPreferences("portal-tv", MODE_PRIVATE);
        tvApplications = new TvApps(getPackageManager(), getPackageName());
        homeIntegration = new HomeIntegration(this, preferences);
        if (BuildConfig.HOME_ENABLED) homeIntegration.rememberPrevious();
        server = preferences.getString("server", "");
        client = new PortalClient(server, preferences.getBoolean("allow-home-http", false));
        discovery = new PortalDiscovery(this);
        favoriteApps = readList("favorite-apps", Arrays.asList("org.smarttube.stable", "com.netflix.ninja", "com.amazon.amazonvideo.livingroom", "com.google.android.youtube.tv"));
        recentApps = readList("recent-apps", Collections.emptyList());
        favoriteExperiences = new LinkedHashSet<>(readList("favorite-experiences", Arrays.asList("web-console", "nutrition-planner")));
        state.page = preferences.getString("last-tab", "Home");
        if (saved != null) {
            state.page = saved.getString("page", "Home");
            state.appReturn = saved.getString("appReturn");
            state.libraryReturn = saved.getString("libraryReturn", "Home");
            state.settingsReturn = saved.getString("settingsReturn", "Home");
            settingsSection = saved.getString("settingsSection", settingsSection); connectionDraft = saved.getString("connectionDraft");
            state.filter = saved.getString("filter", "");
            Bundle focus = saved.getBundle("focused");
            if (focus != null) for (String key : focus.keySet()) state.focused.put(key, focus.getString(key));
            Bundle scroll = saved.getBundle("scroll");
            if (scroll != null) for (String key : scroll.keySet()) state.scroll.put(key, scroll.getInt(key));
        }
        loadCachedCatalogue();
        if (isHomeIntent(getIntent())) state.systemHome();
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY);
        buildShell();
        render(true);
        handleConnection(getIntent());
        refreshTvApps();
        android.os.Looper.myQueue().addIdleHandler(() -> { if (!destroyed) prewarmPage(LIBRARY); return false; });
    }
    private int dp(float value) { return TvTheme.dp(this, value); }
    private TextView text(String value, int size, int color) { return TvTheme.text(this, value, size, color); }
    private LinearLayout column() { return TvTheme.column(this); }
    private LinearLayout row() {
        LinearLayout result = new LinearLayout(this);
        result.setGravity(Gravity.CENTER_VERTICAL);
        result.setClipChildren(false); result.setClipToPadding(false);
        return result;
    }
    private void buildShell() {
        root = new FrameLayout(this);
        ImageView scene = new ImageView(this);
        scene.setImageResource(R.drawable.horizon_dusk);
        scene.setScaleType(ImageView.ScaleType.CENTER_CROP);
        scene.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_NO);
        root.addView(scene, new FrameLayout.LayoutParams(-1, -1));
        View wash = new View(this);
        wash.setBackground(new GradientDrawable(GradientDrawable.Orientation.TOP_BOTTOM, new int[]{0x50071322, 0x00071322, 0xE0081320}));
        root.addView(wash, new FrameLayout.LayoutParams(-1, -1));
        libraryShade = new View(this); libraryShade.setBackgroundColor(0xE6081425);
        root.addView(libraryShade, new FrameLayout.LayoutParams(-1, -1));
        shell = column(); shell.setPadding(dp(48), dp(28), dp(48), dp(30));
        root.addView(shell, new FrameLayout.LayoutParams(-1, -1));
        LinearLayout navigation = row();
        for (String name : new String[]{"Home", "TV apps", LIBRARY, "Inputs"}) {
            TextView tab = new NavigationTab(this, name);
            tab.setGravity(Gravity.CENTER); tab.setPadding(dp(8), dp(8), dp(8), dp(8));
            tab.setFocusable(true); tab.setClickable(true); tab.setId(View.generateViewId());
            tab.setOnClickListener(v -> { if (!name.equals(state.page)) navigate(name); });
            tab.setOnKeyListener((view, key, event) -> {
                if (key != android.view.KeyEvent.KEYCODE_DPAD_LEFT && key != android.view.KeyEvent.KEYCODE_DPAD_RIGHT) return false;
                String next = NavigationState.adjacentTab(name, key == android.view.KeyEvent.KEYCODE_DPAD_LEFT ? -1 : 1);
                if (next == null) return false;
                if (event.getAction() == android.view.KeyEvent.ACTION_DOWN) { tabs.get(next).requestFocus(); navigate(next); }
                return true;
            });
            tab.setOnFocusChangeListener((v, focused) -> {
                tab.setBackground(focused ? TvTheme.surface(this, 0xAD1F3448, true) : null);
            });
            LinearLayout.LayoutParams params = new LinearLayout.LayoutParams(-2, dp(44)); params.rightMargin = dp(20);
            navigation.addView(tab, params); tabs.put(name, tab);
        }
        View space = new View(this); navigation.addView(space, new LinearLayout.LayoutParams(0, 1, 1));
        TextClock clock = new TextClock(this);
        clock.setFormat12Hour("h:mm a"); clock.setFormat24Hour("HH:mm");
        clock.setTextSize(21); clock.setTextColor(TvTheme.SECONDARY); clock.setGravity(Gravity.CENTER_VERTICAL);
        navigation.addView(clock, new LinearLayout.LayoutParams(-2, dp(44)));
        ImageButton settings = new ImageButton(this); settings.setId(View.generateViewId()); settings.setImageResource(R.drawable.ic_settings);
        settings.setContentDescription("Settings"); settings.setFocusable(true); settings.setClickable(true); settings.setPadding(dp(10), dp(10), dp(10), dp(10));
        settings.setBackground(null); settings.setStateListAnimator(null);
        settings.setImageTintList(new ColorStateList(new int[][]{new int[]{android.R.attr.state_focused}, new int[]{android.R.attr.state_selected}, new int[]{}}, new int[]{TvTheme.FOCUS, TvTheme.ACCENT, TvTheme.SECONDARY}));
        settings.setOnFocusChangeListener((v, focused) -> v.setForeground(focused ? TvTheme.surface(this, android.graphics.Color.TRANSPARENT, true) : null));
        settings.setOnClickListener(v -> navigate("Settings"));
        LinearLayout.LayoutParams sp = new LinearLayout.LayoutParams(dp(44), dp(44)); sp.leftMargin = dp(20);
        navigation.addView(settings, sp); tabs.put("Settings", settings);
        shell.addView(navigation, new LinearLayout.LayoutParams(-1, dp(46)));
        content = new FrameLayout(this); content.setClipChildren(true); content.setClipToPadding(true);
        LinearLayout.LayoutParams cp = new LinearLayout.LayoutParams(-1, 0, 1); cp.topMargin = dp(16);
        shell.addView(content, cp); root.setClipChildren(false);
        setContentView(root);
    }
    private void savePosition() {
        if (currentScroll != null) state.scroll.put(state.page, currentScroll.getScrollY());
        if (!"Settings".equals(state.page)) preferences.edit().putString("last-tab", state.page).apply();
    }
    private void navigate(String destination) {
        long started = android.os.SystemClock.elapsedRealtimeNanos();
        savePosition(); state.navigate(destination);
        if ("Settings".equals(destination) || "Inputs".equals(destination)) dirtyPages.add(destination);
        showPage(true);
        UiTimings.record(destination, android.os.SystemClock.elapsedRealtimeNanos() - started);
    }
    private void render(boolean refresh) {
        dirtyPages.add(state.page); showPage(refresh);
    }
    private void showPage(boolean refresh) {
        if (destroyed) return;
        View focusedTab = getCurrentFocus() instanceof NavigationTab ? tabs.get(state.page) : null;
        retainTabFocus = focusedTab != null;
        PageView previous = null;
        for (PageView item : pages.values()) if (item.host.getVisibility() == View.VISIBLE) { previous = item; break; }
        if (previous != null) previous.scroll = currentScroll;
        if (dirtyPages.remove(state.page)) {
            PageView old = pages.remove(state.page);
            if (old != null) content.removeView(old.host);
        }
        for (PageView item : pages.values()) item.host.setVisibility(View.INVISIBLE);
        PageView page = pages.get(state.page);
        boolean cached = page != null;
        if (!cached) { page = new PageView(new FrameLayout(this)); pages.put(state.page, page); content.addView(page.host, new FrameLayout.LayoutParams(-1, -1)); }
        pageHost = page.host; targets = page.targets; currentScroll = page.scroll; pageHost.setVisibility(View.VISIBLE);
        libraryShade.setVisibility("Home".equals(state.page) ? View.GONE : View.VISIBLE);
        for (Map.Entry<String, View> item : tabs.entrySet()) {
            boolean selected = item.getKey().equals(state.page);
            if (item.getValue() instanceof TextView) {
                TextView label = (TextView) item.getValue(); label.setTextColor(selected ? TvTheme.TEXT : TvTheme.SECONDARY);
                label.setTypeface(null, selected ? Typeface.BOLD : Typeface.NORMAL);
            }
            item.getValue().setSelected(selected);
        }
        if (!cached) {
            if ("Home".equals(state.page)) home();
            else if ("TV apps".equals(state.page)) tvApps();
            else if (LIBRARY.equals(state.page)) library(false);
            else if ("Inputs".equals(state.page)) inputs();
            else settings(null);
            page.scroll = currentScroll;
        } else restoreFocus(null);
        if (focusedTab != null) focusedTab.requestFocus();
        View selectedTab = tabs.get(state.page);
        if (selectedTab != null && !targets.isEmpty()) selectedTab.setNextFocusDownId(targets.values().iterator().next().getId());
        retainTabFocus = false;
        UiTimings.page(cached); UiTimings.surface(state.page, browser != null, openingPanel != null && openingPanel.getVisibility() == View.VISIBLE);
        if (refresh && LIBRARY.equals(state.page)) content.post(() -> {
            if (destroyed || !LIBRARY.equals(state.page) || browser != null) return;
            if (!catalogueLoaded || catalogueCheckedAt == 0 || android.os.SystemClock.elapsedRealtime() - catalogueCheckedAt >= (catalogueError == null ? 60000 : 5000)) {
                beginDiscovery(); if (!server.isEmpty()) refreshCatalogue();
            }
        });
    }
    private void prewarmPage(String name) {
        if (destroyed || browser != null || name.equals(state.page) || (pages.containsKey(name) && !dirtyPages.contains(name))) return;
        String previousPage = state.page; FrameLayout previousHost = pageHost; Map<String, View> previousTargets = targets; ScrollView previousScroll = currentScroll;
        boolean previousRetain = retainTabFocus;
        retainTabFocus = true;
        PageView old = pages.remove(name); if (old != null) content.removeView(old.host);
        PageView page = new PageView(new FrameLayout(this)); page.host.setVisibility(View.INVISIBLE);
        pages.put(name, page); content.addView(page.host, new FrameLayout.LayoutParams(-1, -1));
        state.page = name; pageHost = page.host; targets = page.targets; currentScroll = null;
        try {
            if (LIBRARY.equals(name)) library(false); else tvApps();
            page.scroll = currentScroll; dirtyPages.remove(name); UiTimings.page(false);
        } finally {
            state.page = previousPage; pageHost = previousHost; targets = previousTargets; currentScroll = previousScroll;
            retainTabFocus = previousRetain;
        }
    }
    private void refreshTvApps() {
        if (destroyed || packagesRefreshing) return;
        packagesRefreshing = true;
        packageWorker.execute(() -> {
            boolean changed = false;
            try { changed = tvApplications.refresh(); } catch (RuntimeException ignored) { /* Retain the last installed snapshot. */ }
            final boolean updated = changed;
            runOnUiThread(() -> {
                packagesRefreshing = false; if (destroyed) return;
                if (updated || nativeRefreshPending) {
                    dirtyPages.add("Home"); if (updated) dirtyPages.add("TV apps"); nativeRefreshPending = false;
                    if (browser == null && dirtyPages.contains(state.page) && ("Home".equals(state.page) || "TV apps".equals(state.page))) { savePosition(); showPage(false); }
                    android.os.Looper.myQueue().addIdleHandler(() -> { if (!destroyed) prewarmPage("TV apps"); return false; });
                }
            });
        });
    }
    private View remember(View view, String id, String label) {
        final String page = state.page;
        view.setId(View.generateViewId()); view.setTag(id);
        view.setFocusable(true); view.setClickable(true); view.setContentDescription(label);
        view.setOnFocusChangeListener((v, focused) -> {
            TvTheme.focus(v, focused);
            if (focused) state.focused.put(page, id);
        });
        view.setBackground(TvTheme.surface(this, TvTheme.SURFACE, false));
        targets.put(id, view);
        return view;
    }
    private void restoreFocus(String fallback) {
        if (retainTabFocus) return;
        String id = state.focused.get(state.page);
        View target = targets.get(id);
        if (target == null) target = targets.get(fallback);
        if (target == null && !targets.isEmpty()) target = targets.values().iterator().next();
        if (target != null) target.requestFocus();
        if (currentScroll != null) {
            int y = state.scroll.getOrDefault(state.page, 0);
            ScrollView scroll = currentScroll;
            scroll.post(() -> { if (currentScroll == scroll) scroll.scrollTo(0, y); });
        }
    }
    private List<ResolveInfo> installedApps() {
        return tvApplications.installed();
    }
    private LinearLayout appTile(ResolveInfo app, boolean compact) {
        String identity = app.activityInfo.packageName;
        String label = tvApplications.label(app);
        LinearLayout tile = column(); tile.setGravity(Gravity.CENTER); tile.setDescendantFocusability(ViewGroup.FOCUS_BLOCK_DESCENDANTS);
        tile.setPadding(dp(12), dp(10), dp(12), dp(10));
        ImageView image = new ImageView(this); image.setScaleType(ImageView.ScaleType.FIT_CENTER);
        Drawable banner = tvApplications.artwork(app, getResources());
        image.setImageDrawable(banner);
        FrameLayout art = new FrameLayout(this); art.addView(image, new FrameLayout.LayoutParams(-1, -1));
        if (favoriteApps.contains(identity)) {
            ImageView favorite = new ImageView(this); favorite.setImageResource(R.drawable.ic_favorite);
            favorite.setImageTintList(ColorStateList.valueOf(TvTheme.ACCENT)); favorite.setContentDescription("Favorite");
            art.addView(favorite, new FrameLayout.LayoutParams(dp(18), dp(18), Gravity.TOP | Gravity.RIGHT));
        }
        tile.addView(art, new LinearLayout.LayoutParams(-1, dp(compact ? 44 : 42)));
        TextView name = text(label, compact ? 18 : 17, TvTheme.TEXT); name.setGravity(Gravity.CENTER); name.setMaxLines(1); name.setEllipsize(TextUtils.TruncateAt.END);
        LinearLayout.LayoutParams np = new LinearLayout.LayoutParams(-1, -2); np.topMargin = dp(7); tile.addView(name, np);
        remember(tile, "app:" + identity, label + (favoriteApps.contains(identity) ? ", favorite" : "") + ", hold OK for options");
        tile.setOnLongClickListener(v -> { appOptions(app); return true; });
        tile.setOnKeyListener((v, key, event) -> {
            if (key != android.view.KeyEvent.KEYCODE_MENU) return false;
            if (event.getAction() == android.view.KeyEvent.ACTION_UP) appOptions(app);
            return true;
        });
        tile.setOnClickListener(v -> {
            try {
                launchExternal(tvApplications.launch(app));
                nativeRefreshPending = true;
                recentApps.remove(identity); recentApps.add(0, identity);
                while (recentApps.size() > 128) recentApps.remove(recentApps.size() - 1);
                saveList("recent-apps", recentApps);
                dirtyPages.add("Home");
            }
            catch (Exception ignored) { notice("This app is unavailable. Your TV apps list will refresh when you return."); }
        });
        return tile;
    }
    private void home() {
        LinearLayout body = column(); body.setGravity(Gravity.BOTTOM);
        TextView title = text("Your apps", 18, TvTheme.TEXT); title.setContentDescription("Your apps"); title.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_YES);
        if (android.os.Build.VERSION.SDK_INT >= 28) title.setAccessibilityHeading(true);
        title.setPadding(dp(6), 0, 0, dp(14)); body.addView(title);
        HorizontalScrollView strip = new HorizontalScrollView(this); strip.setHorizontalScrollBarEnabled(false); strip.setClipToPadding(false); strip.setSmoothScrollingEnabled(true);
        LinearLayout apps = row(); apps.setPadding(dp(6), dp(5), dp(6), dp(5));
        List<ResolveInfo> installed = installedApps();
        Map<String, ResolveInfo> byPackage = new LinkedHashMap<>(); for (ResolveInfo app : installed) byPackage.put(app.activityInfo.packageName, app);
        List<String> identities = NavigationState.homeApps(new ArrayList<>(byPackage.keySet()), recentApps, favoriteApps);
        for (String identity : identities) {
            LinearLayout.LayoutParams tp = new LinearLayout.LayoutParams(dp(176), dp(100)); tp.rightMargin = dp(16);
            apps.addView(appTile(byPackage.get(identity), true), tp);
        }
        apps.addView(homeDestination("All TV apps", "all-tv", android.R.drawable.ic_menu_agenda, () -> navigate("TV apps")), homeCardParams());
        apps.addView(homeDestination("All Vrooli apps", "all-vrooli", R.drawable.portal, () -> {
            state.filter = ""; state.focused.remove(LIBRARY); state.scroll.put(LIBRARY, 0); dirtyPages.add(LIBRARY); navigate(LIBRARY);
        }), homeCardParams());
        for (int index = 0; index < apps.getChildCount(); index++) {
            View tile = apps.getChildAt(index);
            tile.setNextFocusLeftId(apps.getChildAt(Math.max(0, index - 1)).getId());
            tile.setNextFocusRightId(apps.getChildAt(Math.min(apps.getChildCount() - 1, index + 1)).getId());
        }
        strip.addView(apps, new HorizontalScrollView.LayoutParams(-2, -1));
        body.addView(strip, new LinearLayout.LayoutParams(-1, dp(114)));
        pageHost.addView(body, new FrameLayout.LayoutParams(-1, -1));
        wireUpToNavigation(apps);
        if (identities.isEmpty()) { if (!retainTabFocus) tabs.get("Home").requestFocus(); }
        else restoreFocus("app:" + identities.get(0));
    }
    private LinearLayout.LayoutParams homeCardParams() {
        LinearLayout.LayoutParams value = new LinearLayout.LayoutParams(dp(176), dp(100)); value.rightMargin = dp(16); return value;
    }
    private View homeDestination(String label, String identity, int icon, Runnable open) {
        LinearLayout tile = column(); tile.setGravity(Gravity.CENTER); tile.setPadding(dp(10), dp(10), dp(10), dp(10));
        ImageView mark = new ImageView(this); mark.setImageResource(icon); mark.setScaleType(ImageView.ScaleType.FIT_CENTER); tile.addView(mark, new LinearLayout.LayoutParams(dp(44), dp(44)));
        TextView caption = text(label, 18, TvTheme.TEXT); caption.setGravity(Gravity.CENTER); tile.addView(caption);
        remember(tile, identity, label); tile.setOnClickListener(v -> open.run()); return tile;
    }
    private void heading(LinearLayout body, String title, String description) {
        TextView name = text(title, 28, TvTheme.TEXT); body.addView(name);
        if (description != null) { TextView subtitle = text(description, 16, TvTheme.SECONDARY); subtitle.setPadding(0, dp(6), 0, dp(12)); body.addView(subtitle); }
    }
    private ScrollView grid(LinearLayout body, List<View> cards, int columns, int height) {
        return grid(body, cards.size(), cards::get, columns, height, null);
    }
    private ScrollView grid(LinearLayout body, int count, java.util.function.IntFunction<View> createCard, int columns, int height, CatalogueWindow window) {
        // The parent controls whether the ScrollView's render node clips to its
        // bounds. TvTheme.column disables it for focus effects; grid pages need
        // a real viewport so offscreen rows cannot cover the fixed controls.
        body.setClipChildren(true); body.setClipToPadding(true);
        ScrollView scroll = new ScrollView(this); scroll.setClipChildren(true); scroll.setClipToPadding(true); scroll.setFillViewport(false); scroll.setVerticalScrollBarEnabled(false);
        LinearLayout rows = column(); rows.setPadding(dp(6), dp(6), dp(6), dp(8));
        List<View> cards = new ArrayList<>();
        Runnable append = new Runnable() {
            @Override public void run() {
                if (window != null && !cards.isEmpty()) window.append();
                int end = window == null ? count : window.end();
                for (int offset = cards.size(); offset < end; offset += columns) {
                    LinearLayout line = row();
                    for (int col = 0; col < columns; col++) {
                        int index = offset + col;
                        View item = index < end ? createCard.apply(index) : new View(MainActivity.this);
                        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(0, dp(height), 1); if (col < columns - 1) lp.rightMargin = dp(16);
                        line.addView(item, lp);
                        if (index < end) {
                            cards.add(item);
                            View.OnFocusChangeListener previous = item.getOnFocusChangeListener();
                            item.setOnFocusChangeListener((view, focused) -> {
                                if (previous != null) previous.onFocusChange(view, focused);
                                if (focused && window != null && window.hasMore() && index >= cards.size() - columns * 2) run();
                            });
                        }
                    }
                    LinearLayout.LayoutParams rp = new LinearLayout.LayoutParams(-1, -2); rp.bottomMargin = dp(14); rows.addView(line, rp);
                }
                // Update the old last row as well, so D-pad Down crosses each batch boundary.
                for (int i = 0; i < cards.size(); i++) {
                    View item = cards.get(i);
                    if (i % columns > 0) item.setNextFocusLeftId(cards.get(i - 1).getId());
                    if (i % columns + 1 < columns && i + 1 < cards.size()) item.setNextFocusRightId(cards.get(i + 1).getId());
                    if (i >= columns) item.setNextFocusUpId(cards.get(i - columns).getId());
                    if (i + columns < cards.size()) item.setNextFocusDownId(cards.get(i + columns).getId());
                }
            }
        };
        append.run();
        if (window != null) scroll.setOnScrollChangeListener((view, x, y, oldX, oldY) -> {
            if (currentScroll == scroll && window.hasMore() && y + scroll.getHeight() >= rows.getHeight() - dp((height + 14) * 2)) append.run();
        });
        scroll.addView(rows); body.addView(scroll, new LinearLayout.LayoutParams(-1, 0, 1)); currentScroll = scroll;
        return scroll;
    }
    private void wireUpToNavigation(ViewGroup group) {
        View tab = tabs.get(state.page);
        for (int i = 0; i < group.getChildCount(); i++) if (group.getChildAt(i).isFocusable() && tab != null) group.getChildAt(i).setNextFocusUpId(tab.getId());
    }
    private void tvApps() {
        LinearLayout body = column(); List<View> cards = new ArrayList<>();
        for (ResolveInfo app : NavigationState.libraryApps(installedApps(), value -> value.activityInfo.packageName, tvApplications::label, favoriteApps)) cards.add(appTile(app, false));
        grid(body, cards, 5, 98); pageHost.addView(body, new FrameLayout.LayoutParams(-1, -1));
        for (int i = 0; i < Math.min(5, cards.size()); i++) cards.get(i).setNextFocusUpId(tabs.get("TV apps").getId());
        if (!cards.isEmpty()) restoreFocus((String) cards.get(0).getTag());
    }
    private void appOptions(ResolveInfo app) {
        String identity = app.activityInfo.packageName;
        boolean favorite = favoriteApps.contains(identity);
        boolean system = (app.activityInfo.applicationInfo.flags & android.content.pm.ApplicationInfo.FLAG_SYSTEM) != 0;
        List<String> actions = new ArrayList<>(Arrays.asList(favorite ? "Remove from favorites" : "Add to favorites", "App info / Force stop"));
        if (!system) actions.add("Uninstall…");
        AlertDialog options = new AlertDialog.Builder(this).setTitle(app.loadLabel(getPackageManager())).setItems(actions.toArray(new String[0]), (dialog, which) -> {
            if (which == 0) {
                if (favorite) favoriteApps.remove(identity); else favoriteApps.add(identity);
                saveList("favorite-apps", favoriteApps); dirtyPages.add("Home"); dirtyPages.add("TV apps"); savePosition(); render(false);
            } else {
                try {
                    launchExternal(new Intent(which == 1 ? android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS : Intent.ACTION_UNINSTALL_PACKAGE, Uri.parse("package:" + identity)));
                } catch (Exception ignored) {
                    try { launchExternal(new Intent(android.provider.Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:" + identity))); }
                    catch (Exception unavailable) { notice("Open TV Settings → Apps to manage this app."); }
                }
            }
        }).setNegativeButton("Close", null).create();
        options.setOnShowListener(dialog -> {
            if (options.getWindow() != null) options.getWindow().setBackgroundDrawable(TvTheme.surface(this, TvTheme.INK, false));
            options.getListView().setSelection(0); options.getListView().requestFocus();
        });
        options.show();
    }
    private void library(boolean refresh) {
        LinearLayout body = column();
        LinearLayout toolbar = row();
        EditText search = new EditText(this); search.setSingleLine(true); search.setTextColor(TvTheme.TEXT); search.setTextSize(16); search.setHintTextColor(TvTheme.SECONDARY); search.setHint("Find an experience"); search.setContentDescription("Find an experience"); search.setText(state.filter); search.setSelectAllOnFocus(true); search.setImeOptions(EditorInfo.IME_ACTION_SEARCH);
        LinearLayout.LayoutParams ip = new LinearLayout.LayoutParams(0, dp(42), 1); ip.leftMargin = dp(12); toolbar.addView(search, ip);
        Runnable find = () -> { hideKeyboard(search); state.filter = search.getText().toString().trim(); state.scroll.put(LIBRARY, 0); state.focused.remove(LIBRARY); render(false); };
        search.setOnEditorActionListener((v, action, event) -> { if (action == EditorInfo.IME_ACTION_SEARCH) { find.run(); return true; } return false; });
        TextView findButton = TvTheme.action(this, "Find", find); LinearLayout.LayoutParams fp = new LinearLayout.LayoutParams(dp(68), dp(38)); fp.leftMargin = dp(8); toolbar.addView(findButton, fp);
        TextView clear = TvTheme.action(this, "Clear", () -> { search.setText(""); find.run(); }); clear.setContentDescription("Clear search");
        LinearLayout.LayoutParams clearParams = new LinearLayout.LayoutParams(dp(72), dp(38)); clearParams.leftMargin = dp(8); toolbar.addView(clear, clearParams);
        body.addView(toolbar);
        List<JSONObject> matches = new ArrayList<>();
        String query = state.filter.toLowerCase(Locale.ROOT);
        for (JSONObject app : NavigationState.libraryApps(catalogue, value -> value.optString("scenario"), value -> value.optString("label"), favoriteExperiences)) {
            String identity = app.optString("scenario"), label = app.optString("label");
            if (!(label + " " + app.optString("description") + " " + identity).toLowerCase(Locale.ROOT).contains(query)) continue;
            matches.add(app);
        }
        String status;
        if (server.isEmpty()) status = discoveryStatus;
        else if (catalogueError != null) status = catalogueError + (catalogueLoaded ? " Showing saved apps." : "");
        else if (!catalogueLoaded) status = "Connecting to Portal…";
        else if (matches.isEmpty()) status = "No matches";
        else status = (server.startsWith("http://") ? "Home LAN · unencrypted fallback · " : "Secure Portal · ") + (staleCatalogue ? "Saved library · reconnect to check availability" : "Connected");
        librarySummary = text(status, 14, TvTheme.SECONDARY); librarySummary.setPadding(dp(3), dp(12), 0, dp(9)); body.addView(librarySummary);
        int restoredIndex = -1;
        for (int i = 0; i < matches.size(); i++) if (("experience:" + matches.get(i).optString("scenario")).equals(state.focused.get(LIBRARY))) restoredIndex = i;
        grid(body, matches.size(), index -> experienceTile(matches.get(index)), 3, 108, new CatalogueWindow(matches.size(), restoredIndex));
        LinearLayout footer = row();
        TextView retry = TvTheme.action(this, server.isEmpty() ? "Find Portal" : "Refresh", () -> { beginDiscovery(); if (!server.isEmpty()) { cancelRequest(); catalogueError = null; refreshCatalogue(); } });
        footer.addView(retry, new LinearLayout.LayoutParams(-2, dp(36)));
        TextView connectionSettings = TvTheme.action(this, "Connection", () -> navigate("Settings"));
        LinearLayout.LayoutParams csp = new LinearLayout.LayoutParams(-2, dp(36)); csp.leftMargin = dp(10); footer.addView(connectionSettings, csp);
        if (catalogueError != null && !server.isEmpty()) {
            TextView login = TvTheme.action(this, "Open Portal", () -> openWeb(server, true)); LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(-2, dp(36)); lp.leftMargin = dp(12); footer.addView(login, lp);
        }
        body.addView(footer);
        pageHost.addView(body, new FrameLayout.LayoutParams(-1, -1));
        if (!matches.isEmpty()) restoreFocus("experience:" + matches.get(0).optString("scenario")); else if (!retainTabFocus) retry.requestFocus();
    }
    private View experienceTile(JSONObject app) {
        String identity = app.optString("scenario"), label = app.optString("label", identity);
        PageView owner = pages.get(LIBRARY); PortalClient connection = client;
        LinearLayout card = row(); card.setPadding(dp(16), dp(12), dp(16), dp(12)); card.setDescendantFocusability(ViewGroup.FOCUS_BLOCK_DESCENDANTS);
        FrameLayout art = new FrameLayout(this);
        TextView letter = text(label.isEmpty() ? "V" : label.substring(0, 1).toUpperCase(Locale.ROOT), 30, TvTheme.ACCENT); letter.setGravity(Gravity.CENTER); art.addView(letter, new FrameLayout.LayoutParams(-1, -1));
        ImageView image = new ImageView(this); image.setScaleType(ImageView.ScaleType.FIT_CENTER); art.addView(image, new FrameLayout.LayoutParams(-1, -1));
        if ("web-console".equals(identity)) { image.setImageResource(R.drawable.aquila); letter.setVisibility(View.GONE); }
        String icon = app.optString("iconPath");
        if (!icon.isEmpty()) artwork.load(connection, icon, bitmap -> { if (!destroyed && pages.get(LIBRARY) == owner && client == connection) { image.setImageBitmap(bitmap); letter.setVisibility(View.GONE); } });
        card.addView(art, new LinearLayout.LayoutParams(dp(48), dp(48)));
        LinearLayout details = column(); LinearLayout.LayoutParams dp = new LinearLayout.LayoutParams(0, -2, 1); dp.leftMargin = dp(14); card.addView(details, dp);
        TextView title = text(label, 18, TvTheme.TEXT); title.setTypeface(null, Typeface.BOLD); title.setMaxLines(1); title.setEllipsize(TextUtils.TruncateAt.END); details.addView(title);
        if (icon.isEmpty() && !"web-console".equals(identity)) {
            card.removeView(art); dp.leftMargin = 0; details.setLayoutParams(dp); title.setTextSize(22);
        }
        String description = app.optString("description");
        TextView subtitle = text(description, 14, TvTheme.SECONDARY); subtitle.setMaxLines(1); subtitle.setEllipsize(TextUtils.TruncateAt.END); subtitle.setPadding(0, dp(5), 0, dp(5)); details.addView(subtitle);
        boolean running = "running".equals(app.optString("runtimeState"));
        String availability = staleCatalogue ? "Saved · availability unknown" : running ? "" : "Not running";
        if (!availability.isEmpty()) details.addView(text(availability, 12, TvTheme.SECONDARY));
        FrameLayout tile = new FrameLayout(this); tile.addView(card, new FrameLayout.LayoutParams(-1, -1));
        tile.setDescendantFocusability(ViewGroup.FOCUS_BLOCK_DESCENDANTS);
        if (favoriteExperiences.contains(identity)) {
            ImageView favorite = new ImageView(this); favorite.setImageResource(R.drawable.ic_favorite);
            favorite.setImageTintList(ColorStateList.valueOf(TvTheme.ACCENT)); favorite.setContentDescription("Favorite");
            FrameLayout.LayoutParams star = new FrameLayout.LayoutParams(dp(18), dp(18), Gravity.TOP | Gravity.RIGHT); star.topMargin = dp(8); star.rightMargin = dp(8); tile.addView(favorite, star);
            details.setPadding(0, 0, dp(16), 0);
        }
        remember(tile, "experience:" + identity, (running ? "Open " : "") + label + (favoriteExperiences.contains(identity) ? ", favorite" : "") + (running ? "" : ", Not running"));
        if (android.os.Build.VERSION.SDK_INT >= 30 && !availability.isEmpty()) tile.setStateDescription(availability);
        tile.setOnClickListener(v -> {
            if (!running) { notice("This experience is not running. Open it on your computer first."); return; }
            savePosition(); openApplication(identity);
        });
        tile.setOnLongClickListener(v -> {
            experienceOptions(app); return true;
        });
        return tile;
    }
    private void experienceOptions(JSONObject app) {
        String identity = app.optString("scenario");
        boolean pinned = favoriteExperiences.contains(identity);
        new AlertDialog.Builder(this).setTitle(app.optString("label")).setItems(new String[]{pinned ? "Remove from favorites" : "Add to favorites"}, (dialog, which) -> {
            if (pinned) favoriteExperiences.remove(identity); else favoriteExperiences.add(identity);
            saveList("favorite-experiences", new ArrayList<>(favoriteExperiences)); render(false);
        }).setNegativeButton("Cancel", null).show();
    }
    private void beginDiscovery() {
        if (discovering || destroyed) return;
        discovering = true; discoveryStatus = "Looking for Portal on your home network…";
        discovery.start((origins, status) -> {
            if (destroyed) return;
            discovering = false; discoveredOrigins.clear(); discoveredOrigins.addAll(origins); discoveryStatus = status;
            String paired = preferences.getString("home-server", server);
            List<String> matches = new ArrayList<>();
            for (String origin : origins) if (PortalOrigin.sameHost(paired, origin) || (server.startsWith("http://") && PortalOrigin.secureUpgrade(paired, origin))) matches.add(origin);
            if (matches.isEmpty() && preferences.getBoolean("allow-home-http", false) && server.startsWith("http://")) for (String origin : origins) if (PortalOrigin.sameLocalFallback(paired, origin)) matches.add(origin);
            if (browser == null && LIBRARY.equals(state.page) && matches.size() == 1 && !server.equals(matches.get(0))) {
                cancelRequest(); fetchCatalogue(new PortalClient(matches.get(0), preferences.getBoolean("allow-home-http", false)), true);
            }
            if (browser == null && "Settings".equals(state.page) && !(getCurrentFocus() instanceof EditText)) { savePosition(); render(false); }
            else if (browser == null && LIBRARY.equals(state.page) && server.isEmpty() && librarySummary != null) librarySummary.setText(discoveryStatus);
        });
    }
    private void selectPortal(String origin, boolean home) {
        if (!server.equals(origin)) {
            cancelRequest(); server = origin; client = new PortalClient(origin, preferences.getBoolean("allow-home-http", false));
            catalogue.clear(); catalogueLoaded = false; staleCatalogue = false; catalogueError = null;
            dirtyPages.add(LIBRARY); catalogueCheckedAt = 0;
            preferences.edit().putString("server", origin).apply(); loadCachedCatalogue();
        }
        if (home) preferences.edit().putString("home-server", origin).apply();
    }
    private void refreshCatalogue() { if (!catalogueRefreshing) fetchCatalogue(client, false); }
    private void fetchCatalogue(PortalClient connection, boolean adopt) {
        final int generation = ++requestVersion;
        catalogueRefreshing = true;
        pending = network.submit(() -> {
            try {
                JSONArray values = connection.rpc("ListApplications", new JSONObject()).optJSONArray("applications");
                if (values == null || values.length() > 512) throw new JSONException("invalid_catalogue");
                List<JSONObject> apps = new ArrayList<>(); for (int i = 0; i < values.length(); i++) apps.add(values.getJSONObject(i));
                apps.sort((a, b) -> a.optString("label").compareToIgnoreCase(b.optString("label")));
                String payload = values.toString();
                runOnUiThread(() -> {
                    if (destroyed || generation != requestVersion) return;
                    catalogueRefreshing = false; catalogueCheckedAt = android.os.SystemClock.elapsedRealtime();
                    if (adopt) selectPortal(connection.origin, true);
                    boolean changed = !catalogueLoaded || staleCatalogue || catalogueError != null || !payload.equals(cataloguePayload);
                    savePosition(); catalogue.clear(); catalogue.addAll(apps); catalogueLoaded = true; staleCatalogue = false; catalogueError = null;
                    cataloguePayload = payload;
                    preferences.edit().putString("catalogue-origin", connection.origin).putString("catalogue", payload).apply();
                    if (changed) {
                        dirtyPages.add(LIBRARY);
                        if (LIBRARY.equals(state.page) && browser == null) showPage(false);
                        else android.os.Looper.myQueue().addIdleHandler(() -> { if (!destroyed) prewarmPage(LIBRARY); return false; });
                    }
                });
            } catch (Exception error) {
                String message = failure(error);
                runOnUiThread(() -> { if (!destroyed && generation == requestVersion) { catalogueRefreshing = false; catalogueCheckedAt = android.os.SystemClock.elapsedRealtime(); if (adopt) { discoveryStatus = "The discovered connection could not be verified. Keeping your saved connection."; } else { catalogueError = message; staleCatalogue = catalogueLoaded; } dirtyPages.add(LIBRARY); if (LIBRARY.equals(state.page) && browser == null) showPage(false); } });
            }
        });
    }
    private void inputs() {
        LinearLayout body = column();
        List<View> cards = new ArrayList<>();
        android.media.tv.TvInputManager manager = (android.media.tv.TvInputManager) getSystemService(TV_INPUT_SERVICE);
        List<android.media.tv.TvInputInfo> sources = manager == null ? new ArrayList<>() : new ArrayList<>(manager.getTvInputList());
        sources.sort(Comparator.comparingInt(android.media.tv.TvInputInfo::getType).reversed().thenComparing(input -> input.loadLabel(this).toString(), String.CASE_INSENSITIVE_ORDER));
        for (android.media.tv.TvInputInfo input : sources) {
            if (!input.isPassthroughInput() || input.isHidden(this)) continue;
            String label = input.loadLabel(this).toString();
            int inputState = manager.getInputState(input.getId());
            boolean connected = inputState == android.media.tv.TvInputManager.INPUT_STATE_CONNECTED;
            String connection = connected ? "Connected" : inputState == android.media.tv.TvInputManager.INPUT_STATE_CONNECTED_STANDBY ? "Standby" : "Disconnected";
            LinearLayout card = column(); card.setGravity(Gravity.CENTER); card.setPadding(dp(18), dp(14), dp(18), dp(14)); card.setDescendantFocusability(ViewGroup.FOCUS_BLOCK_DESCENDANTS);
            ImageView icon = new ImageView(this); icon.setImageResource(input.getType() == android.media.tv.TvInputInfo.TYPE_HDMI ? R.drawable.ic_hdmi : R.drawable.ic_input);
            icon.setImageTintList(ColorStateList.valueOf(connected ? TvTheme.ACCENT : TvTheme.SECONDARY));
            card.addView(icon, new LinearLayout.LayoutParams(dp(44), dp(44)));
            TextView name = text(label, 20, TvTheme.TEXT); name.setGravity(Gravity.CENTER); name.setSingleLine(true); name.setEllipsize(TextUtils.TruncateAt.END);
            LinearLayout.LayoutParams np = new LinearLayout.LayoutParams(-1, -2); np.topMargin = dp(12); card.addView(name, np);
            TextView status = text(connection, 14, TvTheme.SECONDARY); status.setPadding(0, dp(6), 0, 0); card.addView(status);
            remember(card, "input:" + input.getId(), label + ", " + connection.toLowerCase(Locale.ROOT));
            card.setOnClickListener(v -> { try { launchExternal(new Intent(Intent.ACTION_VIEW, android.media.tv.TvContract.buildChannelUriForPassthroughInput(input.getId()))); } catch (Exception ignored) { notice("Use the Input button on your TV remote."); } });
            cards.add(card);
        }
        if (cards.isEmpty()) body.addView(text("Your remote’s Input button remains available.", 18, TvTheme.SECONDARY));
        grid(body, cards, 4, 150); pageHost.addView(body, new FrameLayout.LayoutParams(-1, -1));
        for (int i = 0; i < Math.min(4, cards.size()); i++) cards.get(i).setNextFocusUpId(tabs.get("Inputs").getId());
        restoreFocus(null);
    }
    private void settings(String proposed) {
        if (proposed != null) { connectionDraft = proposed; settingsSection = "Portal"; state.focused.put("Settings", "settings-category:Portal"); }
        LinearLayout layout = row(); layout.setGravity(Gravity.TOP);
        LinearLayout sidebar = column(); sidebar.setPadding(dp(4), dp(6), dp(20), 0);
        TextView title = text("Settings", 23, TvTheme.TEXT); title.setPadding(dp(12), 0, 0, dp(24)); sidebar.addView(title);
        settingsPane = new FrameLayout(this); settingsCategories.clear();
        for (String section : BuildConfig.HOME_ENABLED ? new String[]{"Home", "Portal", "Preferences"} : new String[]{"Portal", "Preferences"}) {
            TextView category = TvTheme.action(this, section, () -> showSettingsSection(section));
            category.setGravity(Gravity.LEFT | Gravity.CENTER_VERTICAL);
            remember(category, "settings-category:" + section, section);
            category.setOnFocusChangeListener((view, focused) -> {
                TvTheme.focus(view, focused);
                if (focused) { state.focused.put("Settings", "settings-category:" + section); if (!section.equals(settingsSection)) showSettingsSection(section); }
            });
            LinearLayout.LayoutParams cp = new LinearLayout.LayoutParams(-1, dp(46)); cp.bottomMargin = dp(10); sidebar.addView(category, cp);
            settingsCategories.put(section, category);
        }
        sidebar.addView(new View(this), new LinearLayout.LayoutParams(1, 0, 1));
        TextView version = text("Vrooli TV  " + BuildConfig.VERSION_NAME, 13, TvTheme.SECONDARY); version.setPadding(dp(12), 0, 0, dp(8)); sidebar.addView(version);
        layout.addView(sidebar, new LinearLayout.LayoutParams(dp(180), -1));
        LinearLayout.LayoutParams pp = new LinearLayout.LayoutParams(0, -1, 1); pp.leftMargin = dp(26); layout.addView(settingsPane, pp);
        pageHost.addView(layout, new FrameLayout.LayoutParams(-1, -1));
        showSettingsSection(settingsSection);
        restoreFocus("settings-category:" + settingsSection);
    }
    private void showSettingsSection(String section) {
        settingsSection = section; targets.entrySet().removeIf(entry -> entry.getKey().startsWith("settings-control:"));
        for (Map.Entry<String, TextView> entry : settingsCategories.entrySet()) {
            boolean selected = entry.getKey().equals(section); entry.getValue().setSelected(selected);
            entry.getValue().setTextColor(selected ? TvTheme.ACCENT : TvTheme.TEXT);
        }
        settingsPane.removeAllViews(); LinearLayout body = column(); body.setPadding(dp(4), dp(6), dp(6), dp(6));
        if ("Home".equals(section)) {
            boolean active = homeIntegration.selected() || homeIntegration.overrideRequested() && homeIntegration.overrideConnected();
            String status = active ? "Home active" : homeIntegration.overrideRequested() ? homeIntegration.overrideAllowed() ? "Home override disconnected" : "Permission needed" : "Home override paused";
            settingsStatus(body, status, active ? TvTheme.ACCENT : TvTheme.SECONDARY);
            if (homeIntegration.previousAvailable()) {
                boolean pause = homeIntegration.overrideRequested() && homeIntegration.overrideConnected();
                settingsAction(body, "override", pause ? "Pause override" : homeIntegration.overrideRequested() ? "Reconnect override" : "Enable override", pause ? "Open Google TV" : "Vrooli Home", () -> {
                    try { if (pause) homeIntegration.openPrevious(); else homeIntegration.enableOverride(); }
                    catch (Exception ignored) { notice("Open Android Home or accessibility settings."); }
                });
            }
            settingsAction(body, "default-home", "Default Home app", "Android settings", () -> { try { homeIntegration.chooseHome(); } catch (Exception ignored) { notice("Open Android default app settings."); } });
            settingsAction(body, "accessibility", "Accessibility", "Android service settings", () -> { try { homeIntegration.openAccessibilitySettings(); } catch (Exception ignored) { notice("Open Android accessibility settings."); } });
        } else if ("Portal".equals(section)) {
            String status = server.isEmpty() ? "Not configured" : catalogueError != null ? "Unavailable" : catalogueLoaded && !staleCatalogue ? "Connected" : "Not connected";
            settingsStatus(body, status, "Connected".equals(status) ? TvTheme.ACCENT : TvTheme.SECONDARY);
            settingsAction(body, "address", "Portal address", server.isEmpty() ? "Not set" : server, this::editPortalAddress);
            settingsAction(body, "discovery", "Find Portal", discovering ? "Searching…" : discoveredOrigins.isEmpty() ? "Home network" : discoveredOrigins.size() + " found", () -> { beginDiscovery(); savePosition(); render(false); });
            if (!discoveredOrigins.isEmpty()) settingsAction(body, "found", "Available Portals", Integer.toString(discoveredOrigins.size()), this::chooseDiscoveredPortal);
            LinearLayout fallback = row(); fallback.setPadding(dp(18), dp(12), dp(18), dp(12));
            LinearLayout labels = column(); labels.addView(text("Home LAN fallback", 17, TvTheme.TEXT)); TextView detail = text("Unencrypted HTTP", 13, TvTheme.SECONDARY); detail.setPadding(0, dp(5), 0, 0); labels.addView(detail);
            fallback.addView(labels, new LinearLayout.LayoutParams(0, -2, 1));
            Switch toggle = new Switch(this); toggle.setChecked(preferences.getBoolean("allow-home-http", false)); toggle.setFocusable(false); toggle.setClickable(false); toggle.setImportantForAccessibility(View.IMPORTANT_FOR_ACCESSIBILITY_NO); fallback.addView(toggle);
            remember(fallback, "settings-control:fallback", "Home LAN fallback, " + (toggle.isChecked() ? "on" : "off"));
            fallback.setOnClickListener(view -> { setHomeFallback(!toggle.isChecked()); render(false); });
            settingsRow(body, fallback);
        } else {
            settingsAction(body, "clear-recents", "Recent apps", recentApps.size() + " saved", () -> new AlertDialog.Builder(this).setTitle("Clear recent apps?").setPositiveButton("Clear", (dialog, which) -> { recentApps.clear(); saveList("recent-apps", recentApps); dirtyPages.add("Home"); render(false); }).setNegativeButton("Cancel", null).show());
        }
        ScrollView scroll = new ScrollView(this); scroll.setFillViewport(false); scroll.setClipToPadding(true); scroll.setVerticalScrollBarEnabled(false); scroll.addView(body); currentScroll = scroll;
        settingsPane.addView(scroll, new FrameLayout.LayoutParams(-1, -1));
        View category = settingsCategories.get(section), first = null;
        for (Map.Entry<String, View> entry : targets.entrySet()) if (entry.getKey().startsWith("settings-control:")) { entry.getValue().setNextFocusLeftId(category.getId()); if (first == null) first = entry.getValue(); }
        if (first != null) category.setNextFocusRightId(first.getId());
        for (TextView item : settingsCategories.values()) item.setNextFocusLeftId(item.getId());
        settingsCategories.values().iterator().next().setNextFocusUpId(tabs.get("Settings").getId());
    }
    private void settingsStatus(LinearLayout body, String value, int color) {
        TextView status = text(value, 20, color); status.setPadding(dp(14), dp(3), 0, dp(20)); body.addView(status);
    }
    private void settingsRow(LinearLayout body, View view) {
        LinearLayout.LayoutParams rp = new LinearLayout.LayoutParams(-1, dp(62)); rp.bottomMargin = dp(12); body.addView(view, rp);
    }
    private void settingsAction(LinearLayout body, String identity, String label, String value, Runnable action) {
        LinearLayout item = row(); item.setPadding(dp(18), dp(10), dp(18), dp(10)); item.setDescendantFocusability(ViewGroup.FOCUS_BLOCK_DESCENDANTS);
        LinearLayout labels = column(); labels.addView(text(label, 17, TvTheme.TEXT));
        if (value != null && !value.isEmpty()) { TextView detail = text(value, 13, TvTheme.SECONDARY); detail.setSingleLine(true); detail.setEllipsize(TextUtils.TruncateAt.END); detail.setPadding(0, dp(5), 0, 0); labels.addView(detail); }
        item.addView(labels, new LinearLayout.LayoutParams(0, -2, 1)); TextView arrow = text("›", 25, TvTheme.SECONDARY); item.addView(arrow);
        remember(item, "settings-control:" + identity, label + (value == null ? "" : ", " + value)); item.setOnClickListener(view -> action.run()); settingsRow(body, item);
    }
    private void setHomeFallback(boolean allowed) {
        preferences.edit().putBoolean("allow-home-http", allowed).apply();
        if (!allowed && server.startsWith("http://")) { cancelRequest(); server = ""; preferences.edit().remove("server").apply(); catalogue.clear(); catalogueLoaded = false; staleCatalogue = false; catalogueError = null; }
        dirtyPages.add(LIBRARY);
        client = new PortalClient(server, allowed);
    }
    private void chooseDiscoveredPortal() {
        List<String> origins = new ArrayList<>();
        for (String origin : discoveredOrigins) if (!origin.startsWith("http://") || preferences.getBoolean("allow-home-http", false)) origins.add(origin);
        if (origins.isEmpty()) { notice("These Portals require the home LAN fallback."); return; }
        new AlertDialog.Builder(this).setTitle("Home Portals").setItems(origins.toArray(new String[0]), (dialog, which) -> { selectPortal(origins.get(which), true); navigate(LIBRARY); }).setNegativeButton("Cancel", null).show();
    }
    private void editPortalAddress() {
        EditText address = new EditText(this); address.setSingleLine(true); address.setTextColor(TvTheme.TEXT); address.setTextSize(18); address.setHintTextColor(TvTheme.SECONDARY); address.setHint("https://portal.example.com"); address.setText(connectionDraft == null ? server : connectionDraft); address.setSelectAllOnFocus(true);
        address.setInputType(android.text.InputType.TYPE_CLASS_TEXT | android.text.InputType.TYPE_TEXT_VARIATION_URI); address.setContentDescription("Portal address");
        address.addTextChangedListener(new android.text.TextWatcher() { public void beforeTextChanged(CharSequence value, int start, int count, int after) {} public void onTextChanged(CharSequence value, int start, int before, int count) { connectionDraft = value.toString(); } public void afterTextChanged(android.text.Editable value) {} });
        LinearLayout form = column(); form.setPadding(dp(24), dp(8), dp(24), dp(4)); form.addView(address, new LinearLayout.LayoutParams(-1, dp(56)));
        AlertDialog dialog = new AlertDialog.Builder(this).setTitle("Portal address").setView(form).setPositiveButton("Connect", null).setNegativeButton("Cancel", (d, which) -> hideKeyboard(address)).create();
        dialog.setOnShowListener(ignored -> { dialog.getButton(AlertDialog.BUTTON_POSITIVE).setOnClickListener(view -> {
            try {
                String origin = PortalClient.validateOrigin(address.getText().toString(), BuildConfig.DEBUG, preferences.getBoolean("allow-home-http", false));
                hideKeyboard(address); connectionDraft = null; dialog.dismiss(); selectPortal(origin, false); navigate(LIBRARY);
            } catch (Exception invalid) { address.setError("Use HTTPS, or enable the private home LAN fallback."); }
        }); address.requestFocus(); });
        dialog.show();
    }
    @Override protected void onNewIntent(Intent intent) { super.onNewIntent(intent); setIntent(intent); handleConnection(intent); }
    private void handleConnection(Intent intent) {
        if (BuildConfig.HOME_ENABLED && intent != null && HomeOverrideService.SETTINGS_ACTION.equals(intent.getAction())) { navigate("Settings"); return; }
        if (isHomeIntent(intent)) {
            savePosition(); cancelRequest(); closeWeb(false); state.systemHome(); showPage(false); refreshTvApps(); HomeOverrideService.homeArrived(); return;
        }
        Uri uri = intent == null ? null : intent.getData();
        if (uri != null && "vrooli-portal".equals(uri.getScheme()) && "connect".equals(uri.getHost())) {
            closeWeb(false); savePosition(); cancelRequest(); connectionDraft = uri.getQueryParameter("server"); settingsSection = "Portal"; state.focused.put("Settings", "settings-category:Portal"); navigate("Settings");
        }
    }
    private boolean isHomeIntent(Intent intent) {
        return BuildConfig.HOME_ENABLED && intent != null && Intent.ACTION_MAIN.equals(intent.getAction()) &&
            (intent.hasCategory(Intent.CATEGORY_HOME) || (intent.getComponent() != null && "com.vrooli.portal.tv.HomeActivity".equals(intent.getComponent().getClassName())));
    }
    private void openApplication(String identity) {
        cancelRequest();
        state.appOpened();
        final int generation = ++requestVersion; final PortalClient connection = client;
        showLoading("Opening experience…", () -> { cancelRequest(); closeWeb(false); });
        pending = network.submit(() -> {
            try {
                JSONObject request = new JSONObject(); request.put("scenario", identity); request.put("useGateway", true);
                String address = new URI(connection.origin + "/").resolve(connection.rpc("ResolveApplication", request).getString("url")).toString();
                runOnUiThread(() -> { if (!destroyed && generation == requestVersion) openWeb(address, false); });
            } catch (Exception error) {
                String message = failure(error);
                runOnUiThread(() -> { if (!destroyed && generation == requestVersion) showLoading(message, () -> { cancelRequest(); closeWeb(false); }); });
            }
        });
    }
    private boolean safePage(String address) {
        try {
            URI uri = new URI(address); if (uri.getHost() == null || uri.getRawUserInfo() != null) return false;
            if ("https".equals(uri.getScheme())) return true;
            return "http".equals(uri.getScheme()) && preferences.getBoolean("allow-home-http", false) && server.startsWith("http://") && PortalOrigin.privateIPv4(uri.getHost()) && Objects.equals(new URI(server).getAuthority(), uri.getAuthority());
        } catch (Exception ignored) { return false; }
    }
    private void showLoading(String message, Runnable cancel) {
        if (browser == null) {
            browser = new FrameLayout(this); browser.setBackgroundColor(TvTheme.INK);
            openingPanel = column(); openingPanel.setGravity(Gravity.CENTER); openingPanel.setBackgroundColor(TvTheme.INK);
            openingCaption = text(message, 22, TvTheme.TEXT); openingCaption.setGravity(Gravity.CENTER); openingPanel.addView(openingCaption);
            openingCancel = TvTheme.action(this, "Return to library", cancel); LinearLayout.LayoutParams bp = new LinearLayout.LayoutParams(dp(220), dp(46)); bp.topMargin = dp(24); openingPanel.addView(openingCancel, bp);
            browser.addView(openingPanel, new FrameLayout.LayoutParams(-1, -1)); root.addView(browser, new FrameLayout.LayoutParams(-1, -1));
        }
        openingCaption.setText(message); openingCancel.setOnClickListener(view -> cancel.run()); openingPanel.setVisibility(View.VISIBLE);
        shell.setVisibility(View.GONE); openingCancel.requestFocus(); UiTimings.surface(state.page, true, true);
    }
    @SuppressWarnings("SetJavaScriptEnabled") private void openWeb(String address, boolean signIn) {
        if (!safePage(address)) { showLoading("This address cannot be opened safely.", () -> closeWeb(false)); return; }
        destroyWeb(); webIsSignIn = signIn;
        showLoading("Opening experience…", () -> { cancelRequest(); closeWeb(true); });
        web = new WebView(this);
        if ((getApplicationInfo().flags & android.content.pm.ApplicationInfo.FLAG_DEBUGGABLE) != 0) WebView.setWebContentsDebuggingEnabled(true);
        WebSettings settings = web.getSettings(); settings.setJavaScriptEnabled(true); settings.setDomStorageEnabled(true); settings.setAllowFileAccess(false); settings.setAllowContentAccess(false); settings.setMixedContentMode(WebSettings.MIXED_CONTENT_NEVER_ALLOW);
        browser.addView(web, 0, new FrameLayout.LayoutParams(-1, -1));
        web.setWebViewClient(new WebViewClient() {
            private boolean failed;
            @Override public WebResourceResponse shouldInterceptRequest(WebView view, WebResourceRequest request) {
                Uri destination = request.getUrl();
                if ("http".equals(destination.getScheme()) && !safePage(destination.toString())) return new WebResourceResponse("text/plain", "utf-8", 403, "Blocked", Collections.emptyMap(), new java.io.ByteArrayInputStream(new byte[0]));
                return null;
            }
            @Override public void onPageCommitVisible(WebView view, String url) { if (!failed && view == web && browser != null) { openingPanel.setVisibility(View.GONE); view.requestFocus(); UiTimings.surface(state.page, true, false); } }
            @Override public boolean shouldOverrideUrlLoading(WebView view, WebResourceRequest request) { return !safePage(request.getUrl().toString()); }
            @Override public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
                if (request.isForMainFrame() && view == web) { failed = true; showLoading("This experience is unavailable. Return to your library to try again.", () -> { cancelRequest(); closeWeb(false); }); android.util.Log.w("PortalTV", "Web page error: " + error.getErrorCode()); }
            }
            @Override public void onReceivedHttpError(WebView view, WebResourceRequest request, WebResourceResponse response) {
                if (request.isForMainFrame() && view == web && response.getStatusCode() >= 400) { failed = true; showLoading("This experience could not be opened. Your library is still available.", () -> { cancelRequest(); closeWeb(false); }); }
            }
            @Override public boolean onRenderProcessGone(WebView view, RenderProcessGoneDetail detail) { closeWeb(false); notice("The experience closed. Your library is still available."); return true; }
        });
        web.setWebChromeClient(new WebChromeClient());
        web.loadUrl(address);
    }
    private void closeWeb(boolean retrySignIn) {
        if (browser == null) return;
        boolean retry = retrySignIn && webIsSignIn;
        if (web != null && !server.isEmpty()) client.setCookie(CookieManager.getInstance().getCookie(server));
        state.appReturned();
        destroyWeb(); root.removeView(browser); browser = null; openingPanel = null; openingCaption = null; openingCancel = null; shell.setVisibility(View.VISIBLE); webIsSignIn = false;
        restoreFocus(LIBRARY.equals(state.page) ? "experience:web-console" : "library");
        UiTimings.surface(state.page, false, false);
        if (retry && LIBRARY.equals(state.page)) { catalogueError = null; refreshCatalogue(); }
    }
    private void destroyWeb() { if (web != null) { WebView previous = web; web = null; previous.stopLoading(); if (previous.getParent() instanceof ViewGroup) ((ViewGroup) previous.getParent()).removeView(previous); previous.destroy(); } }
    private void cancelRequest() { ++requestVersion; catalogueRefreshing = false; if (pending != null) pending.cancel(true); pending = null; if (client != null) client.cancel(); }
    private String failure(Exception error) {
        String code = "request_failed", message = "Portal is unavailable. Your TV apps still work.";
        if (error instanceof java.net.SocketTimeoutException) { code = "timeout"; message = "Portal did not respond. Try again when it is available."; }
        else if (error instanceof java.net.UnknownHostException) { code = "unknown_host"; message = "The Portal address could not be found."; }
        else if (error instanceof javax.net.ssl.SSLException) { code = "tls_failed"; message = "Portal’s secure connection could not be verified."; }
        else if (error instanceof JSONException) { code = "invalid_response"; message = "Portal returned an unexpected response."; }
        else if (error.getMessage() != null && error.getMessage().matches("http_[1-5][0-9]{2}")) {
            code = error.getMessage(); message = code.matches("http_(30[12378]|401|403)") ? "Sign in to Portal to open your library." : "Portal is unavailable. Try again later.";
        }
        android.util.Log.w("PortalTV", "Library request: " + code + " (" + error.getClass().getSimpleName() + ")");
        return message;
    }
    private List<String> readList(String key, List<String> fallback) {
        if (!preferences.contains(key)) return new ArrayList<>(fallback);
        List<String> values = new ArrayList<>();
        try { JSONArray json = new JSONArray(preferences.getString(key, "[]")); for (int i = 0; i < Math.min(json.length(), 512); i++) { String value = json.getString(i); if (!values.contains(value)) values.add(value); } } catch (JSONException ignored) { return new ArrayList<>(fallback); }
        return values;
    }
    private void saveList(String key, List<String> values) { preferences.edit().putString(key, new JSONArray(values).toString()).apply(); }
    private void loadCachedCatalogue() {
        if (!server.equals(preferences.getString("catalogue-origin", ""))) return;
        try { cataloguePayload = preferences.getString("catalogue", "[]"); JSONArray values = new JSONArray(cataloguePayload); if (values.length() > 512) return; for (int i = 0; i < values.length(); i++) catalogue.add(values.getJSONObject(i)); catalogueLoaded = !catalogue.isEmpty(); staleCatalogue = catalogueLoaded; } catch (JSONException ignored) { catalogue.clear(); }
    }
    private void notice(String message) { Toast.makeText(this, message, Toast.LENGTH_LONG).show(); }
    private void hideKeyboard(View input) {
        android.view.inputmethod.InputMethodManager manager = (android.view.inputmethod.InputMethodManager) getSystemService(INPUT_METHOD_SERVICE);
        if (manager != null) manager.hideSoftInputFromWindow(input.getWindowToken(), 0);
    }
    @Override public void onBackPressed() {
        if (browser != null) { if (web != null && web.canGoBack()) web.goBack(); else { cancelRequest(); closeWeb(true); } }
        else if (!"Home".equals(state.page)) { String destination = state.back(); savePosition(); cancelRequest(); state.page = destination; render(false); }
        else if (state.exitsOnBack(BuildConfig.HOME_ENABLED)) super.onBackPressed();
    }
    @Override protected void onSaveInstanceState(Bundle out) {
        out.putString("appReturn", state.appReturn);
        savePosition(); out.putString("page", state.page); out.putString("libraryReturn", state.libraryReturn); out.putString("settingsReturn", state.settingsReturn); out.putString("settingsSection", settingsSection); out.putString("connectionDraft", connectionDraft); out.putString("filter", state.filter);
        Bundle focus = new Bundle(); for (Map.Entry<String, String> entry : state.focused.entrySet()) focus.putString(entry.getKey(), entry.getValue()); out.putBundle("focused", focus);
        Bundle scroll = new Bundle(); for (Map.Entry<String, Integer> entry : state.scroll.entrySet()) scroll.putInt(entry.getKey(), entry.getValue()); out.putBundle("scroll", scroll);
        super.onSaveInstanceState(out);
    }
    @Override protected void onPause() { savePosition(); if (web != null) web.onPause(); super.onPause(); }
    @Override protected void onResume() {
        super.onResume();
        if (web != null) web.onResume();
        else if (browser == null && content != null) {
            boolean returning = state.appReturn != null; String before = state.page; state.appReturned();
            if (returning || !before.equals(state.page)) showPage(false);
            // Package availability and banners may change while an app owns the
            // screen. Re-read only local native pages; never trigger Portal here.
            refreshTvApps();
            if ("Settings".equals(state.page)) render(false);
            else if (!(getCurrentFocus() instanceof NavigationTab)) {
                restoreFocus("library");
            }
        }
    }
    private void launchExternal(Intent intent) {
        savePosition(); state.appOpened();
        try { startActivity(intent); } catch (RuntimeException error) { state.appReturned(); throw error; }
    }
    @Override protected void onDestroy() { destroyed = true; discovery.stop(); cancelRequest(); network.shutdownNow(); packageWorker.shutdownNow(); artwork.close(); destroyWeb(); super.onDestroy(); }
}
