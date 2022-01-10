import { OrganizationSortBy } from "@local/shared";
import { OrganizationListItem } from "components";
import { organizationsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Organization } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<OrganizationSortBy>[] = Object.values(OrganizationSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as OrganizationSortBy 
}));

export const SearchOrganizationsPage = () => {
    const [selected, setSelected] = useState<Organization | undefined>(undefined);
    const dialogOpen = Boolean(selected);

    const handleDialogClose = useCallback(() => setSelected(undefined), []);

    const listItemFactory = (node: any, index: number) => (
        <OrganizationListItem 
            key={`organization-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={(selected: Organization) => setSelected(selected)}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Organizations'}
            searchPlaceholder="Search by name, description, or tags..."
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={organizationsQuery}
            listItemFactory={listItemFactory}
            getOptionLabel={(o: any) => o.name}
            onObjectSelect={(selected: Organization) => setSelected(selected)}
        />
    )
}