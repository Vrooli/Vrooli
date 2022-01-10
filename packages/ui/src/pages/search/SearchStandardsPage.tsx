import { StandardSortBy } from "@local/shared";
import { StandardListItem } from "components";
import { standardsQuery } from "graphql/query";
import { Standard } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<StandardSortBy>[] = Object.values(StandardSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as StandardSortBy 
}));

export const SearchStandardsPage = () => {
    const listItemFactory = (node: Standard, index: number) => (
        <StandardListItem 
            key={`standard-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={() => {}}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Standards'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={standardsQuery}
            listItemFactory={listItemFactory}
        />
    )
}