import { type ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./baseConfig.js";
import { type StringifyMode } from "./utils.js";

const LATEST_CONFIG_VERSION = "1.0";

/**
 * Represents all data that can be stored in a Project's stringified config.
 */
export type ProjectVersionConfigObject = BaseConfigObject

/**
 * Top-level API config that encapsulates all API-related configuration data.
 */
export class ProjectVersionConfig extends BaseConfig<ProjectVersionConfigObject> {

    constructor({ config }: { config: ProjectVersionConfigObject }) {
        super(config);
    }

    static parse(
        version: Pick<ResourceVersion, "config">,
        logger: PassableLogger,
        opts?: { mode?: StringifyMode; useFallbacks?: boolean },
    ): ProjectVersionConfig {
        return super.parseBase<ProjectVersionConfigObject, ProjectVersionConfig>(
            version.config,
            logger,
            (cfg) => {
                // Add fallback properties as needed
                return new ProjectVersionConfig({ config: cfg });
            },
            { mode: opts?.mode },
        );
    }

    /**
     * Creates a default ProjectVersionConfig
     */
    static default(): ProjectVersionConfig {
        const config: ProjectVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
        };
        return new ProjectVersionConfig({ config });
    }

    /**
     * Exports the config to a plain object
     */
    override export(): ProjectVersionConfigObject {
        return {
            ...super.export(),
        };
    }
} 
