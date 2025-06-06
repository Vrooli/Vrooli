import { type Logger } from "winston";
import { type RunContext } from "@vrooli/shared";

/**
 * MOISE+ permission check result
 */
export interface PermissionCheckResult {
    permitted: boolean;
    reason?: string;
    requiredRole?: string;
    requiredPermissions?: string[];
}

/**
 * MOISE+ role definition
 */
export interface MOISERole {
    id: string;
    name: string;
    permissions: string[];
    inherits?: string[];
}

/**
 * MOISE+ mission definition
 */
export interface MOISEMission {
    id: string;
    name: string;
    requiredRoles: string[];
    goals: string[];
    constraints: Record<string, unknown>;
}

/**
 * MOISE+ organizational specification
 */
export interface MOISEOrgSpec {
    roles: MOISERole[];
    missions: MOISEMission[];
    norms: MOISENorm[];
}

/**
 * MOISE+ norm (permission/obligation/prohibition)
 */
export interface MOISENorm {
    id: string;
    type: "permission" | "obligation" | "prohibition";
    role: string;
    mission?: string;
    condition?: string;
    resource: string;
    action: string;
}

/**
 * MOISEGate - MOISE+ organizational model for permission validation
 * 
 * This component implements the MOISE+ (Model of Organization for multI-agent
 * SystEms) framework for managing permissions and organizational structure
 * in the execution system.
 * 
 * MOISE+ provides:
 * - Role-based access control
 * - Mission-oriented task allocation
 * - Normative specifications (permissions, obligations, prohibitions)
 * - Dynamic organization adaptation
 * - Multi-level permission inheritance
 * 
 * This ensures that agents and users can only execute steps they are
 * authorized to perform within the organizational context.
 */
export class MOISEGate {
    private readonly logger: Logger;
    private orgSpec: MOISEOrgSpec;
    private roleCache: Map<string, MOISERole> = new Map();
    private permissionCache: Map<string, Set<string>> = new Map();

    constructor(logger: Logger) {
        this.logger = logger;
        this.initializeDefaultOrgSpec();
    }

    /**
     * Checks if a run can execute a specific step
     */
    async checkPermission(
        runId: string,
        stepId: string,
        context: RunContext,
    ): Promise<boolean> {
        const result = await this.performPermissionCheck(runId, stepId, context);
        
        if (!result.permitted) {
            this.logger.warn("[MOISEGate] Permission denied", {
                runId,
                stepId,
                reason: result.reason,
                requiredRole: result.requiredRole,
                requiredPermissions: result.requiredPermissions,
            });
        }

        return result.permitted;
    }

    /**
     * Performs detailed permission check
     */
    async performPermissionCheck(
        runId: string,
        stepId: string,
        context: RunContext,
    ): Promise<PermissionCheckResult> {
        // Extract user role from context
        const userRole = context.variables.userRole as string || "user";
        const userMission = context.variables.userMission as string;
        
        // Get step metadata from context
        const stepType = context.variables[`step_${stepId}_type`] as string || "action";
        const stepResource = context.variables[`step_${stepId}_resource`] as string || stepId;
        
        // Check role permissions
        const rolePermissions = await this.getRolePermissions(userRole);
        
        // Check if user has permission for this step type
        const requiredPermission = `execute:${stepType}`;
        if (!rolePermissions.has(requiredPermission)) {
            return {
                permitted: false,
                reason: "Missing required permission",
                requiredRole: this.findRoleWithPermission(requiredPermission),
                requiredPermissions: [requiredPermission],
            };
        }

        // Check norms (permissions, obligations, prohibitions)
        const normCheck = await this.checkNorms(
            userRole,
            userMission,
            stepResource,
            "execute",
            context,
        );

        if (!normCheck.permitted) {
            return normCheck;
        }

        // Check mission constraints if applicable
        if (userMission) {
            const missionCheck = await this.checkMissionConstraints(
                userMission,
                stepId,
                context,
            );
            
            if (!missionCheck.permitted) {
                return missionCheck;
            }
        }

        return { permitted: true };
    }

    /**
     * Gets all permissions for a role (including inherited)
     */
    async getRolePermissions(roleId: string): Promise<Set<string>> {
        // Check cache
        if (this.permissionCache.has(roleId)) {
            return this.permissionCache.get(roleId)!;
        }

        const permissions = new Set<string>();
        const role = this.roleCache.get(roleId);
        
        if (!role) {
            this.logger.warn("[MOISEGate] Unknown role", { roleId });
            return permissions;
        }

        // Add direct permissions
        for (const permission of role.permissions) {
            permissions.add(permission);
        }

        // Add inherited permissions
        if (role.inherits) {
            for (const parentRoleId of role.inherits) {
                const parentPermissions = await this.getRolePermissions(parentRoleId);
                for (const permission of parentPermissions) {
                    permissions.add(permission);
                }
            }
        }

        // Cache result
        this.permissionCache.set(roleId, permissions);

        return permissions;
    }

    /**
     * Checks norms for permission/prohibition
     */
    private async checkNorms(
        role: string,
        mission: string | undefined,
        resource: string,
        action: string,
        context: RunContext,
    ): Promise<PermissionCheckResult> {
        const applicableNorms = this.orgSpec.norms.filter(norm => {
            // Check if norm applies to this role
            if (norm.role !== role && norm.role !== "*") {
                return false;
            }

            // Check if norm applies to this mission
            if (norm.mission && norm.mission !== mission) {
                return false;
            }

            // Check if norm applies to this resource/action
            if (norm.resource !== resource && norm.resource !== "*") {
                return false;
            }
            
            if (norm.action !== action && norm.action !== "*") {
                return false;
            }

            // Check condition if present
            if (norm.condition) {
                return this.evaluateCondition(norm.condition, context);
            }

            return true;
        });

        // Check for prohibitions first (they override permissions)
        const prohibition = applicableNorms.find(n => n.type === "prohibition");
        if (prohibition) {
            return {
                permitted: false,
                reason: `Action prohibited by norm ${prohibition.id}`,
            };
        }

        // Check for explicit permissions
        const permission = applicableNorms.find(n => n.type === "permission");
        if (permission) {
            return { permitted: true };
        }

        // Check for obligations (must be fulfilled)
        const obligation = applicableNorms.find(n => n.type === "obligation");
        if (obligation) {
            // For now, treat obligations as permissions
            // In a full implementation, we would track obligation fulfillment
            return { permitted: true };
        }

        // Default: check role permissions
        return { permitted: true };
    }

    /**
     * Checks mission-specific constraints
     */
    private async checkMissionConstraints(
        missionId: string,
        stepId: string,
        context: RunContext,
    ): Promise<PermissionCheckResult> {
        const mission = this.orgSpec.missions.find(m => m.id === missionId);
        
        if (!mission) {
            return {
                permitted: false,
                reason: `Unknown mission: ${missionId}`,
            };
        }

        // Check if step contributes to mission goals
        const stepGoal = context.variables[`step_${stepId}_goal`] as string;
        if (stepGoal && !mission.goals.includes(stepGoal)) {
            return {
                permitted: false,
                reason: "Step does not contribute to mission goals",
            };
        }

        // Check mission constraints
        for (const [constraint, value] of Object.entries(mission.constraints)) {
            const contextValue = context.variables[constraint];
            
            // Simple equality check for now
            if (contextValue !== value) {
                return {
                    permitted: false,
                    reason: `Mission constraint not met: ${constraint}`,
                };
            }
        }

        return { permitted: true };
    }

    /**
     * Evaluates a condition expression
     */
    private evaluateCondition(condition: string, context: RunContext): boolean {
        try {
            // Simple condition evaluation
            // In production, use a proper expression evaluator
            const contextVars = { ...context.variables, ...context.blackboard };
            
            // Replace variable references
            let evalCondition = condition;
            for (const [key, value] of Object.entries(contextVars)) {
                evalCondition = evalCondition.replace(
                    new RegExp(`\\$${key}\\b`, "g"),
                    JSON.stringify(value),
                );
            }

            // Evaluate (UNSAFE - use proper parser in production)
            return eval(evalCondition) === true;
        } catch (error) {
            this.logger.error("[MOISEGate] Condition evaluation failed", {
                condition,
                error: error instanceof Error ? error.message : String(error),
            });
            return false;
        }
    }

    /**
     * Finds a role that has a specific permission
     */
    private findRoleWithPermission(permission: string): string | undefined {
        for (const role of this.orgSpec.roles) {
            if (role.permissions.includes(permission)) {
                return role.id;
            }
        }
        return undefined;
    }

    /**
     * Initializes default organizational specification
     */
    private initializeDefaultOrgSpec(): void {
        // Default roles
        const adminRole: MOISERole = {
            id: "admin",
            name: "Administrator",
            permissions: [
                "execute:*",
                "manage:*",
                "view:*",
            ],
        };

        const userRole: MOISERole = {
            id: "user",
            name: "User",
            permissions: [
                "execute:action",
                "execute:decision",
                "execute:loop",
                "view:own",
            ],
        };

        const guestRole: MOISERole = {
            id: "guest",
            name: "Guest",
            permissions: [
                "view:public",
            ],
        };

        // Agent roles
        const agentRole: MOISERole = {
            id: "agent",
            name: "AI Agent",
            permissions: [
                "execute:action",
                "execute:decision",
                "execute:subroutine",
                "execute:parallel",
            ],
            inherits: ["user"],
        };

        // Default missions
        const explorationMission: MOISEMission = {
            id: "exploration",
            name: "Exploration Mission",
            requiredRoles: ["user", "agent"],
            goals: ["discover", "analyze", "report"],
            constraints: {
                maxDepth: 5,
                allowExternalCalls: false,
            },
        };

        const automationMission: MOISEMission = {
            id: "automation",
            name: "Automation Mission",
            requiredRoles: ["agent", "admin"],
            goals: ["optimize", "execute", "monitor"],
            constraints: {
                requireApproval: true,
            },
        };

        // Default norms
        const guestProhibition: MOISENorm = {
            id: "no-guest-execution",
            type: "prohibition",
            role: "guest",
            resource: "*",
            action: "execute",
        };

        const externalCallPermission: MOISENorm = {
            id: "admin-external-calls",
            type: "permission",
            role: "admin",
            resource: "external_api",
            action: "call",
        };

        // Initialize org spec
        this.orgSpec = {
            roles: [adminRole, userRole, guestRole, agentRole],
            missions: [explorationMission, automationMission],
            norms: [guestProhibition, externalCallPermission],
        };

        // Cache roles
        for (const role of this.orgSpec.roles) {
            this.roleCache.set(role.id, role);
        }

        this.logger.info("[MOISEGate] Initialized default organization specification", {
            roleCount: this.orgSpec.roles.length,
            missionCount: this.orgSpec.missions.length,
            normCount: this.orgSpec.norms.length,
        });
    }

    /**
     * Updates organizational specification
     */
    async updateOrgSpec(orgSpec: MOISEOrgSpec): Promise<void> {
        this.orgSpec = orgSpec;
        
        // Clear caches
        this.roleCache.clear();
        this.permissionCache.clear();
        
        // Rebuild role cache
        for (const role of orgSpec.roles) {
            this.roleCache.set(role.id, role);
        }

        this.logger.info("[MOISEGate] Updated organization specification");
    }

    /**
     * Gets current organizational specification
     */
    getOrgSpec(): MOISEOrgSpec {
        return this.orgSpec;
    }
}