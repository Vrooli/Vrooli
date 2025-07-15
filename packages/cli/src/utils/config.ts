import * as fsSync from "fs";
import * as os from "os";
import * as path from "path";
import { logger } from "./logger.js";
import { TIMEOUTS } from "./constants.js";
import type { Session } from "@vrooli/shared";

export interface ProfileConfig {
    url: string;
    authToken?: string;
    refreshToken?: string;
    tokenExpiry?: number;
    userId?: string;
    handle?: string;
    timeZone?: string;
}

export interface CliConfig {
    currentProfile: string;
    profiles: Record<string, ProfileConfig>;
}

export class ConfigManager {
    private configDir: string;
    private configPath: string;
    private config!: CliConfig;
    private debug = false;
    private jsonOutput = false;

    constructor() {
        this.configDir = path.join(os.homedir(), ".vrooli");
        this.configPath = path.join(this.configDir, "config.json");
        this.loadConfig();
    }

    private loadConfig(): void {
        try {
            const configData = JSON.parse(fsSync.readFileSync(this.configPath, "utf-8"));
            this.config = configData;
        } catch (error) {
            // Create default config if it doesn't exist
            this.config = {
                currentProfile: "default",
                profiles: {
                    default: {
                        url: "http://localhost:5329",
                    },
                },
            };
            this.ensureConfigDir();
            this.saveConfig();
        }
    }

    private ensureConfigDir(): void {
        try {
            fsSync.mkdirSync(this.configDir, { recursive: true });
        } catch (error) {
            logger.error("Failed to create config directory", error);
        }
    }

    private saveConfig(): void {
        try {
            this.ensureConfigDir();
            fsSync.writeFileSync(
                this.configPath,
                JSON.stringify(this.config, null, 2),
            );
        } catch (error) {
            logger.error("Failed to save config", error);
            throw new Error("Failed to save configuration");
        }
    }

    public getActiveProfile(): ProfileConfig {
        const profile = this.config.profiles[this.config.currentProfile];
        if (!profile) {
            throw new Error(`Profile '${this.config.currentProfile}' not found`);
        }
        return profile;
    }

    public getActiveProfileName(): string {
        return this.config.currentProfile;
    }

    public setActiveProfile(profileName: string): void {
        if (!this.config.profiles[profileName]) {
            throw new Error(`Profile '${profileName}' does not exist`);
        }
        this.config.currentProfile = profileName;
        this.saveConfig();
    }

    public createProfile(name: string, config: ProfileConfig): void {
        if (this.config.profiles[name]) {
            throw new Error(`Profile '${name}' already exists`);
        }
        this.config.profiles[name] = config;
        this.saveConfig();
    }

    public updateProfile(name: string, updates: Partial<ProfileConfig>): void {
        if (!this.config.profiles[name]) {
            throw new Error(`Profile '${name}' does not exist`);
        }
        this.config.profiles[name] = {
            ...this.config.profiles[name],
            ...updates,
        };
        this.saveConfig();
    }

    public listProfiles(): string[] {
        return Object.keys(this.config.profiles);
    }

    public getAuthToken(): string | undefined {
        const profile = this.getActiveProfile();

        // Check if token is expired
        if (profile.tokenExpiry && Date.now() > profile.tokenExpiry) {
            this.clearAuth();
            return undefined;
        }

        return profile.authToken;
    }

    public getRefreshToken(): string | undefined {
        return this.getActiveProfile().refreshToken;
    }

    public setAuth(authToken: string, refreshToken?: string, expiresIn?: number): void {
        const profileName = this.config.currentProfile;
        const tokenExpiry = expiresIn ? Date.now() + (expiresIn * TIMEOUTS.COMMAND_TIMEOUT_MS) : undefined;

        this.updateProfile(profileName, {
            authToken,
            refreshToken,
            tokenExpiry,
        });
    }

    public setSession(session: Session): void {
        const profileName = this.config.currentProfile;
        const user = session.users?.[0];
        
        this.updateProfile(profileName, {
            userId: user?.publicId,
            handle: user?.handle || undefined,
            timeZone: session.timeZone || undefined,
        });
    }

    public clearAuth(): void {
        const profileName = this.config.currentProfile;
        this.updateProfile(profileName, {
            authToken: undefined,
            refreshToken: undefined,
            tokenExpiry: undefined,
            userId: undefined,
            handle: undefined,
            timeZone: undefined,
        });
    }

    public isDebug(): boolean {
        return this.debug;
    }

    public setDebug(value: boolean): void {
        this.debug = value;
    }

    public isJsonOutput(): boolean {
        return this.jsonOutput;
    }

    public setJsonOutput(value: boolean): void {
        this.jsonOutput = value;
    }

    public getServerUrl(): string {
        return this.getActiveProfile().url;
    }

    public getConfigDir(): string {
        return this.configDir;
    }
}
