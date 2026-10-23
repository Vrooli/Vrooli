package com.vrooli.portal.tv;

import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import org.json.JSONObject;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URI;
import java.net.URL;
import java.nio.charset.StandardCharsets;

/** Bounded, origin-scoped reads; this client never follows authentication redirects. */
final class PortalClient {
    private static final String SERVICE = "/vrooli.portal.v1.surfaces.ApplicationCatalogService/";
    final String origin;
    private volatile String cookie;
    private volatile HttpURLConnection active;
    private final boolean allowLocalHttp;
    PortalClient(String origin, boolean allowLocalHttp) { this.origin = origin; this.allowLocalHttp = allowLocalHttp; }
    void setCookie(String value) { cookie = value; }
    void cancel() { HttpURLConnection previous = active; if (previous != null) previous.disconnect(); }

    JSONObject rpc(String method, JSONObject request) throws Exception {
        HttpURLConnection connection = open(SERVICE + method);
        connection.setRequestMethod("POST");
        connection.setDoOutput(true);
        connection.setRequestProperty("Content-Type", "application/json");
        connection.setRequestProperty("Connect-Protocol-Version", "1");
        active = connection;
        try {
            try (OutputStream out = connection.getOutputStream()) {
                out.write(request.toString().getBytes(StandardCharsets.UTF_8));
            }
            checkStatus(connection);
            return new JSONObject(new String(read(connection.getInputStream(), 2 * 1024 * 1024), StandardCharsets.UTF_8));
        } finally { connection.disconnect(); if (active == connection) active = null; }
    }
    Bitmap artwork(String path) throws Exception {
        // Catalogue artwork is served by Portal, never fetched from arbitrary app URLs.
        if (!path.matches("/api/v1/applications/[a-z][a-z0-9-]*/icon")) return null;
        HttpURLConnection connection = open(path);
        try {
            checkStatus(connection);
            if (!"image/png".equals(connection.getContentType())) return null;
            byte[] data = read(connection.getInputStream(), 512 * 1024);
            BitmapFactory.Options bounds = new BitmapFactory.Options();
            bounds.inJustDecodeBounds = true;
            BitmapFactory.decodeByteArray(data, 0, data.length, bounds);
            if (bounds.outWidth < 1 || bounds.outWidth > 1024 || bounds.outHeight < 1 || bounds.outHeight > 1024) return null;
            BitmapFactory.Options options = new BitmapFactory.Options();
            options.inSampleSize = 2;
            return BitmapFactory.decodeByteArray(data, 0, data.length, options);
        } finally { connection.disconnect(); }
    }
    private HttpURLConnection open(String path) throws Exception {
        validateOrigin(origin, BuildConfig.DEBUG, allowLocalHttp);
        HttpURLConnection connection = (HttpURLConnection) new URL(origin + path).openConnection();
        connection.setConnectTimeout(3000);
        connection.setReadTimeout(5000);
        connection.setInstanceFollowRedirects(false);
        String current = cookie;
        if (current != null) connection.setRequestProperty("Cookie", current);
        return connection;
    }
    private static void checkStatus(HttpURLConnection connection) throws IOException {
        int status = connection.getResponseCode();
        if (status != 200) throw new IOException("http_" + status);
    }
    private static byte[] read(InputStream stream, int limit) throws IOException {
        try (InputStream input = stream; ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            byte[] buffer = new byte[8192];
            for (int size; (size = input.read(buffer)) != -1;) {
                if (output.size() + size > limit) throw new IOException("response_too_large");
                output.write(buffer, 0, size);
            }
            return output.toByteArray();
        }
    }
    static String validateOrigin(String value, boolean debug, boolean allowLocalHttp) throws Exception {
        URI uri = new URI(value.trim());
        String host = uri.getHost();
        if (host == null || uri.getRawUserInfo() != null || uri.getRawQuery() != null || uri.getRawFragment() != null || !(uri.getPath() == null || uri.getPath().isEmpty() || "/".equals(uri.getPath()))) throw new Exception("invalid_origin");
        boolean privateIP = PortalOrigin.privateIPv4(host);
        boolean loopback = "localhost".equals(host) || "127.0.0.1".equals(host) || "[::1]".equals(host);
        if (!"https".equals(uri.getScheme()) && !("http".equals(uri.getScheme()) && ((allowLocalHttp && privateIP) || (debug && (privateIP || loopback))))) throw new Exception("secure_origin_required");
        if (uri.getPort() == 0 || uri.getPort() > 65535) throw new Exception("invalid_port");
        return uri.getScheme() + "://" + uri.getRawAuthority();
    }
}
