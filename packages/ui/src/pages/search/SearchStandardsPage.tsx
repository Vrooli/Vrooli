import { StandardSortBy } from "@local/shared";
import { StandardListItem } from "components";
import { standardsQuery } from "graphql/query";
import { Standard } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: StandardSortBy}[] = Object.values(StandardSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as StandardSortBy }));

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
            defaultSortOption={SORT_OPTIONS[1].value}
            query={standardsQuery}
            listItemFactory={listItemFactory}
        />
    )
}