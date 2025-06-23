/**
 * Authoritative Prisma Type Reference
 * Generated to ensure correct type usage in fixtures
 */

import { type Prisma, type user, type team, type bookmark, type credit_account, type api_key } from "@prisma/client";

// =============================================================================
// TYPE VERIFICATION
// This file serves as verification that types exist and documentation for correct usage
// =============================================================================

// Core Models - CORRECT PATTERN: Direct exports (not in Prisma namespace)
export type UserModel = user;
export type TeamModel = team;
export type BookmarkModel = bookmark;
export type CreditAccountModel = credit_account;
export type ApiKeyModel = api_key;

// Create Input Types - CORRECT PATTERN: In Prisma namespace, snake_case
export type UserCreateInput = Prisma.userCreateInput;
export type TeamCreateInput = Prisma.teamCreateInput;
export type BookmarkCreateInput = Prisma.bookmarkCreateInput;
export type CreditAccountCreateInput = Prisma.credit_accountCreateInput;
export type ApiKeyCreateInput = Prisma.api_keyCreateInput;

// Update Input Types
export type UserUpdateInput = Prisma.userUpdateInput;
export type TeamUpdateInput = Prisma.teamUpdateInput;
export type BookmarkUpdateInput = Prisma.bookmarkUpdateInput;
export type CreditAccountUpdateInput = Prisma.credit_accountUpdateInput;
export type ApiKeyUpdateInput = Prisma.api_keyUpdateInput;

// Include Types  
export type UserInclude = Prisma.userInclude;
export type TeamInclude = Prisma.teamInclude;
export type BookmarkInclude = Prisma.bookmarkInclude;
export type CreditAccountInclude = Prisma.credit_accountInclude;
export type ApiKeyInclude = Prisma.api_keyInclude;

// =============================================================================
// NAMING CONVENTION PATTERNS
// =============================================================================

/**
 * CONFIRMED PATTERNS (CORRECTED):
 * 
 * Model Types: Direct exports, snake_case
 * - user (not Prisma.user)
 * - credit_account (not Prisma.credit_account)
 * - api_key (not Prisma.api_key)
 * 
 * Input Types: In Prisma namespace, camelCase + suffix
 * - Prisma.userCreateInput
 * - Prisma.credit_accountCreateInput  
 * - Prisma.api_keyCreateInput
 * 
 * Include Types: In Prisma namespace, camelCase + Include
 * - Prisma.userInclude
 * - Prisma.credit_accountInclude
 * - Prisma.api_keyInclude
 */

// =============================================================================
// FACTORY TYPE MAPPING HELPER
// =============================================================================

export const PRISMA_TYPE_MAP = {
    // Core Business Objects
    user: {
        model: "user" as const,
        createInput: "userCreateInput" as const,
        updateInput: "userUpdateInput" as const,
        include: "userInclude" as const,
    },
    team: {
        model: "team" as const,
        createInput: "teamCreateInput" as const,
        updateInput: "teamUpdateInput" as const,
        include: "teamInclude" as const,
    },
    
    // Content Objects
    bookmark: {
        model: "bookmark" as const,
        createInput: "bookmarkCreateInput" as const,
        updateInput: "bookmarkUpdateInput" as const,
        include: "bookmarkInclude" as const,
    },
    bookmark_list: {
        model: "bookmark_list" as const,
        createInput: "bookmark_listCreateInput" as const,
        updateInput: "bookmark_listUpdateInput" as const,
        include: "bookmark_listInclude" as const,
    },
    
    // Authentication & User Management
    api_key: {
        model: "api_key" as const,
        createInput: "api_keyCreateInput" as const,
        updateInput: "api_keyUpdateInput" as const,
        include: "api_keyInclude" as const,
    },
    user_auth: {
        model: "user_auth" as const,
        createInput: "user_authCreateInput" as const,
        updateInput: "user_authUpdateInput" as const,
        include: "user_authInclude" as const,
    },
    
    // Billing & Credits
    credit_account: {
        model: "credit_account" as const,
        createInput: "credit_accountCreateInput" as const,
        updateInput: "credit_accountUpdateInput" as const,
        include: "credit_accountInclude" as const,
    },
    credit_ledger_entry: {
        model: "credit_ledger_entry" as const,
        createInput: "credit_ledger_entryCreateInput" as const,
        updateInput: "credit_ledger_entryUpdateInput" as const,
        include: "credit_ledger_entryInclude" as const,
    },
    
    // Team Management
    member: {
        model: "member" as const,
        createInput: "memberCreateInput" as const,
        updateInput: "memberUpdateInput" as const,
        include: "memberInclude" as const,
    },
    member_invite: {
        model: "member_invite" as const,
        createInput: "member_inviteCreateInput" as const,
        updateInput: "member_inviteUpdateInput" as const,
        include: "member_inviteInclude" as const,
    },
    
    // Add more as needed...
} as const;

// =============================================================================
// TYPE HELPER UTILITIES
// =============================================================================

/**
 * Helper type to get the correct Prisma types for a model
 */
export type PrismaTypesFor<T extends keyof typeof PRISMA_TYPE_MAP> = {
    Model: T extends "user" ? user :
           T extends "team" ? team :
           T extends "bookmark" ? bookmark :
           T extends "credit_account" ? credit_account :
           T extends "api_key" ? api_key :
           never;
    
    CreateInput: T extends "user" ? Prisma.userCreateInput :
                 T extends "team" ? Prisma.teamCreateInput :
                 T extends "bookmark" ? Prisma.bookmarkCreateInput :
                 T extends "credit_account" ? Prisma.credit_accountCreateInput :
                 T extends "api_key" ? Prisma.api_keyCreateInput :
                 never;
    
    UpdateInput: T extends "user" ? Prisma.userUpdateInput :
                 T extends "team" ? Prisma.teamUpdateInput :
                 T extends "bookmark" ? Prisma.bookmarkUpdateInput :
                 T extends "credit_account" ? Prisma.credit_accountUpdateInput :
                 T extends "api_key" ? Prisma.api_keyUpdateInput :
                 never;
    
    Include: T extends "user" ? Prisma.userInclude :
             T extends "team" ? Prisma.teamInclude :
             T extends "bookmark" ? Prisma.bookmarkInclude :
             T extends "credit_account" ? Prisma.credit_accountInclude :
             T extends "api_key" ? Prisma.api_keyInclude :
             never;
};

// Export individual types for easy access
export { Prisma };
