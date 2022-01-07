import { DocumentNode } from "graphql";

export type SearchSortBy<SortBy> = { label: string, value: SortBy };

export interface SearchQueryVariables {
    input: UserSearchInput;
}

export interface SearchQueryVariablesInput<SortBy> {
    ids?: string[] | null;
    sortBy?: SortBy | null;
    searchString?: string | null;
    after?: string | null;
    take?: number | null;
}

export interface BaseSearchPageProps<SortBy> {
    title?: string | null;
    sortOptions: SearchSortBy<SortBy>[];
    defaultSortOption: SortBy;
    query: DocumentNode;
    cardFactory: (node: User, index: number) => JSX.Element;
}