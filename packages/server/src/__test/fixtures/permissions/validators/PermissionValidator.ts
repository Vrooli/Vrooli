/**
 * Permission Validator
 * 
 * Utilities for validating permissions and access control in tests.
 */

import { type AuthenticatedSessionData } from "../../../../types.js";
import { 
    type ApiKeyAuthData,
    type PermissionInheritance,
    type PermissionValidator as IPermissionValidator,
} from "../types.js";

/**
 * Implementation of permission validation utilities
 */
export class PermissionValidator implements IPermissionValidator {
    
    /**
     * Check if a session has a specific permission
     */
    hasPermission(
        session: AuthenticatedSessionData | ApiKeyAuthData,
        permission: string,
    ): boolean {
        // Check if it's an API key session
        if (this.isApiKeySession(session)) {
            return this.checkApiKeyPermission(session, permission);
        }

        // For user sessions, check roles
        if (!session.roles || session.roles.length === 0) {
            return false;
        }

        // Check each role
        for (const roleData of session.roles) {
            const permissions = this.parsePermissions(roleData.role.permissions);
            
            // Check for wildcard permission
            if (permissions.includes("*")) {
                return true;
            }

            // Check for exact permission
            if (permissions.includes(permission)) {
                return true;
            }

            // Check for negated permission
            if (permissions.includes(`!${permission}`)) {
                return false;
            }

            // Check for partial wildcard (e.g., "content.*")
            const permissionParts = permission.split(".");
            for (let i = permissionParts.length; i > 0; i--) {
                const wildcardPermission = permissionParts.slice(0, i - 1).join(".") + ".*";
                if (permissions.includes(wildcardPermission)) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Check if a session can perform an action on a resource
     */
    canAccess(
        session: AuthenticatedSessionData | ApiKeyAuthData,
        action: string,
        resource: Record<string, unknown>,
    ): boolean {
        // Basic permission check
        if (!this.hasPermission(session, `${resource.__typename?.toLowerCase()}.${action}`)) {
            return false;
        }

        // Additional checks based on resource ownership
        if (this.isUserSession(session)) {
            // Check if user owns the resource
            if (resource.owner?.id === session.id) {
                return true;
            }

            // Check team membership if resource is team-owned
            if (resource.team && session._testTeamMembership) {
                const membership = session._testTeamMembership as { teamId: string; role: string };
                if (membership.teamId === resource.team.id) {
                    // Owners and admins can do anything
                    if (membership.role === "Owner" || membership.role === "Admin") {
                        return true;
                    }
                    // Members have limited access
                    if (membership.role === "Member" && action === "read") {
                        return true;
                    }
                }
            }
        }

        // Check visibility for read actions
        if (action === "read" && resource.isPublic) {
            return true;
        }

        // Default deny for API keys on non-public resources
        if (this.isApiKeySession(session) && !resource.isPublic) {
            const permLevel = session.permissions[action as keyof typeof session.permissions];
            if (permLevel === "Private" || permLevel === "Auth") {
                // Check if API key belongs to resource owner
                return resource.owner?.id === session.userId;
            }
            return false;
        }

        return false;
    }

    /**
     * Get all permissions for a session
     */
    getPermissions(session: AuthenticatedSessionData | ApiKeyAuthData): string[] {
        if (this.isApiKeySession(session)) {
            return this.getApiKeyPermissions(session);
        }

        const allPermissions: string[] = [];
        
        if (!session.roles || session.roles.length === 0) {
            return allPermissions;
        }

        for (const roleData of session.roles) {
            const permissions = this.parsePermissions(roleData.role.permissions);
            allPermissions.push(...permissions);
        }

        // Remove duplicates and handle negations
        const uniquePermissions = new Set<string>();
        const negatedPermissions = new Set<string>();

        for (const perm of allPermissions) {
            if (perm.startsWith("!")) {
                negatedPermissions.add(perm.substring(1));
            } else {
                uniquePermissions.add(perm);
            }
        }

        // Remove negated permissions
        for (const negated of negatedPermissions) {
            uniquePermissions.delete(negated);
        }

        return Array.from(uniquePermissions);
    }

    /**
     * Validate permission inheritance chain
     */
    validateInheritance(chain: PermissionInheritance[]): boolean {
        const seenNames = new Set<string>();
        
        for (const item of chain) {
            // Check for circular references
            if (item.inherits && seenNames.has(item.inherits)) {
                return false;
            }
            
            // Validate structure
            if (item.adds && !Array.isArray(item.adds)) {
                return false;
            }
            
            if (item.excludes && !Array.isArray(item.excludes)) {
                return false;
            }
            
            // Track seen names
            if (item.inherits) {
                seenNames.add(item.inherits);
            }
        }

        return true;
    }

    /**
     * Check if session is a user session
     */
    private isUserSession(session: AuthenticatedSessionData | ApiKeyAuthData): session is AuthenticatedSessionData {
        return !("__type" in session);
    }

    /**
     * Check if session is an API key session
     */
    private isApiKeySession(session: AuthenticatedSessionData | ApiKeyAuthData): session is ApiKeyAuthData {
        return "__type" in session;
    }

    /**
     * Parse permissions string
     */
    private parsePermissions(permissionsJson: string): string[] {
        try {
            const parsed = JSON.parse(permissionsJson);
            return Array.isArray(parsed) ? parsed : [];
        } catch {
            return [];
        }
    }

    /**
     * Check API key permission
     */
    private checkApiKeyPermission(session: ApiKeyAuthData, permission: string): boolean {
        // Bot API keys have special permissions
        if (session.permissions.bot && permission.startsWith("bot.")) {
            return true;
        }

        // Parse the permission to determine read/write
        const [_resource, action] = permission.split(".");
        
        if (action === "read") {
            return this.checkApiKeyReadPermission(session.permissions.read);
        }
        
        if (action === "write" || action === "create" || action === "update" || action === "delete") {
            return this.checkApiKeyWritePermission(session.permissions.write);
        }

        return false;
    }

    /**
     * Check API key read permission level
     */
    private checkApiKeyReadPermission(level: string): boolean {
        return level !== "None";
    }

    /**
     * Check API key write permission level
     */
    private checkApiKeyWritePermission(level: string): boolean {
        return level !== "None";
    }

    /**
     * Get API key permissions as string array
     */
    private getApiKeyPermissions(session: ApiKeyAuthData): string[] {
        const permissions: string[] = [];

        // Read permissions
        if (session.permissions.read !== "None") {
            permissions.push(`*.read.${session.permissions.read.toLowerCase()}`);
        }

        // Write permissions
        if (session.permissions.write !== "None") {
            permissions.push(`*.write.${session.permissions.write.toLowerCase()}`);
            permissions.push(`*.create.${session.permissions.write.toLowerCase()}`);
            permissions.push(`*.update.${session.permissions.write.toLowerCase()}`);
            permissions.push(`*.delete.${session.permissions.write.toLowerCase()}`);
        }

        // Bot permissions
        if (session.permissions.bot) {
            permissions.push("bot.*");
        }

        return permissions;
    }

    /**
     * Check if a user has any of the specified permissions
     */
    hasAnyPermission(
        session: AuthenticatedSessionData | ApiKeyAuthData,
        permissions: string[],
    ): boolean {
        for (const permission of permissions) {
            if (this.hasPermission(session, permission)) {
                return true;
            }
        }
        return false;
    }

    /**
     * Check if a user has all of the specified permissions
     */
    hasAllPermissions(
        session: AuthenticatedSessionData | ApiKeyAuthData,
        permissions: string[],
    ): boolean {
        for (const permission of permissions) {
            if (!this.hasPermission(session, permission)) {
                return false;
            }
        }
        return true;
    }
}
