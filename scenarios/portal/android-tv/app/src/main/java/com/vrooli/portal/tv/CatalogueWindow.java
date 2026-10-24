package com.vrooli.portal.tv;

/** Bounds view creation without discarding any of the fetched catalogue. */
final class CatalogueWindow {
    private static final int BATCH = 30;
    final int total;
    private int end;

    CatalogueWindow(int total, int restoredIndex) {
        this.total = Math.max(0, total);
        end = Math.min(this.total, restoredIndex < 0 ? 12 : Math.max(12, ((restoredIndex / BATCH) + 1) * BATCH));
    }

    int end() { return end; }
    boolean hasMore() { return end < total; }
    int append() { end = Math.min(total, end + BATCH); return end; }
}
