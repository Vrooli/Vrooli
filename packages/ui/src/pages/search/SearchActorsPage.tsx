import { UserSortBy } from "@local/shared";
import { ActorCard } from "components";
import { usersQuery } from "graphql/query";
import { User } from "types";
import { BaseSearchPage } from "./BaseSearchPage";

const SORT_OPTIONS: {label: string, value: UserSortBy}[] = Object.values(UserSortBy).map((sortOption) => ({ label: sortOption, value: sortOption as UserSortBy }));

export const SearchActorsPage = () => {
    const cardFactory = (node: User, index: number) => <ActorCard key={`actor-card-${index}`} data={node} />

    return (
        <BaseSearchPage 
            title={'Search Actors'}
            sortOptions={SORT_OPTIONS}
            defaultSortOption={SORT_OPTIONS[1].value}
            query={usersQuery}
            cardFactory={cardFactory}
        />
    )
}