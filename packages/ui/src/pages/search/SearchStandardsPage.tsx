import { StandardSortBy } from "@local/shared";
import { StandardListItem } from "components";
import { standardsQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { Standard } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<StandardSortBy>[] = Object.values(StandardSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as StandardSortBy 
}));

export const SearchStandardsPage = () => {
    const [selected, setSelected] = useState<Standard | undefined>(undefined);
    const dialogOpen = Boolean(selected);

    const handleDialogClose = useCallback(() => setSelected(undefined), []);

    const listItemFactory = (node: any, index: number) => (
        <StandardListItem 
            key={`standard-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={(selected: Standard) => setSelected(selected)}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Standards'}
            searchPlaceholder="Search by name, description, or tags..."
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={standardsQuery}
            listItemFactory={listItemFactory}
            getOptionLabel={(o: any) => o.name}
            onObjectSelect={(selected: Standard) => setSelected(selected)}
        />
    )
}