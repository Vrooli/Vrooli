package com.vrooli.portal.tv;
public final class PortalOriginTest {
    public static void main(String[] args) throws Exception {
        if (!PortalOrigin.sameHost("https://portal.home.example:23456", "https://portal.home.example:27890")) throw new AssertionError("Same trusted TLS host must support port changes");
        for (String value : new String[]{"http://portal.home.example:23456", "https://other.example:23456", "https://portal.home.example:0", "https://user@portal.home.example", "https://portal.home.example/path", "https://localhost:23456", "https://127.0.0.1:23456"}) {
            if (PortalOrigin.sameHost("https://portal.home.example:23456", value)) throw new AssertionError("Untrusted hint replaced paired host: " + value);
        }
        if (!PortalOrigin.sameLocalFallback("http://192.168.1.12:23456", "http://192.168.1.12:27890")) throw new AssertionError("Opted-in local fallback must support a port change");
        if (PortalOrigin.sameLocalFallback("https://192.168.1.12:23456", "http://192.168.1.12:27890")) throw new AssertionError("TLS failure must never silently downgrade");
        if (!PortalOrigin.secureUpgrade("http://192.168.1.12:23456", "https://192.168.1.12:27890")) throw new AssertionError("Prefer a verified HTTPS upgrade when available");
        if (PortalOrigin.secureUpgrade("http://192.168.1.12:23456", "https://192.168.1.13:27890")) throw new AssertionError("Discovery cannot change hosts implicitly");
        for (String invalid : new String[]{"8.8.8.8", "192.168.1.999", "172.32.1.1", "10.1.2", "010.1.2.3", "portal.example", "127.0.0.1"}) if (PortalOrigin.privateIPv4(invalid)) throw new AssertionError("Not a private LAN IP: " + invalid);
        try { PortalOrigin.discovered("https://portal.home.example", "other-protocol"); throw new AssertionError("Wrong protocol accepted"); } catch (Exception expected) {}
        System.out.println("PASS: dynamic TLS ports, host isolation, no downgrade, no credential/path/loopback discovery");
    }
}
