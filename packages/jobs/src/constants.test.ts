// AI_CHECK: TEST_COVERAGE=1,TEST_QUALITY=1 | LAST: 2025-06-24
import { describe, expect, it } from "vitest";
import {
    API_BATCH_SIZE,
    BATCH_SIZE_LARGE,
    BATCH_SIZE_SMALL,
    BYTES_PER_KILOBYTE,
    ESTIMATED_ENTRY_OVERHEAD_BYTES,
    MAX_ENTRIES_PER_SITEMAP,
    MAX_SITEMAP_FILE_SIZE_BYTES,
    SITEMAP_SIZE_LIMIT_MB,
} from "./constants.js";

describe("constants", () => {
    describe("batch processing sizes", () => {
        it("should have correct small batch size", () => {
            expect(BATCH_SIZE_SMALL).toBe(100);
        });

        it("should have correct large batch size", () => {
            expect(BATCH_SIZE_LARGE).toBe(1000);
        });

        it("should have large batch size greater than small batch size", () => {
            expect(BATCH_SIZE_LARGE).toBeGreaterThan(BATCH_SIZE_SMALL);
        });
    });

    describe("sitemap generation limits", () => {
        it("should have correct max entries per sitemap", () => {
            expect(MAX_ENTRIES_PER_SITEMAP).toBe(50000);
        });

        it("should have correct bytes per kilobyte", () => {
            expect(BYTES_PER_KILOBYTE).toBe(1024);
        });

        it("should have correct sitemap size limit", () => {
            expect(SITEMAP_SIZE_LIMIT_MB).toBe(50);
        });

        it("should have correct estimated entry overhead", () => {
            expect(ESTIMATED_ENTRY_OVERHEAD_BYTES).toBe(100);
        });

        it("should calculate max sitemap file size correctly", () => {
            const expected = SITEMAP_SIZE_LIMIT_MB * BYTES_PER_KILOBYTE * BYTES_PER_KILOBYTE;
            expect(MAX_SITEMAP_FILE_SIZE_BYTES).toBe(expected);
            expect(MAX_SITEMAP_FILE_SIZE_BYTES).toBe(52428800); // 50MB in bytes
        });
    });

    describe("API limits", () => {
        it("should have correct API batch size", () => {
            expect(API_BATCH_SIZE).toBe(100);
        });

        it("should have API batch size equal to small batch size", () => {
            expect(API_BATCH_SIZE).toBe(BATCH_SIZE_SMALL);
        });
    });

    describe("constant relationships", () => {
        it("should have sensible relationships between constants", () => {
            // Entry overhead should be reasonable compared to typical entry sizes
            expect(ESTIMATED_ENTRY_OVERHEAD_BYTES).toBeLessThan(1000);
            expect(ESTIMATED_ENTRY_OVERHEAD_BYTES).toBeGreaterThan(0);

            // Max entries should allow for reasonable file sizes
            const estimatedMaxSize = MAX_ENTRIES_PER_SITEMAP * ESTIMATED_ENTRY_OVERHEAD_BYTES;
            expect(estimatedMaxSize).toBeLessThanOrEqual(MAX_SITEMAP_FILE_SIZE_BYTES);
        });
    });
});
