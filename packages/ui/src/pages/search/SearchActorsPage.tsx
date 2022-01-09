import { UserSortBy } from "@local/shared";
import { ActorListItem } from "components";
import { usersQuery } from "graphql/query";
import { User } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: UserSortBy}[] = Object.values(UserSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as UserSortBy }));

export const SearchActorsPage = () => {
    const listItemFactory = (node: User, index: number) => (
        <ActorListItem 
            key={`actor-list-item-${index}`} 
            data={node} 
            isStarred={false}
            isOwn={false}
            onClick={() => {}}
            onStarClick={() => {}}
        />)

    return (
        <BaseSearchPage 
            title={'Search Actors'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1].value}
            query={usersQuery}
            listItemFactory={listItemFactory}
        />
    )
}