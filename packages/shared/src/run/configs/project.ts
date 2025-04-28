import { type PassableLogger } from "../../consts/commonTypes.js";
import { LATEST_RUN_CONFIG_VERSION } from "../consts.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { parseObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all data that can be stored in a project's stringified config.
 * 
 * This includes configuration that doesn't need to be queried while searching
 * (e.g., project settings, automation config, etc).
 */
export type ProjectVersionConfigObject = BaseConfigObject

/**
 * Top-level Project config that encapsulates all project-related configuration data.
 */
export class ProjectVersionConfig extends BaseConfig<ProjectVersionConfigObject> {

    constructor(data: ProjectVersionConfigObject) {
        super(data);
    }

    /**
     * Creates a ProjectVersionConfig from a project version object
     */
    static createFromProjectVersion(
        projectVersion: {
            data?: string | null;
        },
        logger: PassableLogger,
        options: { mode?: StringifyMode } = {},
    ): ProjectVersionConfig {
        const mode = options.mode || DEFAULT_STRINGIFY_MODE;
        const obj = projectVersion.data ? parseObject<ProjectVersionConfigObject>(projectVersion.data, mode, logger) : null;
        if (!obj) {
            return ProjectVersionConfig.default();
        }
        return new ProjectVersionConfig(obj);
    }

    /**
     * Creates a default ProjectVersionConfig
     */
    static default(): ProjectVersionConfig {
        const data: ProjectVersionConfigObject = {
            __version: LATEST_RUN_CONFIG_VERSION,
            resources: [],
            metadata: {},
        };
        return new ProjectVersionConfig(data);
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
