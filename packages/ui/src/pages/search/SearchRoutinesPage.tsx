import { RoutineSortBy } from "@local/shared";
import { RoutineListItem } from "components";
import { routinesQuery } from "graphql/query";
import { RoutineDeep } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<RoutineSortBy>[] = Object.values(RoutineSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as RoutineSortBy 
}));

export const SearchRoutinesPage = () => {
    const listItemFactory = (node: RoutineDeep, index: number) => (
        <RoutineListItem 
            key={`routine-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={() => {}}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Routines'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={routinesQuery}
            listItemFactory={listItemFactory}
        />
    )
}