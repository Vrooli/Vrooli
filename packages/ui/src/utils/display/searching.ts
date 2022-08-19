import { CommentSortBy, OrganizationSortBy, ProjectSortBy, RoutineSortBy, RunSortBy, StandardSortBy, StarSortBy, UserSortBy, ViewSortBy } from "@shared/consts";
import { commentSearchSchema, organizationSearchSchema, projectSearchSchema, routineSearchSchema, runSearchSchema, standardSearchSchema, userSearchSchema } from "components/dialogs/AdvancedSearchDialog/schemas";
import { ObjectType } from "utils/navigation";

export const commentSearchInfo = {
    advancedSearchSchema: commentSearchSchema,
    defaultSortBy: CommentSortBy.VotesDesc,
    sortByOptions: CommentSortBy,
}

export const organizationSearchInfo = {
    advancedSearchSchema: organizationSearchSchema,
    defaultSortBy: OrganizationSortBy.StarsDesc,
    sortByOptions: OrganizationSortBy,
}

export const projectSearchInfo = {
    advancedSearchSchema: projectSearchSchema,
    defaultSortBy: ProjectSortBy.VotesDesc,
    sortByOptions: ProjectSortBy,
}

export const routineSearchInfo = {
    advancedSearchSchema: routineSearchSchema,
    defaultSortBy: RoutineSortBy.VotesDesc,
    sortByOptions: RoutineSortBy,
}

export const runSearchInfo = {
    advancedSearchSchema: runSearchSchema,
    defaultSortBy: RunSortBy.DateStartedAsc,
    sortByOptions: RunSortBy,
}

export const standardSearchInfo = {
    advancedSearchSchema: standardSearchSchema,
    defaultSortBy: StandardSortBy.VotesDesc,
    sortByOptions: StandardSortBy,
}

export const starSearchInfo = {
    advancedSearchSchema: null,
    defaultSortBy: StarSortBy.DateUpdatedDesc,
    sortByOptions: StarSortBy,
}

export const userSearchInfo = {
    advancedSearchSchema: userSearchSchema,
    defaultSortBy: UserSortBy.StarsDesc,
    sortByOptions: UserSortBy,
}

export const viewSearchInfo = {
    advancedSearchSchema: null,
    defaultSortBy: ViewSortBy.LastViewedDesc,
    sortByOptions: ViewSortBy,
}

export const objectToSearchInfo = {
    [ObjectType.Comment]: commentSearchInfo,
    [ObjectType.Organization]: organizationSearchInfo,
    [ObjectType.Project]: projectSearchInfo,
    [ObjectType.Routine]: routineSearchInfo,
    [ObjectType.Run]: runSearchInfo,
    [ObjectType.Standard]: standardSearchInfo,
    [ObjectType.Star]: starSearchInfo,
    [ObjectType.User]: userSearchInfo,
    [ObjectType.View]: viewSearchInfo,
}