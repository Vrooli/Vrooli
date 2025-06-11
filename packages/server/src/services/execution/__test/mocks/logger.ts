import { type Logger } from "winston";

export function createMockLogger(): Logger {
    return {
        debug: () => {},
        info: () => {},
        warn: () => {},
        error: () => {},
        log: () => {},
        // Add other Logger interface methods as needed
    } as Logger;
}