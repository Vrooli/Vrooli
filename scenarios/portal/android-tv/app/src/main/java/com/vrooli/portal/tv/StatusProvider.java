package com.vrooli.portal.tv;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;

/** Shell/system-only status metadata. No account, Portal, text or mutation surface. */
public final class StatusProvider extends ContentProvider {
    @Override public boolean onCreate() { return true; }
    @Override public Cursor query(Uri uri, String[] projection, String selection, String[] args, String sort) {
        if (!"/state".equals(uri.getPath()) || selection != null || args != null || sort != null) throw new IllegalArgumentException("status_state_only");
        android.content.SharedPreferences p = getContext().getSharedPreferences("portal-tv", android.content.Context.MODE_PRIVATE);
        MatrixCursor c = new MatrixCursor(new String[]{"key", "value"});
        c.addRow(new Object[]{"home.requested", Boolean.toString(p.getBoolean(HomeOverrideService.ENABLED, false))});
        c.addRow(new Object[]{"home.allowed", Boolean.toString(HomeIntegration.overrideAllowed(getContext()))});
        c.addRow(new Object[]{"home.connected", Boolean.toString(HomeOverrideService.connected())});
        c.addRow(new Object[]{"home.foreground", Boolean.toString(HomeOverrideService.foreground())});
        c.addRow(new Object[]{"home.pause_reason", p.getString("home-override-pause", "none")});
        c.addRow(new Object[]{"home.previous", p.getString("previous-home", "unknown")});
        c.addRow(new Object[]{"home.last_stock_class", HomeOverrideService.lastStockClass()});
        c.addRow(new Object[]{"home.boot_seen", Integer.toString(p.getInt("home-override-boot", -1))});
        return c;
    }
    @Override public String getType(Uri uri) { return "vnd.android.cursor.dir/vnd.vrooli.status"; }
    @Override public Uri insert(Uri uri, ContentValues values) { throw new UnsupportedOperationException("read_only"); }
    @Override public int update(Uri uri, ContentValues values, String selection, String[] args) { throw new UnsupportedOperationException("read_only"); }
    @Override public int delete(Uri uri, String selection, String[] args) { throw new UnsupportedOperationException("read_only"); }
}
