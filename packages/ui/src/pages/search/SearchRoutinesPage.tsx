import { RoutineSortBy } from "@local/shared";
import { RoutineListItem } from "components";
import { routinesQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { RoutineDeep } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<RoutineSortBy>[] = Object.values(RoutineSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as RoutineSortBy 
}));

export const SearchRoutinesPage = () => {
    const [selected, setSelected] = useState<RoutineDeep | undefined>(undefined);
    const dialogOpen = Boolean(selected);

    const handleDialogClose = useCallback(() => setSelected(undefined), []);

    const listItemFactory = (node: any, index: number) => (
        <RoutineListItem 
            key={`routine-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={(selected: RoutineDeep) => setSelected(selected)}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Routines'}
            searchPlaceholder="Search by title, description, instructions, or tags..."
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={routinesQuery}
            listItemFactory={listItemFactory}
            getOptionLabel={(o: any) => o.title}
            onObjectSelect={(selected: RoutineDeep) => setSelected(selected)}
        />
    )
}