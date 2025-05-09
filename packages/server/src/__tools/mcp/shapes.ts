// packages/server/src/services/mcp/mcp_io_shapes.ts
import { TeamConfigObject, TeamSortBy, VisibilityType } from '@local/shared';

// Shared shapes to reduce code duplication
interface CommonFindFilters {
    /** Filter by specific IDs. */
    ids?: string[];
    /** Filter by visibility. */
    visibility?: VisibilityType;
    /** Number of items to return. */
    take?: number;
    /** Cursor for pagination. */
    after?: string;
}

interface CommonAttributes {
    /**
     * Specifies if the resource is private or public.
     * @default false
     */
    isPrivate: boolean;
    /**
     * Optional handle. Must be unique if provided.
     * @minLength 3
     * @maxLength 16
     * @pattern ^[a-zA-Z0-9_-]+$
     */
    handle?: string;
    /**
     * Optional list of tags to associate with the resource.
     */
    tagsConnect?: string[];
    /**
     * Optional list of tag IDs to remove from the resource.
     */
    tagsDisconnect?: string[];
}

/**
 * Represents a single member invitation item for MCP operations.
 * @title MCP Member Invite Item
 */
interface MemberInviteItem {
    /** User ID of the member to invite. */
    userConnect: string;
    /** Optional invitation message. */
    message?: string;
    /** Grant admin privileges to the invited member upon joining. */
    willBeAdmin?: boolean;
}

// MCP tool shapes

/**
 * Attributes for adding a 'Team' resource via MCP.
 * @title MCP Team Add Attributes
 * @description Defines the properties for creating a new team through the MCP.
 *              The 'id' is typically server-generated and not part of the add attributes.
 */
export interface TeamAddAttributes extends Pick<CommonAttributes, "handle" | "isPrivate" | "tagsConnect"> {
    /**
     * Optional team configuration object.
     * This should be the parsed TeamConfigObject, not a stringified version.
     */
    config?: TeamConfigObject;
    /**
     * Optional list of members to invite initially.
     * User IDs are expected.
     */
    memberInvitesCreate?: MemberInviteItem[];
}

/**
 * Attributes for updating a 'Team' resource via MCP.
 * All properties are optional for an update.
 * @title MCP Team Update Attributes
 */
export interface TeamUpdateAttributes extends Pick<CommonAttributes, "handle" | "isPrivate" | "tagsConnect" | "tagsDisconnect"> {
    /** New configuration object for the team. */
    config?: TeamConfigObject;
    /**
     * Optional list of new members to invite.
     * User IDs are expected.
     */
    memberInvitesCreate?: MemberInviteItem[];
    /**
     * Optional list of member invite IDs to delete.
     */
    memberInvitesDelete?: string[];
    /**
     * Optional list of member IDs to remove from the team.
     */
    membersDelete?: string[];
}

/**
 * Filters for finding 'Team' resources via MCP.
 * @title MCP Team Find Filters
 */
export interface TeamFindFilters extends CommonFindFilters {
    /** Filter by team handle (exact match). */
    handle?: string;
    /** Find teams containing specific user IDs as members. */
    memberUserIds?: string[];
    /** General search string for teams. */
    searchString?: string;
    /** Filter teams by associated tags. */
    tags?: string[];
    /** Filter teams by whether they are open to new members. */
    isOpenToNewMembers?: boolean;
    /** Sort order for the results. */
    sortBy?: TeamSortBy;
}

/**
 * Attributes for adding a 'Note' resource via MCP.
 * @title MCP Note Add Attributes
 */
export interface NoteAddAttributes extends Pick<CommonAttributes, "tagsConnect"> {
    /**
     * The name or title of the note.
     */
    name: string;
    /**
     * The content of the note.
     */
    content: string;
}
