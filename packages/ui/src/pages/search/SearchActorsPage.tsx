import { UserSortBy } from "@local/shared";
import { ActorListItem } from "components";
import { usersQuery } from "graphql/query";
import { useCallback, useState } from "react";
import { User } from "types";
import { SortValueToLabelMap } from "utils";
import { BaseSearchPage } from "./BaseSearchPage";
import { SearchSortBy } from "./types";

const SORT_OPTIONS: SearchSortBy<UserSortBy>[] = Object.values(UserSortBy).map((sortOption) => ({ 
    label: SortValueToLabelMap[sortOption], 
    value: sortOption as UserSortBy 
}));

export const SearchActorsPage = () => {
    const [selected, setSelected] = useState<User | undefined>(undefined);
    const dialogOpen = Boolean(selected);

    const handleDialogClose = useCallback(() => setSelected(undefined), []);

    const listItemFactory = (node: User, index: number) => (
        <ActorListItem 
            key={`actor-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={(selected: User) => setSelected(selected)}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Actors'}
            searchPlaceholder="Search by username..."
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1]}
            query={usersQuery}
            listItemFactory={listItemFactory}
            getOptionLabel={(o: any) => o.username}
            onObjectSelect={(selected: User) => setSelected(selected)}
        />
    )
}