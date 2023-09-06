import { CommonKey, LINKS, MemberInviteStatus, RunStatus, ScheduleFor, VisibilityType } from "@local/shared";
import { Palette } from "@mui/material";
import { ApiIcon, CommentIcon, FocusModeIcon, HelpIcon, MonthIcon, NoteIcon, OrganizationIcon, ProjectIcon, ReminderIcon, RoutineIcon, SmartContractIcon, StandardIcon, UserIcon, VisibleIcon } from "icons";
import { YouInflated } from "utils/display/listTools";
import { PolicyTabOption } from "views/legal";
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
    Member = "Member",
    MemberInvite = "MemberInvite",
}

export enum MyStuffPageTabOption {
    All = "All",
    Api = "Api",
    Note = "Note",
    Organization = "Organization",
    Project = "Project",
    Question = "Question",
    Reminder = "Reminder",
    Routine = "Routine",
    SmartContract = "SmartContract",
    Schedule = "Schedule",
    Standard = "Standard",
    User = "User",
}

export enum OrganizationPageTabOption {
    Resource = "Resource",
    Project = "Project",
    Member = "Member",
}

export enum SearchPageTabOption {
    All = "All",
    Api = "Api",
    Note = "Note",
    Organization = "Organization",
    Project = "Project",
    Question = "Question",
    Routine = "Routine",
    SmartContract = "SmartContract",
    Standard = "Standard",
    User = "User",
}

export enum UserPageTabOption {
    Details = "Details",
    Project = "Project",
    Organization = "Organization",
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

export enum ChatPageTabOption {
    Chat = "Chat",
    Routine = "Routine",
    Prompt = "Prompt",
}

export const searchViewTabParams = [{
    Icon: VisibleIcon,
    titleKey: "All" as CommonKey,
    searchType: SearchType.Popular,
    tabType: SearchPageTabOption.All,
    where: () => ({}),
}, {
    Icon: RoutineIcon,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: SearchPageTabOption.Routine,
    where: () => ({ isInternal: false }),
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: SearchPageTabOption.Project,
    where: () => ({}),
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    searchType: SearchType.Question,
    tabType: SearchPageTabOption.Question,
    where: () => ({}),
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    searchType: SearchType.Note,
    tabType: SearchPageTabOption.Note,
    where: () => ({}),
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: SearchPageTabOption.Organization,
    where: () => ({}),
}, {
    Icon: UserIcon,
    titleKey: "User" as CommonKey,
    searchType: SearchType.User,
    tabType: SearchPageTabOption.User,
    where: () => ({}),
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    searchType: SearchType.Standard,
    tabType: SearchPageTabOption.Standard,
    where: () => ({ isInternal: false, type: "JSON" }),
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    searchType: SearchType.Api,
    tabType: SearchPageTabOption.Api,
    where: () => ({}),
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
    searchType: SearchType.SmartContract,
    tabType: SearchPageTabOption.SmartContract,
    where: () => ({}),
}];

export const calendarTabParams = [{
    titleKey: "All" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: CalendarPageTabOption.All,
    where: () => ({}),
}, {
    titleKey: "Meeting" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: CalendarPageTabOption.Meeting,
    where: () => ({ scheduleFor: ScheduleFor.Meeting }),
}, {
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: CalendarPageTabOption.RunRoutine,
    where: () => ({ scheduleFor: ScheduleFor.RunRoutine }),
}, {
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: CalendarPageTabOption.RunProject,
    where: () => ({ scheduleFor: ScheduleFor.RunProject }),
}, {
    titleKey: "FocusMode" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: CalendarPageTabOption.FocusMode,
    where: () => ({ scheduleFor: ScheduleFor.FocusMode }),
}];

export const chatTabParams = [{
    Icon: CommentIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Chat" as CommonKey,
    searchType: SearchType.Chat,
    tabType: ChatPageTabOption.Chat,
    where: () => ({}),
}, {
    Icon: RoutineIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: ChatPageTabOption.Routine,
    where: () => ({}),
}, {
    Icon: StandardIcon,
    color: (palette: Palette) => palette.primary.contrastText,
    titleKey: "Prompt" as CommonKey,
    searchType: SearchType.Standard,
    tabType: ChatPageTabOption.Prompt,
    where: () => ({}),
}];

export const historyTabParams = [{
    titleKey: "View" as CommonKey,
    searchType: SearchType.View,
    tabType: HistoryPageTabOption.Viewed,
    where: () => ({}),
}, {
    titleKey: "Bookmark" as CommonKey,
    searchType: SearchType.BookmarkList,
    tabType: HistoryPageTabOption.Bookmarked,
    where: () => ({}),
}, {
    titleKey: "Active" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsActive,
    where: () => ({ statuses: [RunStatus.InProgress, RunStatus.Scheduled] }),
}, {
    titleKey: "Complete" as CommonKey,
    searchType: SearchType.RunProjectOrRunRoutine,
    tabType: HistoryPageTabOption.RunsCompleted,
    where: () => ({ statuses: [RunStatus.Cancelled, RunStatus.Completed, RunStatus.Failed] }),
}];

export const findObjectTabParams = [
    ...searchViewTabParams,
    {
        Icon: FocusModeIcon,
        titleKey: "FocusMode" as CommonKey,
        searchType: SearchType.FocusMode,
        tabType: CalendarPageTabOption.FocusMode,
        where: () => ({}),
    }, {
        Icon: OrganizationIcon,
        titleKey: "Meeting" as CommonKey,
        searchType: SearchType.Meeting,
        tabType: CalendarPageTabOption.Meeting,
        where: () => ({}),
    }, {
        Icon: RoutineIcon,
        titleKey: "RunRoutine" as CommonKey,
        searchType: SearchType.RunRoutine,
        tabType: CalendarPageTabOption.RunRoutine,
        where: () => ({}),
    }, {
        Icon: ProjectIcon,
        titleKey: "RunProject" as CommonKey,
        searchType: SearchType.RunProject,
        tabType: CalendarPageTabOption.RunProject,
        where: () => ({}),
    },
];

export const inboxTabParams = [{
    titleKey: "Message" as CommonKey,
    searchType: SearchType.Chat,
    tabType: InboxPageTabOption.Message,
    where: () => ({}),
}, {
    titleKey: "Notification" as CommonKey,
    searchType: SearchType.Notification,
    tabType: InboxPageTabOption.Notification,
    where: () => ({}),
}];

export const myStuffTabParams = [{
    Icon: VisibleIcon,
    titleKey: "All" as CommonKey,
    searchType: SearchType.Popular,
    tabType: MyStuffPageTabOption.All,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: RoutineIcon,
    titleKey: "Routine" as CommonKey,
    searchType: SearchType.Routine,
    tabType: MyStuffPageTabOption.Routine,
    where: () => ({ isInternal: false, visibility: VisibilityType.Own }),
}, {
    Icon: ProjectIcon,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: MyStuffPageTabOption.Project,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: MonthIcon,
    titleKey: "Schedule" as CommonKey,
    searchType: SearchType.Schedule,
    tabType: MyStuffPageTabOption.Schedule,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ReminderIcon,
    titleKey: "Reminder" as CommonKey,
    searchType: SearchType.Reminder,
    tabType: MyStuffPageTabOption.Reminder,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: NoteIcon,
    titleKey: "Note" as CommonKey,
    searchType: SearchType.Note,
    tabType: MyStuffPageTabOption.Note,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: HelpIcon,
    titleKey: "Question" as CommonKey,
    searchType: SearchType.Question,
    tabType: MyStuffPageTabOption.Question,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: OrganizationIcon,
    titleKey: "Organization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: MyStuffPageTabOption.Organization,
    where: ({ userId }) => ({ memberUserIds: [userId] }),
}, {
    Icon: UserIcon,
    titleKey: "Bot" as CommonKey,
    searchType: SearchType.User,
    tabType: MyStuffPageTabOption.User,
    where: ({ userId }) => ({ visibility: VisibilityType.Own, isBot: true, excludeIds: [userId] }),
}, {
    Icon: StandardIcon,
    titleKey: "Standard" as CommonKey,
    searchType: SearchType.Standard,
    tabType: MyStuffPageTabOption.Standard,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ApiIcon,
    titleKey: "Api" as CommonKey,
    searchType: SearchType.Api,
    tabType: MyStuffPageTabOption.Api,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: SmartContractIcon,
    titleKey: "SmartContract" as CommonKey,
    searchType: SearchType.SmartContract,
    tabType: MyStuffPageTabOption.SmartContract,
    where: () => ({ visibility: VisibilityType.Own }),
}];

export const memberTabParams = [{
    titleKey: "Member" as CommonKey,
    searchType: SearchType.Member,
    tabType: MemberManagePageTabOption.Member,
    where: (organizationId: string) => ({ organizationId }),
}, {
    titleKey: "Invite" as CommonKey,
    searchType: SearchType.MemberInvite,
    tabType: MemberManagePageTabOption.MemberInvite,
    where: (organizationId: string) => ({ organizationId, statuses: [MemberInviteStatus.Pending, MemberInviteStatus.Declined] }),
}];

type OrganizationTabWhereParams = {
    organizationId: string;
    permissions: YouInflated;
}
const organizationTabColor = (palette: Palette) => ({ active: palette.secondary.main, inactive: palette.background.textSecondary });
export const organizationTabParams = [{
    color: organizationTabColor,
    titleKey: "Resource" as CommonKey,
    searchType: SearchType.Resource,
    tabType: OrganizationPageTabOption.Resource,
    where: () => ({}),
}, {
    color: organizationTabColor,
    titleKey: "Project" as CommonKey,
    searchType: SearchType.Project,
    tabType: OrganizationPageTabOption.Project,
    where: ({ organizationId, permissions }: OrganizationTabWhereParams) => ({ ownedByOrganizationId: organizationId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    color: organizationTabColor,
    titleKey: "Member" as CommonKey,
    searchType: SearchType.Member,
    tabType: OrganizationPageTabOption.Member,
    where: ({ organizationId }: OrganizationTabWhereParams) => ({ organizationId }),
}];

type UserTabWhereParams = {
    userId: string;
    permissions: YouInflated;
}
const userTabColor = (palette: Palette) => ({ active: palette.secondary.main, inactive: palette.background.textSecondary });
export const userTabParams = [{
    color: userTabColor,
    titleKey: "Details" as CommonKey,
    searchType: SearchType.User, // Ignored
    tabType: UserPageTabOption.Details,
    where: () => ({}),
}, {
    color: userTabColor,
    titleKey: "Project" as CommonKey,
    searchPlaceholderKey: "SearchProject" as CommonKey,
    searchType: SearchType.Project,
    tabType: UserPageTabOption.Project,
    where: ({ userId, permissions }: UserTabWhereParams) => ({ ownedByUserId: userId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    color: userTabColor,
    titleKey: "Organization" as CommonKey,
    searchPlaceholderKey: "SearchOrganization" as CommonKey,
    searchType: SearchType.Organization,
    tabType: UserPageTabOption.Organization,
    where: ({ userId }: UserTabWhereParams) => ({ memberUserIds: [userId], visibility: VisibilityType.All }),
}];

export const policyTabParams = [
    {
        titleKey: "Privacy" as CommonKey,
        href: LINKS.Privacy,
        tabType: PolicyTabOption.Privacy,
    }, {
        titleKey: "Terms" as CommonKey,
        href: LINKS.Terms,
        tabType: PolicyTabOption.Terms,
    },
];

/** Maps search types to values needed to query and display results */
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
