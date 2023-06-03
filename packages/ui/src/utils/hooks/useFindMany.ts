import { addSearchParams, exists, lowercaseFirstLetter, parseSearchParams, TimeFrame, useLocation } from "@local/shared";
import { SearchQueryVariablesInput } from "components/lists/types";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AutocompleteOption } from "types";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { SessionContext } from "utils/SessionContext";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";
import { useStableObject } from "./useStableObject";

type UseFindManyProps = {
    canSearch: (where: any) => boolean; // Checking against "where" can be useful to ensure that it has been updated
    controlsUrl?: boolean; // If true, URL search params update to reflect search state
    searchType: SearchType | `${SearchType}`;
    resolve?: (data: any) => any;
    take?: number;
    where?: Record<string, any>;
}

type FullSearchParams = Partial<SearchParams> & {
    searchString: string;
    sortBy: string;
    timeFrame: TimeFrame | undefined;
    where: Record<string, any>;
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
 * Finds updated value for sortBy
 * @returns the sortBy param if it's valid, or searchParams.defaultSortBy if it's not
 */
const updateSortBy = (searchParams: Partial<SearchParams>, sortBy: string) => {
    if (typeof sortBy === "string") {
        if (exists(searchParams.sortByOptions) && sortBy in searchParams.sortByOptions) {
            return sortBy;
        } else {
            return searchParams.defaultSortBy ?? "";
        }
    }
};

/**
 * Logic for displaying search options and querying a findMany endpoint
 */
export const useFindMany = <DataType extends Record<string, any>>({
    canSearch,
    controlsUrl = true,
    searchType,
    resolve,
    take = 20,
    where,
}: UseFindManyProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    const stableResolve = useStableCallback(resolve);
    const stableWhere = useStableObject(where);

    const params = useRef<FullSearchParams>({
        searchString: "",
        sortBy: "",
        timeFrame: undefined,
        where: stableWhere ?? {},
    });
    const [paramsReady, setParamsReady] = useState<boolean>(false);

    /**
     * When sort and filter options change, update the URL
     */
    const updateUrl = useCallback(() => {
        if (!controlsUrl) return;
        const { searchString, sortBy, timeFrame } = params.current;
        addSearchParams(setLocation, {
            search: searchString.length > 0 ? searchString : undefined,
            sort: sortBy,
            time: timeFrame ? {
                after: timeFrame.after?.toISOString() ?? "",
                before: timeFrame.before?.toISOString() ?? "",
            } : undefined,
        });
    }, [controlsUrl, setLocation]);

    /**
     * Read URL search params on first render
     */
    useEffect(() => {
        if (!controlsUrl) return;
        const searchParams = parseSearchParams();
        if (typeof searchParams.search === "string") params.current.searchString = searchParams.search;
        if (typeof searchParams.sort === "string") {
            params.current.sortBy = updateSortBy(params.current, searchParams.sort);
        }
        if (typeof searchParams.time === "object" &&
            !Array.isArray(searchParams.time) &&
            Object.prototype.hasOwnProperty.call(searchParams.time, "after") &&
            Object.prototype.hasOwnProperty.call(searchParams.time, "before")) {
            params.current.timeFrame = {
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            };
        }
        updateUrl();
    }, [controlsUrl, updateUrl]);

    /**
     * Update params when search conditions change
     */
    useEffect(() => {
        const fetchParams = async () => {
            const newParams = searchTypeToParams[searchType];
            if (!newParams) return;
            setParamsReady(false);
            const resolvedParams = await newParams();
            const sortBy = updateSortBy(resolvedParams, params.current.sortBy);
            params.current = {
                ...params.current,
                ...resolvedParams,
                sortBy,
                where: stableWhere ?? {},
            };
            setParamsReady(canSearch(params.current) && sortBy?.length > 0);
        };
        fetchParams();
    }, [canSearch, searchType, stableWhere]);

    /**
     * Cursor for pagination. Resets when search params change.
     * In most cases, there is only one cursor (and thus only one "after" field). But when searching unions, there can be multiple.
     * To decide the name of the "after" field, we use these rules:
     * 1. "endCursor" -> "after"
     * 2. "endCursorSomeObjectType" -> "someObjectTypeAfter"
     */
    const after = useRef<Record<string, string>>({});

    const [advancedSearchParams, setAdvancedSearchParams] = useState<object | null>(null);
    const [getPageData, { data: pageData, loading, error }] = useLazyFetch<SearchQueryVariablesInput<any>, Record<string, any>>(params!.current.endpoint, {
        take,
        sortBy: params.current.sortBy,
        searchString: params.current.searchString,
        createdTimeFrame: (params.current.timeFrame && Object.keys(params.current.timeFrame).length > 0) ? {
            after: params.current.timeFrame.after?.toISOString(),
            before: params.current.timeFrame.before?.toISOString(),
        } : undefined,
        ...after.current,
        ...params.current.where,
        ...advancedSearchParams,
    } as any);
    // Display a snack error message if there is an error
    useDisplayServerError(error);
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

    // On search filters/sort change, reset the page
    useEffect(() => {
        // Reset the pagination cursor and hasMore
        after.current = {};
        setHasMore(true);
        // Only get data if we can search
        if (paramsReady) getPageData();
    }, [advancedSearchParams, paramsReady, getPageData]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!paramsReady || !pageData) return;
        if (!pageData.pageInfo) return [];
        if (pageData.pageInfo?.hasNextPage) {
            // Find every field starting with "endCursor" and add the appropriate "after" field
            after.current = {};
            for (const [key, value] of Object.entries(pageData.pageInfo)) {
                // "endCursor" -> "after"
                if (key === "endCursor") (after.current as any).after = value;
                // "endCursorSomeObjectType" -> "someObjectTypeAfter"
                else if (key.startsWith("endCursor")) {
                    const afterKeyLower = lowercaseFirstLetter(key.replace("endCursor", ""));
                    (after.current as any)[afterKeyLower + "After"] = value;
                }
            }
            getPageData();
        } else {
            setHasMore(false);
        }
    }, [getPageData, pageData, paramsReady]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        const parsedData = parseData(pageData, stableResolve);
        console.log("got parsed data", parsedData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (Object.keys(after.current).length > 0) {
            setAllData(curr => [...curr, ...parsedData]);
        } else {
            setAllData(parsedData);
        }
    }, [pageData, stableResolve]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => listToAutocomplete(allData as any, getUserLanguages(session)), [allData, session]);

    const setSortBy = useCallback((sortBy: string) => {
        setParamsReady(false);
        params.current.sortBy = updateSortBy(params.current, sortBy);
        updateUrl();
        setParamsReady(canSearch(params.current));
    }, [canSearch, updateUrl]);

    const setSearchString = useCallback((searchString: string) => {
        setParamsReady(false);
        params.current.searchString = searchString;
        updateUrl();
        setParamsReady(canSearch(params.current));
    }, [canSearch, updateUrl]);

    const setTimeFrame = useCallback((timeFrame: TimeFrame | undefined) => {
        setParamsReady(false);
        params.current.timeFrame = timeFrame;
        updateUrl();
        setParamsReady(canSearch(params.current));
    }, [canSearch, updateUrl]);

    return {
        advancedSearchParams,
        advancedSearchSchema: params?.current?.advancedSearchSchema,
        allData,
        autocompleteOptions,
        defaultSortBy: params?.current?.defaultSortBy,
        hasMore,
        loading,
        loadMore,
        pageData,
        parseData,
        searchString: params?.current?.searchString,
        setAdvancedSearchParams,
        setAllData,
        setSortBy,
        setSearchString,
        setTimeFrame,
        sortBy: params?.current?.sortBy,
        sortByOptions: params?.current?.sortByOptions,
        timeFrame: params?.current?.timeFrame,
    };
};
