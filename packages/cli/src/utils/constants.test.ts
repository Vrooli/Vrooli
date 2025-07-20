// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-07-13
import { describe, expect, it } from "vitest";
import {
    CHAT_CONFIG,
    CONTEXT_CONFIG,
    DISPLAY_CONFIG,
    EXPORT_CONFIG,
    HTTP_STATUS,
    LIMITS,
    SLASH_COMMANDS_CONFIG,
    TERMINAL_DIMENSIONS,
    TIMEOUTS,
    TOOL_APPROVAL,
    UI,
} from "./constants.js";

describe("constants", () => {
    describe("UI constants", () => {
        it("should have valid UI values", () => {
            expect(UI.TERMINAL_WIDTH).toBe(80);
            expect(UI.SEPARATOR_LENGTH).toBe(50);
            expect(UI.POLLING_INTERVAL_MS).toBe(50);
            expect(UI.CONTENT_WIDTH).toBe(98);
            expect(UI.DESCRIPTION_WIDTH).toBe(30);
            expect(UI.WIDTH_ADJUSTMENT).toBe(8);
            expect(UI.PADDING.TOP).toBe(8);
            expect(UI.PADDING.RIGHT).toBe(20);
            expect(UI.PADDING.BOTTOM).toBe(30);
            expect(UI.PADDING.LEFT).toBe(20);
        });
    });

    describe("TERMINAL_DIMENSIONS constants", () => {
        it("should have valid terminal dimensions", () => {
            expect(TERMINAL_DIMENSIONS.DEFAULT_WIDTH).toBe(80);
            expect(TERMINAL_DIMENSIONS.DEFAULT_HEIGHT).toBe(24);
            expect(TERMINAL_DIMENSIONS.FALLBACK_WIDTH).toBe(80);
            expect(TERMINAL_DIMENSIONS.MAX_MESSAGE_LENGTH).toBe(2000);
        });
    });

    describe("CHAT_CONFIG constants", () => {
        it("should have valid chat configuration", () => {
            expect(CHAT_CONFIG.DEFAULT_LIMIT).toBe(10);
            expect(CHAT_CONFIG.PRIORITY_OFFSET).toBe(-10);
            expect(CHAT_CONFIG.PAGINATION_LIMIT).toBe(10);
        });
    });

    describe("HTTP_STATUS constants", () => {
        it("should have standard HTTP status codes", () => {
            expect(HTTP_STATUS.OK).toBe(200);
            expect(HTTP_STATUS.CREATED).toBe(201);
            expect(HTTP_STATUS.BAD_REQUEST).toBe(400);
            expect(HTTP_STATUS.UNAUTHORIZED).toBe(401);
            expect(HTTP_STATUS.FORBIDDEN).toBe(403);
            expect(HTTP_STATUS.NOT_FOUND).toBe(404);
            expect(HTTP_STATUS.INTERNAL_SERVER_ERROR).toBe(500);
        });
    });

    describe("TIMEOUTS constants", () => {
        it("should have valid timeout values", () => {
            expect(TIMEOUTS.CHAT_ENGINE_DEFAULT).toBe(2000);
            expect(TIMEOUTS.CHAT_ENGINE_MINUTES_TO_MS).toBe(60000);
            expect(TIMEOUTS.TOOL_APPROVAL_DEFAULT).toBe(1000);
            expect(TIMEOUTS.TOOL_APPROVAL_SHORT).toBe(50);
            expect(TIMEOUTS.TOOL_APPROVAL_MEDIUM).toBe(500);
            expect(TIMEOUTS.CONFIG_SAVE_DEBOUNCE).toBe(1000);
            expect(TIMEOUTS.MAX_WAIT_SECONDS).toBe(300);
            expect(TIMEOUTS.COMMAND_TIMEOUT_MS).toBe(30000);
        });
    });

    describe("LIMITS constants", () => {
        it("should have valid limit values", () => {
            expect(LIMITS.DEFAULT_PAGE_SIZE).toBe(20);
            expect(LIMITS.MAX_CONTEXT_SIZE_BYTES).toBe(10485760);
            expect(LIMITS.MAX_CONTEXT_MESSAGES).toBe(100);
            expect(LIMITS.MAX_MESSAGE_DISPLAY).toBe(10);
            expect(LIMITS.MAX_RESULT_DISPLAY_LENGTH).toBe(500);
        });
    });

    describe("DISPLAY_CONFIG constants", () => {
        it("should have valid display configuration", () => {
            expect(DISPLAY_CONFIG.ROUTINE_WIDTH_PADDING).toBe(8);
            expect(DISPLAY_CONFIG.MAX_DISPLAY_WIDTH).toBe(98);
            expect(DISPLAY_CONFIG.TABLE_COLUMN_WIDTHS.SMALL).toBe(8);
            expect(DISPLAY_CONFIG.TABLE_COLUMN_WIDTHS.MEDIUM).toBe(20);
            expect(DISPLAY_CONFIG.TABLE_COLUMN_WIDTHS.LARGE).toBe(30);
            expect(DISPLAY_CONFIG.TABLE_COLUMN_WIDTHS.EXTRA_LARGE).toBe(20);
        });
    });

    describe("CONTEXT_CONFIG constants", () => {
        it("should have valid context configuration", () => {
            expect(CONTEXT_CONFIG.MAX_CONTEXT_ENTRIES).toBe(10);
            expect(CONTEXT_CONFIG.CONTEXT_SIZE_LIMIT).toBe(1048576);
            expect(CONTEXT_CONFIG.CONTEXT_BUFFER_SIZE).toBe(1024);
            expect(CONTEXT_CONFIG.MAX_RECENT_CONVERSATIONS).toBe(20);
        });
    });

    describe("EXPORT_CONFIG constants", () => {
        it("should have valid export configuration", () => {
            expect(EXPORT_CONFIG.TIMESTAMP_MINUTES).toBe(60);
        });
    });

    describe("SLASH_COMMANDS_CONFIG constants", () => {
        it("should have valid slash commands configuration", () => {
            expect(SLASH_COMMANDS_CONFIG.HELP_LIMIT).toBe(10);
        });
    });

    describe("TOOL_APPROVAL constants", () => {
        it("should have valid tool approval configuration", () => {
            expect(TOOL_APPROVAL.POLLING_INTERVAL_MS).toBe(50);
            expect(TOOL_APPROVAL.SEPARATOR_LENGTH).toBe(50);
        });
    });
});
