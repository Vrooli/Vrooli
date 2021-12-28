import { useQuery } from "@apollo/client";
import { useState } from "react";
import { BaseSearchPageProps, SortOption } from "./types";


type QueryInput = {
    first?: number;
    ids?: number[];
    searchString?: string;
    skip?: number;
    sortBy?: { [x: string]: string }
    userId?: number;
};

export const BaseSearchPage = ({
    searchQuery,
    sortOptions,
}: BaseSearchPageProps) => {
    const [searchString, setSearchString] = useState<string>('');
    const [sortBy, setSortBy] = useState<SortOption | undefined>(sortOptions.length > 0 ? sortOptions[0] : undefined);
    const { data: searchData } = useQuery<any, any>(searchQuery, { variables: { sortBy, searchString }, pollInterval: 50000 });
}