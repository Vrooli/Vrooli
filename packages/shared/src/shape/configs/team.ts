import { type Team } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";


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
export type TeamConfigObject = BaseConfigObject & {
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

function defaultStructure(): TeamConfigObject["structure"] {
    return {
        type: "MOISE+",
        version: "1.0",
        content: "",
    };
}

/**
 * Top-level Team config that encapsulates all team-related configuration data.
 */
export class TeamConfig extends BaseConfig<TeamConfigObject> {
    structure?: TeamConfigObject["structure"];

    constructor({ config }: { config: TeamConfigObject }) {
        super({ config });
        this.structure = config.structure;
    }

    static parse(
        version: Pick<Team, "config">,
        logger: PassableLogger,
        opts?: { useFallbacks?: boolean },
    ): TeamConfig {
        return super.parseBase<TeamConfigObject, TeamConfig>(
            version.config,
            logger,
            ({ config }) => {
                if (opts?.useFallbacks ?? true) {
                    config.structure ??= defaultStructure();
                }
                return new TeamConfig({ config });
            },
        );
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
