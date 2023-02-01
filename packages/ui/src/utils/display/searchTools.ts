import { ApiSortBy, ApiVersionSortBy, CommentSortBy, InputType, IssueSortBy, LabelSortBy, MeetingInviteSortBy, MeetingSortBy, MemberInviteSortBy, MemberSortBy, NoteSortBy, NoteVersionSortBy, NotificationSortBy, NotificationSubscriptionSortBy, OrganizationSortBy, PostSortBy, ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, ProjectSortBy, ProjectVersionSortBy, PullRequestSortBy, QuestionAnswerSortBy, QuestionSortBy, QuizAttemptSortBy, QuizQuestionResponseSortBy, QuizQuestionSortBy, QuizSortBy, ReminderListSortBy, ReminderSortBy, ReportResponseSortBy, ReportSortBy, ReputationHistorySortBy, ResourceListSortBy, ResourceSortBy, RoleSortBy, RoutineSortBy, RoutineVersionSortBy, RunProjectOrRunRoutineSortBy, RunProjectScheduleSortBy, RunProjectSortBy, RunRoutineInputSortBy, RunRoutineScheduleSortBy, RunRoutineSortBy, RunStatus, Session, SmartContractSortBy, SmartContractVersionSortBy, StandardSortBy, StandardVersionSortBy, StarSortBy, StatsApiSortBy, StatsOrganizationSortBy, StatsProjectSortBy, StatsQuizSortBy, StatsRoutineSortBy, StatsSiteSortBy, StatsSmartContractSortBy, StatsStandardSortBy, StatsUserSortBy, Tag, TagSortBy, TransferSortBy, UserScheduleSortBy, UserSortBy, ViewSortBy, VoteSortBy } from '@shared/consts';
import { FormSchema } from 'forms/types';
import { DocumentNode } from 'graphql';
import { getLocalStorageKeys } from 'utils/localStorage';
import { PubSub } from 'utils/pubsub';
import { SnackSeverity } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { apiFindMany } from 'api/generated/endpoints/api';
import { apiVersionFindMany } from 'api/generated/endpoints/apiVersion';
import { commentFindMany } from 'api/generated/endpoints/comment';
import { issueFindMany } from 'api/generated/endpoints/issue';
import { labelFindMany } from 'api/generated/endpoints/label';
import { meetingFindMany } from 'api/generated/endpoints/meeting';
import { meetingInviteFindMany } from 'api/generated/endpoints/meetingInvite';
import { memberFindMany } from 'api/generated/endpoints/member';
import { memberInviteFindMany } from 'api/generated/endpoints/memberInvite';
import { noteFindMany } from 'api/generated/endpoints/note';
import { noteVersionFindMany } from 'api/generated/endpoints/noteVersion';
import { notificationFindMany } from 'api/generated/endpoints/notification';
import { notificationSubscriptionFindMany } from 'api/generated/endpoints/notificationSubscription';
import { organizationFindMany } from 'api/generated/endpoints/organization';
import { postFindMany } from 'api/generated/endpoints/post';
import { projectFindMany } from 'api/generated/endpoints/project';
import { projectVersionFindMany } from 'api/generated/endpoints/projectVersion';
import { projectOrOrganizationFindMany } from 'api/generated/endpoints/projectOrOrganization';
import { projectOrRoutineFindMany } from 'api/generated/endpoints/projectOrRoutine';
import { pullRequestFindMany } from 'api/generated/endpoints/pullRequest';
import { questionFindMany } from 'api/generated/endpoints/question';
import { questionAnswerFindMany } from 'api/generated/endpoints/questionAnswer';
import { quizFindMany } from 'api/generated/endpoints/quiz';
import { quizQuestionFindMany } from 'api/generated/endpoints/quizQuestion';
import { quizQuestionResponseFindMany } from 'api/generated/endpoints/quizQuestionResponse';
import { quizAttemptFindMany } from 'api/generated/endpoints/quizAttempt';
import { reminderFindMany } from 'api/generated/endpoints/reminder';
import { reminderListFindMany } from 'api/generated/endpoints/reminderList';
import { reportFindMany } from 'api/generated/endpoints/report';
import { reportResponseFindMany } from 'api/generated/endpoints/reportResponse';
import { getOperationName } from '@apollo/client/utilities';
import { reputationHistoryFindMany } from 'api/generated/endpoints/reputationHistory';
import { resourceFindMany } from 'api/generated/endpoints/resource';
import { resourceListFindMany } from 'api/generated/endpoints/resourceList';
import { roleFindMany } from 'api/generated/endpoints/role';
import { routineFindMany } from 'api/generated/endpoints/routine';
import { routineVersionFindMany } from 'api/generated/endpoints/routineVersion';
import { runProjectFindMany } from 'api/generated/endpoints/runProject';
import { runProjectOrRunRoutineFindMany } from 'api/generated/endpoints/runProjectOrRunRoutine';
import { runProjectScheduleFindMany } from 'api/generated/endpoints/runProjectSchedule';
import { runRoutineFindMany } from 'api/generated/endpoints/runRoutine';
import { runRoutineInputFindMany } from 'api/generated/endpoints/runRoutineInput';
import { runRoutineScheduleFindMany } from 'api/generated/endpoints/runRoutineSchedule';
import { smartContractFindMany } from 'api/generated/endpoints/smartContract';
import { smartContractVersionFindMany } from 'api/generated/endpoints/smartContractVersion';
import { standardFindMany } from 'api/generated/endpoints/standard';
import { standardVersionFindMany } from 'api/generated/endpoints/standardVersion';
import { starFindMany } from 'api/generated/endpoints/star';
import { statsApiFindMany } from 'api/generated/endpoints/statsApi';
import { statsOrganizationFindMany } from 'api/generated/endpoints/statsOrganization';
import { voteFindMany } from 'api/generated/endpoints/vote';
import { viewFindMany } from 'api/generated/endpoints/view';
import { userFindMany } from 'api/generated/endpoints/user';
import { userScheduleFindMany } from 'api/generated/endpoints/userSchedule';
import { transferFindMany } from 'api/generated/endpoints/transfer';
import { tagFindMany } from 'api/generated/endpoints/tag';
import { statsUserFindMany } from 'api/generated/endpoints/statsUser';
import { statsStandardFindMany } from 'api/generated/endpoints/statsStandard';
import { statsSmartContractFindMany } from 'api/generated/endpoints/statsSmartContract';
import { statsSiteFindMany } from 'api/generated/endpoints/statsSite';
import { statsRoutineFindMany } from 'api/generated/endpoints/statsRoutine';
import { statsQuizFindMany } from 'api/generated/endpoints/statsQuiz';
import { statsProjectFindMany } from 'api/generated/endpoints/statsProject';

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    endpoint: string;
    sortByOptions: any;
    query: DocumentNode;
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
    RunProjectOrRunRoutine = 'RunProjectOrRunRoutine',
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

const starsDescription = `Stars are a way to bookmark an object. They don't affect the ranking of an object in default searches, but are still useful to get a feel for how popular an object is.`;
const votesDescription = `Votes are a way to show support for an object, which affect the ranking of an object in default searches.`;
const languagesDescription = `Filter results by the language(s) they are written in.`;
const tagsDescription = `Filter results by the tags they are associated with.`;
const simplicityDescription = `Simplicity is a mathematical measure of the shortest path to complete a routine. 

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;
const complexityDescription = `Complexity is a mathematical measure of the longest path to complete a routine.

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;

export const apiSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Apis",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const apiVersionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Api Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Issues",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const labelSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Labels",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const meetingSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Meetings",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const meetingInviteSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Meeting Invites",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const memberSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Members",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const memberInviteSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Member Invites",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const noteSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Notes",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const noteVersionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Note Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const notificationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Notifications",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const notificationSubscriptionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Subscriptions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Posts",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Project Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const pullRequestSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Pull Requests",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const questionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Questions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const questionAnswerSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Answers",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const quizSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Quizzes",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const quizAttemptSearchSchema: FormSchema = {
    formLayout: {
        title: "Search QuizAttempts",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const quizQuestionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Quiz Questions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const quizQuestionResponseSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Quiz Question Responses",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const reminderSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Reminders",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const reminderListSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Reminder Lists",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const reportSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Reports",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const reportResponseSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Report Responses",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const reputationHistorySearchSchema: FormSchema = {
    formLayout: {
        title: "Search Reputation History",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const resourceSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Resources",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const resourceListSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Resource Lists",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const roleSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Roles",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Routine Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const runProjectSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Runs (Projects)",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const runProjectOrRunRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Runs",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [] //TODO
}

export const runProjectScheduleSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Run Schedules (Projects)",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const runRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Runs (Routines)",
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
    formLayout: {
        title: "Search Run Inputs (Routine)",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const runRoutineScheduleSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Run Schedules (Routine)",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const smartContractSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Smart Contracts",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const smartContractVersionSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Smart Contract Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Standard Versions",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const starSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Stars",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsApiSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Api Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsOrganizationSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Organization Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsProjectSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Project Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsQuizSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Quiz Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsRoutineSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Routine Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsSiteSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Site Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsSmartContractSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Smart Contract Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsStandardSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Standard Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const statsUserSearchSchema: FormSchema = {
    formLayout: {
        title: "Search User Stats",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const tagSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Tags",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const transferSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Transfers",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const userScheduleSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Your Schedules",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
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
    formLayout: {
        title: "Search Views",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

export const voteSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Votes",
        direction: "column",
        spacing: 4,
    },
    containers: [], //TODO
    fields: [], //TODO
}

/**
 * Converts shorthand search info to SearchParams object
 */
const toParams = (
    advancedSearchSchema: FormSchema,
    query: DocumentNode,
    sortByOptions: { [key: string]: string },
    defaultSortBy: string,
): SearchParams => ({
    advancedSearchSchema,
    defaultSortBy,
    endpoint: getOperationName(query) ?? '',
    query,
    sortByOptions,
})

/**
 * Maps search types to values needed to query and display results
 */
export const searchTypeToParams: { [key in SearchType]: SearchParams } = {
    'Api': toParams(apiSearchSchema, apiFindMany, ApiSortBy, ApiSortBy.ScoreDesc),
    'ApiVersion': toParams(apiVersionSearchSchema, apiVersionFindMany, ApiVersionSortBy, ApiVersionSortBy.DateCreatedDesc),
    'Comment': toParams(commentSearchSchema, commentFindMany, CommentSortBy, CommentSortBy.ScoreDesc),
    'Issue': toParams(issueSearchSchema, issueFindMany, IssueSortBy, IssueSortBy.ScoreDesc),
    'Label': toParams(labelSearchSchema, labelFindMany, LabelSortBy, LabelSortBy.DateCreatedDesc),
    'Meeting': toParams(meetingSearchSchema, meetingFindMany, MeetingSortBy, MeetingSortBy.EventStartDesc),
    'MeetingInvite': toParams(meetingInviteSearchSchema, meetingInviteFindMany, MeetingInviteSortBy, MeetingInviteSortBy.DateCreatedDesc),
    'Member': toParams(memberSearchSchema, memberFindMany, MemberSortBy, MemberSortBy.DateCreatedDesc),
    'MemberInvite': toParams(memberInviteSearchSchema, memberInviteFindMany, MemberInviteSortBy, MemberInviteSortBy.DateCreatedDesc),
    'Note': toParams(noteSearchSchema, noteFindMany, NoteSortBy, NoteSortBy.ScoreDesc),
    'NoteVersion': toParams(noteVersionSearchSchema, noteVersionFindMany, NoteVersionSortBy, NoteVersionSortBy.DateCreatedDesc),
    'Notification': toParams(notificationSearchSchema, notificationFindMany, NotificationSortBy, NotificationSortBy.DateCreatedDesc),
    'NotificationSubscription': toParams(notificationSubscriptionSearchSchema, notificationSubscriptionFindMany, NotificationSubscriptionSortBy, NotificationSubscriptionSortBy.DateCreatedDesc),
    'Organization': toParams(organizationSearchSchema, organizationFindMany, OrganizationSortBy, OrganizationSortBy.StarsDesc),
    'Post': toParams(postSearchSchema, postFindMany, PostSortBy, PostSortBy.DateCreatedDesc),
    'Project': toParams(projectSearchSchema, projectFindMany, ProjectSortBy, ProjectSortBy.ScoreDesc),
    'ProjectVersion': toParams(projectVersionSearchSchema, projectVersionFindMany, ProjectVersionSortBy, ProjectVersionSortBy.DateCreatedDesc),
    'ProjectOrOrganization': toParams(projectOrOrganizationSearchSchema, projectOrOrganizationFindMany, ProjectOrOrganizationSortBy, ProjectOrOrganizationSortBy.StarsDesc),
    'ProjectOrRoutine': toParams(projectOrRoutineSearchSchema, projectOrRoutineFindMany, ProjectOrRoutineSortBy, ProjectOrRoutineSortBy.StarsDesc),
    'PullRequest': toParams(pullRequestSearchSchema, pullRequestFindMany, PullRequestSortBy, PullRequestSortBy.DateCreatedDesc),
    'Question': toParams(questionSearchSchema, questionFindMany, QuestionSortBy, QuestionSortBy.ScoreDesc),
    'QuestionAnswer': toParams(questionAnswerSearchSchema, questionAnswerFindMany, QuestionAnswerSortBy, QuestionAnswerSortBy.ScoreDesc),
    'Quiz': toParams(quizSearchSchema, quizFindMany, QuizSortBy, QuizSortBy.StarsDesc),
    'QuizQuestion': toParams(quizQuestionSearchSchema, quizQuestionFindMany, QuizQuestionSortBy, QuizQuestionSortBy.OrderAsc),
    'QuizQuestionResponse': toParams(quizQuestionResponseSearchSchema, quizQuestionResponseFindMany, QuizQuestionResponseSortBy, QuizQuestionResponseSortBy.DateCreatedDesc),
    'QuizAttempt': toParams(quizAttemptSearchSchema, quizAttemptFindMany, QuizAttemptSortBy, QuizAttemptSortBy.DateCreatedDesc),
    'Reminder': toParams(reminderSearchSchema, reminderFindMany, ReminderSortBy, ReminderSortBy.DueDateAsc),
    'ReminderList': toParams(reminderListSearchSchema, reminderListFindMany, ReminderListSortBy, ReminderListSortBy.DateCreatedDesc),
    'Report': toParams(reportSearchSchema, reportFindMany, ReportSortBy, ReportSortBy.DateCreatedDesc),
    'ReportResponse': toParams(reportResponseSearchSchema, reportResponseFindMany, ReportResponseSortBy, ReportResponseSortBy.DateCreatedDesc),
    'ReputationHistory': toParams(reputationHistorySearchSchema, reputationHistoryFindMany, ReputationHistorySortBy, ReputationHistorySortBy.DateCreatedDesc),
    'Resource': toParams(resourceSearchSchema, resourceFindMany, ResourceSortBy, ResourceSortBy.DateCreatedDesc),
    'ResourceList': toParams(resourceListSearchSchema, resourceListFindMany, ResourceListSortBy, ResourceListSortBy.DateCreatedDesc),
    'Role': toParams(roleSearchSchema, roleFindMany, RoleSortBy, RoleSortBy.DateCreatedDesc),
    'Routine': toParams(routineSearchSchema, routineFindMany, RoutineSortBy, RoutineSortBy.ScoreDesc),
    'RoutineVersion': toParams(routineVersionSearchSchema, routineVersionFindMany, RoutineVersionSortBy, RoutineVersionSortBy.DateCreatedDesc),
    'RunProject': toParams(runProjectSearchSchema, runProjectFindMany, RunProjectSortBy, RunProjectSortBy.DateStartedDesc),
    'RunProjectOrRunRoutine': toParams(runProjectOrRunRoutineSearchSchema, runProjectOrRunRoutineFindMany, RunProjectOrRunRoutineSortBy, RunProjectOrRunRoutineSortBy.DateStartedDesc),
    'RunProjectSchedule': toParams(runProjectScheduleSearchSchema, runProjectScheduleFindMany, RunProjectScheduleSortBy, RunProjectScheduleSortBy.WindowStartAsc),
    'RunRoutine': toParams(runRoutineSearchSchema, runRoutineFindMany, RunRoutineSortBy, RunRoutineSortBy.DateStartedAsc),
    'RunRoutineInput': toParams(runRoutineInputSearchSchema, runRoutineInputFindMany, RunRoutineInputSortBy, RunRoutineInputSortBy.DateCreatedDesc),
    'RunRoutineSchedule': toParams(runRoutineScheduleSearchSchema, runRoutineScheduleFindMany, RunRoutineScheduleSortBy, RunRoutineScheduleSortBy.WindowStartAsc),
    'SmartContract': toParams(smartContractSearchSchema, smartContractFindMany, SmartContractSortBy, SmartContractSortBy.ScoreDesc),
    'SmartContractVersion': toParams(smartContractVersionSearchSchema, smartContractVersionFindMany, SmartContractVersionSortBy, SmartContractVersionSortBy.DateCreatedDesc),
    'Standard': toParams(standardSearchSchema, standardFindMany, StandardSortBy, StandardSortBy.ScoreDesc),
    'StandardVersion': toParams(standardVersionSearchSchema, standardVersionFindMany, StandardVersionSortBy, StandardVersionSortBy.DateCreatedDesc),
    'Star': toParams(starSearchSchema, starFindMany, StarSortBy, StarSortBy.DateUpdatedDesc),
    'StatsApi': toParams(statsApiSearchSchema, statsApiFindMany, StatsApiSortBy, StatsApiSortBy.DateUpdatedDesc),
    'StatsOrganization': toParams(statsOrganizationSearchSchema, statsOrganizationFindMany, StatsOrganizationSortBy, StatsOrganizationSortBy.DateUpdatedDesc),
    'StatsProject': toParams(statsProjectSearchSchema, statsProjectFindMany, StatsProjectSortBy, StatsProjectSortBy.DateUpdatedDesc),
    'StatsQuiz': toParams(statsQuizSearchSchema, statsQuizFindMany, StatsQuizSortBy, StatsQuizSortBy.DateUpdatedDesc),
    'StatsRoutine': toParams(statsRoutineSearchSchema, statsRoutineFindMany, StatsRoutineSortBy, StatsRoutineSortBy.DateUpdatedDesc),
    'StatsSite': toParams(statsSiteSearchSchema, statsSiteFindMany, StatsSiteSortBy, StatsSiteSortBy.DateUpdatedDesc),
    'StatsSmartContract': toParams(statsSmartContractSearchSchema, statsSmartContractFindMany, StatsSmartContractSortBy, StatsSmartContractSortBy.DateUpdatedDesc),
    'StatsStandard': toParams(statsStandardSearchSchema, statsStandardFindMany, StatsStandardSortBy, StatsStandardSortBy.DateUpdatedDesc),
    'StatsUser': toParams(statsUserSearchSchema, statsUserFindMany, StatsUserSortBy, StatsUserSortBy.DateUpdatedDesc),
    'Tag': toParams(tagSearchSchema, tagFindMany, TagSortBy, TagSortBy.StarsDesc),
    'Transfer': toParams(transferSearchSchema, transferFindMany, TransferSortBy, TransferSortBy.DateCreatedDesc),
    'UserSchedule': toParams(userScheduleSearchSchema, userScheduleFindMany, UserScheduleSortBy, UserScheduleSortBy.EventStartAsc),
    'User': toParams(userSearchSchema, userFindMany, UserSortBy, UserSortBy.StarsDesc),
    'View': toParams(viewSearchSchema, viewFindMany, ViewSortBy, ViewSortBy.LastViewedDesc),
    'Vote': toParams(voteSearchSchema, voteFindMany, VoteSortBy, VoteSortBy.DateUpdatedDesc),
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