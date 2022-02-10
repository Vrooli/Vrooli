import { ValueOf } from ".";

//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================

export const AccountStatus: { [x: string]: "Deleted" | "Unlocked" | "SoftLocked" | "HardLocked" } = {
    Deleted: "Deleted",
    Unlocked: "Unlocked",
    SoftLocked: "SoftLocked",
    HardLocked: "HardLocked"
}
export type AccountStatus = ValueOf<typeof AccountStatus>;

export const NodeType: { [x: string]: "Combine" | "Decision" | "End" | "Loop" | "RoutineList" | "Redirect" | "Start" } = {
    Combine: "Combine",
    Decision: "Decision",
    End: "End",
    Loop: "Loop",
    RoutineList: "RoutineList",
    Redirect: "Redirect",
    Start: "Start",
}
export type NodeType = ValueOf<typeof NodeType>;

export const StandardType: { [x: string]: "String" | "Number" | "Boolean" | "Object" | "Array" | "File" | "Url" } = {
    String: "String",
    Number: "Number",
    Boolean: "Boolean",
    Object: "Object",
    Array: "Array",
    File: "File",
    Url: "Url",
}
export type StandardType = ValueOf<typeof StandardType>;

//==============================================================
/* #endregion Database Enums*/
//==============================================================

//==============================================================
/* #region Sort Options */
//==============================================================
// Each sort option should be ordered in the order they should appear in, 
// NOT alphabetically
//==============================================================

export enum OrganizationSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum ProjectSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    VotesDesc = "VotesDesc",
    VotesAsc = "VotesAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    ForksDesc = "ForksDesc",
    ForksAsc = "ForksAsc",
    CommentsDesc = "CommentsDesc",
    CommentsAsc = "CommentsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum ResourceSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum RoutineSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    VotesDesc = "VotesDesc",
    VotesAsc = "VotesAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    ForksDesc = "ForksDesc",
    ForksAsc = "ForksAsc",
    CommentsDesc = "CommentsDesc",
    CommentsAsc = "CommentsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum StandardSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    VotesDesc = "VotesDesc",
    VotesAsc = "VotesAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    CommentsDesc = "CommentsDesc",
    CommentsAsc = "CommentsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum TagSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum UserSortBy {
    AlphabeticalDesc = "AlphabeticalDesc",
    AlphabeticalAsc = "AlphabeticalAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

//==============================================================
/* #endregion Sort Options */
//==============================================================

//==============================================================
/* #region Node Data */
//==============================================================
export type CombineNodeData = {
    from: string[],
    to: string[],
}

export type DecisionNodeDataDecision = {
    id: string,
    title: string,
    description: string | null,
    next: string,
    when: {
        id: string,
        condition: string,
    }
}

export type DecisionNodeData = {
    decisions: DecisionNodeDataDecision[]
}

export type EndNodeData = {
    wasSuccessful: boolean,
}

//TODO: define loop data
export type LoopNodeData = {}

export type RoutineListNodeData = {
    isOrdered: boolean,
    isOptional: boolean,
    routines: RoutineListNodeItemData[]
};

export type RoutineListNodeItemData = {
    id: string,
    isOptional: boolean,
    title: string | null,
    description: string | null,
    routine: {
        id: string,
        title: string,
        description: string | null,
        isAutomatable: boolean,
    }
}

export type RedirectNodeData = {}

export type StartNodeData = {}

export type NodeDataData = CombineNodeData | DecisionNodeData | EndNodeData | LoopNodeData | RoutineListNodeData | RedirectNodeData | StartNodeData;

export type NodeData = {
    id: string,
    type: NodeType,
    title: string | null,
    description: string | null,
    previous: string | null,
    next: string | null,
    data: NodeDataData | null,
}
//==============================================================
/* #endregion Node Data */
//==============================================================

export const CommentFor = {
    Organization: "Organization",
    Project: "Project",
    Routine: "Routine",
    Standard: "Standard",
    User: "User",
}
export type CommentFor = ValueOf<typeof CommentFor>;

export const ReportFor = {
    Comment: "Comment",
    Organization: "Organization",
    Project: "Project",
    Routine: "Routine",
    Standard: "Standard",
    Tag: "Tag",
    User: "User",
}
export type ReportFor = ValueOf<typeof ReportFor>;

export const ResourceFor = {
    Organization: "Organization",
    Project: "Project",
    RoutineContextual: "RoutineContextual",
    RoutineExternal: "RoutineExternal",
    User: "User",
}
export type ResourceFor = ValueOf<typeof ResourceFor>;

export const ResourceUsedFor = {
    Community: "Community",
    Context: "Context",
    Donation: "Donation",
    Learning: "Learning",
    OfficialWebsite: "OfficialWebsite",
    Proposal: "Proposal",
    Related: "Related",
    Social: "Social",
    Tutorial: "Tutorial",
}
export type ResourceUsedFor = ValueOf<typeof ResourceUsedFor>;

export const MemberRole = {
    Admin: "Admin",
    Member: "Member",
    Owner: "Owner",
}
export type MemberRole = ValueOf<typeof MemberRole>;

export const ROLES = {
    Actor: "Actor",
    Guest: "Guest",
}
export type ROLES = ValueOf<typeof ROLES>;

export type OrchestrationData = {
    title: string,
    description?: string,
    nodes: NodeData[],
}

export const VoteFor = {
    Comment: "Comment",
    Project: "Project",
    Routine: "Routine",
    Standard: "Standard",
    Tag: "Tag",
}
export type VoteFor = ValueOf<typeof VoteFor>;

export const StarFor = {
    Comment: "Comment",
    Organization: "Organization",
    Project: "Project",
    Routine: "Routine",
    Standard: "Standard",
    Tag: "Tag",
    User: "User",
}
export type StarFor = ValueOf<typeof StarFor>;