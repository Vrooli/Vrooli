import { addSearchParams, exists, lowercaseFirstLetter, parseSearchParams, TimeFrame, useLocation } from "@local/shared";
import { SearchQueryVariablesInput } from "components/lists/types";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { AutocompleteOption } from "types";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { SessionContext } from "utils/SessionContext";
import { useDebounce } from "./useDebounce";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";
import { useStableObject } from "./useStableObject";

type UseFindManyProps = {
    canSearch: (where: any) => boolean; // Checking against "where" can be useful to ensure that it has been updated
    controlsUrl?: boolean; // If true, URL search params update to reflect search state
    debounceMs?: number;
    searchType: SearchType | `${SearchType}`;
    resolve?: (data: any) => any;
    take?: number;
    where?: Record<string, any>;
}

type FullSearchParams = Partial<SearchParams> & {
    hasMore: boolean;
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
 * Determines if we have sufficient search params to perform a search
 * @param canSearch function for determining if we can search
 * @param params search params
 */
const readyToSearch = (canSearch: (where: any) => boolean, params: FullSearchParams) => {
    const { endpoint, hasMore, sortBy } = params;
    if (!hasMore) return false;
    if (!endpoint || endpoint.length === 0) return false;
    if (!sortBy || sortBy.length === 0) return false;
    if (!canSearch(params.where)) return false;
    return true;
};

/**
 * Logic for displaying search options and querying a findMany endpoint
 */
export const useFindMany = <DataType extends Record<string, any>>({
    canSearch,
    controlsUrl = true,
    debounceMs = 100,
    searchType,
    resolve,
    take = 20,
    where,
}: UseFindManyProps, deps: any[] = []) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    // Limit rerenders of params
    const stableCanSearch = useStableCallback(canSearch);
    const stableResolve = useStableCallback(resolve);
    const stableWhere = useStableObject(where);

    // Store search params
    const params = useRef<FullSearchParams>({
        endpoint: "",
        hasMore: true,
        searchString: "",
        sortBy: "",
        timeFrame: undefined,
        where: stableWhere ?? {},
    });

    // Handle URL search params
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
     * Cursor for pagination. Resets when search params change.
     * In most cases, there is only one cursor (and thus only one "after" field). But when searching unions, there can be multiple.
     * To decide the name of the "after" field, we use these rules:
     * 1. "endCursor" -> "after"
     * 2. "endCursorSomeObjectType" -> "someObjectTypeAfter"
     */
    const after = useRef<Record<string, string>>({});

    // Handle advanced search params
    const [advancedSearchParams, setAdvancedSearchParams] = useState<object | null>(null);

    // Handle fetching data
    const [getPageData, { data: pageData, loading, errors }] = useLazyFetch<SearchQueryVariablesInput<any>, Record<string, any>>({
        endpoint: params.current.endpoint,
        inputs: {
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
        },
    } as any);
    useDisplayServerError(errors);
    const debouncedGetPageData = useDebounce(() => {
        getPageData();
    }, debounceMs);

    const [allData, setAllData] = useState<DataType[]>(() => {
        // TODO Check if we just navigated back to this page from an object page. If so, use results stored in sessionStorage. Also TODO for storing results in sessionStorage
        const lastPath = sessionStorage.getItem("lastPath");
        const lastSearchParams = sessionStorage.getItem("lastSearchParams");
        console.log("lastPath", lastPath);
        console.log("lastSearchParams", lastSearchParams);
        return [];
    });

    // On search filters/sort change, reset the page
    useEffect(() => {
        // Reset the pagination cursor and hasMore
        after.current = {};
        params.current.hasMore = true;
        updateUrl();
        // Only get data if we can search
        if (readyToSearch(stableCanSearch, params.current)) debouncedGetPageData({});
    }, [advancedSearchParams, stableCanSearch, debouncedGetPageData, updateUrl]);

    /**
     * Update params when search conditions change
     */
    useEffect(() => {
        const fetchParams = async () => {
            const newParams = searchTypeToParams[searchType];
            if (!newParams) return;
            const resolvedParams = await newParams();
            const sortBy = updateSortBy(resolvedParams, params.current.sortBy);
            after.current = {};
            params.current = {
                ...params.current,
                ...resolvedParams,
                sortBy,
                hasMore: true,
            };
            updateUrl();
            if (readyToSearch(stableCanSearch, params.current)) debouncedGetPageData({});
        };
        fetchParams();
    }, [stableCanSearch, debouncedGetPageData, updateUrl, searchType]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!readyToSearch(stableCanSearch, params.current) || !pageData) return;
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
            debouncedGetPageData({});
        } else {
            params.current.hasMore = false;
        }
    }, [stableCanSearch, debouncedGetPageData, pageData]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        const parsedData = parseData(pageData, stableResolve);
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
        params.current.sortBy = updateSortBy(params.current, sortBy);
        updateUrl();
        debouncedGetPageData({});
    }, [debouncedGetPageData, updateUrl]);

    const setSearchString = useCallback((searchString: string) => {
        params.current.searchString = searchString;
        updateUrl();
        debouncedGetPageData({});
    }, [debouncedGetPageData, updateUrl]);

    const setTimeFrame = useCallback((timeFrame: TimeFrame | undefined) => {
        params.current.timeFrame = timeFrame;
        updateUrl();
        debouncedGetPageData({});
    }, [debouncedGetPageData, updateUrl]);

    return {
        advancedSearchParams,
        advancedSearchSchema: params?.current?.advancedSearchSchema,
        allData,
        autocompleteOptions,
        defaultSortBy: params?.current?.defaultSortBy,
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
