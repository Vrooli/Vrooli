import { commentSearchSchema, organizationSearchSchema, projectOrOrganizationSearchSchema, projectOrRoutineSearchSchema, projectSearchSchema, routineSearchSchema, runSearchSchema, standardSearchSchema, userSearchSchema } from 'components';
import { CommentSortBy, InputType, OrganizationSortBy, ProjectOrOrganizationSortBy, ProjectOrRoutineSortBy, ProjectSortBy, RoutineSortBy, RunSortBy, StandardSortBy, StarSortBy, UserSortBy, ViewSortBy } from '@shared/consts';
import { Tag } from 'types';
import { FormSchema } from 'forms/types';
import { commentsQuery, organizationsQuery, projectOrOrganizationsQuery, projectsQuery, routinesQuery, runsQuery, standardsQuery, starsQuery, usersQuery, viewsQuery } from 'graphql/query';
import { DocumentNode } from 'graphql';
import { projectOrRoutinesQuery } from 'graphql/query/projectOrRoutines';

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

type SearchParams = {
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
 * Converts an array of items to a search URL value
 * @param arr The array of items
 * @param convert The function to convert each item to a search URL value
 * @returns The items of the array converted, or undefined if the array is empty or undefined
 */
function arrayToSearch<T>(arr: T[] | undefined, convert?: (T) => string): string[] | undefined {
    if (Array.isArray(arr)) {
        if (arr.length === 0) return undefined;
        return convert ? arr.map(convert) : arr as unknown as string[];
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
    [InputType.LanguageInput]: (value: string[]) => arrayToSearch(value),
    [InputType.Markdown]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
    [InputType.Radio]: (value: string) => radioValueToSearch(value),
    [InputType.Selector]: (value) => value, //TODO
    [InputType.Slider]: (value) => value, //TODO
    [InputType.Switch]: (value) => value, //TODO 
    [InputType.TagSelector]: (value: Tag[]) => arrayToSearch(value, ({ tag }) => tag),
    [InputType.TextField]: (value: string) => typeof value === 'string' && value.trim().length > 0 ? value.trim() : undefined,
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