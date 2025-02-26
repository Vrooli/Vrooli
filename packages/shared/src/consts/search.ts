import { ApiSearchInput, ApiVersionSearchInput, BookmarkListSearchInput, BookmarkSearchInput, ChatInviteSearchInput, ChatMessageSearchInput, ChatParticipantSearchInput, ChatSearchInput, CodeSearchInput, CodeVersionSearchInput, CommentSearchInput, FocusModeSearchInput, IssueSearchInput, LabelSearchInput, MeetingInviteSearchInput, MeetingSearchInput, MemberInviteSearchInput, MemberSearchInput, NoteSearchInput, NoteVersionSearchInput, NotificationSearchInput, NotificationSubscriptionSearchInput, PopularSearchInput, PostSearchInput, ProjectOrRoutineSearchInput, ProjectOrTeamSearchInput, ProjectSearchInput, ProjectVersionSearchInput, PullRequestSearchInput, QuestionAnswerSearchInput, QuestionSearchInput, QuizAttemptSearchInput, QuizQuestionResponseSearchInput, QuizSearchInput, ReactionSearchInput, ReminderSearchInput, ReportResponseSearchInput, ReportSearchInput, ReputationHistorySearchInput, ResourceListSearchInput, ResourceSearchInput, RoleSearchInput, RoutineSearchInput, RoutineVersionSearchInput, RunProjectOrRunRoutineSearchInput, RunProjectSearchInput, RunRoutineIOSearchInput, RunRoutineSearchInput, ScheduleSearchInput, StandardSearchInput, StandardVersionSearchInput, StatsApiSearchInput, StatsCodeSearchInput, StatsProjectSearchInput, StatsQuizSearchInput, StatsRoutineSearchInput, StatsSiteSearchInput, StatsStandardSearchInput, StatsTeamSearchInput, StatsUserSearchInput, TagSearchInput, TeamSearchInput, TransferSearchInput, UserSearchInput, ViewSearchInput } from "../api/types.js";

export type SearchTypeToSearchInput = {
    Api: ApiSearchInput;
    ApiVersion: ApiVersionSearchInput;
    Bookmark: BookmarkSearchInput;
    BookmarkList: BookmarkListSearchInput;
    Chat: ChatSearchInput;
    ChatInvite: ChatInviteSearchInput;
    ChatMessage: ChatMessageSearchInput;
    ChatParticipant: ChatParticipantSearchInput;
    Code: CodeSearchInput;
    CodeVersion: CodeVersionSearchInput;
    Comment: CommentSearchInput;
    FocusMode: FocusModeSearchInput;
    Issue: IssueSearchInput;
    Label: LabelSearchInput;
    MeetingInvite: MeetingInviteSearchInput;
    Meeting: MeetingSearchInput;
    MemberInvite: MemberInviteSearchInput;
    Member: MemberSearchInput;
    Note: NoteSearchInput;
    NoteVersion: NoteVersionSearchInput;
    Notification: NotificationSearchInput;
    NotificationSubscription: NotificationSubscriptionSearchInput;
    Popular: PopularSearchInput;
    Post: PostSearchInput;
    ProjectOrRoutine: ProjectOrRoutineSearchInput;
    ProjectOrTeam: ProjectOrTeamSearchInput;
    Project: ProjectSearchInput;
    ProjectVersion: ProjectVersionSearchInput;
    // ProjectVersionDirectory: ProjectVersionDirectorySearchInput;
    PullRequest: PullRequestSearchInput;
    Question: QuestionSearchInput;
    QuestionAnswer: QuestionAnswerSearchInput;
    Quiz: QuizSearchInput;
    QuizAttempt: QuizAttemptSearchInput;
    QuizQuestionResponse: QuizQuestionResponseSearchInput;
    Reaction: ReactionSearchInput;
    Reminder: ReminderSearchInput;
    ReportResponse: ReportResponseSearchInput;
    Report: ReportSearchInput;
    ReputationHistory: ReputationHistorySearchInput;
    Resource: ResourceSearchInput;
    ResourceList: ResourceListSearchInput;
    Role: RoleSearchInput;
    Routine: RoutineSearchInput;
    RoutineVersion: RoutineVersionSearchInput;
    RunProject: RunProjectSearchInput;
    RunProjectOrRunRoutine: RunProjectOrRunRoutineSearchInput;
    RunRoutine: RunRoutineSearchInput;
    RunRoutineIO: RunRoutineIOSearchInput;
    Schedule: ScheduleSearchInput;
    Standard: StandardSearchInput;
    StandardVersion: StandardVersionSearchInput;
    StatsApi: StatsApiSearchInput;
    StatsCode: StatsCodeSearchInput;
    StatsProject: StatsProjectSearchInput;
    StatsQuiz: StatsQuizSearchInput;
    StatsRoutine: StatsRoutineSearchInput;
    StatsSite: StatsSiteSearchInput;
    StatsStandard: StatsStandardSearchInput;
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
    Question = "Question",
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
    Question = "Question",
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
    FocusMode = "FocusMode",
    Meeting = "Meeting",
    RunProject = "RunProject",
    RunRoutine = "RunRoutine",
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
