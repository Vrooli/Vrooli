import { RoutineSortBy } from "@local/shared";
import { RoutineListItem } from "components";
import { routinesQuery } from "graphql/query";
import { RoutineDeep } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: RoutineSortBy}[] = Object.values(RoutineSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as RoutineSortBy }));

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
            defaultSortOption={SORT_OPTIONS[1].value}
            query={routinesQuery}
            listItemFactory={listItemFactory}
        />
    )
}