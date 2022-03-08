import { ValueOf } from ".";

//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================

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

export enum CommentSortBy {
    VotesAsc = "VotesAsc",
    VotesDesc = "VotesDesc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum OrganizationSortBy {
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum ProjectSortBy {
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

export enum ReportSortBy {
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum ResourceSortBy {
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum RoutineSortBy {
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
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum UserSortBy {
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
    Routine: "Routine",
    User: "User",
}
export type ResourceFor = ValueOf<typeof ResourceFor>;

export const ResourceUsedFor = {
    Community: "Community",
    Context: "Context",
    Developer: "Developer",
    Donation: "Donation",
    ExternalService: "ExternalService",
    Install: "Install",
    Learning: "Learning",
    Notes : "Notes",
    OfficialWebsite: "OfficialWebsite",
    Proposal: "Proposal",
    Related: "Related",
    Scheduling: "Scheduling",
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