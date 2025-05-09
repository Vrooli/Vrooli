// packages/server/src/services/mcp/mcp_io_shapes.ts
import { TeamConfigObject } from '@local/shared';

/**
 * Attributes for adding a 'Team' resource via MCP.
 * @title MCP Team Add Attributes
 * @description Defines the properties for creating a new team through the MCP.
 *              The 'id' is typically server-generated and not part of the add attributes.
 */
export interface McpTeamAddAttributes {
    /**
     * Specifies if the team is private or public.
     * @default false
     */
    isPrivate: boolean;

    /**
     * Optional team handle. Must be unique if provided.
     * @maxLength 50
     * @pattern ^[a-zA-Z0-9_-]+$
     */
    handle?: string;

    /**
     * Optional team configuration object.
     * This should be the parsed TeamConfigObject, not a stringified version.
     */
    config?: TeamConfigObject; // The generator will create a schema for TeamConfigObject too

    /**
     * Optional list of members to invite initially.
     * User IDs are expected.
     */
    memberInvites?: Array<{
        /** User ID of the member to invite. */
        userId: string;
        /** Optional invitation message. */
        message?: string;
        /** Grant admin privileges to the invited member upon joining. */
        willBeAdmin?: boolean;
    }>;

    // You would not include 'id' here as it's an input for creation.
    // Fields like bannerImage/profileImage (Upload) would need special handling
    // or a simplified representation (e.g., a URL string if MCP handles upload separately).
}

/**
 * Attributes for updating a 'Team' resource via MCP.
 * All properties are optional for an update.
 * @title MCP Team Update Attributes
 */
export interface McpTeamUpdateAttributes {
    /** New privacy status for the team. */
    isPrivate?: boolean;
    /** New handle for the team. */
    handle?: string;
    /** New configuration object for the team. */
    config?: TeamConfigObject;
    // Note: Managing memberInvites directly in an update might be complex.
    // You might have separate MCP ops for 'inviteMemberToTeam', 'removeMemberFromTeam'.
    // Or, if it's a full replacement, define that behavior.
}

/**
 * Filters for finding 'Team' resources via MCP.
 * @title MCP Team Find Filters
 */
export interface McpTeamFindFilters {
    /** Filter by team handle (exact match). */
    handle?: string;
    /** Filter by privacy status. */
    isPrivate?: boolean;
    /** Find teams containing specific user IDs as members. */
    memberUserId?: string;
    // Add other relevant filters for teams
}

// --- Note Variant ---
/**
 * Attributes for adding a 'Note' resource via MCP.
 * @title MCP Note Add Attributes
 */
export interface McpNoteAddAttributes {
    /** The name or title of the note. */
    name: string;
    /** The content of the note. */
    content: string;
    /** Optional tags for categorizing the note. */
    tags?: string[];
}

// ... Define similar interfaces for McpNoteUpdateAttributes, McpNoteFindFilters,
// ... and for all other variants (RoutineApi, RoutineCode, etc.) and their operations.