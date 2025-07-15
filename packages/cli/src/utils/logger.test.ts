import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { logger } from "./logger.js";

describe("logger", () => {
    // Store original console methods
    const originalConsole = {
        log: console.log,
        warn: console.warn,
        error: console.error,
    };

    beforeEach(() => {
        // Mock console methods
        console.log = vi.fn();
        console.warn = vi.fn();
        console.error = vi.fn();
        // Reset logger to default level
        logger.setLevel("info");
    });

    afterEach(() => {
        // Restore original console methods
        console.log = originalConsole.log;
        console.warn = originalConsole.warn;
        console.error = originalConsole.error;
    });

    describe("log level filtering", () => {
        it("should log messages at or above the current level", () => {
            logger.setLevel("info");
            
            logger.debug("debug message");
            expect(console.log).not.toHaveBeenCalled();
            
            logger.info("info message");
            expect(console.log).toHaveBeenCalledTimes(1);
            expect(console.log).toHaveBeenCalledWith(
                expect.stringContaining("[INFO] info message"),
            );
            
            logger.warn("warn message");
            expect(console.warn).toHaveBeenCalledTimes(1);
            expect(console.warn).toHaveBeenCalledWith(
                expect.stringContaining("[WARN] warn message"),
            );
            
            logger.error("error message");
            expect(console.error).toHaveBeenCalledTimes(1);
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[ERROR] error message"),
            );
        });

        it("should log all messages when level is debug", () => {
            logger.setLevel("debug");
            
            logger.debug("debug message");
            expect(console.log).toHaveBeenCalledTimes(1);
            expect(console.log).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] debug message"),
            );
            
            logger.info("info message");
            expect(console.log).toHaveBeenCalledTimes(2);
            expect(console.log).toHaveBeenLastCalledWith(
                expect.stringContaining("[INFO] info message"),
            );
        });

        it("should only log errors when level is error", () => {
            logger.setLevel("error");
            
            logger.debug("debug message");
            logger.info("info message");
            logger.warn("warn message");
            
            expect(console.log).not.toHaveBeenCalled();
            expect(console.warn).not.toHaveBeenCalled();
            
            logger.error("error message");
            expect(console.error).toHaveBeenCalledTimes(1);
            expect(console.error).toHaveBeenCalledWith(
                expect.stringContaining("[ERROR] error message"),
            );
        });
    });

    describe("additional arguments", () => {
        it("should pass additional arguments to console methods", () => {
            const obj = { foo: "bar" };
            const arr = [1, 2, 3];
            
            logger.info("message with data", obj, arr);
            
            expect(console.log).toHaveBeenCalledWith(
                expect.stringContaining("[INFO] message with data"),
                obj,
                arr,
            );
        });
    });

    describe("setLevel", () => {
        it("should change the logging level", () => {
            // Start with info level
            logger.debug("should not appear");
            expect(console.log).not.toHaveBeenCalled();
            
            // Change to debug level
            logger.setLevel("debug");
            logger.debug("should appear");
            expect(console.log).toHaveBeenCalledWith(
                expect.stringContaining("[DEBUG] should appear"),
            );
        });
    });
});
