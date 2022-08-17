import { CommentSortBy, OrganizationSortBy, ProjectSortBy, RoutineSortBy, StandardSortBy, UserSortBy } from "@shared/consts";
import { commentSearchSchema, organizationSearchSchema, projectSearchSchema, routineSearchSchema, standardSearchSchema, userSearchSchema } from "components/dialogs/AdvancedSearchDialog/schemas";
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

export const standardSearchInfo = {
    advancedSearchSchema: standardSearchSchema,
    defaultSortBy: StandardSortBy.VotesDesc,
    sortByOptions: StandardSortBy,
}

export const userSearchInfo = {
    advancedSearchSchema: userSearchSchema,
    defaultSortBy: UserSortBy.StarsDesc,
    sortByOptions: UserSortBy,
}

export const objectToSearchInfo = {
    [ObjectType.Comment]: commentSearchInfo,
    [ObjectType.Organization]: organizationSearchInfo,
    [ObjectType.Project]: projectSearchInfo,
    [ObjectType.Routine]: routineSearchInfo,
    [ObjectType.Standard]: standardSearchInfo,
    [ObjectType.User]: userSearchInfo,
}