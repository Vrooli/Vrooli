import { apiSearchParams } from "./schemas/api";
import { apiVersionSearchParams } from "./schemas/apiVersion";
import { SearchParams } from "./schemas/base";
import { bookmarkSearchParams } from "./schemas/bookmark";
import { bookmarkListSearchParams } from "./schemas/bookmarkList";
import { chatSearchParams } from "./schemas/chat";
import { chatInviteSearchParams } from "./schemas/chatInvite";
import { chatMessageSearchParams } from "./schemas/chatMessage";
import { chatParticipantSearchParams } from "./schemas/chatParticipant";
import { commentSearchParams } from "./schemas/comment";
import { focusModeSearchParams } from "./schemas/focusMode";
import { issueSearchParams } from "./schemas/issue";
import { labelSearchParams } from "./schemas/label";
import { meetingSearchParams } from "./schemas/meeting";
import { meetingInviteSearchParams } from "./schemas/meetingInvite";
import { memberSearchParams } from "./schemas/member";
import { memberInviteSearchParams } from "./schemas/memberInvite";
import { noteSearchParams } from "./schemas/note";
import { noteVersionSearchParams } from "./schemas/noteVersion";
import { notificationSearchParams } from "./schemas/notification";
import { notificationSubscriptionSearchParams } from "./schemas/notificationSubscription";
import { organizationSearchParams } from "./schemas/organization";
import { popularSearchParams } from "./schemas/popular";
import { postSearchParams } from "./schemas/post";
import { projectSearchParams } from "./schemas/project";
import { projectOrOrganizationSearchParams } from "./schemas/projectOrOrganization";
import { projectOrRoutineSearchParams } from "./schemas/projectOrRoutine";
import { projectVersionSearchParams } from "./schemas/projectVersion";
import { pullRequestSearchParams } from "./schemas/pullRequest";
import { questionSearchParams } from "./schemas/question";
import { questionAnswerSearchParams } from "./schemas/questionAnswer";
import { quizSearchParams } from "./schemas/quiz";
import { quizAttemptSearchParams } from "./schemas/quizAttempt";
import { quizQuestionSearchParams } from "./schemas/quizQuestion";
import { quizQuestionResponseSearchParams } from "./schemas/quizQuestionResponse";
import { reactionSearchParams } from "./schemas/reaction";
import { reminderSearchParams } from "./schemas/reminder";
import { reportSearchParams } from "./schemas/report";
import { reportResponseSearchParams } from "./schemas/reportResponse";
import { reputationHistorySearchParams } from "./schemas/reputationHistory";
import { resourceSearchParams } from "./schemas/resource";
import { resourceListSearchParams } from "./schemas/resourceList";
import { roleSearchParams } from "./schemas/role";
import { routineSearchParams } from "./schemas/routine";
import { routineVersionSearchParams } from "./schemas/routineVersion";
import { runProjectSearchParams } from "./schemas/runProject";
import { runProjectOrRunRoutineSearchParams } from "./schemas/runProjectOrRunRoutine";
import { runRoutineSearchParams } from "./schemas/runRoutine";
import { runRoutineInputSearchParams } from "./schemas/runRoutineInput";
import { scheduleSearchParams } from "./schemas/schedule";
import { smartContractSearchParams } from "./schemas/smartContract";
import { smartContractVersionSearchParams } from "./schemas/smartContractVersion";
import { standardSearchParams } from "./schemas/standard";
import { standardVersionSearchParams } from "./schemas/standardVersion";
import { statsApiSearchParams } from "./schemas/statsApi";
import { statsOrganizationSearchParams } from "./schemas/statsOrganization";
import { statsProjectSearchParams } from "./schemas/statsProject";
import { statsQuizSearchParams } from "./schemas/statsQuiz";
import { statsRoutineSearchParams } from "./schemas/statsRoutine";
import { statsSiteSearchParams } from "./schemas/statsSite";
import { statsSmartContractSearchParams } from "./schemas/statsSmartContract";
import { statsStandardSearchParams } from "./schemas/statsStandard";
import { statsUserSearchParams } from "./schemas/statsUser";
import { tagSearchParams } from "./schemas/tag";
import { transferSearchParams } from "./schemas/transfer";
import { userSearchParams } from "./schemas/user";
import { viewSearchParams } from "./schemas/view";

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

export enum MemberManagePageTabOption {
    Members = "Members",
    MemberInvites = "MemberInvites",
}

export enum MyStuffPageTabOption {
    Apis = "Apis",
    Notes = "Notes",
    Organizations = "Organizations",
    Projects = "Projects",
    Questions = "Questions",
    Reminders = "Reminders",
    Routines = "Routines",
    SmartContracts = "SmartContracts",
    Schedules = "Schedules",
    Standards = "Standards",
    Users = "Users",
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
    All = "All",
    FocusModes = "FocusModes",
    Meetings = "Meetings",
    RunProjects = "RunProjects",
    RunRoutines = "RunRoutines",
}

export enum InboxPageTabOption {
    Notifications = "Notifications",
    Messages = "Messages",
}


/**
 * Maps search types to values needed to query and display results
 */
export const searchTypeToParams: { [key in SearchType]: () => SearchParams } = {
    Api: apiSearchParams,
    ApiVersion: apiVersionSearchParams,
    Bookmark: bookmarkSearchParams,
    BookmarkList: bookmarkListSearchParams,
    Chat: chatSearchParams,
    ChatInvite: chatInviteSearchParams,
    ChatMessage: chatMessageSearchParams,
    ChatParticipant: chatParticipantSearchParams,
    Comment: commentSearchParams,
    FocusMode: focusModeSearchParams,
    Issue: issueSearchParams,
    Label: labelSearchParams,
    Meeting: meetingSearchParams,
    MeetingInvite: meetingInviteSearchParams,
    Member: memberSearchParams,
    MemberInvite: memberInviteSearchParams,
    Note: noteSearchParams,
    NoteVersion: noteVersionSearchParams,
    Notification: notificationSearchParams,
    NotificationSubscription: notificationSubscriptionSearchParams,
    Organization: organizationSearchParams,
    Popular: popularSearchParams,
    Post: postSearchParams,
    Project: projectSearchParams,
    ProjectOrOrganization: projectOrOrganizationSearchParams,
    ProjectOrRoutine: projectOrRoutineSearchParams,
    ProjectVersion: projectVersionSearchParams,
    // ProjectVersionDirectory: projectVersionDirectorySearchParams,
    PullRequest: pullRequestSearchParams,
    Question: questionSearchParams,
    QuestionAnswer: questionAnswerSearchParams,
    Quiz: quizSearchParams,
    QuizAttempt: quizAttemptSearchParams,
    QuizQuestion: quizQuestionSearchParams,
    QuizQuestionResponse: quizQuestionResponseSearchParams,
    Reaction: reactionSearchParams,
    Reminder: reminderSearchParams,
    Report: reportSearchParams,
    ReportResponse: reportResponseSearchParams,
    ReputationHistory: reputationHistorySearchParams,
    Resource: resourceSearchParams,
    ResourceList: resourceListSearchParams,
    Role: roleSearchParams,
    Routine: routineSearchParams,
    RoutineVersion: routineVersionSearchParams,
    RunProject: runProjectSearchParams,
    RunProjectOrRunRoutine: runProjectOrRunRoutineSearchParams,
    RunRoutine: runRoutineSearchParams,
    RunRoutineInput: runRoutineInputSearchParams,
    Schedule: scheduleSearchParams,
    SmartContract: smartContractSearchParams,
    SmartContractVersion: smartContractVersionSearchParams,
    Standard: standardSearchParams,
    StandardVersion: standardVersionSearchParams,
    StatsApi: statsApiSearchParams,
    StatsOrganization: statsOrganizationSearchParams,
    StatsProject: statsProjectSearchParams,
    StatsQuiz: statsQuizSearchParams,
    StatsRoutine: statsRoutineSearchParams,
    StatsSite: statsSiteSearchParams,
    StatsSmartContract: statsSmartContractSearchParams,
    StatsStandard: statsStandardSearchParams,
    StatsUser: statsUserSearchParams,
    Tag: tagSearchParams,
    Transfer: transferSearchParams,
    User: userSearchParams,
    View: viewSearchParams,
};
