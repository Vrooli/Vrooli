import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { LATEST_CONFIG_VERSION, parseObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all data that can be stored in a team's stringified config.
 *
 * This includes configuration that doesn't need to be queried while searching
 * (e.g., team settings, automation config, etc).
 *
 * ## 🔍 New in vNEXT – `organizationModel`
 * Use this section to declare **how the team is structured** so autonomous agents can
 * reason about leadership, roles, and obligations.
 *
 * ```json5
 * {
 *   "organizationModel": {
 *     "type": "MOISE+",      // 👉 recommended default
 *     "version": "1.0",      // optional – spec or dialect version
 *     "content": """          // raw model payload (string)
 *     structure Swarm {
 *       group devTeam {
 *         role leader  cardinality 1..1
 *         role member  cardinality 2..*
 *         link leader > member
 *       }
 *     }
 *     """
 *   }
 * }
 * ```
 */
export interface TeamConfigObject extends BaseConfigObject {
    /**
     * Roles available to team mmembers
     */
    /**
     * Declarative description of the team's organisational structure.
     *
     * * **type** – the standard you are using (`"MOISE+"`, `"FIPA ACL"`, `"ArchiMate"`, etc.).
     *   *If you are unsure, start with **MOISE+** – purpose‑built for multi‑agent systems.*
     * * **version** – (optional) version of the chosen spec/dialect.
     * * **content** – raw payload (string). Could be a MOISE+ structural layer snippet, a FIPA ACL role ontology in Turtle, an ArchiMate XML export, etc.
     */
    structure?: {
        type: string; // e.g. "MOISE+", "FIPA ACL"
        version?: string;
        content: string;
    };
}

/**
 * Top-level Team config that encapsulates all team-related configuration data.
 */
export class TeamConfig extends BaseConfig<TeamConfigObject> {
    structure?: TeamConfigObject["structure"];

    constructor(data: TeamConfigObject) {
        super(data);
        this.structure = data.structure;
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
            __version: LATEST_CONFIG_VERSION,
        };
        return new TeamConfig(data);
    }

    /**
     * Exports the config to a plain object
     */
    override export(): TeamConfigObject {
        return {
            ...super.export(),
            structure: this.structure,
        };
    }

    /**
     * Sets or replaces the organisation‑model definition.
     */
    setStructure(structure: TeamConfigObject["structure"]): void {
        this.structure = structure;
    }
}
