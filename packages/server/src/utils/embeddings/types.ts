
/**
 * All object types that can be embedded. For any versioned object, 
 * the embedding is for the latest version.
 */
export type EmbeddableType =
    | "Api"
    | "Chat"
    | "Issue"
    | "Meeting"
    | "Note"
    | "Organization"
    | "Post"
    | "Project"
    | "Question"
    | "Quiz"
    | "Reminder"
    | "Routine"
    | "RunProject"
    | "RunRoutine"
    | "SmartContract"
    | "Standard"
    | "Tag"
    | "User";

export enum EmbedSortOption {
    EmbedDateCreatedAsc = "EmbedDateCreatedAsc",
    EmbedDateCreatedDesc = "EmbedDateCreatedDesc",
    EmbedDateUpdatedAsc = "EmbedDateUpdatedAsc",
    EmbedDateUpdatedDesc = "EmbedDateUpdatedDesc",
    EmbedTopAsc = "EmbedTopAsc",
    EmbedTopDesc = "EmbedTopDesc",
}

export enum SearchTimePeriod {
    Day = "Day",
    Week = "Week",
    Month = "Month",
    Year = "Year",
    AllTime = "AllTime"
}
