import { type PassableLogger } from "../../consts/commonTypes.js";
import { LATEST_RUN_CONFIG_VERSION } from "../consts.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { parseObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all data that can be stored in a team's stringified config.
 * 
 * This includes configuration that doesn't need to be queried while searching
 * (e.g., team settings, automation config, etc).
 */
export interface TeamConfigObject extends BaseConfigObject {
    /** Access control settings */
    access?: {
        /** Role-based access control */
        rbac?: {
            /** Roles and their permissions */
            roles?: Record<string, string[]>;
        };
    };
    /** Integration settings with other systems */
    integrations?: {
        /** External services integration */
        services?: Record<string, unknown>;
        /** Webhook configurations */
        webhooks?: Array<{
            /** Webhook URL */
            url: string;
            /** Events that trigger this webhook */
            events: string[];
            /** Secret for webhook verification */
            secret?: string;
        }>;
    };
}

/**
 * Top-level Team config that encapsulates all team-related configuration data.
 */
export class TeamConfig extends BaseConfig<TeamConfigObject> {
    access?: TeamConfigObject["access"];
    integrations?: TeamConfigObject["integrations"];

    constructor(data: TeamConfigObject) {
        super(data);
        this.access = data.access;
        this.integrations = data.integrations;
    }

    /**
     * Creates a TeamConfig from a team object
     */
    static createFromTeam(
        team: {
            config?: string | null;
        },
        logger: PassableLogger,
        options: { mode?: StringifyMode } = {},
    ): TeamConfig {
        const mode = options.mode || DEFAULT_STRINGIFY_MODE;
        const obj = team.config ? parseObject<TeamConfigObject>(team.config, mode, logger) : null;
        if (!obj) {
            return TeamConfig.default();
        }
        return new TeamConfig(obj);
    }

    /**
     * Creates a default TeamConfig
     */
    static default(): TeamConfig {
        const data: TeamConfigObject = {
            __version: LATEST_RUN_CONFIG_VERSION,
            resources: [],
            metadata: {},
        };
        return new TeamConfig(data);
    }

    /**
     * Exports the config to a plain object
     */
    override export(): TeamConfigObject {
        return {
            ...super.export(),
            access: this.access,
            integrations: this.integrations,
        };
    }

    /**
     * Sets access control settings
     */
    setAccess(access: TeamConfigObject["access"]): void {
        this.access = access;
    }

    /**
     * Sets integration settings
     */
    setIntegrations(integrations: TeamConfigObject["integrations"]): void {
        this.integrations = integrations;
    }
}
