/* c8 ignore start */
import { type BookmarkListSearchInput, type BookmarkSearchInput, type ChatInviteSearchInput, type ChatMessageSearchInput, type ChatParticipantSearchInput, type ChatSearchInput, type CommentSearchInput, type IssueSearchInput, type MeetingInviteSearchInput, type MeetingSearchInput, type MemberInviteSearchInput, type MemberSearchInput, type NotificationSearchInput, type NotificationSubscriptionSearchInput, type PopularSearchInput, type PullRequestSearchInput, type ReactionSearchInput, type ReminderSearchInput, type ReportResponseSearchInput, type ReportSearchInput, type ReputationHistorySearchInput, type ResourceSearchInput, type ResourceVersionSearchInput, type RunIOSearchInput, type RunSearchInput, type ScheduleSearchInput, type StatsResourceSearchInput, type StatsSiteSearchInput, type StatsTeamSearchInput, type StatsUserSearchInput, type TagSearchInput, type TeamSearchInput, type TransferSearchInput, type UserSearchInput, type ViewSearchInput } from "../api/types.js";

export type SearchTypeToSearchInput = {
    Bookmark: BookmarkSearchInput;
    BookmarkList: BookmarkListSearchInput;
    Chat: ChatSearchInput;
    ChatInvite: ChatInviteSearchInput;
    ChatMessage: ChatMessageSearchInput;
    ChatParticipant: ChatParticipantSearchInput;
    Comment: CommentSearchInput;
    Issue: IssueSearchInput;
    MeetingInvite: MeetingInviteSearchInput;
    Meeting: MeetingSearchInput;
    MemberInvite: MemberInviteSearchInput;
    Member: MemberSearchInput;
    Notification: NotificationSearchInput;
    NotificationSubscription: NotificationSubscriptionSearchInput;
    Popular: PopularSearchInput;
    PullRequest: PullRequestSearchInput;
    Reaction: ReactionSearchInput;
    Reminder: ReminderSearchInput;
    ReportResponse: ReportResponseSearchInput;
    Report: ReportSearchInput;
    ReputationHistory: ReputationHistorySearchInput;
    Resource: ResourceSearchInput;
    ResourceVersion: ResourceVersionSearchInput;
    Run: RunSearchInput;
    RunIO: RunIOSearchInput;
    Schedule: ScheduleSearchInput;
    StatsResource: StatsResourceSearchInput;
    StatsSite: StatsSiteSearchInput;
    StatsTeam: StatsTeamSearchInput;
    StatsUser: StatsUserSearchInput;
    Tag: TagSearchInput;
    Team: TeamSearchInput;
    Transfer: TransferSearchInput;
    User: UserSearchInput;
    View: ViewSearchInput;
};
export type SearchType = keyof SearchTypeToSearchInput;

export enum HistoryPageTabOption {
    RunsActive = "RunsActive",
    RunsCompleted = "RunsCompleted",
    Viewed = "Viewed",
    Bookmarked = "Bookmarked",
}

export enum MemberManagePageTabOption {
    Members = "Members",
    Invites = "Invites",
    NonMembers = "NonMembers",
}

export enum MyStuffPageTabOption {
    All = "All",
    Api = "Api",
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Reminder = "Reminder",
    Routine = "Routine",
    Schedule = "Schedule",
    Standard = "Standard",
    Team = "Team",
    User = "User",
}

export enum TeamPageTabOption {
    Resource = "Resource",
    Project = "Project",
    Member = "Member",
}

export enum ParticipantManagePageTabOption {
    ChatParticipant = "ChatParticipant",
    ChatInvite = "ChatInvite",
    Add = "Add",
}

export enum SearchPageTabOption {
    All = "All",
    Api = "Api",
    DataConverter = "DataConverter",
    DataStructure = "DataStructure",
    Note = "Note",
    Project = "Project",
    Prompt = "Prompt",
    RoutineSingleStep = "RoutineSingleStep",
    RoutineMultiStep = "RoutineMultiStep",
    SmartContract = "SmartContract",
    Team = "Team",
    User = "User",
}

export enum SearchVersionPageTabOption {
    ApiVersion = "ApiVersion",
    DataConverterVersion = "DataConverterVersion",
    DataStructureVersion = "DataStructureVersion",
    NoteVersion = "NoteVersion",
    ProjectVersion = "ProjectVersion",
    PromptVersion = "PromptVersion",
    RoutineSingleStepVersion = "RoutineSingleStepVersion",
    RoutineMultiStepVersion = "RoutineMultiStepVersion",
    SmartContractVersion = "SmartContractVersion",
}

export enum UserPageTabOption {
    Details = "Details",
    Project = "Project",
    Team = "Team",
}

export enum CalendarPageTabOption {
    All = "All",
    Meeting = "Meeting",
    Run = "Run",
}

export enum InboxPageTabOption {
    Message = "Message",
    Notification = "Notification",
}

export enum SignUpPageTabOption {
    SignUp = "SignUp",
    MoreInfo = "MoreInfo",
}

export enum ChatPageTabOption {
    Chat = "Chat",
    History = "History",
    Prompt = "Prompt",
    Routine = "Routine",
}
