import { OrganizationSortBy } from "@local/shared";
import { OrganizationListItem } from "components";
import { organizationsQuery } from "graphql/query";
import { Organization } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<OrganizationSortBy>[] = Object.values(OrganizationSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as OrganizationSortBy 
}));

export const SearchOrganizationsPage = () => {
    const listItemFactory = (node: Organization, index: number) => (
        <OrganizationListItem 
            key={`organization-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={() => {}}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Organizations'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={organizationsQuery}
            listItemFactory={listItemFactory}
        />
    )
}