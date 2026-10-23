package com.vrooli.portal.tv;

import java.util.*;

public final class CatalogueWindowTest {
    private static void equal(Object expected, Object actual, String message) {
        if (!Objects.equals(expected, actual)) throw new AssertionError(message + ": expected " + expected + ", got " + actual);
    }
    public static void main(String[] args) {
        for (int count : new int[]{0, 1, 59, 60, 61, 90, 129, 512}) {
            CatalogueWindow window = new CatalogueWindow(count, -1);
            List<Integer> rendered = new ArrayList<>();
            while (true) {
                for (int i = rendered.size(); i < window.end(); i++) rendered.add(i);
                if (!window.hasMore()) break;
                int before = window.end();
                window.append();
                if (window.end() <= before) throw new AssertionError("Scrolling must make progress");
            }
            List<Integer> expected = new ArrayList<>(); for (int i = 0; i < count; i++) expected.add(i);
            equal(expected, rendered, "Every catalogue entry is reachable once and in order for " + count + " apps");
            equal(count, window.append(), "Further bottom events cannot overrun catalogue");
        }
        CatalogueWindow large = new CatalogueWindow(512, -1);
        equal(60, large.end(), "Initial view creation remains bounded");
        equal(true, large.hasMore(), "A large catalogue remains extensible");
        CatalogueWindow restored = new CatalogueWindow(129, 128);
        equal(129, restored.end(), "Returning to the final card restores a reachable focus target");
        equal(1, new CatalogueWindow(1, 128).end(), "A changed filter or smaller refresh clamps restored position");
        System.out.println("PASS: full catalogue traversal, bounded batches, empty/filter results and deep focus restoration");
    }
}
