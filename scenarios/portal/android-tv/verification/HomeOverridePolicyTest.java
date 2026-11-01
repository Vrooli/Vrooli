package com.vrooli.portal.tv;

public final class HomeOverridePolicyTest {
    private static final String HOME = "tv.stock/tv.stock.Home";
    private static void expect(HomeOverridePolicy.Decision expected, HomeOverridePolicy.Decision actual) {
        if (expected != actual) throw new AssertionError("expected " + expected + ", got " + actual);
    }
    public static void main(String[] args) {
        try {
            String xml = java.nio.file.Files.readString(java.nio.file.Path.of(System.getProperty("tv.projectRoot", "."), "app/src/main/res/xml/home_override_service.xml"));
            if (xml.contains("android:packageNames=")) throw new AssertionError("Android11 package filters are additive: initial own-package filter prevents stock Home events");
            var factory = javax.xml.parsers.DocumentBuilderFactory.newInstance();
            factory.setNamespaceAware(true);
            var manifest = factory.newDocumentBuilder().parse(java.nio.file.Path.of(System.getProperty("tv.projectRoot", "."), "app/src/main/AndroidManifest.xml").toFile());
            String android = "http://schemas.android.com/apk/res/android";
            var permissions = manifest.getElementsByTagName("uses-permission");
            boolean bootPermission = false;
            for (int i = 0; i < permissions.getLength(); i++) {
                var element = (org.w3c.dom.Element) permissions.item(i);
                bootPermission |= "android.permission.RECEIVE_BOOT_COMPLETED".equals(element.getAttributeNS(android, "name"));
            }
            boolean receiver = false;
            var receivers = manifest.getElementsByTagName("receiver");
            for (int i = 0; i < receivers.getLength(); i++) {
                var element = (org.w3c.dom.Element) receivers.item(i);
                if (!"com.vrooli.portal.tv.HomeOverrideService$BootReceiver".equals(element.getAttributeNS(android, "name"))) continue;
                var actions = element.getElementsByTagName("action");
                receiver = "${homeEnabled}".equals(element.getAttributeNS(android, "enabled"))
                    && "false".equals(element.getAttributeNS(android, "exported"))
                    && actions.getLength() == 1
                    && "android.intent.action.BOOT_COMPLETED".equals(((org.w3c.dom.Element) actions.item(0)).getAttributeNS(android, "name"));
            }
            if (!bootPermission || !receiver) throw new AssertionError("Daily Home must have a non-exported boot diagnostic receiver; preview must not enable it");
            for (String capability : new String[]{"canRetrieveWindowContent", "canRequestFilterKeyEvents", "canPerformGestures"}) {
                if (!xml.contains("android:" + capability + "=\"false\"")) throw new AssertionError("Home metadata service must not request " + capability);
            }
        } catch (Exception unavailable) { throw new AssertionError("Actual Home service resource must be verified", unavailable); }
        HomeOverridePolicy p = new HomeOverridePolicy();
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(false,true,true,HOME,"tv.stock","tv.stock.Home",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,false,true,HOME,"tv.stock","tv.stock.Home",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,true,false,HOME,"tv.stock","tv.stock.Home",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,true,true,HOME,"tv.stock","tv.stock.Settings",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,true,true,HOME,"media.app","tv.stock.Home",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,true,true,HOME,null,null,0));
        expect(HomeOverridePolicy.Decision.OPEN_HOME, p.window(true,true,true,HOME,"tv.stock",".Home",0));
        expect(HomeOverridePolicy.Decision.IGNORE, p.window(true,true,true,HOME,"tv.stock","tv.stock.Home",100));
        for (long now : new long[]{500,1000,1500}) expect(HomeOverridePolicy.Decision.OPEN_HOME,p.window(true,true,true,HOME,"tv.stock","tv.stock.Home",now));
        expect(HomeOverridePolicy.Decision.PAUSE,p.window(true,true,true,HOME,"tv.stock","tv.stock.Home",2000));
        expect(HomeOverridePolicy.Decision.IGNORE,p.window(false,true,true,HOME,"tv.stock","tv.stock.Home",2500));
        expect(HomeOverridePolicy.Decision.OPEN_HOME,p.window(true,true,true,HOME,"tv.stock","tv.stock.Home",7000));
        p.homeArrived();
        for (int i=1;i<=8;i++) {
            expect(HomeOverridePolicy.Decision.OPEN_HOME,p.window(true,true,true,HOME,"tv.stock","tv.stock.Home",7000+i*400));
            p.homeArrived();
        }
        expect(HomeOverridePolicy.Decision.IGNORE,p.window(true,false,true,HOME,"tv.stock","tv.stock.Home",12000));
        expect(HomeOverridePolicy.Decision.OPEN_HOME,p.wake(true,true,true,13000));
        p.homeArrived();
        expect(HomeOverridePolicy.Decision.IGNORE,p.window(true,false,true,HOME,"tv.stock","tv.stock.Home",14000));
        expect(HomeOverridePolicy.Decision.IGNORE,p.window(true,true,true,HOME,"media.app","media.app.Player",14500));
        expect(HomeOverridePolicy.Decision.IGNORE,p.wake(true,true,true,15000));
        HomeOverridePolicy google = new HomeOverridePolicy();
        String googleHome = "com.google.android.apps.tv.launcherx/com.google.android.apps.tv.launcherx.home.HomeActivity";
        String googlePackage = "com.google.android.apps.tv.launcherx";
        expect(HomeOverridePolicy.Decision.OPEN_HOME, google.window(true,true,true,googleHome,googlePackage,"com.google.android.apps.tv.launcherx.profile.chooser.ProfileChooserActivity",0));
        google.homeArrived();
        expect(HomeOverridePolicy.Decision.IGNORE, google.window(true,true,true,googleHome,googlePackage,"com.google.android.apps.tv.launcherx.settings.SettingsActivity",500));
        expect(HomeOverridePolicy.Decision.IGNORE, google.window(true,true,true,HOME,googlePackage,"com.google.android.apps.tv.launcherx.profile.chooser.ProfileChooserActivity",1000));
        expect(HomeOverridePolicy.Decision.IGNORE, google.window(false,true,true,googleHome,googlePackage,"com.google.android.apps.tv.launcherx.profile.chooser.ProfileChooserActivity",1500));
        if (!HomeOverridePolicy.startForBoot(67,66) || HomeOverridePolicy.startForBoot(66,66) || HomeOverridePolicy.startForBoot(-1,66)) throw new AssertionError("Boot entry must be new and known; ordinary reconnect must preserve another app");
        System.out.println("PASS: exact Home only, paused/locked/screen-off passthrough, duplicate suppression, bounded fail-open burst and recovery");
    }
}
