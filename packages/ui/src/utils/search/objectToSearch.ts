import { CalendarPageTabOption, ChatInviteStatus, CodeType, HistoryPageTabOption, InboxPageTabOption, LINKS, MemberInviteStatus, MemberManagePageTabOption, MyStuffPageTabOption, ParticipantManagePageTabOption, RoutineType, RunStatus, ScheduleFor, SearchPageTabOption, SearchType, SearchTypeToSearchInput, SearchVersionPageTabOption, SignUpPageTabOption, StandardType, TeamPageTabOption, TranslationKeyCommon, UserPageTabOption, VisibilityType, YouInflated } from "@local/shared";
import { Palette } from "@mui/material";
import { IconInfo } from "../../icons/Icons.js";
import { PolicyTabOption } from "../../views/PolicyView/PolicyView.js";
import { apiSearchParams } from "./schemas/api.js";
import { apiVersionSearchParams } from "./schemas/apiVersion.js";
import { SearchParams } from "./schemas/base.js";
import { bookmarkSearchParams } from "./schemas/bookmark.js";
import { bookmarkListSearchParams } from "./schemas/bookmarkList.js";
import { chatSearchParams } from "./schemas/chat.js";
import { chatInviteSearchParams } from "./schemas/chatInvite.js";
import { chatMessageSearchParams } from "./schemas/chatMessage.js";
import { chatParticipantSearchParams } from "./schemas/chatParticipant.js";
import { codeSearchParams } from "./schemas/code.js";
import { codeVersionSearchParams } from "./schemas/codeVersion.js";
import { commentSearchParams } from "./schemas/comment.js";
import { focusModeSearchParams } from "./schemas/focusMode.js";
import { issueSearchParams } from "./schemas/issue.js";
import { labelSearchParams } from "./schemas/label.js";
import { meetingSearchParams } from "./schemas/meeting.js";
import { meetingInviteSearchParams } from "./schemas/meetingInvite.js";
import { memberSearchParams } from "./schemas/member.js";
import { memberInviteSearchParams } from "./schemas/memberInvite.js";
import { noteSearchParams } from "./schemas/note.js";
import { noteVersionSearchParams } from "./schemas/noteVersion.js";
import { notificationSearchParams } from "./schemas/notification.js";
import { notificationSubscriptionSearchParams } from "./schemas/notificationSubscription.js";
import { popularSearchParams } from "./schemas/popular.js";
import { postSearchParams } from "./schemas/post.js";
import { projectSearchParams } from "./schemas/project.js";
import { projectOrRoutineSearchParams } from "./schemas/projectOrRoutine.js";
import { projectOrTeamSearchParams } from "./schemas/projectOrTeam.js";
import { projectVersionSearchParams } from "./schemas/projectVersion.js";
import { pullRequestSearchParams } from "./schemas/pullRequest.js";
import { questionSearchParams } from "./schemas/question.js";
import { questionAnswerSearchParams } from "./schemas/questionAnswer.js";
import { quizSearchParams } from "./schemas/quiz.js";
import { quizAttemptSearchParams } from "./schemas/quizAttempt.js";
import { quizQuestionResponseSearchParams } from "./schemas/quizQuestionResponse.js";
import { reactionSearchParams } from "./schemas/reaction.js";
import { reminderSearchParams } from "./schemas/reminder.js";
import { reportSearchParams } from "./schemas/report.js";
import { reportResponseSearchParams } from "./schemas/reportResponse.js";
import { reputationHistorySearchParams } from "./schemas/reputationHistory.js";
import { resourceSearchParams } from "./schemas/resource.js";
import { resourceListSearchParams } from "./schemas/resourceList.js";
import { roleSearchParams } from "./schemas/role.js";
import { routineSearchParams } from "./schemas/routine.js";
import { routineVersionSearchParams } from "./schemas/routineVersion.js";
import { runProjectSearchParams } from "./schemas/runProject.js";
import { runProjectOrRunRoutineSearchParams } from "./schemas/runProjectOrRunRoutine.js";
import { runRoutineSearchParams } from "./schemas/runRoutine.js";
import { runRoutineIOSearchParams } from "./schemas/runRoutineIO.js";
import { scheduleSearchParams } from "./schemas/schedule.js";
import { standardSearchParams } from "./schemas/standard.js";
import { standardVersionSearchParams } from "./schemas/standardVersion.js";
import { statsApiSearchParams } from "./schemas/statsApi.js";
import { statsCodeSearchParams } from "./schemas/statsCode.js";
import { statsProjectSearchParams } from "./schemas/statsProject.js";
import { statsQuizSearchParams } from "./schemas/statsQuiz.js";
import { statsRoutineSearchParams } from "./schemas/statsRoutine.js";
import { statsSiteSearchParams } from "./schemas/statsSite.js";
import { statsStandardSearchParams } from "./schemas/statsStandard.js";
import { statsTeamSearchParams } from "./schemas/statsTeam.js";
import { statsUserSearchParams } from "./schemas/statsUser.js";
import { tagSearchParams } from "./schemas/tag.js";
import { teamSearchParams } from "./schemas/team.js";
import { transferSearchParams } from "./schemas/transfer.js";
import { userSearchParams } from "./schemas/user.js";
import { viewSearchParams } from "./schemas/view.js";

export type TabsInfo = {
    Key: string;
    Payload: object | undefined;
    WhereParams: object | undefined;
}

export type TabStateColors = {
    active: string;
    inactive: string;
}

type Exact<T, Shape> =
    T extends Shape
    ? (Shape extends T ? T : never)
    : never;

export type TabParamBase<TabList extends TabsInfo> = {
    color?: (palette: Palette) => (string | TabStateColors)
    href?: string;
    iconInfo?: IconInfo,
    key: TabList["Key"];
    titleKey: TranslationKeyCommon;
};
export type TabParamSearchable<TabList extends TabsInfo, S extends SearchType = SearchType> = TabParamBase<TabList> & {
    searchPlaceholderKey?: TranslationKeyCommon;
    searchType: S;
    where: (params: TabList["WhereParams"]) => Exact<SearchTypeToSearchInput[S], SearchTypeToSearchInput[S]>;
};
export type TabParamPayload<TabList extends TabsInfo> = TabParamBase<TabList> & {
    data: TabList["Payload"];
};
export type TabParam<TabList extends TabsInfo> = TabParamBase<TabList> | TabParamSearchable<TabList> | TabParamPayload<TabList>;

export type TabParamSearchableList<
    TabList extends TabsInfo,
    S extends Array<SearchType>
> = {
    [Index in keyof S]: S[Index] extends SearchType
    ? TabParamSearchable<TabList, S[Index]>
    : never;
}[number][];

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type TabListType = readonly TabParamBase<any>[];

export type SearchViewTabsInfo = {
    Key: SearchPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const searchViewTabParams: TabParamSearchableList<SearchViewTabsInfo, ["Popular", "Routine", "Project", "Question", "Note", "Team", "User", "Standard", "Api", "Code"]> = [
    {
        iconInfo: { name: "Visible", type: "Common" } as const,
        key: SearchPageTabOption.All,
        titleKey: "All",
        searchType: "Popular",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Routine", type: "Routine" } as const,
        key: SearchPageTabOption.RoutineMultiStep,
        titleKey: "RoutineMultiStep",
        searchType: "Routine",
        where: () => ({ isInternal: false, latestVersionRoutineType: RoutineType.MultiStep } as const),
    },
    {
        iconInfo: { name: "Action", type: "Common" } as const,
        key: SearchPageTabOption.RoutineSingleStep,
        titleKey: "RoutineSingleStep",
        searchType: "Routine",
        where: () => ({ isInternal: false, latestVersionRoutineTypes: Object.values(RoutineType).filter(type => type !== RoutineType.MultiStep) }),
    },
    {
        iconInfo: { name: "Project", type: "Common" } as const,
        key: SearchPageTabOption.Project,
        titleKey: "Project",
        searchType: "Project",
        where: () => ({}),
    },
    // {
    //     iconInfo: { name: "Help", type: "Common" } as const,
    //     key: SearchPageTabOption.Question,
    //     titleKey: "Question",
    //     searchType: "Question",
    //     where: () => ({}),
    // },
    {
        iconInfo: { name: "Note", type: "Common" } as const,
        key: SearchPageTabOption.Note,
        titleKey: "Note",
        searchType: "Note",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Team", type: "Common" } as const,
        key: SearchPageTabOption.Team,
        titleKey: "Team",
        searchType: "Team",
        where: () => ({}),
    },
    {
        iconInfo: { name: "User", type: "Common" } as const,
        key: SearchPageTabOption.User,
        titleKey: "User",
        searchType: "User",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Article", type: "Common" } as const,
        key: SearchPageTabOption.Prompt,
        titleKey: "Prompt",
        searchType: "Standard",
        where: () => ({ isInternal: false, variantLatestVersion: StandardType.Prompt }),
    },
    {
        iconInfo: { name: "Object", type: "Common" } as const,
        key: SearchPageTabOption.DataStructure,
        titleKey: "DataStructure",
        searchType: "Standard",
        where: () => ({ isInternal: false, variantLatestVersion: StandardType.DataStructure }),
    },
    {
        iconInfo: { name: "Api", type: "Common" } as const,
        key: SearchPageTabOption.Api,
        titleKey: "Api",
        searchType: "Api",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Terminal", type: "Common" } as const,
        key: SearchPageTabOption.DataConverter,
        titleKey: "DataConverter",
        searchType: "Code",
        where: () => ({ codeTypeLatestVersion: CodeType.DataConvert }),
    },
    {
        iconInfo: { name: "SmartContract", type: "Common" } as const,
        key: SearchPageTabOption.SmartContract,
        titleKey: "SmartContract",
        searchType: "Code",
        where: () => ({ codeTypeLatestVersion: CodeType.SmartContract }),
    },
];

export type SearchVersionViewTabsInfo = {
    Key: SearchVersionPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const searchVersionViewTabParams: TabParamSearchableList<SearchVersionViewTabsInfo, ["RoutineVersion", "ProjectVersion", "NoteVersion", "StandardVersion", "ApiVersion", "CodeVersion"]> = [
    {
        iconInfo: { name: "Routine", type: "Routine" } as const,
        key: SearchVersionPageTabOption.RoutineMultiStepVersion,
        titleKey: "RoutineMultiStep",
        searchType: "RoutineVersion",
        where: () => ({ isInternalWithRoot: false, routineType: RoutineType.MultiStep }),
    },
    {
        iconInfo: { name: "Action", type: "Common" } as const,
        key: SearchVersionPageTabOption.RoutineSingleStepVersion,
        titleKey: "RoutineSingleStep",
        searchType: "RoutineVersion",
        where: () => ({ isInternalWithRoot: false, routineTypes: Object.values(RoutineType).filter(type => type !== RoutineType.MultiStep) }),
    },
    {
        iconInfo: { name: "Project", type: "Common" } as const,
        key: SearchVersionPageTabOption.ProjectVersion,
        titleKey: "Project",
        searchType: "ProjectVersion",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Note", type: "Common" } as const,
        key: SearchVersionPageTabOption.NoteVersion,
        titleKey: "Note",
        searchType: "NoteVersion",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Article", type: "Common" } as const,
        key: SearchVersionPageTabOption.PromptVersion,
        titleKey: "Prompt",
        searchType: "StandardVersion",
        where: () => ({ isInternalWithRoot: false, variant: StandardType.Prompt }),
    },
    {
        iconInfo: { name: "Object", type: "Common" } as const,
        key: SearchVersionPageTabOption.DataStructureVersion,
        titleKey: "DataStructure",
        searchType: "StandardVersion",
        where: () => ({ isInternalWithRoot: false, variant: StandardType.DataStructure }),
    },
    {
        iconInfo: { name: "Api", type: "Common" } as const,
        key: SearchVersionPageTabOption.ApiVersion,
        titleKey: "Api",
        searchType: "ApiVersion",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Terminal", type: "Common" } as const,
        key: SearchVersionPageTabOption.DataConverterVersion,
        titleKey: "DataConverter",
        searchType: "CodeVersion",
        where: () => ({ codeType: CodeType.DataConvert }),
    },
    {
        iconInfo: { name: "SmartContract", type: "Common" } as const,
        key: SearchVersionPageTabOption.SmartContractVersion,
        titleKey: "SmartContract",
        searchType: "CodeVersion",
        where: () => ({ codeType: CodeType.SmartContract }),
    },
];

export type CalendarTabsInfo = {
    Key: CalendarPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const calendarTabParams: TabParamSearchableList<CalendarTabsInfo, ["Schedule"]> = [
    {
        key: CalendarPageTabOption.All,
        titleKey: "All",
        searchType: "Schedule",
        where: () => ({}),
    },
    {
        key: CalendarPageTabOption.Meeting,
        titleKey: "Meeting",
        searchType: "Schedule",
        where: () => ({ scheduleFor: ScheduleFor.Meeting }),
    },
    {
        key: CalendarPageTabOption.RunRoutine,
        titleKey: "Routine",
        searchType: "Schedule",
        where: () => ({ scheduleFor: ScheduleFor.RunRoutine }),
    },
    {
        key: CalendarPageTabOption.RunProject,
        titleKey: "Project",
        searchType: "Schedule",
        where: () => ({ scheduleFor: ScheduleFor.RunProject }),
    },
    {
        key: CalendarPageTabOption.FocusMode,
        titleKey: "FocusMode",
        searchType: "Schedule",
        where: () => ({ scheduleFor: ScheduleFor.FocusMode }),
    },
];

export type HistoryTabsInfo = {
    Key: HistoryPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const historyTabParams: TabParamSearchableList<HistoryTabsInfo, ["View", "BookmarkList", "RunProjectOrRunRoutine"]> = [
    {
        key: HistoryPageTabOption.Viewed,
        titleKey: "View",
        searchType: "View",
        where: () => ({}), // Always "Own" visibility
    },
    {
        key: HistoryPageTabOption.Bookmarked,
        titleKey: "Bookmark",
        searchType: "BookmarkList",
        where: () => ({}), // Always "Own" visibility
    },
    {
        key: HistoryPageTabOption.RunsActive,
        titleKey: "Active",
        searchType: "RunProjectOrRunRoutine",
        where: () => ({
            statuses: [RunStatus.InProgress, RunStatus.Scheduled],
            visibility: VisibilityType.Own,
        }),
    },
    {
        key: HistoryPageTabOption.RunsCompleted,
        titleKey: "Complete",
        searchType: "RunProjectOrRunRoutine",
        where: () => ({
            statuses: [RunStatus.Cancelled, RunStatus.Completed, RunStatus.Failed],
            visibility: VisibilityType.Own,
        }),
    },
];

export type FindObjectTabsInfo = {
    Key: SearchPageTabOption | CalendarPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const findObjectTabParams: TabParamSearchableList<FindObjectTabsInfo, ["Popular", "Routine", "Project", "Question", "Note", "Team", "User", "Standard", "Api", "Code", "FocusMode", "Meeting", "RunRoutine", "RunProject"]> = [
    ...searchViewTabParams,
    {
        iconInfo: { name: "FocusMode", type: "Common" } as const,
        key: CalendarPageTabOption.FocusMode,
        titleKey: "FocusMode",
        searchType: "FocusMode",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Team", type: "Common" } as const,
        key: CalendarPageTabOption.Meeting,
        titleKey: "Meeting",
        searchType: "Meeting",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Routine", type: "Routine" } as const,
        key: CalendarPageTabOption.RunRoutine,
        titleKey: "RunRoutine",
        searchType: "RunRoutine",
        where: () => ({}),
    },
    {
        iconInfo: { name: "Project", type: "Common" } as const,
        key: CalendarPageTabOption.RunProject,
        titleKey: "RunProject",
        searchType: "RunProject",
        where: () => ({}),
    },
];

export type InboxTabsInfo = {
    Key: InboxPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const inboxTabParams: TabParamSearchableList<InboxTabsInfo, ["Chat", "Notification"]> = [
    {
        key: InboxPageTabOption.Message,
        titleKey: "Message",
        searchType: "Chat",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        key: InboxPageTabOption.Notification,
        titleKey: "Notification",
        searchType: "Notification",
        where: () => ({ visibility: VisibilityType.Own }),
    },
];

export type SignUpTabsInfo = {
    Key: SignUpPageTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const signUpTabParams: TabParamBase<SignUpTabsInfo>[] = [
    {
        key: SignUpPageTabOption.SignUp,
        titleKey: "SignUp",
    },
    {
        key: SignUpPageTabOption.MoreInfo,
        titleKey: "MoreInfo",
    },
];

export type MyStuffTabsInfo = {
    Key: MyStuffPageTabOption;
    Payload: undefined;
    WhereParams: { userId: string };
}

export const myStuffTabParams: TabParamSearchableList<MyStuffTabsInfo, ["Popular", "Project", "Routine", "Schedule", "Reminder", "Note", "Question", "Team", "User", "Standard", "Api", "Code"]> = [
    {
        iconInfo: { name: "Visible", type: "Common" } as const,
        key: MyStuffPageTabOption.All,
        titleKey: "All",
        searchType: "Popular",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Project", type: "Common" } as const,
        key: MyStuffPageTabOption.Project,
        titleKey: "Project",
        searchType: "Project",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Routine", type: "Routine" } as const,
        key: MyStuffPageTabOption.Routine,
        titleKey: "Routine",
        searchType: "Routine",
        where: () => ({ isInternal: false, visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Month", type: "Common" } as const,
        key: MyStuffPageTabOption.Schedule,
        titleKey: "Schedule",
        searchType: "Schedule",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Reminder", type: "Common" } as const,
        key: MyStuffPageTabOption.Reminder,
        titleKey: "Reminder",
        searchType: "Reminder",
        where: () => ({}), // Always "Own" visibility
    },
    {
        iconInfo: { name: "Note", type: "Common" } as const,
        key: MyStuffPageTabOption.Note,
        titleKey: "Note",
        searchType: "Note",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    // {
    //     iconInfo: { name: "Help", type: "Common" } as const,
    //     key: MyStuffPageTabOption.Question,
    //     titleKey: "Question",
    //     searchType: "Question",
    //     where: () => ({ visibility: VisibilityType.Own }),
    // },
    {
        iconInfo: { name: "Team", type: "Common" } as const,
        key: MyStuffPageTabOption.Team,
        titleKey: "Team",
        searchType: "Team",
        where: ({ userId }) => ({ memberUserIds: [userId] }),
    },
    {
        iconInfo: { name: "User", type: "Common" } as const,
        key: MyStuffPageTabOption.User,
        titleKey: "Bot",
        searchType: "User",
        where: ({ userId }) => ({ visibility: VisibilityType.Own, isBot: true, excludeIds: [userId] }),
    },
    {
        iconInfo: { name: "Standard", type: "Common" } as const,
        key: MyStuffPageTabOption.Standard,
        titleKey: "Standard",
        searchType: "Standard",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Api", type: "Common" } as const,
        key: MyStuffPageTabOption.Api,
        titleKey: "Api",
        searchType: "Api",
        where: () => ({ visibility: VisibilityType.Own }),
    },
    {
        iconInfo: { name: "Terminal", type: "Common" } as const,
        key: MyStuffPageTabOption.Code,
        titleKey: "Code",
        searchType: "Code",
        where: () => ({ visibility: VisibilityType.Own }),
    },
];

export type MemberTabsInfo = {
    Key: MemberManagePageTabOption;
    Payload: undefined;
    WhereParams: { teamId: string };
}

export const memberTabParams: TabParamSearchableList<MemberTabsInfo, ["Member", "MemberInvite", "User"]> = [
    {
        key: MemberManagePageTabOption.Members,
        titleKey: "Member",
        searchType: "Member",
        where: ({ teamId }) => ({ teamId }),
    },
    {
        key: MemberManagePageTabOption.Invites,
        titleKey: "Invite",
        searchType: "MemberInvite",
        where: ({ teamId }) => ({ teamId, statuses: [MemberInviteStatus.Pending, MemberInviteStatus.Declined] }),
    },
    {
        key: MemberManagePageTabOption.NonMembers,
        titleKey: "NonMembers",
        searchType: "User",
        where: ({ teamId }) => ({ notMemberInTeamId: teamId, notInvitedToTeamId: teamId }),
    },
];

export type TeamTabsInfo = {
    Key: TeamPageTabOption;
    Payload: undefined;
    WhereParams: {
        teamId: string;
        permissions: YouInflated;
    };
}

function teamTabColor(palette: Palette) {
    return {
        active: palette.secondary.main,
        inactive: palette.background.textSecondary,
    } as const;
}

export const teamTabParams: TabParamSearchableList<TeamTabsInfo, ["Resource", "Project", "Member"]> = [
    {
        color: teamTabColor,
        key: TeamPageTabOption.Resource,
        titleKey: "Resource",
        searchType: "Resource",
        where: () => ({}),
    },
    {
        color: teamTabColor,
        key: TeamPageTabOption.Project,
        titleKey: "Project",
        searchType: "Project",
        where: ({ teamId, permissions }) => ({ ownedByTeamId: teamId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.OwnOrPublic }),
    },
    {
        color: teamTabColor,
        key: TeamPageTabOption.Member,
        titleKey: "Member",
        searchType: "Member",
        where: ({ teamId }) => ({ teamId }),
    },
];

export type UserTabsInfo = {
    Key: UserPageTabOption;
    Payload: undefined;
    WhereParams: {
        userId: string;
        permissions: YouInflated;
    };
}

function userTabColor(palette: Palette) {
    return {
        active: palette.secondary.main,
        inactive: palette.background.textSecondary,
    } as const;
}

export const userTabParams: TabParamSearchableList<UserTabsInfo, ["User", "Project", "Team"]> = [
    {
        color: userTabColor,
        key: UserPageTabOption.Details,
        titleKey: "Details",
        searchType: "User", // Ignored
        where: () => ({}),
    },
    {
        color: userTabColor,
        key: UserPageTabOption.Project,
        titleKey: "Project",
        searchPlaceholderKey: "SearchProject",
        searchType: "Project",
        where: ({ userId, permissions }) => ({ ownedByUserId: userId, hasCompleteVersion: !permissions.canUpdate ? true : undefined, visibility: VisibilityType.OwnOrPublic }),
    },
    {
        color: userTabColor,
        key: UserPageTabOption.Team,
        titleKey: "Team",
        searchPlaceholderKey: "SearchTeam",
        searchType: "Team",
        where: ({ userId }) => ({ memberUserIds: [userId], visibility: VisibilityType.OwnOrPublic }),
    },
];

export type ParticipantTabsInfo = {
    Key: ParticipantManagePageTabOption;
    Payload: undefined;
    WhereParams: { chatId: string };
}

export const participantTabParams: TabParamSearchableList<ParticipantTabsInfo, ["ChatParticipant", "ChatInvite", "User"]> = [
    {
        key: ParticipantManagePageTabOption.ChatParticipant,
        titleKey: "Participant",
        searchType: "ChatParticipant",
        where: ({ chatId }) => ({ chatId }),
    },
    {
        key: ParticipantManagePageTabOption.ChatInvite,
        titleKey: "Invite",
        searchType: "ChatInvite",
        where: ({ chatId }) => ({ chatId, statuses: [ChatInviteStatus.Pending, ChatInviteStatus.Declined] }),
    },
    {
        iconInfo: { name: "Add", type: "Common" } as const,
        key: ParticipantManagePageTabOption.Add,
        titleKey: "SearchUser",
        searchType: "User",
        where: ({ chatId }) => ({ notInChatId: chatId }),
    },
];

export type PolicyTabsInfo = {
    Key: PolicyTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

export const policyTabParams: TabParamBase<PolicyTabsInfo>[] = [
    {
        href: LINKS.Privacy,
        key: PolicyTabOption.Privacy,
        titleKey: "Privacy",
    }, {
        href: LINKS.Terms,
        key: PolicyTabOption.Terms,
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
    RunRoutineIO: runRoutineIOSearchParams,
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
