//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================
export const AccountStatus = {
    Deleted: 'Deleted',
    Unlocked: 'Unlocked',
    SoftLocked: 'SoftLocked',
    HardLocked: 'HardLocked'
};
export const NodeType = {
    Combine: 'Combine',
    Decision: 'Decision',
    End: 'End',
    Loop: 'Loop',
    RoutineList: 'RoutineList',
    Redirect: 'Redirect',
    Start: 'Start',
};
export const StandardType = {
    String: 'String',
    Number: 'Number',
    Boolean: 'Boolean',
    Object: 'Object',
    Array: 'Array',
    File: 'File',
    Url: 'Url',
};
export const TagVoteType = {
    Organization: 'Organization',
    Project: 'Project',
    Resource: 'Resource',
    Routine: 'Routine',
    Standard: 'Standard',
};
//==============================================================
/* #endregion Database Enums*/
//==============================================================
//==============================================================
/* #region Sort Options */
//==============================================================
export var OrganizationSortBy;
(function (OrganizationSortBy) {
    OrganizationSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    OrganizationSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    OrganizationSortBy["CommentsAsc"] = "CommentsAsc";
    OrganizationSortBy["CommentsDesc"] = "CommentsDesc";
    OrganizationSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    OrganizationSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    OrganizationSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    OrganizationSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    OrganizationSortBy["StarsAsc"] = "StarsAsc";
    OrganizationSortBy["StarsDesc"] = "StarsDesc";
    OrganizationSortBy["VotesAsc"] = "VotesAsc";
    OrganizationSortBy["VotesDesc"] = "VotesDesc";
})(OrganizationSortBy || (OrganizationSortBy = {}));
export var ProjectSortBy;
(function (ProjectSortBy) {
    ProjectSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    ProjectSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    ProjectSortBy["CommentsAsc"] = "CommentsAsc";
    ProjectSortBy["CommentsDesc"] = "CommentsDesc";
    ProjectSortBy["ForksAsc"] = "ForksAsc";
    ProjectSortBy["ForksDesc"] = "ForksDesc";
    ProjectSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    ProjectSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    ProjectSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    ProjectSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    ProjectSortBy["StarsAsc"] = "StarsAsc";
    ProjectSortBy["StarsDesc"] = "StarsDesc";
    ProjectSortBy["VotesAsc"] = "VotesAsc";
    ProjectSortBy["VotesDesc"] = "VotesDesc";
})(ProjectSortBy || (ProjectSortBy = {}));
export var ResourceSortBy;
(function (ResourceSortBy) {
    ResourceSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    ResourceSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    ResourceSortBy["CommentsAsc"] = "CommentsAsc";
    ResourceSortBy["CommentsDesc"] = "CommentsDesc";
    ResourceSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    ResourceSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    ResourceSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    ResourceSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    ResourceSortBy["StarsAsc"] = "StarsAsc";
    ResourceSortBy["StarsDesc"] = "StarsDesc";
})(ResourceSortBy || (ResourceSortBy = {}));
export var RoutineSortBy;
(function (RoutineSortBy) {
    RoutineSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    RoutineSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    RoutineSortBy["CommentsAsc"] = "CommentsAsc";
    RoutineSortBy["CommentsDesc"] = "CommentsDesc";
    RoutineSortBy["ForksAsc"] = "ForksAsc";
    RoutineSortBy["ForksDesc"] = "ForksDesc";
    RoutineSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    RoutineSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    RoutineSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    RoutineSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    RoutineSortBy["StarsAsc"] = "StarsAsc";
    RoutineSortBy["StarsDesc"] = "StarsDesc";
    RoutineSortBy["VotesAsc"] = "VotesAsc";
    RoutineSortBy["VotesDesc"] = "VotesDesc";
})(RoutineSortBy || (RoutineSortBy = {}));
export var StandardSortBy;
(function (StandardSortBy) {
    StandardSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    StandardSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    StandardSortBy["CommentsAsc"] = "CommentsAsc";
    StandardSortBy["CommentsDesc"] = "CommentsDesc";
    StandardSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    StandardSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    StandardSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    StandardSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    StandardSortBy["StarsAsc"] = "StarsAsc";
    StandardSortBy["StarsDesc"] = "StarsDesc";
    StandardSortBy["VotesAsc"] = "VotesAsc";
    StandardSortBy["VotesDesc"] = "VotesDesc";
})(StandardSortBy || (StandardSortBy = {}));
export var TagSortBy;
(function (TagSortBy) {
    TagSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    TagSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    TagSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    TagSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    TagSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    TagSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    TagSortBy["StarsAsc"] = "StarsAsc";
    TagSortBy["StarsDesc"] = "StarsDesc";
})(TagSortBy || (TagSortBy = {}));
export var UserSortBy;
(function (UserSortBy) {
    UserSortBy["AlphabeticalAsc"] = "AlphabeticalAsc";
    UserSortBy["AlphabeticalDesc"] = "AlphabeticalDesc";
    UserSortBy["CommentsAsc"] = "CommentsAsc";
    UserSortBy["CommentsDesc"] = "CommentsDesc";
    UserSortBy["DateCreatedAsc"] = "DateCreatedAsc";
    UserSortBy["DateCreatedDesc"] = "DateCreatedDesc";
    UserSortBy["DateUpdatedAsc"] = "DateUpdatedAsc";
    UserSortBy["DateUpdatedDesc"] = "DateUpdatedDesc";
    UserSortBy["StarsAsc"] = "StarsAsc";
    UserSortBy["StarsDesc"] = "StarsDesc";
})(UserSortBy || (UserSortBy = {}));
//==============================================================
/* #endregion Node Data */
//==============================================================
export const TimeFrame = {
    Day: 'Day',
    Week: 'Week',
    Month: 'Month',
    Year: 'Year',
};
export const ResourceFor = {
    Actor: 'Actor',
    Organization: 'Organization',
    Project: 'Project',
    RoutineContextual: 'RoutineContextual',
    RoutineExternal: 'RoutineExternal',
    RoutineDonation: 'RoutineDonation',
};
export const ROLES = {
    Actor: 'Actor',
    Guest: 'Guest',
};
