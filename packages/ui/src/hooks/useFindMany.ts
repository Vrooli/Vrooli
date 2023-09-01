import { deepClone, exists, lowercaseFirstLetter, TimeFrame } from "@local/shared";
import { SearchQueryVariablesInput } from "components/lists/types";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { AutocompleteOption } from "types";
import { ListObject, listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { SearchType, searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";
import { useStableObject } from "./useStableObject";

type UseFindManyProps = {
    canSearch?: (where: any) => boolean; // Checking against "where" can be useful to ensure that it has been updated
    controlsUrl?: boolean; // If true, URL search params update to reflect search state
    debounceMs?: number;
    searchType: SearchType | `${SearchType}`;
    resolve?: (data: any, searchType: SearchType | `${SearchType}`) => any;
    take?: number;
    where?: Record<string, any>;
}

type FullSearchParams = Partial<SearchParams> & {
    advancedSearchParams: object | null;
    canSearch: boolean;
    hasMore: boolean;
    loading: boolean;
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
export const parseData = (
    data: any,
    searchType: SearchType | `${SearchType}`,
    resolve?: (data: any, searchType: SearchType | `${SearchType}`) => object[],
) => {
    console.log("parseData 1", data, typeof resolve);
    if (!data) return [];
    // query result is always returned as an object with a single key (the endpoint name), where 
    // the value is the actual data. If this is not the case, then (hopefully) it was already 
    // deconstructed earlier in the chain
    const queryData: any = (Object.keys(data).length === 1 && !["success", "count"].includes(Object.keys(data)[0])) ? Object.values(data)[0] : data;
    // If there is a custom resolver, use it
    console.log("parseData 2", queryData, typeof resolve);
    if (resolve) return resolve(queryData, searchType);
    // Otherwise, treat as typically-shaped paginated data
    if (!queryData || !queryData.edges) return [];
    return queryData.edges.map((edge: any) => edge.node);
};

/**
 * Finds updated value for sortBy
 * @returns the sortBy param if it's valid, or searchParams.defaultSortBy if it's not
 */
const updateSortBy = (searchParams: Pick<FullSearchParams, "defaultSortBy" | "sortByOptions">, sortBy: string) => {
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
const readyToSearch = (params: FullSearchParams) => {
    return params.canSearch &&
        !params.loading &&
        params.hasMore &&
        params.endpoint && params.endpoint.length > 0 &&
        params.sortBy && params.sortBy.length > 0;
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
}: UseFindManyProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    console.log("search type", searchType);

    // Limit rerenders of params
    const stableCanSearch = useStableCallback(canSearch);
    const stableResolve = useStableCallback(resolve);
    const stableWhere = useStableObject(where);

    const getUrlParams = useCallback(() => {
        const result: { searchString?: string, sortBy?: string, timeFrame?: TimeFrame } = {};
        if (!controlsUrl) return {};
        const searchParams = parseSearchParams();
        if (typeof searchParams.search === "string") result.searchString = searchParams.search;
        if (typeof searchParams.sort === "string") {
            result.sortBy = updateSortBy(searchTypeToParams[searchType](), searchParams.sort);
        }
        if (typeof searchParams.time === "object" &&
            !Array.isArray(searchParams.time) &&
            Object.prototype.hasOwnProperty.call(searchParams.time, "after") &&
            Object.prototype.hasOwnProperty.call(searchParams.time, "before")) {
            result.timeFrame = {
                after: new Date((searchParams.time as any).after),
                before: new Date((searchParams.time as any).before),
            };
        }
        return result;
    }, [controlsUrl, searchType]);

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

    // Store search params
    const params = useRef<FullSearchParams>({
        advancedSearchParams: null,
        canSearch: typeof stableCanSearch === "function" ? stableCanSearch(stableWhere ?? {}) : true,
        endpoint: "",
        hasMore: true,
        // Start loading as "true" if we're allowed to search, to prevent flicker
        loading: typeof stableCanSearch === "function" ? stableCanSearch(stableWhere ?? {}) : true,
        searchString: getUrlParams().searchString ?? "",
        sortBy: getUrlParams().sortBy ?? "",
        timeFrame: getUrlParams().timeFrame,
        where: stableWhere ?? {},
    });
    // Store params for the last search
    const lastParams = useRef<FullSearchParams>(deepClone(params.current));

    /**
     * Cursor for pagination. Resets when search params change.
     * In most cases, there is only one cursor (and thus only one "after" field). But when searching unions, there can be multiple.
     * To decide the name of the "after" field, we use these rules:
     * 1. "endCursor" -> "after"
     * 2. "endCursorSomeObjectType" -> "someObjectTypeAfter"
     */
    const after = useRef<Record<string, string>>({});

    // Handle fetching data
    const [getPageData, { data: pageData, loading, errors }] = useLazyFetch<SearchQueryVariablesInput<string>, Record<string, any>>({});
    useDisplayServerError(errors);
    /** Function for fetching new data, only when there isn't data currently being fetched */
    const getData = useCallback(() => {
        if (!readyToSearch(params.current)) return;
        console.log("getting data and setting lastParms.current", params.current);
        lastParams.current = deepClone(params.current);
        params.current.loading = true;
        getPageData({
            take,
            sortBy: params.current.sortBy,
            searchString: params.current.searchString,
            createdTimeFrame: (params.current.timeFrame && Object.keys(params.current.timeFrame).length > 0) ? {
                after: params.current.timeFrame.after?.toISOString(),
                before: params.current.timeFrame.before?.toISOString(),
            } : undefined,
            ...after.current,
            ...params.current.where,
            ...params.current.advancedSearchParams,
        }, params.current.endpoint as string);
    }, [getPageData, take]);
    useEffect(() => {
        console.log("settings params.current.loading", loading);
        params.current.loading = loading;
    }, [loading]);

    // Fetch data when canSearch changes to true
    useEffect(() => {
        const oldCanSearch = params.current.canSearch;
        const newCanSearch = typeof stableCanSearch === "function" ? stableCanSearch(stableWhere) : true;
        params.current.canSearch = newCanSearch;
        // Get data if we couldn't before
        console.log("calling getData from here? 1", readyToSearch(params.current), oldCanSearch, newCanSearch, params.current.endpoint);
        if (readyToSearch(params.current) && !oldCanSearch && newCanSearch) getData();
    }, [getData, stableCanSearch, stableWhere]);

    const [allData, setAllData] = useState<DataType[]>([]);
    const updateItem = useCallback((item: DataType) => {
        setAllData(curr => {
            const index = curr.findIndex(i => i.id === item.id);
            if (index === -1) return curr;
            const copy = [...curr];
            copy[index] = item;
            return copy;
        });
    }, []);
    const removeItem = useCallback((id: string, idField?: string) => {
        console.log("in usefindmany removeItem", id);
        setAllData(curr => {
            const index = curr.findIndex(i => i[idField ?? "id"] === id);
            console.log("in usefindmany removeItem - found item?", index);
            if (index === -1) return curr;
            const copy = [...curr];
            copy.splice(index, 1);
            return copy;
        });
    }, []);

    // Handle advanced search params
    const [advancedSearchParams, setSearchParams] = useState<object | null>(params.current.advancedSearchParams);
    const setAdvancedSearchParams = useCallback((advancedSearchParams: object | null) => {
        const oldAdvancedSearchParams = params.current.advancedSearchParams;
        // Update params
        params.current.advancedSearchParams = advancedSearchParams;
        // If the params haven't changed, return
        if (JSON.stringify(oldAdvancedSearchParams) === JSON.stringify(advancedSearchParams)) return;
        // Reset the pagination cursor and hasMore
        after.current = {};
        params.current.hasMore = true;
        // Update the URL
        updateUrl();
        // Fetch new data
        console.log("calling getData from here? 2", readyToSearch(params.current), params.current.endpoint, advancedSearchParams);
        getData();
        // Update state
        console.log("setting search params from useFindMany", advancedSearchParams);
        setSearchParams(advancedSearchParams);
    }, [getData, updateUrl]);

    /**
     * Update params when search conditions change
     */
    useEffect(() => {
        const newParams = searchTypeToParams[searchType]();
        if (!newParams) return;
        const sortBy = updateSortBy(newParams, params.current.sortBy);
        after.current = {};
        params.current = {
            ...params.current,
            ...newParams,
            sortBy,
            hasMore: true,
        };
        setAllData([]);
        updateUrl();
        console.log("calling getData from here? 3", readyToSearch(params.current), params.current.endpoint);
        getData();
    }, [updateUrl, searchType, getData]);

    // Fetch more data by setting "after"
    const loadMore = useCallback(() => {
        if (!readyToSearch(params.current) || !pageData) return;
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
            console.log("calling getData from here? 4", readyToSearch(params.current), params.current.endpoint);
            getData();
        } else {
            params.current.hasMore = false;
        }
    }, [getData, pageData]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        // params.current.loading = false;
        // If params have changed since this fetch started, invalidate it and refetch
        const last = {
            advancedSearchParams: lastParams.current.advancedSearchParams,
            endpoint: lastParams.current.endpoint,
            searchString: lastParams.current.searchString,
            sortBy: lastParams.current.sortBy,
            timeFrame: lastParams.current.timeFrame,
            where: lastParams.current.where,
        };
        const current = {
            advancedSearchParams: params.current.advancedSearchParams,
            endpoint: params.current.endpoint,
            searchString: params.current.searchString,
            sortBy: params.current.sortBy,
            timeFrame: params.current.timeFrame,
            where: params.current.where,
        };
        if (JSON.stringify(last) !== JSON.stringify(current)) {
            setAllData([]);
            console.log("calling getData from here? 5", readyToSearch(params.current), params.current.endpoint);
            getData();
            return;
        }
        // Parse data
        const parsedData = parseData(pageData, searchType, stableResolve);
        console.log("got parsed data", parsedData, current, searchType);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        // If there is "after" data, append it to the existing data
        if (Object.keys(after.current).length > 0) {
            // In case some of the data is already in allData, we must make sure to not add duplicates. 
            // From testing, this hash-based method is significantly faster than other methods like Set or filtering a joined array. 
            setAllData(curr => {
                const hash = {};
                // Create hash from curr data
                for (const item of curr) {
                    hash[item.id] = item;
                }
                // Add unique items from parsedData
                for (const item of parsedData) {
                    if (!hash[item.id]) {
                        hash[item.id] = item;
                    }
                }
                return Object.values(hash);
            });
        }
        // Otherwise, assume this is a new search and replace the existing data
        else {
            setAllData(parsedData);
        }
    }, [getData, pageData, searchType, stableResolve]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => listToAutocomplete(allData as unknown as ListObject[], getUserLanguages(session)), [allData, session]);

    const setSortBy = useCallback((sortBy: string) => {
        const newSortBy = updateSortBy(params.current, sortBy);
        if (newSortBy === params.current.sortBy) return;
        params.current.sortBy = newSortBy;
        setAllData([]);
        updateUrl();
        console.log("calling getData from here? 6", readyToSearch(params.current));
        getData();
    }, [getData, updateUrl]);

    const setSearchString = useCallback((searchString: string) => {
        if (searchString === params.current.searchString) return;
        params.current.searchString = searchString;
        console.log("setting searcch string", searchString);
        setAllData([]);
        updateUrl();
        console.log("calling getData from here? 7", readyToSearch(params.current));
        getData();
    }, [getData, updateUrl]);

    const setTimeFrame = useCallback((timeFrame: TimeFrame | undefined) => {
        if (timeFrame === params.current.timeFrame) return;
        params.current.timeFrame = timeFrame;
        setAllData([]);
        updateUrl();
        console.log("calling getData from here? 8", readyToSearch(params.current));
        getData();
    }, [getData, updateUrl]);

    return {
        advancedSearchParams,
        advancedSearchSchema: params?.current?.advancedSearchSchema,
        allData,
        autocompleteOptions,
        defaultSortBy: params?.current?.defaultSortBy,
        loading,
        loadMore,
        removeItem,
        searchString: params?.current?.searchString,
        setAdvancedSearchParams,
        setAllData,
        setSortBy,
        setSearchString,
        setTimeFrame,
        sortBy: params?.current?.sortBy,
        sortByOptions: params?.current?.sortByOptions,
        timeFrame: params?.current?.timeFrame,
        updateItem,
    };
};
