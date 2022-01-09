import { OrganizationSortBy } from "@local/shared";
import { OrganizationListItem } from "components";
import { organizationsQuery } from "graphql/query";
import { Organization } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: OrganizationSortBy}[] = Object.values(OrganizationSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as OrganizationSortBy }));

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
            defaultSortOption={SORT_OPTIONS[1].value}
            query={organizationsQuery}
            listItemFactory={listItemFactory}
        />
    )
}