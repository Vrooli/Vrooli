import { ModelStrategy } from "../../../shape/configs/base.js";
import { type TeamConfigObject, STANDARD_RESOURCE_QUOTAS } from "../../../shape/configs/team.js";
import { LATEST_CONFIG_VERSION } from "../../../shape/configs/utils.js";
import { type ConfigTestFixtures, mergeWithBaseDefaults } from "./baseConfigFixtures.js";

/**
 * Team configuration fixtures for testing team organizational structures and settings
 */
export const teamConfigFixtures: ConfigTestFixtures<TeamConfigObject> = {
    minimal: {
        __version: LATEST_CONFIG_VERSION,
        deploymentType: "development",
        goal: "Test team for development purposes",
        businessPrompt: "A minimal test team configuration",
        resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
        targetProfitPerMonth: "1000000", // $1 in micro-dollars (stringified bigint)
        stats: {
            totalInstances: 0,
            totalProfit: "0",
            totalCosts: "0",
            averageKPIs: {},
            activeInstances: 0,
            lastUpdated: Date.now(),
        },
    },

    complete: {
        __version: LATEST_CONFIG_VERSION,
        deploymentType: "saas",
        goal: "Build and manage software development teams with autonomous AI agents",
        businessPrompt: "Optimize team productivity and code quality while maintaining cost efficiency. Focus on delivering high-quality software through collaborative AI-human teams.",
        resourceQuota: STANDARD_RESOURCE_QUOTAS.standard,
        targetProfitPerMonth: "50000000000", // $50k in micro-dollars (stringified bigint)
        costLimit: "10000000000", // $10k monthly cost limit
        verticalPackage: {
            industry: "software-development",
            complianceRequirements: ["SOC2", "ISO27001"],
            defaultWorkflows: ["code-review", "testing", "deployment"],
            dataPrivacyLevel: "standard",
            terminology: {
                "project": "repository",
                "task": "issue",
                "worker": "developer",
            },
            regulatoryBodies: ["ISO", "AICPA"],
        },
        marketSegment: "enterprise",
        isolation: {
            sandboxLevel: "full-container",
            networkPolicy: "restricted",
            secretsAccess: ["/secrets/github", "/secrets/aws"],
            auditLogging: true,
            securityPolicies: ["no-external-network", "code-signing-required"],
        },
        modelConfig: {
            strategy: ModelStrategy.QUALITY_FIRST,
            preferredModel: "gpt-4",
            offlineOnly: false,
        },
        defaultLimits: {
            maxToolCallsPerBotResponse: 10,
            maxToolCalls: 50,
            maxCreditsPerBotResponse: "1000000",
            maxCredits: "5000000",
            maxDurationPerBotResponseMs: 300000,
            maxDurationMs: 3600000,
            delayBetweenProcessingCyclesMs: 0,
        },
        defaultScheduling: {
            defaultDelayMs: 0,
            toolSpecificDelays: { "expensive-api": 1000 },
            requiresApprovalTools: ["deploy", "delete"],
            approvalTimeoutMs: 600000,
            autoRejectOnTimeout: true,
        },
        defaultPolicy: {
            visibility: "restricted",
            acl: ["team-members", "project-managers"],
        },
        stats: {
            totalInstances: 42,
            totalProfit: "125000000000", // $125k
            totalCosts: "25000000000", // $25k
            averageKPIs: {
                "code-quality": 0.92,
                "delivery-speed": 0.85,
                "cost-efficiency": 0.78,
            },
            activeInstances: 8,
            lastUpdated: Date.now(),
        },
        blackboard: [
            {
                id: "insight-1",
                value: {
                    type: "optimization",
                    data: { recommendation: "Increase parallel processing for code reviews" },
                    confidence: 0.85,
                },
                created_at: new Date().toISOString(),
            },
        ],
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
        deploymentType: "development",
        goal: "Default test team configuration",
        businessPrompt: "A test team with default settings",
        resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
        targetProfitPerMonth: "0", // No profit target for test
        stats: {
            totalInstances: 0,
            totalProfit: "0",
            totalCosts: "0",
            averageKPIs: {},
            activeInstances: 0,
            lastUpdated: Date.now(),
        },
        structure: {
            type: "MOISE+",
            version: "1.0",
            content: "",
        },
    },

    invalid: {
        missingVersion: {
            // Missing __version
            deploymentType: "development",
            goal: "Test invalid config",
            businessPrompt: "Testing missing version",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "0",
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: "structure Test {}",
            },
        },
        invalidVersion: {
            __version: "0.1", // Invalid version
            deploymentType: "development",
            goal: "Test invalid version",
            businessPrompt: "Testing invalid version",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "0",
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
            structure: {
                type: "MOISE+",
                version: "1.0",
                content: "",
            },
        },
        malformedStructure: {
            __version: LATEST_CONFIG_VERSION,
            deploymentType: "development",
            goal: "Test malformed structure",
            businessPrompt: "Testing malformed structure",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "0",
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
            structure: "string instead of object", // Wrong type
        },
        invalidTypes: {
            __version: LATEST_CONFIG_VERSION,
            deploymentType: "development",
            goal: "Test invalid types",
            businessPrompt: "Testing invalid types",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "0",
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
            structure: {
                // @ts-expect-error - Intentionally invalid type for testing
                type: 123, // Should be string
                version: true as any, // Should be string - invalid type for testing
                content: null as any, // Should be string - invalid type for testing
            },
        },
    },

    variants: {
        // Minimal team with just a leader and members
        simpleTeam: {
            __version: LATEST_CONFIG_VERSION,
            deploymentType: "development",
            goal: "Simple team structure for basic coordination",
            businessPrompt: "Coordinate simple tasks between leader and team members",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "5000000000", // $5k in micro-dollars
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "saas",
            goal: "Build MVP and scale startup operations",
            businessPrompt: "Optimize for rapid iteration and growth while maintaining lean operations",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.standard,
            targetProfitPerMonth: "25000000000", // $25k in micro-dollars
            stats: {
                totalInstances: 3,
                totalProfit: "15000000000",
                totalCosts: "3000000000",
                averageKPIs: {
                    "velocity": 0.88,
                    "burn-rate": 0.65,
                },
                activeInstances: 2,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "appliance",
            goal: "Conduct academic research and publish findings",
            businessPrompt: "Focus on research quality and publication output while managing grants efficiently",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.heavy,
            targetProfitPerMonth: "0", // Research teams typically don't target profit
            stats: {
                totalInstances: 1,
                totalProfit: "0",
                totalCosts: "8000000000",
                averageKPIs: {
                    "publications": 0.75,
                    "grant-efficiency": 0.82,
                },
                activeInstances: 1,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "development",
            goal: "Enable collaborative decision-making without hierarchy",
            businessPrompt: "Foster autonomous collaboration with equal participation from all contributors",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.standard,
            targetProfitPerMonth: "10000000000", // $10k in micro-dollars
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "saas",
            goal: "Implement agent communication using FIPA ACL standards",
            businessPrompt: "Use standardized agent communication protocols for interoperability",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.standard,
            targetProfitPerMonth: "15000000000", // $15k in micro-dollars
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "appliance",
            goal: "Manage complex enterprise operations across multiple departments",
            businessPrompt: "Optimize enterprise-wide efficiency while maintaining departmental autonomy and compliance",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.ultra,
            targetProfitPerMonth: "500000000000", // $500k in micro-dollars
            costLimit: "100000000000", // $100k cost limit
            stats: {
                totalInstances: 25,
                totalProfit: "750000000000",
                totalCosts: "150000000000",
                averageKPIs: {
                    "operational-efficiency": 0.89,
                    "compliance-score": 0.95,
                    "roi": 0.8,
                },
                activeInstances: 20,
                lastUpdated: Date.now(),
            },
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
            deploymentType: "development",
            goal: "Operate without formal organizational structure",
            businessPrompt: "Allow organic team formation and task distribution",
            resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
            targetProfitPerMonth: "1000000000", // $1k in micro-dollars
            stats: {
                totalInstances: 0,
                totalProfit: "0",
                totalCosts: "0",
                averageKPIs: {},
                activeInstances: 0,
                lastUpdated: Date.now(),
            },
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
        deploymentType: "development",
        goal: "Custom team structure for testing",
        businessPrompt: "Test team with custom organizational structure",
        resourceQuota: STANDARD_RESOURCE_QUOTAS.light,
        targetProfitPerMonth: "1000000000", // $1k in micro-dollars
        stats: {
            totalInstances: 0,
            totalProfit: "0",
            totalCosts: "0",
            averageKPIs: {},
            activeInstances: 0,
            lastUpdated: Date.now(),
        },
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
