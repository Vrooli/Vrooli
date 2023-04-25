import { addSearchParams, exists, parseSearchParams, TimeFrame, useLocation } from "@local/shared";
import { useCustomLazyQuery } from "api";
import { SearchQueryVariablesInput } from "components/lists/types";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AutocompleteOption } from "types";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { SessionContext } from "utils/SessionContext";
import { useDisplayApolloError } from "./useDisplayApolloError";
import { useStableObject } from "./useStableObject";

type UseFindManyProps = {
    canSearch: boolean;
    searchType: SearchType | `${SearchType}`;
    resolve?: (data: any) => any;
    take?: number;
    where?: any;
}

/**
 * Helper method for converting fetched data to an array of object data
 * @param data data returned from a findMany query
 * @param resolve function for resolving the data
 */
export const parseData = (data: any, resolve?: (data: any) => any) => {
    if (!data) return [];
    // query result is always returned as an object with a single key (the endpoint name), where 
    // the value is the actual data. If this is not the case, then (hopefully) it was already 
    // deconstructed earlier in the chain
    const queryData: any = (Object.keys(data).length === 1 && !["success", "count"].includes(Object.keys(data)[0])) ? Object.values(data)[0] : data;
    // If there is a custom resolver, use it
    if (resolve) return resolve(queryData);
    // Otherwise, treat as typically-shaped paginated data
    if (!queryData || !queryData.edges) return [];
    return queryData.edges.map((edge: any, index: number) => edge.node);
};

/**
 * Logic for displaying search options and querying a findMany endpoint
 */
export const useFindMany = <DataType extends Record<string, any>>({
    canSearch,
    searchType,
    resolve,
    take = 20,
    where,
}: UseFindManyProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    const stableResolve = useStableObject(resolve);
    const stableWhere = useStableObject(where);

    const [params, setParams] = useState<Partial<Partial<SearchParams> & { where: any }>>({});
    useEffect(() => {
        const fetchParams = async () => {
            const newParams = searchTypeToParams[searchType];
            if (!newParams) return;
            const resolvedParams = await newParams();
            setParams({ ...resolvedParams, where: stableWhere });
        };
        fetchParams();
    }, [searchType, stableWhere]);

    const [sortBy, setSortBy] = useState<string>(params?.defaultSortBy ?? "");
    const [searchString, setSearchString] = useState<string>("");
    const [timeFrame, setTimeFrame] = useState<TimeFrame | undefined>(undefined);
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (typeof searchParams.search === "string") setSearchString(searchParams.search);
        if (typeof searchParams.sort === "string") {
            // Check if sortBy is valid
            if (exists(params?.sortByOptions) && searchParams.sort in params.sortByOptions) {
                setSortBy(searchParams.sort);
            } else {
                setSortBy(params?.defaultSortBy ?? "");
            }
        }
        if (typeof searchParams.time === "object" &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty("after") &&
            searchParams.time.hasOwnProperty("before")) {
            setTimeFrame({
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            });
        }
    }, [params?.defaultSortBy, params?.sortByOptions]);

    /**
     * When sort and filter options change, update the URL
     */
    useEffect(() => {
        addSearchParams(setLocation, {
            search: searchString.length > 0 ? searchString : undefined,
            sort: sortBy,
            time: timeFrame ? {
                after: timeFrame.after?.toISOString() ?? "",
                before: timeFrame.before?.toISOString() ?? "",
            } : undefined,
        });
    }, [searchString, sortBy, timeFrame, setLocation]);

    /**
     * Cursor for pagination. Resets when search params change
     */
    const after = useRef<string | undefined>(undefined);

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object | null>(null);
    const [getPageData, { data: pageData, loading, error }] = useCustomLazyQuery<Record<string, any>, SearchQueryVariablesInput<any>>(params!.query, {
        variables: ({
            after: after.current,
            take,
            sortBy,
            searchString,
            createdTimeFrame: (timeFrame && Object.keys(timeFrame).length > 0) ? {
                after: timeFrame.after?.toISOString(),
                before: timeFrame.before?.toISOString(),
            } : undefined,
            ...params.where,
            ...advancedSearchParams,
        } as any),
        errorPolicy: "all",
    });
    // Display a snack error message if there is an error
    useDisplayApolloError(error);
    const [allData, setAllData] = useState<DataType[]>(() => {
        // TODO Check if we just navigated back to this page from an object page. If so, use results stored in sessionStorage. Also TODO for storing results in sessionStorage
        const lastPath = sessionStorage.getItem("lastPath");
        const lastSearchParams = sessionStorage.getItem("lastSearchParams");
        console.log("lastPath", lastPath);
        console.log("lastSearchParams", lastSearchParams);
        return [];
    });

    // Track if there is more data to fetch
    const [hasMore, setHasMore] = useState<boolean>(true);
    // Reset hasMore when search params change
    useEffect(() => {
        console.log("resetting hasMore");
        setHasMore(true);
    }, [advancedSearchParams, searchString, sortBy, timeFrame]);

    // On search filters/sort change, reset the page
    useEffect(() => {
        // Reset the pagination cursor
        after.current = undefined;
        // Only get data if we can search
        if (canSearch && sortBy?.length > 0) getPageData();
    }, [advancedSearchParams, canSearch, searchString, sortBy, timeFrame, getPageData]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!pageData || !canSearch) return;
        if (!pageData.pageInfo) return [];
        if (pageData.pageInfo?.hasNextPage) {
            const { endCursor } = pageData.pageInfo;
            if (endCursor) {
                after.current = endCursor;
                getPageData();
            }
        } else {
            setHasMore(false);
        }
    }, [canSearch, getPageData, pageData]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        const parsedData = parseData(pageData, stableResolve);
        console.log("got parsed data", parsedData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (after.current) {
            setAllData(curr => [...curr, ...parsedData]);
        } else {
            setAllData(parsedData);
        }
    }, [pageData, stableResolve]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        console.log("LISTTOAUTOCOMPLETE allData", allData);
        return listToAutocomplete(allData as any, getUserLanguages(session)).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks; //TODO not all objects have bookmarks
        });
    }, [allData, session]);

    return {
        advancedSearchParams,
        advancedSearchSchema: params?.advancedSearchSchema,
        allData,
        autocompleteOptions,
        defaultSortBy: params?.defaultSortBy,
        hasMore,
        loading,
        loadMore,
        pageData,
        parseData,
        searchString,
        setAdvancedSearchParams,
        setAllData,
        setSortBy,
        setSearchString,
        setTimeFrame,
        sortBy,
        sortByOptions: params?.sortByOptions,
        timeFrame,
    };
};
