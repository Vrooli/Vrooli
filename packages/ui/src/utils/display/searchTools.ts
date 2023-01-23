import { ApiSortBy, ApiVersionSortBy, CommentSortBy, InputType, IssueSortBy, LabelSortBy, MeetingInviteSortBy, MeetingSortBy, MemberInviteSortBy, MemberSortBy, NoteSortBy, NoteVersionSortBy, NotificationSortBy, NotificationSubscriptionSortBy, OrganizationSortBy, PostSortBy, ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, ProjectSortBy, ProjectVersionSortBy, PullRequestSortBy, QuestionAnswerSortBy, QuestionSortBy, QuizAttemptSortBy, QuizQuestionResponseSortBy, QuizQuestionSortBy, QuizSortBy, ReminderListSortBy, ReminderSortBy, ReportResponseSortBy, ReportSortBy, ReputationHistorySortBy, ResourceListSortBy, ResourceSortBy, RoleSortBy, RoutineSortBy, RoutineVersionSortBy, RunProjectOrRunRoutineSortBy, RunProjectScheduleSortBy, RunProjectSortBy, RunRoutineInputSortBy, RunRoutineScheduleSortBy, RunRoutineSortBy, RunStatus, Session, SmartContractSortBy, SmartContractVersionSortBy, StandardSortBy, StandardVersionSortBy, StarSortBy, StatsApiSortBy, StatsOrganizationSortBy, StatsProjectSortBy, StatsQuizSortBy, StatsRoutineSortBy, StatsSiteSortBy, StatsSmartContractSortBy, StatsStandardSortBy, StatsUserSortBy, Tag, TagSortBy, TransferSortBy, UserScheduleSortBy, UserSortBy, ViewSortBy, VoteSortBy } from '@shared/consts';
import { FormSchema } from 'forms/types';
import { DocumentNode } from 'graphql';
import { getLocalStorageKeys } from 'utils/localStorage';
import { PubSub } from 'utils/pubsub';
import { SnackSeverity } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { apiEndpoint, apiVersionEndpoint, commentEndpoint, issueEndpoint, labelEndpoint, meetingEndpoint, meetingInviteEndpoint, memberEndpoint, memberInviteEndpoint, noteEndpoint, noteVersionEndpoint, notificationEndpoint, notificationSubscriptionEndpoint, organizationEndpoint, postEndpoint, projectEndpoint, projectVersionEndpoint, pullRequestEndpoint, questionAnswerEndpoint, questionEndpoint, quizAttemptEndpoint, quizEndpoint, quizQuestionEndpoint, quizQuestionResponseEndpoint, reminderEndpoint, reminderListEndpoint, reportEndpoint, reportResponseEndpoint, reputationHistoryEndpoint, resourceEndpoint, resourceListEndpoint, roleEndpoint, routineEndpoint, routineVersionEndpoint, runProjectEndpoint, runProjectScheduleEndpoint, runRoutineEndpoint, runRoutineInputEndpoint, runRoutineScheduleEndpoint, smartContractEndpoint, smartContractVersionEndpoint, standardEndpoint, standardVersionEndpoint, starEndpoint, statsApiEndpoint, statsOrganizationEndpoint, statsProjectEndpoint, statsQuizEndpoint, statsRoutineEndpoint, statsSiteEndpoint, statsSmartContractEndpoint, statsStandardEndpoint, statsUserEndpoint, tagEndpoint, transferEndpoint, unionEndpoint, userEndpoint, userScheduleEndpoint, viewEndpoint, voteEndpoint } from 'api/endpoints';

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

export type SearchParams = {
    advancedSearchSchema: FormSchema | null;
    defaultSortBy: any;
    endpoint: string;
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
        endpoint: apiEndpoint.findMany[1],
        query: apiEndpoint.findMany[0],
        sortByOptions: ApiSortBy,
    },
    [SearchType.ApiVersion]: {
        advancedSearchSchema: apiVersionSearchSchema,
        defaultSortBy: ApiVersionSortBy.DateCreatedDesc,
        endpoint: apiVersionEndpoint.findMany[1],
        query: apiVersionEndpoint.findMany[0],
        sortByOptions: ApiVersionSortBy,
    },
    [SearchType.Comment]: {
        advancedSearchSchema: commentSearchSchema,
        defaultSortBy: CommentSortBy.ScoreDesc,
        endpoint: commentEndpoint.findMany[1],
        query: commentEndpoint.findMany[0],
        sortByOptions: CommentSortBy,
    },
    [SearchType.Issue]: {
        advancedSearchSchema: issueSearchSchema,
        defaultSortBy: IssueSortBy.ScoreDesc,
        endpoint: issueEndpoint.findMany[1],
        query: issueEndpoint.findMany[0],
        sortByOptions: IssueSortBy,
    },
    [SearchType.Label]: {
        advancedSearchSchema: labelSearchSchema,
        defaultSortBy: LabelSortBy.DateCreatedDesc,
        endpoint: labelEndpoint.findMany[1],
        query: labelEndpoint.findMany[0],
        sortByOptions: LabelSortBy,
    },
    [SearchType.Meeting]: {
        advancedSearchSchema: meetingSearchSchema,
        defaultSortBy: MeetingSortBy.EventStartDesc,
        endpoint: meetingEndpoint.findMany[1],
        query: meetingEndpoint.findMany[0],
        sortByOptions: MeetingSortBy,
    },
    [SearchType.MeetingInvite]: {
        advancedSearchSchema: meetingInviteSearchSchema,
        defaultSortBy: MeetingInviteSortBy.DateCreatedDesc,
        endpoint: meetingInviteEndpoint.findMany[1],
        query: meetingInviteEndpoint.findMany[0],
        sortByOptions: MeetingInviteSortBy,
    },
    [SearchType.Member]: {
        advancedSearchSchema: memberSearchSchema,
        defaultSortBy: MemberSortBy.DateCreatedDesc,
        endpoint: memberEndpoint.findMany[1],
        query: memberEndpoint.findMany[0],
        sortByOptions: MemberSortBy,
    },
    [SearchType.MemberInvite]: {
        advancedSearchSchema: memberInviteSearchSchema,
        defaultSortBy: MemberInviteSortBy.DateCreatedDesc,
        endpoint: memberInviteEndpoint.findMany[1],
        query: memberInviteEndpoint.findMany[0],
        sortByOptions: MemberInviteSortBy,
    },
    [SearchType.Note]: {
        advancedSearchSchema: noteSearchSchema,
        defaultSortBy: NoteSortBy.ScoreDesc,
        endpoint: noteEndpoint.findMany[1],
        query: noteEndpoint.findMany[0],
        sortByOptions: NoteSortBy,
    },
    [SearchType.NoteVersion]: {
        advancedSearchSchema: noteVersionSearchSchema,
        defaultSortBy: NoteVersionSortBy.DateCreatedDesc,
        endpoint: noteVersionEndpoint.findMany[1],
        query: noteVersionEndpoint.findMany[0],
        sortByOptions: NoteVersionSortBy,
    },
    [SearchType.Notification]: {
        advancedSearchSchema: notificationSearchSchema,
        defaultSortBy: NotificationSortBy.DateCreatedDesc,
        endpoint: notificationEndpoint.findMany[1],
        query: notificationEndpoint.findMany[0],
        sortByOptions: NotificationSortBy,
    },
    [SearchType.NotificationSubscription]: {
        advancedSearchSchema: notificationSubscriptionSearchSchema,
        defaultSortBy: NotificationSubscriptionSortBy.DateCreatedDesc,
        endpoint: notificationSubscriptionEndpoint.findMany[1],
        query: notificationSubscriptionEndpoint.findMany[0],
        sortByOptions: NotificationSubscriptionSortBy,
    },
    [SearchType.Organization]: {
        advancedSearchSchema: organizationSearchSchema,
        defaultSortBy: OrganizationSortBy.StarsDesc,
        endpoint: organizationEndpoint.findMany[1],
        query: organizationEndpoint.findMany[0],
        sortByOptions: OrganizationSortBy,
    },
    [SearchType.Post]: {
        advancedSearchSchema: postSearchSchema,
        defaultSortBy: PostSortBy.DateCreatedDesc,
        endpoint: postEndpoint.findMany[1],
        query: postEndpoint.findMany[0],
        sortByOptions: PostSortBy,
    },
    [SearchType.Project]: {
        advancedSearchSchema: projectSearchSchema,
        defaultSortBy: ProjectSortBy.ScoreDesc,
        endpoint: projectEndpoint.findMany[1],
        query: projectEndpoint.findMany[0],
        sortByOptions: ProjectSortBy,
    },
    [SearchType.ProjectVersion]: {
        advancedSearchSchema: projectVersionSearchSchema,
        defaultSortBy: ProjectVersionSortBy.DateCreatedDesc,
        endpoint: projectVersionEndpoint.findMany[1],
        query: projectVersionEndpoint.findMany[0],
        sortByOptions: ProjectVersionSortBy,
    },
    [SearchType.ProjectOrOrganization]: {
        advancedSearchSchema: projectOrOrganizationSearchSchema,
        defaultSortBy: ProjectOrOrganizationSortBy.StarsDesc,
        endpoint: unionEndpoint.projectOrOrganizations[1],
        query: unionEndpoint.projectOrOrganizations[0],
        sortByOptions: ProjectOrOrganizationSortBy,
    },
    [SearchType.ProjectOrRoutine]: {
        advancedSearchSchema: projectOrRoutineSearchSchema,
        defaultSortBy: ProjectOrRoutineSortBy.StarsDesc,
        endpoint: unionEndpoint.projectOrRoutines[1],
        query: unionEndpoint.projectOrRoutines[0],
        sortByOptions: ProjectOrRoutineSortBy,
    },
    [SearchType.PullRequest]: {
        advancedSearchSchema: pullRequestSearchSchema,
        defaultSortBy: PullRequestSortBy.DateCreatedDesc,
        endpoint: pullRequestEndpoint.findMany[1],
        query: pullRequestEndpoint.findMany[0],
        sortByOptions: PullRequestSortBy,
    },
    [SearchType.Question]: {
        advancedSearchSchema: questionSearchSchema,
        defaultSortBy: QuestionSortBy.ScoreDesc,
        endpoint: questionEndpoint.findMany[1],
        query: questionEndpoint.findMany[0],
        sortByOptions: QuestionSortBy,
    },
    [SearchType.QuestionAnswer]: {
        advancedSearchSchema: questionAnswerSearchSchema,
        defaultSortBy: QuestionAnswerSortBy.ScoreDesc,
        endpoint: questionAnswerEndpoint.findMany[1],
        query: questionAnswerEndpoint.findMany[0],
        sortByOptions: QuestionAnswerSortBy,
    },
    [SearchType.Quiz]: {
        advancedSearchSchema: quizSearchSchema,
        defaultSortBy: QuizSortBy.StarsDesc,
        endpoint: quizEndpoint.findMany[1],
        query: quizEndpoint.findMany[0],
        sortByOptions: QuizSortBy,
    },
    [SearchType.QuizQuestion]: {
        advancedSearchSchema: quizQuestionSearchSchema,
        defaultSortBy: QuizQuestionSortBy.OrderAsc,
        endpoint: quizQuestionEndpoint.findMany[1],
        query: quizQuestionEndpoint.findMany[0],
        sortByOptions: QuizQuestionSortBy,
    },
    [SearchType.QuizQuestionResponse]: {
        advancedSearchSchema: quizQuestionResponseSearchSchema,
        defaultSortBy: QuizQuestionResponseSortBy.DateCreatedDesc,
        endpoint: quizQuestionResponseEndpoint.findMany[1],
        query: quizQuestionResponseEndpoint.findMany[0],
        sortByOptions: QuizQuestionResponseSortBy,
    },
    [SearchType.QuizAttempt]: {
        advancedSearchSchema: quizAttemptSearchSchema,
        defaultSortBy: QuizAttemptSortBy.DateCreatedDesc,
        endpoint: quizAttemptEndpoint.findMany[1],
        query: quizAttemptEndpoint.findMany[0],
        sortByOptions: QuizAttemptSortBy,
    },
    [SearchType.Reminder]: {
        advancedSearchSchema: reminderSearchSchema,
        defaultSortBy: ReminderSortBy.DueDateAsc,
        endpoint: reminderEndpoint.findMany[1],
        query: reminderEndpoint.findMany[0],
        sortByOptions: ReminderSortBy,
    },
    [SearchType.ReminderList]: {
        advancedSearchSchema: reminderListSearchSchema,
        defaultSortBy: ReminderListSortBy.DateCreatedDesc,
        endpoint: reminderListEndpoint.findMany[1],
        query: reminderListEndpoint.findMany[0],
        sortByOptions: ReminderListSortBy,
    },
    [SearchType.Report]: {
        advancedSearchSchema: reportSearchSchema,
        defaultSortBy: ReportSortBy.DateCreatedDesc,
        endpoint: reportEndpoint.findMany[1],
        query: reportEndpoint.findMany[0],
        sortByOptions: ReportSortBy,
    },
    [SearchType.ReportResponse]: {
        advancedSearchSchema: reportResponseSearchSchema,
        defaultSortBy: ReportResponseSortBy.DateCreatedDesc,
        endpoint: reportResponseEndpoint.findMany[1],
        query: reportResponseEndpoint.findMany[0],
        sortByOptions: ReportResponseSortBy,
    },
    [SearchType.ReputationHistory]: {
        advancedSearchSchema: reputationHistorySearchSchema,
        defaultSortBy: ReputationHistorySortBy.DateCreatedDesc,
        endpoint: reputationHistoryEndpoint.findMany[1],
        query: reputationHistoryEndpoint.findMany[0],
        sortByOptions: ReputationHistorySortBy,
    },
    [SearchType.Resource]: {
        advancedSearchSchema: resourceSearchSchema,
        defaultSortBy: ResourceSortBy.DateCreatedDesc,
        endpoint: resourceEndpoint.findMany[1],
        query: resourceEndpoint.findMany[0],
        sortByOptions: ResourceSortBy,
    },
    [SearchType.ResourceList]: {
        advancedSearchSchema: resourceListSearchSchema,
        defaultSortBy: ResourceListSortBy.DateCreatedDesc,
        endpoint: resourceListEndpoint.findMany[1],
        query: resourceListEndpoint.findMany[0],
        sortByOptions: ResourceListSortBy,
    },
    [SearchType.Role]: {
        advancedSearchSchema: roleSearchSchema,
        defaultSortBy: RoleSortBy.DateCreatedDesc,
        endpoint: roleEndpoint.findMany[1],
        query: roleEndpoint.findMany[0],
        sortByOptions: RoleSortBy,
    },
    [SearchType.Routine]: {
        advancedSearchSchema: routineSearchSchema,
        defaultSortBy: RoutineSortBy.ScoreDesc,
        endpoint: routineEndpoint.findMany[1],
        query: routineEndpoint.findMany[0],
        sortByOptions: RoutineSortBy,
    },
    [SearchType.RoutineVersion]: {
        advancedSearchSchema: routineVersionSearchSchema,
        defaultSortBy: RoutineVersionSortBy.DateCreatedDesc,
        endpoint: routineVersionEndpoint.findMany[1],
        query: routineVersionEndpoint.findMany[0],
        sortByOptions: RoutineVersionSortBy,
    },
    [SearchType.RunProject]: {
        advancedSearchSchema: runProjectSearchSchema,
        defaultSortBy: RunProjectSortBy.DateStartedAsc,
        endpoint: runProjectEndpoint.findMany[1],
        query: runProjectEndpoint.findMany[0],
        sortByOptions: RunProjectSortBy,
    },
    [SearchType.RunProjectOrRunRoutine]: {
        advancedSearchSchema: {} as any,/// TODO runProjectOrRunRoutineSearchSchema,
        defaultSortBy: RunProjectOrRunRoutineSortBy.DateStartedAsc,
        endpoint: unionEndpoint.runProjectOrRunRoutines[1],
        query: unionEndpoint.runProjectOrRunRoutines[0],
        sortByOptions: RunProjectOrRunRoutineSortBy,
    },
    [SearchType.RunProjectSchedule]: {
        advancedSearchSchema: runProjectScheduleSearchSchema,
        defaultSortBy: RunProjectScheduleSortBy.WindowStartAsc,
        endpoint: runProjectScheduleEndpoint.findMany[1],
        query: runProjectScheduleEndpoint.findMany[0],
        sortByOptions: RunProjectScheduleSortBy,
    },
    [SearchType.RunRoutine]: {
        advancedSearchSchema: runRoutineSearchSchema,
        defaultSortBy: RunRoutineSortBy.DateStartedAsc,
        endpoint: runRoutineEndpoint.findMany[1],
        query: runRoutineEndpoint.findMany[0],
        sortByOptions: RunRoutineSortBy,
    },
    [SearchType.RunRoutineInput]: {
        advancedSearchSchema: runRoutineInputSearchSchema,
        defaultSortBy: RunRoutineInputSortBy.DateCreatedDesc,
        endpoint: runRoutineInputEndpoint.findMany[1],
        query: runRoutineInputEndpoint.findMany[0],
        sortByOptions: RunRoutineInputSortBy,
    },
    [SearchType.RunRoutineSchedule]: {
        advancedSearchSchema: runRoutineScheduleSearchSchema,
        defaultSortBy: RunRoutineScheduleSortBy.WindowStartAsc,
        endpoint: runRoutineScheduleEndpoint.findMany[1],
        query: runRoutineScheduleEndpoint.findMany[0],
        sortByOptions: RunRoutineScheduleSortBy,
    },
    [SearchType.SmartContract]: {
        advancedSearchSchema: smartContractSearchSchema,
        defaultSortBy: SmartContractSortBy.ScoreDesc,
        endpoint: smartContractEndpoint.findMany[1],
        query: smartContractEndpoint.findMany[0],
        sortByOptions: SmartContractSortBy,
    },
    [SearchType.SmartContractVersion]: {
        advancedSearchSchema: smartContractVersionSearchSchema,
        defaultSortBy: SmartContractVersionSortBy.DateCreatedDesc,
        endpoint: smartContractVersionEndpoint.findMany[1],
        query: smartContractVersionEndpoint.findMany[0],
        sortByOptions: SmartContractVersionSortBy,
    },
    [SearchType.Standard]: {
        advancedSearchSchema: standardSearchSchema,
        defaultSortBy: StandardSortBy.ScoreDesc,
        endpoint: standardEndpoint.findMany[1],
        query: standardEndpoint.findMany[0],
        sortByOptions: StandardSortBy,
    },
    [SearchType.StandardVersion]: {
        advancedSearchSchema: standardVersionSearchSchema,
        defaultSortBy: StandardVersionSortBy.DateCreatedDesc,
        endpoint: standardVersionEndpoint.findMany[1],
        query: standardVersionEndpoint.findMany[0],
        sortByOptions: StandardVersionSortBy,
    },
    [SearchType.Star]: {
        advancedSearchSchema: null,
        defaultSortBy: StarSortBy.DateUpdatedDesc,
        endpoint: starEndpoint.stars[1],
        query: starEndpoint.stars[0],
        sortByOptions: StarSortBy,
    },
    [SearchType.StatsApi]: {
        advancedSearchSchema: statsApiSearchSchema,
        defaultSortBy: StatsApiSortBy.DateUpdatedDesc,
        endpoint: statsApiEndpoint.findMany[1],
        query: statsApiEndpoint.findMany[0],
        sortByOptions: StatsApiSortBy,
    },
    [SearchType.StatsOrganization]: {
        advancedSearchSchema: statsOrganizationSearchSchema,
        defaultSortBy: StatsOrganizationSortBy.DateUpdatedDesc,
        endpoint: statsOrganizationEndpoint.findMany[1],
        query: statsOrganizationEndpoint.findMany[0],
        sortByOptions: StatsOrganizationSortBy,
    },
    [SearchType.StatsProject]: {
        advancedSearchSchema: statsProjectSearchSchema,
        defaultSortBy: StatsProjectSortBy.DateUpdatedDesc,
        endpoint: statsProjectEndpoint.findMany[1],
        query: statsProjectEndpoint.findMany[0],
        sortByOptions: StatsProjectSortBy,
    },
    [SearchType.StatsQuiz]: {
        advancedSearchSchema: statsQuizSearchSchema,
        defaultSortBy: StatsQuizSortBy.DateUpdatedDesc,
        endpoint: statsQuizEndpoint.findMany[1],
        query: statsQuizEndpoint.findMany[0],
        sortByOptions: StatsQuizSortBy,
    },
    [SearchType.StatsRoutine]: {
        advancedSearchSchema: statsRoutineSearchSchema,
        defaultSortBy: StatsRoutineSortBy.DateUpdatedDesc,
        endpoint: statsRoutineEndpoint.findMany[1],
        query: statsRoutineEndpoint.findMany[0],
        sortByOptions: StatsRoutineSortBy,
    },
    [SearchType.StatsSite]: {
        advancedSearchSchema: statsSiteSearchSchema,
        defaultSortBy: StatsSiteSortBy.DateUpdatedDesc,
        endpoint: statsSiteEndpoint.findMany[1],
        query: statsSiteEndpoint.findMany[0],
        sortByOptions: StatsSiteSortBy,
    },
    [SearchType.StatsSmartContract]: {
        advancedSearchSchema: statsSmartContractSearchSchema,
        defaultSortBy: StatsSmartContractSortBy.DateUpdatedDesc,
        endpoint: statsSmartContractEndpoint.findMany[1],
        query: statsSmartContractEndpoint.findMany[0],
        sortByOptions: StatsSmartContractSortBy,
    },
    [SearchType.StatsStandard]: {
        advancedSearchSchema: statsStandardSearchSchema,
        defaultSortBy: StatsStandardSortBy.DateUpdatedDesc,
        endpoint: statsStandardEndpoint.findMany[1],
        query: statsStandardEndpoint.findMany[0],
        sortByOptions: StatsStandardSortBy,
    },
    [SearchType.StatsUser]: {
        advancedSearchSchema: statsUserSearchSchema,
        defaultSortBy: StatsUserSortBy.DateUpdatedDesc,
        endpoint: statsUserEndpoint.findMany[1],
        query: statsUserEndpoint.findMany[0],
        sortByOptions: StatsUserSortBy,
    },
    [SearchType.Tag]: {
        advancedSearchSchema: tagSearchSchema,
        defaultSortBy: TagSortBy.StarsDesc,
        endpoint: tagEndpoint.findMany[1],
        query: tagEndpoint.findMany[0],
        sortByOptions: TagSortBy,
    },
    [SearchType.Transfer]: {
        advancedSearchSchema: transferSearchSchema,
        defaultSortBy: TransferSortBy.DateCreatedDesc,
        endpoint: transferEndpoint.findMany[1],
        query: transferEndpoint.findMany[0],
        sortByOptions: TransferSortBy,
    },
    [SearchType.UserSchedule]: {
        advancedSearchSchema: userScheduleSearchSchema,
        defaultSortBy: UserScheduleSortBy.EventStartAsc,
        endpoint: userScheduleEndpoint.findMany[1],
        query: userScheduleEndpoint.findMany[0],
        sortByOptions: UserScheduleSortBy,
    },
    [SearchType.User]: {
        advancedSearchSchema: userSearchSchema,
        defaultSortBy: UserSortBy.StarsDesc,
        endpoint: userEndpoint.findMany[1],
        query: userEndpoint.findMany[0],
        sortByOptions: UserSortBy,
    },
    [SearchType.View]: {
        advancedSearchSchema: null,
        defaultSortBy: ViewSortBy.LastViewedDesc,
        endpoint: viewEndpoint.views[1],
        query: viewEndpoint.views[0],
        sortByOptions: ViewSortBy,
    },
    [SearchType.Vote]: {
        advancedSearchSchema: voteSearchSchema,
        defaultSortBy: VoteSortBy.DateUpdatedDesc,
        endpoint: voteEndpoint.votes[1],
        query: voteEndpoint.votes[0],
        sortByOptions: VoteSortBy,
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