import { SearchParams } from "./schemas/base";

export enum SearchType {
    Api = "Api",
    ApiVersion = "ApiVersion",
    Bookmark = "Bookmark",
    BookmarkList = "BookmarkList",
    Chat = "Chat",
    ChatInvite = "ChatInvite",
    ChatMessage = "ChatMessage",
    ChatParticipant = "ChatParticipant",
    Comment = "Comment",
    FocusMode = "FocusMode",
    Issue = "Issue",
    Label = "Label",
    MeetingInvite = "MeetingInvite",
    Meeting = "Meeting",
    MemberInvite = "MemberInvite",
    Member = "Member",
    Note = "Note",
    NoteVersion = "NoteVersion",
    Notification = "Notification",
    NotificationSubscription = "NotificationSubscription",
    Organization = "Organization",
    Popular = "Popular",
    Post = "Post",
    ProjectOrOrganization = "ProjectOrOrganization",
    ProjectOrRoutine = "ProjectOrRoutine",
    Project = "Project",
    ProjectVersion = "ProjectVersion",
    // ProjectVersionDirectory = 'ProjectVersionDirectory',
    PullRequest = "PullRequest",
    Question = "Question",
    QuestionAnswer = "QuestionAnswer",
    Quiz = "Quiz",
    QuizAttempt = "QuizAttempt",
    QuizQuestion = "QuizQuestion",
    QuizQuestionResponse = "QuizQuestionResponse",
    Reaction = "Reaction",
    Reminder = "Reminder",
    ReportResponse = "ReportResponse",
    Report = "Report",
    ReputationHistory = "ReputationHistory",
    ResourceList = "ResourceList",
    Resource = "Resource",
    Role = "Role",
    Routine = "Routine",
    RoutineVersion = "RoutineVersion",
    RunProject = "RunProject",
    RunProjectOrRunRoutine = "RunProjectOrRunRoutine",
    RunRoutine = "RunRoutine",
    RunRoutineInput = "RunRoutineInput",
    Schedule = "Schedule",
    SmartContract = "SmartContract",
    SmartContractVersion = "SmartContractVersion",
    Standard = "Standard",
    StandardVersion = "StandardVersion",
    StatsApi = "StatsApi",
    StatsOrganization = "StatsOrganization",
    StatsProject = "StatsProject",
    StatsQuiz = "StatsQuiz",
    StatsRoutine = "StatsRoutine",
    StatsSite = "StatsSite",
    StatsSmartContract = "StatsSmartContract",
    StatsStandard = "StatsStandard",
    StatsUser = "StatsUser",
    Tag = "Tag",
    Transfer = "Transfer",
    User = "User",
    View = "View",
}

export enum HistoryPageTabOption {
    RunsActive = "RunsActive",
    RunsCompleted = "RunsCompleted",
    Viewed = "Viewed",
    Bookmarked = "Bookmarked",
}

export enum SearchPageTabOption {
    Apis = "Apis",
    Notes = "Notes",
    Organizations = "Organizations",
    Projects = "Projects",
    Questions = "Questions",
    Routines = "Routines",
    SmartContracts = "SmartContracts",
    Standards = "Standards",
    Users = "Users",
}

export enum CalendarPageTabOption {
    FocusModes = "FocusModes",
    Meetings = "Meetings",
    RunProjects = "RunProjects",
    RunRoutines = "RunRoutines",
}


/**
 * Maps search types to values needed to query and display results
 */
export const searchTypeToParams: { [key in SearchType]: () => Promise<SearchParams> } = {
    Api: async () => (await import("./schemas/api")).apiSearchParams(),
    ApiVersion: async () => (await import("./schemas/apiVersion")).apiVersionSearchParams(),
    Bookmark: async () => (await import("./schemas/bookmark")).bookmarkSearchParams(),
    BookmarkList: async () => (await import("./schemas/bookmarkList")).bookmarkListSearchParams(),
    Chat: async () => (await import("./schemas/chat")).chatSearchParams(),
    ChatInvite: async () => (await import("./schemas/chatInvite")).chatInviteSearchParams(),
    ChatMessage: async () => (await import("./schemas/chatMessage")).chatMessageSearchParams(),
    ChatParticipant: async () => (await import("./schemas/chatParticipant")).chatParticipantSearchParams(),
    Comment: async () => (await import("./schemas/comment")).commentSearchParams(),
    FocusMode: async () => (await import("./schemas/focusMode")).focusModeSearchParams(),
    Issue: async () => (await import("./schemas/issue")).issueSearchParams(),
    Label: async () => (await import("./schemas/label")).labelSearchParams(),
    Meeting: async () => (await import("./schemas/meeting")).meetingSearchParams(),
    MeetingInvite: async () => (await import("./schemas/meetingInvite")).meetingInviteSearchParams(),
    Member: async () => (await import("./schemas/member")).memberSearchParams(),
    MemberInvite: async () => (await import("./schemas/memberInvite")).memberInviteSearchParams(),
    Note: async () => (await import("./schemas/note")).noteSearchParams(),
    NoteVersion: async () => (await import("./schemas/noteVersion")).noteVersionSearchParams(),
    Notification: async () => (await import("./schemas/notification")).notificationSearchParams(),
    NotificationSubscription: async () => (await import("./schemas/notificationSubscription")).notificationSubscriptionSearchParams(),
    Organization: async () => (await import("./schemas/organization")).organizationSearchParams(),
    Popular: async () => (await import("./schemas/popular")).popularSearchParams(),
    Post: async () => (await import("./schemas/post")).postSearchParams(),
    Project: async () => (await import("./schemas/project")).projectSearchParams(),
    ProjectOrOrganization: async () => (await import("./schemas/projectOrOrganization")).projectOrOrganizationSearchParams(),
    ProjectOrRoutine: async () => (await import("./schemas/projectOrRoutine")).projectOrRoutineSearchParams(),
    ProjectVersion: async () => (await import("./schemas/projectVersion")).projectVersionSearchParams(),
    // ProjectVersionDirectory: async () => (await import('./schemas/projectVersionDirectory')).projectVersionDirectorySearchParams(),
    PullRequest: async () => (await import("./schemas/pullRequest")).pullRequestSearchParams(),
    Question: async () => (await import("./schemas/question")).questionSearchParams(),
    QuestionAnswer: async () => (await import("./schemas/questionAnswer")).questionAnswerSearchParams(),
    Quiz: async () => (await import("./schemas/quiz")).quizSearchParams(),
    QuizAttempt: async () => (await import("./schemas/quizAttempt")).quizAttemptSearchParams(),
    QuizQuestion: async () => (await import("./schemas/quizQuestion")).quizQuestionSearchParams(),
    QuizQuestionResponse: async () => (await import("./schemas/quizQuestionResponse")).quizQuestionResponseSearchParams(),
    Reaction: async () => (await import("./schemas/reaction")).reactionSearchParams(),
    Reminder: async () => (await import("./schemas/reminder")).reminderSearchParams(),
    Report: async () => (await import("./schemas/report")).reportSearchParams(),
    ReportResponse: async () => (await import("./schemas/reportResponse")).reportResponseSearchParams(),
    ReputationHistory: async () => (await import("./schemas/reputationHistory")).reputationHistorySearchParams(),
    Resource: async () => (await import("./schemas/resource")).resourceSearchParams(),
    ResourceList: async () => (await import("./schemas/resourceList")).resourceListSearchParams(),
    Role: async () => (await import("./schemas/role")).roleSearchParams(),
    Routine: async () => (await import("./schemas/routine")).routineSearchParams(),
    RoutineVersion: async () => (await import("./schemas/routineVersion")).routineVersionSearchParams(),
    RunProject: async () => (await import("./schemas/runProject")).runProjectSearchParams(),
    RunProjectOrRunRoutine: async () => (await import("./schemas/runProjectOrRunRoutine")).runProjectOrRunRoutineSearchParams(),
    RunRoutine: async () => (await import("./schemas/runRoutine")).runRoutineSearchParams(),
    RunRoutineInput: async () => (await import("./schemas/runRoutineInput")).runRoutineInputSearchParams(),
    Schedule: async () => (await import("./schemas/schedule")).scheduleSearchParams(),
    SmartContract: async () => (await import("./schemas/smartContract")).smartContractSearchParams(),
    SmartContractVersion: async () => (await import("./schemas/smartContractVersion")).smartContractVersionSearchParams(),
    Standard: async () => (await import("./schemas/standard")).standardSearchParams(),
    StandardVersion: async () => (await import("./schemas/standardVersion")).standardVersionSearchParams(),
    StatsApi: async () => (await import("./schemas/statsApi")).statsApiSearchParams(),
    StatsOrganization: async () => (await import("./schemas/statsOrganization")).statsOrganizationSearchParams(),
    StatsProject: async () => (await import("./schemas/statsProject")).statsProjectSearchParams(),
    StatsQuiz: async () => (await import("./schemas/statsQuiz")).statsQuizSearchParams(),
    StatsRoutine: async () => (await import("./schemas/statsRoutine")).statsRoutineSearchParams(),
    StatsSite: async () => (await import("./schemas/statsSite")).statsSiteSearchParams(),
    StatsSmartContract: async () => (await import("./schemas/statsSmartContract")).statsSmartContractSearchParams(),
    StatsStandard: async () => (await import("./schemas/statsStandard")).statsStandardSearchParams(),
    StatsUser: async () => (await import("./schemas/statsUser")).statsUserSearchParams(),
    Tag: async () => (await import("./schemas/tag")).tagSearchParams(),
    Transfer: async () => (await import("./schemas/transfer")).transferSearchParams(),
    User: async () => (await import("./schemas/user")).userSearchParams(),
    View: async () => (await import("./schemas/view")).viewSearchParams(),
};
