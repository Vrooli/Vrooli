package com.vrooli.portal.tv;

import java.net.URI;

/** Endpoint hints never grant trust or silently downgrade a saved TLS route. */
final class PortalOrigin {
    static String discovered(String address, String protocol) throws Exception {
        if (!"portal-catalog-v1".equals(protocol)) throw new Exception("wrong_protocol");
        URI uri = new URI(address);
        String host = uri.getHost();
        if (!("https".equals(uri.getScheme()) || ("http".equals(uri.getScheme()) && privateIPv4(host))) || host == null || host.equalsIgnoreCase("localhost") || host.equals("127.0.0.1") || host.equals("[::1]") || uri.getRawUserInfo() != null || uri.getRawQuery() != null || uri.getRawFragment() != null || !(uri.getPath().isEmpty() || "/".equals(uri.getPath())) || uri.getPort() == 0 || uri.getPort() > 65535) throw new Exception("secure_origin_required");
        return uri.getScheme() + "://" + uri.getRawAuthority();
    }
    static boolean privateIPv4(String host) {
        if (host == null) return false;
        String[] parts = host.split("\\.", -1);
        if (parts.length != 4) return false;
        int[] octets = new int[4];
        try { for (int i = 0; i < 4; i++) { if (!parts[i].matches("0|[1-9][0-9]{0,2}")) return false; octets[i] = Integer.parseInt(parts[i]); if (octets[i] > 255) return false; } } catch (NumberFormatException ignored) { return false; }
        return octets[0] == 10 || (octets[0] == 192 && octets[1] == 168) || (octets[0] == 172 && octets[1] >= 16 && octets[1] <= 31);
    }
    static boolean sameLocalFallback(String saved, String candidate) {
        try {
            URI previous = new URI(discovered(saved, "portal-catalog-v1")), next = new URI(discovered(candidate, "portal-catalog-v1"));
            return "http".equals(previous.getScheme()) && "http".equals(next.getScheme()) && previous.getHost().equals(next.getHost());
        } catch (Exception ignored) { return false; }
    }
    static boolean secureUpgrade(String saved, String candidate) {
        try {
            URI previous = new URI(discovered(saved, "portal-catalog-v1")), next = new URI(discovered(candidate, "portal-catalog-v1"));
            return "http".equals(previous.getScheme()) && "https".equals(next.getScheme()) && previous.getHost().equalsIgnoreCase(next.getHost());
        } catch (Exception ignored) { return false; }
    }
    static boolean sameHost(String paired, String candidate) {
        try {
            URI previous = new URI(discovered(paired, "portal-catalog-v1"));
            URI next = new URI(discovered(candidate, "portal-catalog-v1"));
            return "https".equals(previous.getScheme()) && "https".equals(next.getScheme()) && previous.getHost().equalsIgnoreCase(next.getHost());
        } catch (Exception ignored) { return false; }
    }
}
