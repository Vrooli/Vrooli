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
            for (String capability : new String[]{"canRetrieveWindowContent", "canRequestFilterKeyEvents", "canPerformGestures"}) {
                if (!xml.contains("android:" + capability + "=\"false\"")) throw new AssertionError("Home metadata service must not request " + capability);
            }
        } catch (java.io.IOException unavailable) { throw new AssertionError("Actual Home service resource must be verified", unavailable); }
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
        if (!HomeOverridePolicy.startForBoot(67,66) || HomeOverridePolicy.startForBoot(66,66) || HomeOverridePolicy.startForBoot(-1,66)) throw new AssertionError("Boot entry must be new and known; ordinary reconnect must preserve another app");
        System.out.println("PASS: exact Home only, paused/locked/screen-off passthrough, duplicate suppression, bounded fail-open burst and recovery");
    }
}
