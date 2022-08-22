
//==============================================================
/* #region Database Enums  */
//==============================================================
// These enums need additional typing information to be compatible with Prisma.
// For more information, see: https://github.com/prisma/prisma/discussions/9215
//==============================================================

/**
 * The different types of input components supported for forms and standards. 
 * If more specific types are needed (e.g. URLs, email addresses, etc.), these 
 * are set using Yup validation checks.
 */
 export enum InputType {
    Checkbox = 'Checkbox',
    Dropzone = 'Dropzone',
    JSON = 'JSON',
    LanguageInput = 'LanguageInput',
    Markdown = 'Markdown',
    Radio = 'Radio',
    Selector = 'Selector',
    Slider = 'Slider',
    Switch = 'Switch',
    TagSelector = 'TagSelector',
    TextField = 'TextField',
    QuantityBox = 'QuantityBox',
}

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

export enum CopyType {
    Node = "Node",
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
}

export enum DeleteOneType {
    Comment = "Comment",
    Email = "Email",
    Node = "Node",
    Organization = "Organization",
    Project = "Project",
    Report = "Report",
    Routine = "Routine",
    Run = "Run",
    Standard = "Standard",
    Wallet = "Wallet",
} 

export enum ForkType {
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
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

export enum ProjectOrOrganizationSortBy {
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
}

export enum ProjectOrRoutineSortBy {
    VotesDesc = "VotesDesc",
    VotesAsc = "VotesAsc",
    StarsDesc = "StarsDesc",
    StarsAsc = "StarsAsc",
    ForksDesc = "ForksDesc",
    ForksAsc = "ForksAsc",
    CommentsDesc = "CommentsDesc",
    CommentsAsc = "CommentsAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCompletedAsc = "DateCompletedAsc",
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
    IndexAsc = "IndexAsc",
    IndexDesc = "IndexDesc",
}

export enum ResourceListSortBy {
    DateCreatedDesc = "DateCreatedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    IndexAsc = "IndexAsc",
    IndexDesc = "IndexDesc",
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

export enum RunSortBy {
    DateStartedAsc = "DateStartedAsc",
    DateStartedDesc = "DateStartedDesc",
    DateCompletedAsc = "DateCompletedAsc",
    DateCompletedDesc = "DateCompletedDesc",
    DateCreatedAsc = "DateCreatedAsc",
    DateCreatedDesc = "DateCreatedDesc",
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
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

export enum StarSortBy {
    DateUpdatedAsc = "DateUpdatedAsc",
    DateUpdatedDesc = "DateUpdatedDesc",
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

export enum ViewSortBy {
    LastViewedAsc = "LastViewedAsc",
    LastViewedDesc = "LastViewedDesc",
}

//==============================================================
/* #endregion Sort Options */
//==============================================================

export enum ReportFor {
    Comment = "Comment",
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    Tag = "Tag",
    User = "User",
}

export enum ResourceFor {
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    User = "User",
}

export enum ResourceUsedFor {
    Community = "Community",
    Context = "Context",
    Developer = "Developer",
    Donation = "Donation",
    ExternalService = "ExternalService",
    Feed = "Feed",
    Install = "Install",
    Learning = "Learning",
    Notes = "Notes",
    OfficialWebsite = "OfficialWebsite",
    Proposal = "Proposal",
    Related = "Related",
    Researching = "Researching",
    Scheduling = "Scheduling",
    Social = "Social",
    Tutorial = "Tutorial",
}

export enum ResourceListUsedFor {
    Custom = "Custom",
    Display = "Display",
    Learn = "Learn",
    Research = "Research",
    Develop = "Develop",
}

export enum RunStepStatus {
    InProgress = "InProgress",
    Completed = "Completed",
    Skipped = "Skipped",
}

export enum VoteFor {
    Comment = "Comment",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    Tag = "Tag",
}

export enum StarFor {
    Comment = "Comment",
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    Tag = "Tag",
    User = "User",
}

export enum ViewFor {
    Organization = "Organization",
    Project = "Project",
    Routine = "Routine",
    Standard = "Standard",
    User = "User",
}