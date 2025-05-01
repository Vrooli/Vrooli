import { ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { type StringifyMode } from "./utils.js";

const LATEST_CONFIG_VERSION = "1.0";

/**
 * Represents all data that can be stored in a note's stringified config.
 */
export interface NoteVersionConfigObject extends BaseConfigObject {
    // Add properties as needed
}

/**
 * Top-level API config that encapsulates all API-related configuration data.
 */
export class NoteVersionConfig extends BaseConfig<NoteVersionConfigObject> {

    constructor({ config }: { config: NoteVersionConfigObject }) {
        super(config);
    }

    static deserialize(
        version: Pick<ResourceVersion, "config">,
        logger: PassableLogger,
        opts?: { mode?: StringifyMode; useFallbacks?: boolean }
    ): NoteVersionConfig {
        return this.parseConfig<NoteVersionConfigObject, NoteVersionConfig>(
            version.config,
            logger,
            (cfg) => {
                // Add fallback properties as needed
                return new NoteVersionConfig({ config: cfg });
            },
            { mode: opts?.mode }
        );
    }

    /**
     * Creates a default NoteVersionConfig
     */
    static default(): NoteVersionConfig {
        const config: NoteVersionConfigObject = {
            __version: LATEST_CONFIG_VERSION,
            resources: [],
        };
        return new NoteVersionConfig({ config });
    }

    /**
     * Exports the config to a plain object
     */
    override export(): NoteVersionConfigObject {
        return {
            ...super.export(),
        };
    }
} 
