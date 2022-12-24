import { ApiSortBy, ApiVersionSortBy, CommentSortBy, InputType, IssueSortBy, LabelSortBy, MeetingInviteSortBy, MeetingSortBy, MemberInviteSortBy, MemberSortBy, NoteSortBy, NoteVersionSortBy, NotificationSortBy, NotificationSubscriptionSortBy, OrganizationSortBy, PostSortBy, ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, ProjectSortBy, ProjectVersionSortBy, PullRequestSortBy, QuestionAnswerSortBy, QuestionSortBy, QuizAttemptSortBy, QuizQuestionResponseSortBy, QuizQuestionSortBy, QuizSortBy, ReminderListSortBy, ReminderSortBy, ReportResponseSortBy, ReportSortBy, ReputationHistorySortBy, ResourceListSortBy, ResourceSortBy, RoleSortBy, RoutineSortBy, RoutineVersionSortBy, RunProjectScheduleSortBy, RunProjectSortBy, RunRoutineInputSortBy, RunRoutineScheduleSortBy, RunRoutineSortBy, RunStatus, Session, SmartContractSortBy, SmartContractVersionSortBy, StandardSortBy, StandardVersionSortBy, StarSortBy, StatsApiSortBy, StatsOrganizationSortBy, StatsProjectSortBy, StatsQuizSortBy, StatsRoutineSortBy, StatsSiteSortBy, StatsSmartContractSortBy, StatsStandardSortBy, StatsUserSortBy, Tag, TagSortBy, TransferSortBy, UserScheduleSortBy, UserSortBy, ViewSortBy, VoteSortBy } from '@shared/consts';
import { FormSchema } from 'forms/types';
import { DocumentNode } from 'graphql';
import { getLocalStorageKeys } from 'utils/localStorage';
import { PubSub } from 'utils/pubsub';
import { SnackSeverity } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { apiEndpoint, apiVersionEndpoint, commentEndpoint, issueEndpoint, labelEndpoint, meetingEndpoint, meetingInviteEndpoint, memberEndpoint, memberInviteEndpoint, noteEndpoint, noteVersionEndpoint, notificationEndpoint, notificationSubscriptionEndpoint, organizationEndpoint, postEndpoint, projectEndpoint, projectVersionEndpoint, pullRequestEndpoint, questionAnswerEndpoint, questionEndpoint, quizAttemptEndpoint, quizEndpoint, quizQuestionEndpoint, quizQuestionResponseEndpoint, reminderEndpoint, reminderListEndpoint, reportEndpoint, reportResponseEndpoint, reputationHistoryEndpoint, resourceEndpoint, resourceListEndpoint, roleEndpoint, routineEndpoint, routineVersionEndpoint, runProjectEndpoint, runProjectScheduleEndpoint, runRoutineEndpoint, runRoutineInputEndpoint, runRoutineScheduleEndpoint, smartContractEndpoint, smartContractVersionEndpoint, standardEndpoint, standardVersionEndpoint, starEndpoint, statsApiEndpoint, statsOrganizationEndpoint, statsProjectEndpoint, statsQuizEndpoint, statsRoutineEndpoint, statsSiteEndpoint, statsSmartContractEndpoint, statsStandardEndpoint, statsUserEndpoint, tagEndpoint, transferEndpoint, unionEndpoint, userEndpoint, userScheduleEndpoint, viewEndpoint, voteEndpoint } from 'graphql/endpoints';

const starsDescription = `Stars are a way to bookmark an object. They don't affect the ranking of an object in default searches, but are still useful to get a feel for how popular an object is.`;
const votesDescription = `Votes are a way to show support for an object, which affect the ranking of an object in default searches.`;
const languagesDescription = `Filter results by the language(s) they are written in.`;
const tagsDescription = `Filter results by the tags they are associated with.`;
const simplicityDescription = `Simplicity is a mathematical measure of the shortest path to complete a routine. 

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;
const complexityDescription = `Complexity is a mathematical measure of the longest path to complete a routine.

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;

export const apiSearchSchema: FormSchema = {

}

export const apiVersionSearchSchema: FormSchema = {
}

export const commentSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Comments",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            title: "Votes",
            description: votesDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
    ],
    fields: [
        {
            fieldName: "minVotes",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxVotes",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
    ]
}

export const issueSearchSchema: FormSchema = {
}

export const labelSearchSchema: FormSchema = {
}

export const meetingSearchSchema: FormSchema = {
}

export const meetingInviteSearchSchema: FormSchema = {
}

export const memberSearchSchema: FormSchema = {
}

export const memberInviteSearchSchema: FormSchema = {
}

export const noteSearchSchema: FormSchema = {
}

export const noteVersionSearchSchema: FormSchema = {
}

export const notificationSearchSchema: FormSchema = {
}

export const notificationSubscriptionSearchSchema: FormSchema = {
}

export const organizationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Organizations",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const postSearchSchema: FormSchema = {
}

export const projectOrOrganizationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Projects/Organizations",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "objectType",
            label: "Object Type",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Project", value: 'Project' },
                    { label: "Organization", value: 'Organization' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const projectOrRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Projects/Routines",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Votes",
            description: votesDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Simplicity (Routines only)",
            description: simplicityDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Complexity (Routines only)",
            description: complexityDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "objectType",
            label: "Object Type",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Project", value: 'Project' },
                    { label: "Routine", value: 'Routine' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxStars",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxScore",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minSimplicity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxSimplicity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minComplexity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxComplexity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const projectSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Projects",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Votes",
            description: votesDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const projectVersionSearchSchema: FormSchema = {
}

export const pullRequestSearchSchema: FormSchema = {
}

export const questionSearchSchema: FormSchema = {
}

export const questionAnswerSearchSchema: FormSchema = {
}

export const quizSearchSchema: FormSchema = {
}

export const quizAttemptSearchSchema: FormSchema = {
}

export const quizQuestionSearchSchema: FormSchema = {
}

export const quizQuestionResponseSearchSchema: FormSchema = {
}

export const reminderSearchSchema: FormSchema = {
}

export const reminderListSearchSchema: FormSchema = {
}

export const reportSearchSchema: FormSchema = {
}

export const reportResponseSearchSchema: FormSchema = {
}

export const reputationHistorySearchSchema: FormSchema = {
}

export const resourceSearchSchema: FormSchema = {
}

export const resourceListSearchSchema: FormSchema = {
}

export const roleSearchSchema: FormSchema = {
}

export const routineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Routines",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Votes",
            description: votesDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Simplicity",
            description: simplicityDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Complexity",
            description: complexityDescription,
            totalItems: 2,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "isComplete",
            label: "Is Complete?",
            type: InputType.Radio,
            props: {
                defaultValue: 'undefined',
                row: true,
                options: [
                    { label: "Yes", value: 'true' },
                    { label: "No", value: 'false' },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxStars",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxScore",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minSimplicity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxSimplicity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minComplexity",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "maxComplexity",
            label: "Max",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const routineVersionSearchSchema: FormSchema = {
}

export const runProjectSearchSchema: FormSchema = {
}

export const runProjectScheduleSearchSchema: FormSchema = {
}

export const runRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Routine Runs",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            totalItems: 1,
        },
    ],
    fields: [
        {
            fieldName: "status",
            label: "Status",
            type: InputType.Radio,
            props: {
                defaultValue: RunStatus.InProgress,
                row: true,
                options: [
                    { label: "In Progress", value: RunStatus.InProgress },
                    { label: "Completed", value: RunStatus.Completed },
                    { label: "Scheduled", value: RunStatus.Scheduled },
                    { label: "Failed", value: RunStatus.Failed },
                    { label: "Cancelled", value: RunStatus.Cancelled },
                    { label: "Don't Care", value: 'undefined' },
                ]
            }
        },
    ]
}

export const runRoutineInputSearchSchema: FormSchema = {
}

export const runRoutineScheduleSearchSchema: FormSchema = {
}

export const smartContractSearchSchema: FormSchema = {
}

export const smartContractVersionSearchSchema: FormSchema = {
}

export const standardSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Standards",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Votes",
            description: votesDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
        {
            title: "Tags",
            description: tagsDescription,
            totalItems: 1
        }
    ],
    fields: [
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "minScore",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
        {
            fieldName: "tags",
            label: "Tags",
            type: InputType.TagSelector,
            props: {}
        },
    ]
}

export const standardVersionSearchSchema: FormSchema = {
}

export const starSearchSchema: FormSchema = {
}

export const statsApiSearchSchema: FormSchema = {
}

export const statsOrganizationSearchSchema: FormSchema = {
}

export const statsProjectSearchSchema: FormSchema = {
}

export const statsQuizSearchSchema: FormSchema = {
}

export const statsRoutineSearchSchema: FormSchema = {
}

export const statsSiteSearchSchema: FormSchema = {
}

export const statsSmartContractSearchSchema: FormSchema = {
}

export const statsStandardSearchSchema: FormSchema = {
}

export const statsUserSearchSchema: FormSchema = {
}

export const tagSearchSchema: FormSchema = {
}

export const transferSearchSchema: FormSchema = {
}

export const userScheduleSearchSchema: FormSchema = {
}

export const userSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Users",
        direction: "column",
        spacing: 4,
    },
    containers: [
        {
            title: "Stars",
            description: starsDescription,
            totalItems: 1,
            spacing: 2,
        },
        {
            title: "Languages",
            description: languagesDescription,
            totalItems: 1
        },
    ],
    fields: [
        {
            fieldName: "minStars",
            label: "Min",
            type: InputType.QuantityBox,
            props: {
                min: 0,
                defaultValue: 0,
            }
        },
        {
            fieldName: "languages",
            label: "Languages",
            type: InputType.LanguageInput,
            props: {},
        },
    ]
}

export const viewSearchSchema: FormSchema = {
}

export const voteSearchSchema: FormSchema = {
}

export enum SearchType {
    Api = 'Api',
    ApiVersion = 'ApiVersion',
    Comment = 'Comment',
    Issue = 'Issue',
    Label = 'Label',
    MeetingInvite = 'MeetingInvite',
    Meeting = 'Meeting',
    MemberInvite = 'MemberInvite',
    Member = 'Member',
    Note = 'Note',
    NoteVersion = 'NoteVersion',
    Notification = 'Notification',
    NotificationSubscription = 'NotificationSubscription',
    Organization = 'Organization',
    Post = 'Post',
    ProjectOrOrganization = 'ProjectOrOrganization',
    ProjectOrRoutine = 'ProjectOrRoutine',
    Project = 'Project',
    ProjectVersion = 'ProjectVersion',
    PullRequest = 'PullRequest',
    Question = 'Question',
    QuestionAnswer = 'QuestionAnswer',
    Quiz = 'Quiz',
    QuizAttempt = 'QuizAttempt',
    QuizQuestion = 'QuizQuestion',
    QuizQuestionResponse = 'QuizQuestionResponse',
    ReminderList = 'ReminderList',
    Reminder = 'Reminder',
    ReportResponse = 'ReportResponse',
    Report = 'Report',
    ReputationHistory = 'ReputationHistory',
    ResourceList = 'ResourceList',
    Resource = 'Resource',
    Role = 'Role',
    Routine = 'Routine',
    RoutineVersion = 'RoutineVersion',
    RunProject = 'RunProject',
    RunProjectSchedule = 'RunProjectSchedule',
    RunRoutine = 'RunRoutine',
    RunRoutineSchedule = 'RunRoutineSchedule',
    RunRoutineInput = 'RunRoutineInput',
    SmartContract = 'SmartContract',
    SmartContractVersion = 'SmartContractVersion',
    Standard = 'Standard',
    StandardVersion = 'StandardVersion',
    Star = 'Star',
    StatsApi = 'StatsApi',
    StatsOrganization = 'StatsOrganization',
    StatsProject = 'StatsProject',
    StatsQuiz = 'StatsQuiz',
    StatsRoutine = 'StatsRoutine',
    StatsSite = 'StatsSite',
    StatsSmartContract = 'StatsSmartContract',
    StatsStandard = 'StatsStandard',
    StatsUser = 'StatsUser',
    Tag = 'Tag',
    Transfer = 'Transfer',
    User = 'User',
    UserSchedule = 'UserSchedule',
    View = 'View',
    Vote = 'Vote',
}

export enum DevelopSearchPageTabOption {
    InProgress = 'InProgress',
    Recent = 'Recent',
    Completed = 'Completed',
}

export enum HistorySearchPageTabOption {
    Runs = 'Runs',
    Viewed = 'Viewed',
    Starred = 'Starred',
}

export enum SearchPageTabOption {
    Organizations = 'Organizations',
    Projects = 'Projects',
    Routines = 'Routines',
    Standards = 'Standards',
    Users = 'Users',
}

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    sortByOptions: any;
    query: DocumentNode;
}

/**
 * Maps search types to values needed to query and display results
 */
export const searchTypeToParams: { [key in SearchType]: SearchParams } = {
    [SearchType.Api]: {
        advancedSearchSchema: apiSearchSchema,
        defaultSortBy: ApiSortBy.ScoreDesc,
        sortByOptions: ApiSortBy,
        query: apiEndpoint.findMany,
    },
    [SearchType.ApiVersion]: {
        advancedSearchSchema: apiVersionSearchSchema,
        defaultSortBy: ApiVersionSortBy.DateCreatedDesc,
        sortByOptions: ApiVersionSortBy,
        query: apiVersionEndpoint.findMany,
    },
    [SearchType.Comment]: {
        advancedSearchSchema: commentSearchSchema,
        defaultSortBy: CommentSortBy.ScoreDesc,
        sortByOptions: CommentSortBy,
        query: commentEndpoint.findMany,
    },
    [SearchType.Issue]: {
        advancedSearchSchema: issueSearchSchema,
        defaultSortBy: IssueSortBy.ScoreDesc,
        sortByOptions: IssueSortBy,
        query: issueEndpoint.findMany,
    },
    [SearchType.Label]: {
        advancedSearchSchema: labelSearchSchema,
        defaultSortBy: LabelSortBy.DateCreatedDesc,
        sortByOptions: LabelSortBy,
        query: labelEndpoint.findMany,
    },
    [SearchType.Meeting]: {
        advancedSearchSchema: meetingSearchSchema,
        defaultSortBy: MeetingSortBy.EventStartDesc,
        sortByOptions: MeetingSortBy,
        query: meetingEndpoint.findMany,
    },
    [SearchType.MeetingInvite]: {
        advancedSearchSchema: meetingInviteSearchSchema,
        defaultSortBy: MeetingInviteSortBy.DateCreatedDesc,
        sortByOptions: MeetingInviteSortBy,
        query: meetingInviteEndpoint.findMany,
    },
    [SearchType.Member]: {
        advancedSearchSchema: memberSearchSchema,
        defaultSortBy: MemberSortBy.DateCreatedDesc,
        sortByOptions: MemberSortBy,
        query: memberEndpoint.findMany,
    },
    [SearchType.MemberInvite]: {
        advancedSearchSchema: memberInviteSearchSchema,
        defaultSortBy: MemberInviteSortBy.DateCreatedDesc,
        sortByOptions: MemberInviteSortBy,
        query: memberInviteEndpoint.findMany,
    },
    [SearchType.Note]: {
        advancedSearchSchema: noteSearchSchema,
        defaultSortBy: NoteSortBy.ScoreDesc,
        sortByOptions: NoteSortBy,
        query: noteEndpoint.findMany,
    },
    [SearchType.NoteVersion]: {
        advancedSearchSchema: noteVersionSearchSchema,
        defaultSortBy: NoteVersionSortBy.DateCreatedDesc,
        sortByOptions: NoteVersionSortBy,
        query: noteVersionEndpoint.findMany,
    },
    [SearchType.Notification]: {
        advancedSearchSchema: notificationSearchSchema,
        defaultSortBy: NotificationSortBy.DateCreatedDesc,
        sortByOptions: NotificationSortBy,
        query: notificationEndpoint.findMany,
    },
    [SearchType.NotificationSubscription]: {
        advancedSearchSchema: notificationSubscriptionSearchSchema,
        defaultSortBy: NotificationSubscriptionSortBy.DateCreatedDesc,
        sortByOptions: NotificationSubscriptionSortBy,
        query: notificationSubscriptionEndpoint.findMany,
    },
    [SearchType.Organization]: {
        advancedSearchSchema: organizationSearchSchema,
        defaultSortBy: OrganizationSortBy.StarsDesc,
        sortByOptions: OrganizationSortBy,
        query: organizationEndpoint.findMany,
    },
    [SearchType.Post]: {
        advancedSearchSchema: postSearchSchema,
        defaultSortBy: PostSortBy.DateCreatedDesc,
        sortByOptions: PostSortBy,
        query: postEndpoint.findMany,
    },
    [SearchType.Project]: {
        advancedSearchSchema: projectSearchSchema,
        defaultSortBy: ProjectSortBy.ScoreDesc,
        sortByOptions: ProjectSortBy,
        query: projectEndpoint.findMany,
    },
    [SearchType.ProjectVersion]: {
        advancedSearchSchema: projectVersionSearchSchema,
        defaultSortBy: ProjectVersionSortBy.DateCreatedDesc,
        sortByOptions: ProjectVersionSortBy,
        query: projectVersionEndpoint.findMany,
    },
    [SearchType.ProjectOrOrganization]: {
        advancedSearchSchema: projectOrOrganizationSearchSchema,
        defaultSortBy: ProjectOrOrganizationSortBy.StarsDesc,
        sortByOptions: ProjectOrOrganizationSortBy,
        query: unionEndpoint.projectOrOrganizations,
    },
    [SearchType.ProjectOrRoutine]: {
        advancedSearchSchema: projectOrRoutineSearchSchema,
        defaultSortBy: ProjectOrRoutineSortBy.StarsDesc,
        sortByOptions: ProjectOrRoutineSortBy,
        query: unionEndpoint.projectOrRoutines,
    },
    [SearchType.PullRequest]: {
        advancedSearchSchema: pullRequestSearchSchema,
        defaultSortBy: PullRequestSortBy.DateCreatedDesc,
        sortByOptions: PullRequestSortBy,
        query: pullRequestEndpoint.findMany,
    },
    [SearchType.Question]: {
        advancedSearchSchema: questionSearchSchema,
        defaultSortBy: QuestionSortBy.ScoreDesc,
        sortByOptions: QuestionSortBy,
        query: questionEndpoint.findMany,
    },
    [SearchType.QuestionAnswer]: {
        advancedSearchSchema: questionAnswerSearchSchema,
        defaultSortBy: QuestionAnswerSortBy.ScoreDesc,
        sortByOptions: QuestionAnswerSortBy,
        query: questionAnswerEndpoint.findMany,
    },
    [SearchType.Quiz]: {
        advancedSearchSchema: quizSearchSchema,
        defaultSortBy: QuizSortBy.StarsDesc,
        sortByOptions: QuizSortBy,
        query: quizEndpoint.findMany,
    },
    [SearchType.QuizQuestion]: {
        advancedSearchSchema: quizQuestionSearchSchema,
        defaultSortBy: QuizQuestionSortBy.OrderAsc,
        sortByOptions: QuizQuestionSortBy,
        query: quizQuestionEndpoint.findMany,
    },
    [SearchType.QuizQuestionResponse]: {
        advancedSearchSchema: quizQuestionResponseSearchSchema,
        defaultSortBy: QuizQuestionResponseSortBy.DateCreatedDesc,
        sortByOptions: QuizQuestionResponseSortBy,
        query: quizQuestionResponseEndpoint.findMany,
    },
    [SearchType.QuizAttempt]: {
        advancedSearchSchema: quizAttemptSearchSchema,
        defaultSortBy: QuizAttemptSortBy.DateCreatedDesc,
        sortByOptions: QuizAttemptSortBy,
        query: quizAttemptEndpoint.findMany,
    },
    [SearchType.Reminder]: {
        advancedSearchSchema: reminderSearchSchema,
        defaultSortBy: ReminderSortBy.DueDateAsc,
        sortByOptions: ReminderSortBy,
        query: reminderEndpoint.findMany,
    },
    [SearchType.ReminderList]: {
        advancedSearchSchema: reminderListSearchSchema,
        defaultSortBy: ReminderListSortBy.DateCreatedDesc,
        sortByOptions: ReminderListSortBy,
        query: reminderListEndpoint.findMany,
    },
    [SearchType.Report]: {
        advancedSearchSchema: reportSearchSchema,
        defaultSortBy: ReportSortBy.DateCreatedDesc,
        sortByOptions: ReportSortBy,
        query: reportEndpoint.findMany,
    },
    [SearchType.ReportResponse]: {
        advancedSearchSchema: reportResponseSearchSchema,
        defaultSortBy: ReportResponseSortBy.DateCreatedDesc,
        sortByOptions: ReportResponseSortBy,
        query: reportResponseEndpoint.findMany,
    },
    [SearchType.ReputationHistory]: {
        advancedSearchSchema: reputationHistorySearchSchema,
        defaultSortBy: ReputationHistorySortBy.DateCreatedDesc,
        sortByOptions: ReputationHistorySortBy,
        query: reputationHistoryEndpoint.findMany,
    },
    [SearchType.Resource]: {
        advancedSearchSchema: resourceSearchSchema,
        defaultSortBy: ResourceSortBy.DateCreatedDesc,
        sortByOptions: ResourceSortBy,
        query: resourceEndpoint.findMany,
    },
    [SearchType.ResourceList]: {
        advancedSearchSchema: resourceListSearchSchema,
        defaultSortBy: ResourceListSortBy.DateCreatedDesc,
        sortByOptions: ResourceListSortBy,
        query: resourceListEndpoint.findMany,
    },
    [SearchType.Role]: {
        advancedSearchSchema: roleSearchSchema,
        defaultSortBy: RoleSortBy.DateCreatedDesc,
        sortByOptions: RoleSortBy,
        query: roleEndpoint.findMany,
    },
    [SearchType.Routine]: {
        advancedSearchSchema: routineSearchSchema,
        defaultSortBy: RoutineSortBy.ScoreDesc,
        sortByOptions: RoutineSortBy,
        query: routineEndpoint.findMany,
    },
    [SearchType.RoutineVersion]: {
        advancedSearchSchema: routineVersionSearchSchema,
        defaultSortBy: RoutineVersionSortBy.DateCreatedDesc,
        sortByOptions: RoutineVersionSortBy,
        query: routineVersionEndpoint.findMany,
    },
    [SearchType.RunProject]: {
        advancedSearchSchema: runProjectSearchSchema,
        defaultSortBy: RunProjectSortBy.DateStartedAsc,
        sortByOptions: RunProjectSortBy,
        query: runProjectEndpoint.findMany,
    },
    [SearchType.RunProjectSchedule]: {
        advancedSearchSchema: runProjectScheduleSearchSchema,
        defaultSortBy: RunProjectScheduleSortBy.WindowStartAsc,
        sortByOptions: RunProjectScheduleSortBy,
        query: runProjectScheduleEndpoint.findMany,
    },
    [SearchType.RunRoutine]: {
        advancedSearchSchema: runRoutineSearchSchema,
        defaultSortBy: RunRoutineSortBy.DateStartedAsc,
        sortByOptions: RunRoutineSortBy,
        query: runRoutineEndpoint.findMany,
    },
    [SearchType.RunRoutineInput]: {
        advancedSearchSchema: runRoutineInputSearchSchema,
        defaultSortBy: RunRoutineInputSortBy.DateCreatedDesc,
        sortByOptions: RunRoutineInputSortBy,
        query: runRoutineInputEndpoint.findMany,
    },
    [SearchType.RunRoutineSchedule]: {
        advancedSearchSchema: runRoutineScheduleSearchSchema,
        defaultSortBy: RunRoutineScheduleSortBy.WindowStartAsc,
        sortByOptions: RunRoutineScheduleSortBy,
        query: runRoutineScheduleEndpoint.findMany,
    },
    [SearchType.SmartContract]: {
        advancedSearchSchema: smartContractSearchSchema,
        defaultSortBy: SmartContractSortBy.ScoreDesc,
        sortByOptions: SmartContractSortBy,
        query: smartContractEndpoint.findMany,
    },
    [SearchType.SmartContractVersion]: {
        advancedSearchSchema: smartContractVersionSearchSchema,
        defaultSortBy: SmartContractVersionSortBy.DateCreatedDesc,
        sortByOptions: SmartContractVersionSortBy,
        query: smartContractVersionEndpoint.findMany,
    },
    [SearchType.Standard]: {
        advancedSearchSchema: standardSearchSchema,
        defaultSortBy: StandardSortBy.ScoreDesc,
        sortByOptions: StandardSortBy,
        query: standardEndpoint.findMany,
    },
    [SearchType.StandardVersion]: {
        advancedSearchSchema: standardVersionSearchSchema,
        defaultSortBy: StandardVersionSortBy.DateCreatedDesc,
        sortByOptions: StandardVersionSortBy,
        query: standardVersionEndpoint.findMany,
    },
    [SearchType.Star]: {
        advancedSearchSchema: null,
        defaultSortBy: StarSortBy.DateUpdatedDesc,
        sortByOptions: StarSortBy,
        query: starEndpoint.stars,
    },
    [SearchType.StatsApi]: {
        advancedSearchSchema: statsApiSearchSchema,
        defaultSortBy: StatsApiSortBy.DateUpdatedDesc,
        sortByOptions: StatsApiSortBy,
        query: statsApiEndpoint.findMany,
    },
    [SearchType.StatsOrganization]: {
        advancedSearchSchema: statsOrganizationSearchSchema,
        defaultSortBy: StatsOrganizationSortBy.DateUpdatedDesc,
        sortByOptions: StatsOrganizationSortBy,
        query: statsOrganizationEndpoint.findMany,
    },
    [SearchType.StatsProject]: {
        advancedSearchSchema: statsProjectSearchSchema,
        defaultSortBy: StatsProjectSortBy.DateUpdatedDesc,
        sortByOptions: StatsProjectSortBy,
        query: statsProjectEndpoint.findMany,
    },
    [SearchType.StatsQuiz]: {
        advancedSearchSchema: statsQuizSearchSchema,
        defaultSortBy: StatsQuizSortBy.DateUpdatedDesc,
        sortByOptions: StatsQuizSortBy,
        query: statsQuizEndpoint.findMany,
    },
    [SearchType.StatsRoutine]: {
        advancedSearchSchema: statsRoutineSearchSchema,
        defaultSortBy: StatsRoutineSortBy.DateUpdatedDesc,
        sortByOptions: StatsRoutineSortBy,
        query: statsRoutineEndpoint.findMany,
    },
    [SearchType.StatsSite]: {
        advancedSearchSchema: statsSiteSearchSchema,
        defaultSortBy: StatsSiteSortBy.DateUpdatedDesc,
        sortByOptions: StatsSiteSortBy,
        query: statsSiteEndpoint.findMany,
    },
    [SearchType.StatsSmartContract]: {
        advancedSearchSchema: statsSmartContractSearchSchema,
        defaultSortBy: StatsSmartContractSortBy.DateUpdatedDesc,
        sortByOptions: StatsSmartContractSortBy,
        query: statsSmartContractEndpoint.findMany,
    },
    [SearchType.StatsStandard]: {
        advancedSearchSchema: statsStandardSearchSchema,
        defaultSortBy: StatsStandardSortBy.DateUpdatedDesc,
        sortByOptions: StatsStandardSortBy,
        query: statsStandardEndpoint.findMany,
    },
    [SearchType.StatsUser]: {
        advancedSearchSchema: statsUserSearchSchema,
        defaultSortBy: StatsUserSortBy.DateUpdatedDesc,
        sortByOptions: StatsUserSortBy,
        query: statsUserEndpoint.findMany,
    },
    [SearchType.Tag]: {
        advancedSearchSchema: tagSearchSchema,
        defaultSortBy: TagSortBy.StarsDesc,
        sortByOptions: TagSortBy,
        query: tagEndpoint.findMany,
    },
    [SearchType.Transfer]: {
        advancedSearchSchema: transferSearchSchema,
        defaultSortBy: TransferSortBy.DateCreatedDesc,
        sortByOptions: TransferSortBy,
        query: transferEndpoint.findMany,
    },
    [SearchType.UserSchedule]: {
        advancedSearchSchema: userScheduleSearchSchema,
        defaultSortBy: UserScheduleSortBy.EventStartAsc,
        sortByOptions: UserScheduleSortBy,
        query: userScheduleEndpoint.findMany,
    },
    [SearchType.User]: {
        advancedSearchSchema: userSearchSchema,
        defaultSortBy: UserSortBy.StarsDesc,
        sortByOptions: UserSortBy,
        query: userEndpoint.findMany,
    },
    [SearchType.View]: {
        advancedSearchSchema: null,
        defaultSortBy: ViewSortBy.LastViewedDesc,
        sortByOptions: ViewSortBy,
        query: viewEndpoint.views,
    },
    [SearchType.Vote]: {
        advancedSearchSchema: voteSearchSchema,
        defaultSortBy: VoteSortBy.DateUpdatedDesc,
        sortByOptions: VoteSortBy,
        query: voteEndpoint.votes,
    },
};

/**
 * Converts a radio button value (which can only be a string) to its underlying search URL value
 * @param value The value of the radio button
 * @returns The value as-is, a boolean, or undefined
 */
const radioValueToSearch = (value: string): string | boolean | undefined => {
    // Check if value should be a boolean
    if (value === 'true' || value === 'false') return value === 'true';
    // Check if value should be undefined
    if (value === 'undefined') return undefined;
    // Otherwise, return as-is
    return value;
};

/**
 * Converts a radio button search value to its radio button value (i.e. stringifies)
 * @param value The value of the radio button
 * @returns The value stringified
 */
const searchToRadioValue = (value: string | boolean | undefined): string => value + '';

/**
 * Converts an array of items to a search URL value
 * @param arr The array of items
 * @param convert The function to convert each item to a search URL value
 * @returns The items of the array converted, or undefined if the array is empty or undefined
 */
function arrayConvert<T, U>(arr: T[] | undefined, convert?: (T) => U): U[] | undefined {
    if (Array.isArray(arr)) {
        if (arr.length === 0) return undefined;
        return convert ? arr.map(convert) : arr as unknown as U[];
    }
    return undefined;
};

/**
 * Map for converting input type values to search URL values
 */
const inputTypeToSearch: { [key in InputType]: (value: any) => any } = {
    [InputType.Checkbox]: (value) => value, //TODO
    [InputType.Dropzone]: (value) => value, //TODO
    [InputType.JSON]: (value) => value, //TODO
    [InputType.LanguageInput]: (value: string[]) => arrayConvert(value),
    [InputType.Markdown]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.Radio]: (value: string) => radioValueToSearch(value),
    [InputType.Selector]: (value) => value, //TODO
    [InputType.Slider]: (value) => value, //TODO
    [InputType.Switch]: (value) => value, //TODO 
    [InputType.TagSelector]: (value: Tag[]) => arrayConvert(value, ({ tag }) => tag),
    [InputType.TextField]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.QuantityBox]: (value) => value, //TODO
}

/**
 * Map for converting search URL values to input type values
 */
const searchToInputType: { [key in InputType]: (value: any) => any } = {
    [InputType.Checkbox]: (value) => value, //TODO
    [InputType.Dropzone]: (value) => value, //TODO
    [InputType.JSON]: (value) => value, //TODO
    [InputType.LanguageInput]: (value: string[]) => arrayConvert(value),
    [InputType.Markdown]: (value: string) => value,
    [InputType.Radio]: (value: string) => searchToRadioValue(value),
    [InputType.Selector]: (value) => value, //TODO
    [InputType.Slider]: (value) => value, //TODO
    [InputType.Switch]: (value) => value, //TODO
    [InputType.TagSelector]: (value: string[]) => arrayConvert(value, (tag) => ({ tag })),
    [InputType.TextField]: (value: string) => value,
    [InputType.QuantityBox]: (value) => value, //TODO
}

/**
 * Converts formik values to search URL values 
 * @param values The formik values
 * @param schema The form schema to use
 * @returns Values converted for stringifySearchParams
 */
export const convertFormikForSearch = (values: { [x: string]: any }, schema: FormSchema): { [x: string]: any } => {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through all fields in the schema
    for (const field of schema.fields) {
        // If field in values, convert and add to result
        if (values[field.fieldName]) {
            const value = inputTypeToSearch[field.type](values[field.fieldName]);
            if (value !== undefined) result[field.fieldName] = value;
        }
    }
    // Return result
    return result;
}

/**
 * Converts search URL values to formik values
 * @param values The search URL values
 * @param schema The form schema to use
 * @returns Values converted for formik
 */
export const convertSearchForFormik = (values: { [x: string]: any }, schema: FormSchema): { [x: string]: any } => {
    // Initialize result
    const result: { [x: string]: any } = {};
    // Loop through all fields in the schema
    for (const field of schema.fields) {
        // If field in values, convert and add to result
        if (values[field.fieldName]) {
            const value = searchToInputType[field.type](values[field.fieldName]);
            result[field.fieldName] = value;
        }
        // Otherwise, set as undefined
        else result[field.fieldName] = undefined;
    }
    // Return result
    return result;
}

/**
 * Clears search history from all search bars
 */
export const clearSearchHistory = (session: Session) => {
    const { id } = getCurrentUser(session);
    // Find all search history objects in localStorage
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: 'search-history-',
        suffix: id ?? '',
    });
    // Clear them
    searchHistoryKeys.forEach(key => {
        localStorage.removeItem(key);
    });
    PubSub.get().publishSnack({ messageKey: 'SearchHistoryCleared', severity: SnackSeverity.Success });
}