import { BookmarkFor, ChatInviteStatus, CommonKey, LINKS, MemberInviteStatus, RunStatus, ScheduleFor, VisibilityType, YouInflated } from "@local/shared";
import { Palette } from "@mui/material";
import { PageTab } from "hooks/useTabs";
import { AddIcon, ApiIcon, ArticleIcon, BookmarkFilledIcon, CommentIcon, FocusModeIcon, HelpIcon, MonthIcon, NoteIcon, ProjectIcon, ReminderIcon, RoutineIcon, StandardIcon, TeamIcon, TerminalIcon, UserIcon, VisibleIcon } from "icons";
import { SvgComponent } from "types";
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
import { codeSearchParams } from "./schemas/code";
import { codeVersionSearchParams } from "./schemas/codeVersion";
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
import { popularSearchParams } from "./schemas/popular";
import { postSearchParams } from "./schemas/post";
import { projectSearchParams } from "./schemas/project";
import { projectOrRoutineSearchParams } from "./schemas/projectOrRoutine";
import { projectOrTeamSearchParams } from "./schemas/projectOrTeam";
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
import { standardSearchParams } from "./schemas/standard";
import { standardVersionSearchParams } from "./schemas/standardVersion";
import { statsApiSearchParams } from "./schemas/statsApi";
import { statsCodeSearchParams } from "./schemas/statsCode";
import { statsProjectSearchParams } from "./schemas/statsProject";
import { statsQuizSearchParams } from "./schemas/statsQuiz";
import { statsRoutineSearchParams } from "./schemas/statsRoutine";
import { statsSiteSearchParams } from "./schemas/statsSite";
import { statsStandardSearchParams } from "./schemas/statsStandard";
import { statsTeamSearchParams } from "./schemas/statsTeam";
import { statsUserSearchParams } from "./schemas/statsUser";
import { tagSearchParams } from "./schemas/tag";
import { teamSearchParams } from "./schemas/team";
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
    Code = "Code",
    CodeVersion = "CodeVersion",
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
    Popular = "Popular",
    Post = "Post",
    ProjectOrRoutine = "ProjectOrRoutine",
    ProjectOrTeam = "ProjectOrTeam",
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
    Standard = "Standard",
    StandardVersion = "StandardVersion",
    StatsApi = "StatsApi",
    StatsCode = "StatsCode",
    StatsProject = "StatsProject",
    StatsQuiz = "StatsQuiz",
    StatsRoutine = "StatsRoutine",
    StatsSite = "StatsSite",
    StatsStandard = "StatsStandard",
    StatsTeam = "StatsTeam",
    StatsUser = "StatsUser",
    Tag = "Tag",
    Team = "Team",
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
    Code = "Code",
    Note = "Note",
    Project = "Project",
    Question = "Question",
    Routine = "Routine",
    Standard = "Standard",
    Team = "Team",
    User = "User",
}

export enum SearchVersionPageTabOption {
    ApiVersion = "ApiVersion",
    CodeVersion = "CodeVersion",
    NoteVersion = "NoteVersion",
    ProjectVersion = "ProjectVersion",
    RoutineVersion = "RoutineVersion",
    StandardVersion = "StandardVersion",
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
    Favorite = "Favorite",
    Routine = "Routine",
    Prompt = "Prompt",
}

export type TabsInfo = {
    IsSearchable: boolean;
    Key: string;
    Payload: object | undefined;
    WhereParams: object | undefined;
}

export type TabStateColors = {
    active: string;
    inactive: string;
}

export type TabParam<TabList extends TabsInfo> = {
    color?: (palette: Palette) => (string | TabStateColors)
    href?: string;
    Icon?: SvgComponent,
    key: TabList["Key"] | `${TabList["Key"]}`;
    titleKey: CommonKey;
} & (TabList["IsSearchable"] extends true ? {
    searchPlaceholderKey?: CommonKey;
    searchType: SearchType | `${SearchType}`;
    where: TabList["WhereParams"] extends undefined ? () => { [x: string]: any } : (params: TabList["WhereParams"]) => { [x: string]: any };
} : object) & (TabList["Payload"] extends undefined ? object : {
    data: TabList["Payload"];
});

export type SearchViewTabsInfo = {
    IsSearchable: true;
    Key: SearchPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export function getTabIcon<T extends TabsInfo>(option: PageTab<T>) {
    return option.Icon;
}

export function getTabLabel<T extends TabsInfo>(option: PageTab<T>) {
    return option.label;
}

export const searchViewTabParams: TabParam<SearchViewTabsInfo>[] = [{
    Icon: VisibleIcon,
    key: "All",
    titleKey: "All",
    searchType: "Popular",
    where: () => ({}),
}, {
    Icon: RoutineIcon,
    key: "Routine",
    titleKey: "Routine",
    searchType: "Routine",
    where: () => ({ isInternal: false }),
}, {
    Icon: ProjectIcon,
    key: "Project",
    titleKey: "Project",
    searchType: "Project",
    where: () => ({}),
}, {
    Icon: HelpIcon,
    key: "Question",
    titleKey: "Question",
    searchType: "Question",
    where: () => ({}),
}, {
    Icon: NoteIcon,
    key: "Note",
    titleKey: "Note",
    searchType: "Note",
    where: () => ({}),
}, {
    Icon: TeamIcon,
    key: "Team",
    titleKey: "Team",
    searchType: "Team",
    where: () => ({}),
}, {
    Icon: UserIcon,
    key: "User",
    titleKey: "User",
    searchType: "User",
    where: () => ({}),
}, {
    Icon: StandardIcon,
    key: "Standard",
    titleKey: "Standard",
    searchType: "Standard",
    where: () => ({ isInternal: false, type: "JSON" }),
}, {
    Icon: ApiIcon,
    key: "Api",
    titleKey: "Api",
    searchType: "Api",
    where: () => ({}),
}, {
    Icon: TerminalIcon,
    key: "Code",
    titleKey: "Code",
    searchType: "Code",
    where: () => ({}),
}];

export type SearchVersionViewTabsInfo = {
    IsSearchable: true;
    Key: SearchVersionPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const searchVersionViewTabParams: TabParam<SearchVersionViewTabsInfo>[] = [{
    Icon: RoutineIcon,
    key: "RoutineVersion",
    titleKey: "Routine",
    searchType: "RoutineVersion",
    where: () => ({ isInternalWithRoot: false }),
}, {
    Icon: ProjectIcon,
    key: "ProjectVersion",
    titleKey: "Project",
    searchType: "ProjectVersion",
    where: () => ({}),
}, {
    Icon: NoteIcon,
    key: "NoteVersion",
    titleKey: "Note",
    searchType: "NoteVersion",
    where: () => ({}),
}, {
    Icon: StandardIcon,
    key: "StandardVersion",
    titleKey: "Standard",
    searchType: "StandardVersion",
    where: () => ({ isInternalWithRoot: false }),
}, {
    Icon: ApiIcon,
    key: "ApiVersion",
    titleKey: "Api",
    searchType: "ApiVersion",
    where: () => ({}),
}, {
    Icon: TerminalIcon,
    key: "CodeVersion",
    titleKey: "Code",
    searchType: "CodeVersion",
    where: () => ({}),
}];

export type CalendarTabsInfo = {
    IsSearchable: true;
    Key: CalendarPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const calendarTabParams: TabParam<CalendarTabsInfo>[] = [{
    key: "All",
    titleKey: "All",
    searchType: "Schedule",
    where: () => ({}),
}, {
    key: "Meeting",
    titleKey: "Meeting",
    searchType: "Schedule",
    where: () => ({ scheduleFor: ScheduleFor.Meeting }),
}, {
    key: "RunRoutine",
    titleKey: "Routine",
    searchType: "Schedule",
    where: () => ({ scheduleFor: ScheduleFor.RunRoutine }),
}, {
    key: "RunProject",
    titleKey: "Project",
    searchType: "Schedule",
    where: () => ({ scheduleFor: ScheduleFor.RunProject }),
}, {
    key: "FocusMode",
    titleKey: "FocusMode",
    searchType: "Schedule",
    where: () => ({ scheduleFor: ScheduleFor.FocusMode }),
}];

export type ChatTabsInfo = {
    IsSearchable: true;
    Key: ChatPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const chatTabParams: TabParam<ChatTabsInfo>[] = [{
    color: (palette) => palette.primary.contrastText,
    Icon: CommentIcon,
    key: "Chat",
    titleKey: "Chat",
    searchType: "Chat",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    color: (palette) => palette.primary.contrastText,
    Icon: BookmarkFilledIcon,
    key: "Favorite",
    titleKey: "Favorite",
    searchType: "Bookmark",
    where: () => ({ label: "Favorites", limitTo: [BookmarkFor.Routine, BookmarkFor.Standard] }), // Routines to run them, and standards for prompts
}, {
    color: (palette) => palette.primary.contrastText,
    Icon: RoutineIcon,
    key: "Routine",
    titleKey: "Routine",
    searchType: "Routine",
    where: () => ({
        Public: {},
        My: { visibility: VisibilityType.Own },
    }),
}, {
    color: (palette) => palette.primary.contrastText,
    Icon: ArticleIcon,
    key: "Prompt",
    titleKey: "Prompt",
    searchType: "Standard",
    where: () => ({
        Public: {},
        My: { visibility: VisibilityType.Own },
    }),
}];

export type HistoryTabsInfo = {
    IsSearchable: true;
    Key: HistoryPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const historyTabParams: TabParam<HistoryTabsInfo>[] = [{
    key: "Viewed",
    titleKey: "View",
    searchType: "View",
    where: () => ({}),
}, {
    key: "Bookmarked",
    titleKey: "Bookmark",
    searchType: "BookmarkList",
    where: () => ({}),
}, {
    key: "RunsActive",
    titleKey: "Active",
    searchType: "RunProjectOrRunRoutine",
    where: () => ({ statuses: [RunStatus.InProgress, RunStatus.Scheduled] }),
}, {
    key: "RunsCompleted",
    titleKey: "Complete",
    searchType: "RunProjectOrRunRoutine",
    where: () => ({ statuses: [RunStatus.Cancelled, RunStatus.Completed, RunStatus.Failed] }),
}];

export type FindObjectTabsInfo = {
    IsSearchable: true;
    Key: SearchPageTabOption | CalendarPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const findObjectTabParams: TabParam<FindObjectTabsInfo>[] = [
    ...searchViewTabParams,
    {
        Icon: FocusModeIcon,
        key: "FocusMode",
        titleKey: "FocusMode",
        searchType: "FocusMode",
        where: () => ({}),
    }, {
        Icon: TeamIcon,
        key: "Meeting",
        titleKey: "Meeting",
        searchType: "Meeting",
        where: () => ({}),
    }, {
        Icon: RoutineIcon,
        key: "RunRoutine",
        titleKey: "RunRoutine",
        searchType: "RunRoutine",
        where: () => ({}),
    }, {
        Icon: ProjectIcon,
        key: "RunProject",
        titleKey: "RunProject",
        searchType: "RunProject",
        where: () => ({}),
    },
];

export type InboxTabsInfo = {
    IsSearchable: true;
    Key: InboxPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const inboxTabParams: TabParam<InboxTabsInfo>[] = [{
    key: "Message",
    titleKey: "Message",
    searchType: "Chat",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    key: "Notification",
    titleKey: "Notification",
    searchType: "Notification",
    where: () => ({ visibility: VisibilityType.Own }),
}];

export type SignUpTabsInfo = {
    IsSearchable: false;
    Key: SignUpPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const signUpTabParams: TabParam<SignUpTabsInfo>[] = [{
    key: "SignUp",
    titleKey: "SignUp",
}, {
    key: "MoreInfo",
    titleKey: "MoreInfo",
}];

export type MyStuffTabsInfo = {
    IsSearchable: true;
    Key: MyStuffPageTabOption;
    Payload: undefined;
    WhereParams: { userId: string };
}

export const myStuffTabParams: TabParam<MyStuffTabsInfo>[] = [{
    Icon: VisibleIcon,
    key: "All",
    titleKey: "All",
    searchType: "Popular",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ProjectIcon,
    key: "Project",
    titleKey: "Project",
    searchType: "Project",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: RoutineIcon,
    key: "Routine",
    titleKey: "Routine",
    searchType: "Routine",
    where: () => ({ isInternal: false, visibility: VisibilityType.Own }),
}, {
    Icon: MonthIcon,
    key: "Schedule",
    titleKey: "Schedule",
    searchType: "Schedule",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ReminderIcon,
    key: "Reminder",
    titleKey: "Reminder",
    searchType: "Reminder",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: NoteIcon,
    key: "Note",
    titleKey: "Note",
    searchType: "Note",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: HelpIcon,
    key: "Question",
    titleKey: "Question",
    searchType: "Question",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: TeamIcon,
    key: "Team",
    titleKey: "Team",
    searchType: "Team",
    where: ({ userId }) => ({ memberUserIds: [userId] }),
}, {
    Icon: UserIcon,
    key: "User",
    titleKey: "Bot",
    searchType: "User",
    where: ({ userId }) => ({ visibility: VisibilityType.Own, isBot: true, excludeIds: [userId] }),
}, {
    Icon: StandardIcon,
    key: "Standard",
    titleKey: "Standard",
    searchType: "Standard",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: ApiIcon,
    key: "Api",
    titleKey: "Api",
    searchType: "Api",
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    Icon: TerminalIcon,
    key: "Code",
    titleKey: "Code",
    searchType: "Code",
    where: () => ({ visibility: VisibilityType.Own }),
}];

export type MemberTabsInfo = {
    IsSearchable: true;
    Key: MemberManagePageTabOption;
    Payload: undefined;
    WhereParams: { teamId: string };
}

export const memberTabParams: TabParam<MemberTabsInfo>[] = [{
    key: "Member",
    titleKey: "Member",
    searchType: "Member",
    where: ({ teamId }) => ({ teamId }),
}, {
    key: "MemberInvite",
    titleKey: "Invite",
    searchType: "MemberInvite",
    where: ({ teamId }) => ({ teamId, statuses: [MemberInviteStatus.Pending, MemberInviteStatus.Declined] }),
}];

export type TeamTabsInfo = {
    IsSearchable: true;
    Key: TeamPageTabOption;
    Payload: undefined;
    WhereParams: {
        teamId: string;
        permissions: YouInflated;
    };
}

const teamTabColor = (palette: Palette) => ({ active: palette.secondary.main, inactive: palette.background.textSecondary });
export const teamTabParams: TabParam<TeamTabsInfo>[] = [{
    color: teamTabColor,
    key: "Resource",
    titleKey: "Resource",
    searchType: "Resource",
    where: () => ({}),
}, {
    color: teamTabColor,
    key: "Project",
    titleKey: "Project",
    searchType: "Project",
    where: ({ teamId, permissions }) => ({ ownedByTeamId: teamId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    color: teamTabColor,
    key: "Member",
    titleKey: "Member",
    searchType: "Member",
    where: ({ teamId }) => ({ teamId }),
}];

export type UserTabsInfo = {
    IsSearchable: true;
    Key: UserPageTabOption;
    Payload: undefined;
    WhereParams: {
        userId: string;
        permissions: YouInflated;
    };
}

const userTabColor = (palette: Palette) => ({ active: palette.secondary.main, inactive: palette.background.textSecondary });
export const userTabParams: TabParam<UserTabsInfo>[] = [{
    color: userTabColor,
    key: "Details",
    titleKey: "Details",
    searchType: "User", // Ignored
    where: () => ({}),
}, {
    color: userTabColor,
    key: "Project",
    titleKey: "Project",
    searchPlaceholderKey: "SearchProject",
    searchType: "Project",
    where: ({ userId, permissions }) => ({ ownedByUserId: userId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.All }),
}, {
    color: userTabColor,
    key: "Team",
    titleKey: "Team",
    searchPlaceholderKey: "SearchTeam",
    searchType: "Team",
    where: ({ userId }) => ({ memberUserIds: [userId], visibility: VisibilityType.All }),
}];

export type ParticipantTabsInfo = {
    IsSearchable: true;
    Key: ParticipantManagePageTabOption;
    Payload: undefined;
    WhereParams: { chatId: string };
}

export const participantTabParams: TabParam<ParticipantTabsInfo>[] = [{
    key: "ChatParticipant",
    titleKey: "Participant",
    searchType: "ChatParticipant",
    where: ({ chatId }) => ({ chatId }),
}, {
    key: "ChatInvite",
    titleKey: "Invite",
    searchType: "ChatInvite",
    where: ({ chatId }) => ({ chatId, statuses: [ChatInviteStatus.Pending, ChatInviteStatus.Declined] }),
}, {
    Icon: AddIcon,
    key: "Add",
    titleKey: "SearchUser",
    searchType: "User",
    where: ({ chatId }) => ({ notInChatId: chatId }),
}];

export type PolicyTabsInfo = {
    IsSearchable: false;
    Key: PolicyTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const policyTabParams: TabParam<PolicyTabsInfo>[] = [
    {
        href: LINKS.Privacy,
        key: "Privacy",
        titleKey: "Privacy",
    }, {
        href: LINKS.Terms,
        key: "Terms",
        titleKey: "Terms",
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
    Code: codeSearchParams,
    CodeVersion: codeVersionSearchParams,
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
    Popular: popularSearchParams,
    Post: postSearchParams,
    Project: projectSearchParams,
    ProjectOrRoutine: projectOrRoutineSearchParams,
    ProjectOrTeam: projectOrTeamSearchParams,
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
    Standard: standardSearchParams,
    StandardVersion: standardVersionSearchParams,
    StatsApi: statsApiSearchParams,
    StatsCode: statsCodeSearchParams,
    StatsProject: statsProjectSearchParams,
    StatsQuiz: statsQuizSearchParams,
    StatsRoutine: statsRoutineSearchParams,
    StatsSite: statsSiteSearchParams,
    StatsStandard: statsStandardSearchParams,
    StatsTeam: statsTeamSearchParams,
    StatsUser: statsUserSearchParams,
    Tag: tagSearchParams,
    Team: teamSearchParams,
    Transfer: transferSearchParams,
    User: userSearchParams,
    View: viewSearchParams,
};
