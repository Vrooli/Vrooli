import { ValueOf } from '.';

//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================

export const AccountStatus: { [x: string]: 'Deleted' | 'Unlocked' | 'SoftLocked' | 'HardLocked' } = {
    Deleted: 'Deleted',
    Unlocked: 'Unlocked',
    SoftLocked: 'SoftLocked',
    HardLocked: 'HardLocked'
}
export type AccountStatus = ValueOf<typeof AccountStatus>;

export const NodeType: { [x: string]: 'Combine' | 'Decision' | 'End' | 'Loop' | 'RoutineList' | 'Redirect' | 'Start' } = {
    Combine: 'Combine',
    Decision: 'Decision',
    End: 'End',
    Loop: 'Loop',
    RoutineList: 'RoutineList',
    Redirect: 'Redirect',
    Start: 'Start',
}
export type NodeType = ValueOf<typeof NodeType>;

export const StandardType: { [x: string]: 'String' | 'Number' | 'Boolean' | 'Object' | 'Array' | 'File' | 'Url' } = {
    String: 'String',
    Number: 'Number',
    Boolean: 'Boolean',
    Object: 'Object',
    Array: 'Array',
    File: 'File',
    Url: 'Url',
}
export type StandardType = ValueOf<typeof StandardType>;

export const TagVoteType = {
    Organization: 'Organization',
    Project: 'Project',
    Resource: 'Resource',
    Routine: 'Routine',
    Standard: 'Standard',
}
export type TagVoteType = ValueOf<typeof TagVoteType>;

//==============================================================
/* #endregion Database Enums*/
//==============================================================

//==============================================================
/* #region Sort Options */
//==============================================================

export const OrganizationSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
    VotesAsc: 'VotesAsc',
    VotesDesc: 'VotesDesc',
}
export type OrganizationSortBy = ValueOf<typeof OrganizationSortBy>;

export const ProjectSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    ForksAsc: 'ForksAsc',
    ForksDesc: 'ForksDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
    VotesAsc: 'VotesAsc',
    VotesDesc: 'VotesDesc',
}
export type ProjectSortBy = ValueOf<typeof ProjectSortBy>;

export const ResourceSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
}
export type ResourceSortBy = ValueOf<typeof ResourceSortBy>;

export const RoutineSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    ForksAsc: 'ForksAsc',
    ForksDesc: 'ForksDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
    VotesAsc: 'VotesAsc',
    VotesDesc: 'VotesDesc',
}
export type RoutineSortBy = ValueOf<typeof RoutineSortBy>;

export const StandardSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
    VotesAsc: 'VotesAsc',
    VotesDesc: 'VotesDesc',
}
export type StandardSortBy = ValueOf<typeof StandardSortBy>;

export const TagSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
}
export type TagSortBy = ValueOf<typeof TagSortBy>;

export const UserSortBy = {
    AlphabeticalAsc: 'AlphabeticalAsc',
    AlphabeticalDesc: 'AlphabeticalDesc',
    CommentsAsc: 'CommentsAsc',
    CommentsDesc: 'CommentsDesc',
    DateCreatedAsc: 'DateCreatedAsc',
    DateCreatedDesc: 'DateCreatedDesc',
    DateUpdatedAsc: 'DateUpdatedAsc',
    DateUpdatedDesc: 'DateUpdatedDesc',
    StarsAsc: 'StarsAsc',
    StarsDesc: 'StarsDesc',
}
export type UserSortBy = ValueOf<typeof UserSortBy>;

//==============================================================
/* #endregion Sort Options */
//==============================================================

//==============================================================
/* #region Node Data */
//==============================================================
export type COMBINE_NODE_DATA = null // TODO: define combine data

export type DECISION_NODE_DATA = {
    title: string,
    description: string | null,
    decisions: Array<{
        id: string,
        title: string,
        next: string,
        when: Array<any> //TODO: define
    }>
}

export type END_NODE_DATA = {
    title: string,
    description: string | null,
    wasSuccessful: boolean,
}

export type LOOP_NODE_DATA = null; //TODO: define loop data

export type ROUTINE_LIST_NODE_DATA = {
    title: string,
    description: string | null,
    isOrdered: boolean,
    isOptional: boolean,
    routines: Array<ROUTINE_LIST_NODE_ITEM_DATA>
};

export type ROUTINE_LIST_NODE_ITEM_DATA = {
    id: string,
    title: string,
    description: string | null,
    isOptional: boolean,
}

export type REDIRECT_NODE_DATA = null; // Supposed to be null

export type START_NODE_DATA = null; // Supposed to be null

export type NODE_DATA = {
    id: string,
    type: NodeType,
    previous: string | null,
    next: string | null,
    data: COMBINE_NODE_DATA | DECISION_NODE_DATA | END_NODE_DATA | LOOP_NODE_DATA | ROUTINE_LIST_NODE_DATA | REDIRECT_NODE_DATA | START_NODE_DATA,
}
//==============================================================
/* #endregion Node Data */
//==============================================================

export const MetricTimeFrame = {
    Daily: 'Daily',
    Weekly: 'Weekly',
    Monthly: 'Monthly',
    Yearly: 'Yearly',
}
export type MetricTimeFrame = ValueOf<typeof MetricTimeFrame>;

export const ResourceFor = {
    Actor: 'Actor',
    Organization: 'Organization',
    Project: 'Project',
    RoutineContextual: 'RoutineContextual',
    RoutineExternal: 'RoutineExternal',
    RoutineDonation: 'RoutineDonation',
}
export type ResourceFor = ValueOf<typeof ResourceFor>;

export const ROLES = {
    Actor: 'Actor',
    Guest: 'Guest',
}
export type ROLES = ValueOf<typeof ROLES>;

export type ORCHESTRATION_DATA = {
    title: string,
    description?: string,
    nodes: Array<NODE_DATA>,
}