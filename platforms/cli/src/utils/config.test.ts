// AI_CHECK: TEST_COVERAGE=2 | LAST: 2025-01-12
// AI_CHECK: TEST_QUALITY=1 | LAST: 2025-07-16
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import * as fsSync from "fs";
import * as os from "os";
import * as path from "path";
import { ConfigManager } from "./config.js";
import { logger } from "./logger.js";
import type { Session } from "@vrooli/shared";

// Mock dependencies
vi.mock("fs");
vi.mock("os");
vi.mock("./logger.js", () => ({
    logger: {
        error: vi.fn(),
    },
}));

describe("ConfigManager", () => {
    let config: ConfigManager;
    const mockHomedir = "/home/testuser";
    const mockConfigDir = path.join(mockHomedir, ".vrooli");
    const mockConfigPath = path.join(mockConfigDir, "config.json");

    const defaultConfig = {
        currentProfile: "default",
        profiles: {
            default: {
                url: "http://localhost:5329",
            },
        },
    };

    const existingConfig = {
        currentProfile: "default",
        profiles: {
            default: {
                url: "http://localhost:5329",
                authToken: "default-token",
                refreshToken: "default-refresh",
                tokenExpiry: Date.now() + 3600000,
                userId: "user-123",
                handle: "testuser",
                timeZone: "UTC",
            },
            production: {
                url: "https://api.vrooli.com",
                authToken: "prod-token",
            },
        },
    };

    beforeEach(() => {
        vi.clearAllMocks();
        vi.mocked(os.homedir).mockReturnValue(mockHomedir);
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    describe("constructor and initialization", () => {
        it("should load existing config file", () => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(existingConfig));

            config = new ConfigManager();

            expect(fsSync.readFileSync).toHaveBeenCalledWith(mockConfigPath, "utf-8");
        });

        it("should create default config when file doesn't exist", () => {
            vi.mocked(fsSync.readFileSync).mockImplementation(() => {
                throw new Error("ENOENT: no such file");
            });
            vi.mocked(fsSync.mkdirSync).mockImplementation(() => undefined);
            vi.mocked(fsSync.writeFileSync).mockImplementation(() => undefined);

            config = new ConfigManager();

            expect(fsSync.mkdirSync).toHaveBeenCalledWith(mockConfigDir, { recursive: true });
            expect(fsSync.writeFileSync).toHaveBeenCalledWith(
                mockConfigPath,
                JSON.stringify(defaultConfig, null, 2),
            );
        });

        it("should handle directory creation errors gracefully", () => {
            vi.mocked(fsSync.readFileSync).mockImplementation(() => {
                throw new Error("ENOENT");
            });
            vi.mocked(fsSync.mkdirSync).mockImplementation(() => {
                throw new Error("Permission denied");
            });

            config = new ConfigManager();

            expect(logger.error).toHaveBeenCalledWith(
                "Failed to create config directory",
                expect.any(Error),
            );
        });
    });

    describe("profile management", () => {
        beforeEach(() => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(existingConfig));
            vi.mocked(fsSync.writeFileSync).mockImplementation(() => undefined);
            config = new ConfigManager();
        });

        it("should get active profile", () => {
            const profile = config.getActiveProfile();

            expect(profile).toEqual(existingConfig.profiles.default);
        });

        it("should throw error for non-existent active profile", () => {
            // Mock a corrupted config
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify({
                currentProfile: "nonexistent",
                profiles: {},
            }));
            config = new ConfigManager();

            expect(() => config.getActiveProfile()).toThrow("Profile 'nonexistent' not found");
        });

        it("should get active profile name", () => {
            expect(config.getActiveProfileName()).toBe("default");
        });

        it("should set active profile", () => {
            config.setActiveProfile("production");

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.currentProfile).toBe("production");
        });

        it("should throw error when setting non-existent profile", () => {
            expect(() => config.setActiveProfile("staging")).toThrow(
                "Profile 'staging' does not exist",
            );
        });

        it("should create new profile", () => {
            const newProfile = {
                url: "https://staging.vrooli.com",
            };

            config.createProfile("staging", newProfile);

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.profiles.staging).toEqual(newProfile);
        });

        it("should throw error when creating duplicate profile", () => {
            expect(() => config.createProfile("default", { url: "test" })).toThrow(
                "Profile 'default' already exists",
            );
        });

        it("should update existing profile", () => {
            config.updateProfile("default", { url: "http://localhost:8080" });

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.profiles.default.url).toBe("http://localhost:8080");
            expect(savedConfig.profiles.default.authToken).toBe("default-token");
        });

        it("should throw error when updating non-existent profile", () => {
            expect(() => config.updateProfile("staging", { url: "test" })).toThrow(
                "Profile 'staging' does not exist",
            );
        });

        it("should list all profiles", () => {
            const profiles = config.listProfiles();

            expect(profiles).toEqual(["default", "production"]);
        });
    });

    describe("authentication management", () => {
        beforeEach(() => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(existingConfig));
            vi.mocked(fsSync.writeFileSync).mockImplementation(() => undefined);
            config = new ConfigManager();
        });

        it("should get auth token", () => {
            const token = config.getAuthToken();

            expect(token).toBe("default-token");
        });

        it("should return undefined for expired token", () => {
            // Create config with expired token
            const expiredConfig = {
                ...existingConfig,
                profiles: {
                    default: {
                        ...existingConfig.profiles.default,
                        tokenExpiry: Date.now() - 1000, // Expired
                    },
                },
            };
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(expiredConfig));
            config = new ConfigManager();

            const token = config.getAuthToken();

            expect(token).toBeUndefined();
            expect(fsSync.writeFileSync).toHaveBeenCalled(); // Should clear auth
        });

        it("should get refresh token", () => {
            const token = config.getRefreshToken();

            expect(token).toBe("default-refresh");
        });

        it("should set auth tokens", () => {
            config.setAuth("new-token", "new-refresh", 3600);

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.profiles.default.authToken).toBe("new-token");
            expect(savedConfig.profiles.default.refreshToken).toBe("new-refresh");
            expect(savedConfig.profiles.default.tokenExpiry).toBeGreaterThan(Date.now());
        });

        it("should set session data", () => {
            const session: Session = {
                id: "session-123",
                userId: "user-456",
                timeZone: "America/New_York",
                users: [{
                    id: "internal-id",
                    publicId: "user-456",
                    handle: "newuser",
                    name: "New User",
                }],
            } as any;

            config.setSession(session);

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.profiles.default.userId).toBe("user-456");
            expect(savedConfig.profiles.default.handle).toBe("newuser");
            expect(savedConfig.profiles.default.timeZone).toBe("America/New_York");
        });

        it("should clear auth data", () => {
            config.clearAuth();

            expect(fsSync.writeFileSync).toHaveBeenCalled();
            const savedConfig = JSON.parse(
                vi.mocked(fsSync.writeFileSync).mock.calls[0][1] as string,
            );
            expect(savedConfig.profiles.default.authToken).toBeUndefined();
            expect(savedConfig.profiles.default.refreshToken).toBeUndefined();
            expect(savedConfig.profiles.default.tokenExpiry).toBeUndefined();
            expect(savedConfig.profiles.default.userId).toBeUndefined();
            expect(savedConfig.profiles.default.handle).toBeUndefined();
            expect(savedConfig.profiles.default.timeZone).toBeUndefined();
        });
    });

    describe("runtime settings", () => {
        beforeEach(() => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(defaultConfig));
            config = new ConfigManager();
        });

        it("should manage debug mode", () => {
            expect(config.isDebug()).toBe(false);

            config.setDebug(true);
            expect(config.isDebug()).toBe(true);

            config.setDebug(false);
            expect(config.isDebug()).toBe(false);
        });

        it("should manage JSON output mode", () => {
            expect(config.isJsonOutput()).toBe(false);

            config.setJsonOutput(true);
            expect(config.isJsonOutput()).toBe(true);

            config.setJsonOutput(false);
            expect(config.isJsonOutput()).toBe(false);
        });
    });

    describe("server URL", () => {
        beforeEach(() => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(existingConfig));
            config = new ConfigManager();
        });

        it("should get server URL from active profile", () => {
            expect(config.getServerUrl()).toBe("http://localhost:5329");

            config.setActiveProfile("production");
            expect(config.getServerUrl()).toBe("https://api.vrooli.com");
        });
    });

    describe("error handling", () => {
        beforeEach(() => {
            vi.mocked(fsSync.readFileSync).mockReturnValue(JSON.stringify(defaultConfig));
            config = new ConfigManager();
        });

        it("should throw error when save fails", () => {
            vi.mocked(fsSync.writeFileSync).mockImplementation(() => {
                throw new Error("Permission denied");
            });

            expect(() => config.setActiveProfile("default")).toThrow(
                "Failed to save configuration",
            );
            expect(logger.error).toHaveBeenCalledWith(
                "Failed to save config",
                expect.any(Error),
            );
        });
    });
});
