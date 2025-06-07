import { type ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

const LATEST_CONFIG_VERSION = "1.0";

/**
 * Represents all data that can be stored in a note's stringified config.
 */
export type NoteVersionConfigObject = BaseConfigObject

/**
 * Top-level API config that encapsulates all API-related configuration data.
 */
export class NoteVersionConfig extends BaseConfig<NoteVersionConfigObject> {

    constructor({ config }: { config: NoteVersionConfigObject }) {
        super(config);
    }

    static parse(
        version: Pick<ResourceVersion, "config">,
        logger: PassableLogger,
        _opts?: { useFallbacks?: boolean },
    ): NoteVersionConfig {
        return super.parseBase<NoteVersionConfigObject, NoteVersionConfig>(
            version.config,
            logger,
            (cfg) => {
                // Add fallback properties as needed
                return new NoteVersionConfig({ config: cfg });
            },
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
