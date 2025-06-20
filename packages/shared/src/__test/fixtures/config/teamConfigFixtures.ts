import { type TeamConfigObject } from "../../../shape/configs/team.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Team configuration fixtures for testing team organizational structures and settings
 */
export const teamConfigFixtures: ConfigTestFixtures<TeamConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        structure: {
            type: "MOISE+",
            version: "1.0",
            content: `structure VrooliTeam {
                group devTeam {
                    role teamLead cardinality 1..1
                    role developer cardinality 2..10
                    role designer cardinality 1..3
                    role qaEngineer cardinality 1..5
                    
                    link teamLead > developer
                    link teamLead > designer
                    link teamLead > qaEngineer
                    
                    subgroup frontend {
                        role frontendLead cardinality 1..1
                        role frontendDev cardinality 2..5
                        link frontendLead > frontendDev
                    }
                    
                    subgroup backend {
                        role backendLead cardinality 1..1
                        role backendDev cardinality 2..5
                        link backendLead > backendDev
                    }
                }
            }`,
        },
        resources: [
            {
                link: "https://github.com/Vrooli/Vrooli",
                usedFor: "Developer",
                translations: [{
                    language: "en",
                    name: "Team Repository",
                    description: "Main codebase for the team",
                }],
            },
            {
                link: "https://vrooli.com/team-handbook",
                usedFor: "OfficialWebsite",
                translations: [{
                    language: "en",
                    name: "Team Handbook",
                    description: "Team policies and procedures",
                }],
            },
        ],
    },

    withDefaults: {
        __version: LATEST_CONFIG_VERSION,
        structure: {
            type: "MOISE+",
            version: "1.0",
            content: "",
        },
    },

    invalid: {
        missingVersion: {
            // Missing __version
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: "structure Test {}",
            },
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: "",
            },
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            structure: "string instead of object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: 123, // Should be string
                version: true, // Should be string
                content: null, // Should be string
            },
        },
    },

    variants: {
        // Minimal team with just a leader and members
        simpleTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: `structure SimpleTeam {
                    group team {
                        role leader cardinality 1..1
                        role member cardinality 1..*
                        link leader > member
                    }
                }`,
            },
        },

        // Startup team structure
        startupTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: `structure StartupTeam {
                    group startup {
                        role founder cardinality 1..2
                        role developer cardinality 1..3
                        role marketer cardinality 0..1
                        role advisor cardinality 0..5
                        
                        link founder > developer
                        link founder > marketer
                        link advisor > founder
                    }
                }`,
            },
            resources: [{
                link: "https://startup.example.com",
                usedFor: "OfficialWebsite",
                translations: [{
                    language: "en",
                    name: "Startup Homepage",
                }],
            }],
        },

        // Research team structure
        researchTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: `structure ResearchTeam {
                    group researchLab {
                        role principalInvestigator cardinality 1..1
                        role postdoc cardinality 1..5
                        role phdStudent cardinality 2..10
                        role researchAssistant cardinality 0..5
                        
                        link principalInvestigator > postdoc
                        link principalInvestigator > phdStudent
                        link postdoc > researchAssistant
                        link phdStudent > researchAssistant
                    }
                }`,
            },
        },

        // Flat organization (no hierarchy)
        flatOrganization: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: `structure FlatOrg {
                    group collective {
                        role contributor cardinality 3..*
                        // No links - all contributors are equal
                    }
                }`,
            },
        },

        // Using FIPA ACL instead of MOISE+
        fipaTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "FIPA ACL",
                version: "2.0",
                content: `@prefix fipa: <http://www.fipa.org/ontology#> .
                @prefix team: <http://example.com/team#> .
                
                team:ProjectManager a fipa:AgentRole ;
                    fipa:hasCapability team:Planning ;
                    fipa:hasCapability team:Coordination .
                
                team:Developer a fipa:AgentRole ;
                    fipa:hasCapability team:Coding ;
                    fipa:hasCapability team:Testing .`,
            },
        },

        // Complex organization with multiple departments
        enterpriseTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: `structure Enterprise {
                    group company {
                        role ceo cardinality 1..1
                        role cto cardinality 1..1
                        role cfo cardinality 1..1
                        
                        link ceo > cto
                        link ceo > cfo
                        
                        subgroup engineering {
                            role vpEngineering cardinality 1..1
                            role engineeringManager cardinality 2..5
                            role seniorEngineer cardinality 5..20
                            role engineer cardinality 10..50
                            
                            link cto > vpEngineering
                            link vpEngineering > engineeringManager
                            link engineeringManager > seniorEngineer
                            link engineeringManager > engineer
                        }
                        
                        subgroup finance {
                            role vpFinance cardinality 1..1
                            role controller cardinality 1..2
                            role accountant cardinality 2..10
                            
                            link cfo > vpFinance
                            link vpFinance > controller
                            link controller > accountant
                        }
                    }
                }`,
            },
            resources: [
                {
                    link: "https://enterprise.example.com",
                    usedFor: "OfficialWebsite",
                    translations: [{
                        language: "en",
                        name: "Corporate Website",
                        description: "Public-facing corporate site",
                    }],
                },
                {
                    link: "https://wiki.enterprise.example.com",
                    usedFor: "Learning",
                    translations: [{
                        language: "en",
                        name: "Internal Wiki",
                        description: "Team knowledge base",
                    }],
                },
            ],
        },

        // Empty structure (team exists but no formal organization defined)
        unstructuredTeam: {
            __version: LATEST_CONFIG_VERSION,
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: "",
            },
        },
    },
};

/**
 * Create a team config with a specific organizational structure
 */
export function createTeamConfigWithStructure(
    type: string,
    content: string,
    version?: string,
): TeamConfigObject {
    return mergeWithBaseDefaults<TeamConfigObject>({
        structure: {
            type,
            version: version ?? "1.0",
            content,
        },
    });
}

/**
 * Create a MOISE+ team structure with custom roles
 */
export function createMoiseTeamStructure(
    groupName: string,
    roles: Array<{ name: string; min?: number; max?: number }>,
): TeamConfigObject {
    const roleDefinitions = roles.map(role =>
        `role ${role.name} cardinality ${role.min ?? 1}..${role.max ?? "*"}`,
    ).join("\n        ");

    const content = `structure ${groupName}Team {
    group ${groupName} {
        ${roleDefinitions}
    }
}`;

    return createTeamConfigWithStructure("MOISE+", content);
}

/**
 * Create a hierarchical team structure
 */
export function createHierarchicalTeamStructure(
    levels: Array<{ role: string; reportsTo?: string; count?: { min: number; max: number } }>,
): TeamConfigObject {
    const roles = levels.map(level =>
        `role ${level.role} cardinality ${level.count?.min ?? 1}..${level.count?.max ?? "*"}`,
    ).join("\n        ");

    const links = levels
        .filter(level => level.reportsTo)
        .map(level => `link ${level.reportsTo} > ${level.role}`)
        .join("\n        ");

    const content = `structure HierarchicalTeam {
    group organization {
        ${roles}
        ${links ? "\n        " + links : ""}
    }
}`;

    return createTeamConfigWithStructure("MOISE+", content);
}
