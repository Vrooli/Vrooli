import { CommentSortBy, InputType, OrganizationSortBy, ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, ProjectSortBy, RoutineSortBy, RunSortBy, StandardSortBy, StarSortBy, UserSortBy, ViewSortBy } from '@shared/consts';
import { Session, Tag } from 'types';
import { FormSchema } from 'forms/types';
import { commentsQuery, organizationsQuery, projectOrOrganizationsQuery, projectsQuery, routinesQuery, runsQuery, standardsQuery, starsQuery, usersQuery, viewsQuery } from 'graphql/query';
import { DocumentNode } from 'graphql';
import { projectOrRoutinesQuery } from 'graphql/query/projectOrRoutines';
import { RunStatus } from 'graphql/generated/globalTypes';
import { getLocalStorageKeys } from 'utils/localStorage';
import { PubSub } from 'utils/pubsub';
import { SnackSeverity } from 'components';
import { getCurrentUser } from 'utils/authentication';

const starsDescription = `Stars are a way to bookmark an object. They don't affect the ranking of an object in default searches, but are still useful to get a feel for how popular an object is.`;
const votesDescription = `Votes are a way to show support for an object, which affect the ranking of an object in default searches.`;
const languagesDescription = `Filter results by the language(s) they are written in.`;
const tagsDescription = `Filter results by the tags they are associated with.`;
const simplicityDescription = `Simplicity is a mathematical measure of the shortest path to complete a routine. 

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;
const complexityDescription = `Complexity is a mathematical measure of the longest path to complete a routine.

For the curious, it is calculated using a weighted, directed, cyclic graph. Each node is a subroutine list or decision, and each weight represents the number of steps the node takes to complete`;

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

export const runSearchSchema: FormSchema = {
    formLayout: {
        title: "Search Runs",
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

export enum SearchType {
    Comment = 'Comment',
    Organization = 'Organization',
    Project = 'Project',
    ProjectOrOrganization = 'ProjectOrOrganization',
    ProjectOrRoutine = 'ProjectOrRoutine',
    Routine = 'Routine',
    Run = 'Run',
    Standard = 'Standard',
    Star = 'Star',
    User = 'User',
    View = 'View',
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
    [SearchType.Comment]: {
        advancedSearchSchema: commentSearchSchema,
        defaultSortBy: CommentSortBy.VotesDesc,
        sortByOptions: CommentSortBy,
        query: commentsQuery,
    },
    [SearchType.Organization]: {
        advancedSearchSchema: organizationSearchSchema,
        defaultSortBy: OrganizationSortBy.StarsDesc,
        sortByOptions: OrganizationSortBy,
        query: organizationsQuery,
    },
    [SearchType.Project]: {
        advancedSearchSchema: projectSearchSchema,
        defaultSortBy: ProjectSortBy.VotesDesc,
        sortByOptions: ProjectSortBy,
        query: projectsQuery,
    },
    [SearchType.ProjectOrOrganization]: {
        advancedSearchSchema: projectOrOrganizationSearchSchema,
        defaultSortBy: ProjectOrOrganizationSortBy.StarsDesc,
        sortByOptions: ProjectOrOrganizationSortBy,
        query: projectOrOrganizationsQuery,
    },
    [SearchType.ProjectOrRoutine]: {
        advancedSearchSchema: projectOrRoutineSearchSchema,
        defaultSortBy: ProjectOrRoutineSortBy.StarsDesc,
        sortByOptions: ProjectOrRoutineSortBy,
        query: projectOrRoutinesQuery,
    },
    [SearchType.Routine]: {
        advancedSearchSchema: routineSearchSchema,
        defaultSortBy: RoutineSortBy.VotesDesc,
        sortByOptions: RoutineSortBy,
        query: routinesQuery,
    },
    [SearchType.Run]: {
        advancedSearchSchema: runSearchSchema,
        defaultSortBy: RunSortBy.DateStartedAsc,
        sortByOptions: RunSortBy,
        query: runsQuery,
    },
    [SearchType.Standard]: {
        advancedSearchSchema: standardSearchSchema,
        defaultSortBy: StandardSortBy.VotesDesc,
        sortByOptions: StandardSortBy,
        query: standardsQuery,
    },
    [SearchType.Star]: {
        advancedSearchSchema: null,
        defaultSortBy: StarSortBy.DateUpdatedDesc,
        sortByOptions: StarSortBy,
        query: starsQuery,
    },
    [SearchType.User]: {
        advancedSearchSchema: userSearchSchema,
        defaultSortBy: UserSortBy.StarsDesc,
        sortByOptions: UserSortBy,
        query: usersQuery,
    },
    [SearchType.View]: {
        advancedSearchSchema: null,
        defaultSortBy: ViewSortBy.LastViewedDesc,
        sortByOptions: ViewSortBy,
        query: viewsQuery,
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
    PubSub.get().publishSnack({ message: 'Search history cleared.', severity: SnackSeverity.Success });
}