import { DocumentNode } from "graphql";

export interface BaseSearchPageProps {
    searchQuery: DocumentNode
    sortOptions: SortOption[]
}

export interface SortOption {
    label: string
    value: string
}