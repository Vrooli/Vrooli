import { AutocompleteOption, ListObject, SearchType, TimeFrame, addToArray, deepClone, deleteArrayIndex, exists, lowercaseFirstLetter, parseSearchParams, updateArray } from "@local/shared";
import { SearchQueryVariablesInput } from "components/lists/types";
import { SessionContext } from "contexts";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { SetLocation, addSearchParams, useLocation } from "route";
import { listToAutocomplete } from "utils/display/listTools";
import { getUserLanguages } from "utils/display/translationTools";
import { searchTypeToParams } from "utils/search/objectToSearch";
import { SearchParams } from "utils/search/schemas/base";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";
import { useStableObject } from "./useStableObject";

type UseFindManyProps = {
    canSearch?: (where: any) => boolean; // Checking against "where" can be useful to ensure that it has been updated
    controlsUrl?: boolean; // If true, URL search params update to reflect search state
    searchType: SearchType | `${SearchType}`;
    resolve?: (data: any) => any;
    take?: number;
    where?: Record<string, any>;
}

export type UseFindManyResult<DataType extends Record<string, any>> = {
    addItem: (item: DataType) => unknown;
    advancedSearchParams: object | null;
    advancedSearchSchema: any;
    allData: DataType[];
    autocompleteOptions: AutocompleteOption[];
    defaultSortBy: string;
    loading: boolean;
    loadMore: () => unknown;
    removeItem: (id: string, idField?: string) => unknown;
    searchString: string;
    searchType: SearchType | `${SearchType}`;
    setAdvancedSearchParams: (advancedSearchParams: object | null) => unknown;
    setAllData: React.Dispatch<React.SetStateAction<DataType[]>>;
    setSortBy: (sortBy: string) => unknown;
    setSearchString: (searchString: string) => unknown;
    setTimeFrame: (timeFrame: TimeFrame | undefined) => unknown;
    sortBy: string;
    sortByOptions: Record<string, string>;
    timeFrame: TimeFrame | undefined;
    updateItem: (item: DataType, idField?: string) => unknown;
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

type GetUrlSearchParamsResult = {
    searchString: string;
    sortBy: string;
    timeFrame: TimeFrame | undefined;
}

const DEFAULT_TAKE = 20;

/**
 * Helper method for converting fetched data to an array of object data
 * @param data data returned from a findMany query
 * @param resolve function for resolving the data
 * @returns List of objects from the data, without any pagination information
 */
export function parseData(
    data: object | undefined,
    resolve?: (data: any) => object[],
) {
    // Ensure data is properly formatted
    if (!data || typeof data !== "object") return [];
    // If there is a custom resolver, use it
    if (resolve) return resolve(data);
    // Otherwise, treat as typically-shaped paginated data
    if (!Array.isArray((data as { edges?: unknown }).edges)) return [];
    return (data as { edges: any[] }).edges.map((edge) => edge.node);
}

/**
 * Finds updated value for sortBy
 * @returns the sortBy param if it's valid, or searchParams.defaultSortBy if it's not
 */
export function updateSortBy(searchParams: Pick<FullSearchParams, "defaultSortBy" | "sortByOptions">, sortBy: string) {
    if (typeof sortBy === "string") {
        if (exists(searchParams.sortByOptions) && sortBy in searchParams.sortByOptions) {
            return sortBy;
        } else {
            return searchParams.defaultSortBy ?? "";
        }
    }
}

/**
 * Determines if we have sufficient search params to perform a search
 * @param params search params
 */
export function readyToSearch({ canSearch, findManyEndpoint, hasMore, loading, sortBy }: FullSearchParams) {
    return Boolean(
        canSearch &&
        !loading &&
        hasMore &&
        findManyEndpoint && findManyEndpoint.length > 0 &&
        sortBy && sortBy.length > 0,
    );
}

/**
 * Extracts and returns basic search-related parameters from the URL, if we're currently
 * controlling the URL. This is the case when the search results are the main focus of the page, 
 * and this is not in a modal or other secondary context.
 *
 * @param controlsUrl Indicates whether the URL is currently under control, which 
 * impacts whether search parameters should be read from the URL.
 * @param searchType Specifies the type of search, which is used to determine 
 * how the 'sort' parameter is validated and processed.
 *
 * @returns An object containing the extracted search parameters:
 *         - searchString: Extracted search string; defaults to an empty string if not specified.
 *         - sortBy: Validated sorting parameter based on the search type; defaults to an empty string.
 *         - timeFrame: Optional object containing 'after' and/or 'before' date limits; included if either
 *                      date is validly specified in the URL.
 */
export function getUrlSearchParams(controlsUrl: boolean, searchType: string): GetUrlSearchParamsResult {
    const result: GetUrlSearchParamsResult = {
        searchString: "",
        sortBy: "",
        timeFrame: undefined,
    };
    if (!controlsUrl) return result;
    const searchParams = parseSearchParams();
    // Find searchString
    if (typeof searchParams.search === "string") result.searchString = searchParams.search;
    // Find sortBy
    const searchParamsConfig = searchTypeToParams[searchType];
    if (typeof searchParamsConfig === "function") {
        result.sortBy = updateSortBy(searchParamsConfig(), typeof searchParams.sort === "string" ? searchParams.sort : "");
    } else {
        console.warn(`Invalid search type provided: ${searchType}`);
    }
    // Find timeFrame
    if (typeof searchParams.time === "object" && !Array.isArray(searchParams.time)) {
        const timeFrame: TimeFrame = {};
        if (Object.prototype.hasOwnProperty.call(searchParams.time, "after")) {
            timeFrame.after = new Date((searchParams.time as { after: string }).after);
        }
        if (Object.prototype.hasOwnProperty.call(searchParams.time, "before")) {
            timeFrame.before = new Date((searchParams.time as { before: string }).before);
        }
        if (Object.keys(timeFrame).length > 0) {
            result.timeFrame = timeFrame;
        }
    }
    return result;
}

/**
 * The opposite of getUrlSearchParams. Updates the URL with the current search parameters.
 * @param controlsUrl Indicates whether the URL is currently under control, which
 * impacts whether search parameters should be written to the URL.
 * @param searchString The current search string.
 * @param sortBy The current sorting parameter.
 * @param timeFrame The current time frame.
 * @param setLocation Function for updating the URL.
 */
export function updateSearchUrl(
    controlsUrl: boolean,
    { searchString, sortBy, timeFrame }: FullSearchParams,
    setLocation: SetLocation,
) {
    if (!controlsUrl) return;
    addSearchParams(setLocation, {
        search: searchString.length > 0 ? searchString : undefined,
        sort: sortBy,
        time: timeFrame ? {
            after: timeFrame.after?.toISOString() ?? "",
            before: timeFrame.before?.toISOString() ?? "",
        } : undefined,
    });
}

/**
 * Logic for displaying search options and querying a findMany endpoint
 */
export function useFindMany<DataType extends Record<string, any>>({
    canSearch,
    controlsUrl = true,
    searchType,
    resolve,
    take = DEFAULT_TAKE,
    where,
}: UseFindManyProps): UseFindManyResult<DataType> {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();

    // Limit rerenders of params
    const stableCanSearch = useStableCallback(canSearch);
    const stableResolve = useStableCallback(resolve);
    const stableWhere = useStableObject(where);
    const controlsUrlRef = useRef(controlsUrl);
    controlsUrlRef.current = controlsUrl;

    // Store search params
    const params = useRef<FullSearchParams>({
        advancedSearchParams: null,
        canSearch: typeof stableCanSearch === "function" ? stableCanSearch(stableWhere ?? {}) : true,
        findManyEndpoint: "",
        hasMore: true,
        loading: false,
        ...getUrlSearchParams(controlsUrlRef.current, searchType),
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
    const [getPageData, { data: pageData, loading }] = useLazyFetch<SearchQueryVariablesInput<string>, Record<string, any>>({});
    /** Function for fetching new data, only when there isn't data currently being fetched */
    const getData = useCallback(() => {
        if (!readyToSearch(params.current)) return;
        lastParams.current = deepClone(params.current);
        // params.current.loading = true;
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
        }, { endpointOverride: params.current.findManyEndpoint as string });
    }, [getPageData, take]);
    useEffect(() => {
        params.current.loading = loading;
    }, [loading]);

    useEffect(function fetchWhenCanSearchIsTrue() {
        const oldCanSearch = params.current.canSearch;
        const newCanSearch = typeof stableCanSearch === "function" ? stableCanSearch(stableWhere) : true;
        params.current.canSearch = newCanSearch;
        // Get data if we couldn't before
        if (readyToSearch(params.current) && !oldCanSearch && newCanSearch) getData();
    }, [getData, stableCanSearch, stableWhere]);

    const [allData, setAllData] = useState<DataType[]>([]);
    const removeItem = useCallback((id: string, idField?: string) => {
        setAllData(curr => deleteArrayIndex(curr, curr.findIndex(i => i[idField ?? "id"] === id)));
    }, []);
    const updateItem = useCallback((item: DataType, idField?: string) => {
        setAllData(curr => updateArray(curr, curr.findIndex(i => i[idField ?? "id"] === item[idField ?? "id"]), item));
    }, []);
    const addItem = useCallback((item: DataType) => {
        setAllData(curr => addToArray(curr, item));
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
        updateSearchUrl(controlsUrlRef.current, params.current, setLocation);
        // Fetch new data
        getData();
        // Update state
        setSearchParams(advancedSearchParams);
    }, [getData, setLocation]);

    /**
     * Update params when search conditions change
     */
    useEffect(() => {
        const newParamsFunc = searchTypeToParams[searchType];
        if (typeof newParamsFunc !== "function") return;
        const newParams = newParamsFunc();
        if (!newParams) return;
        const sortBy = updateSortBy(newParams, params.current.sortBy);
        after.current = {};
        params.current = {
            ...params.current,
            ...newParams,
            sortBy,
            hasMore: true,
            where: stableWhere ?? {},
        };
        setAllData([]);
        updateSearchUrl(controlsUrlRef.current, params.current, setLocation);
        getData();
    }, [searchType, stableWhere, getData, setLocation]);

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
            getData();
        } else {
            params.current.hasMore = false;
        }
    }, [getData, pageData]);

    // Parse newly fetched data, and determine if it should be appended to the existing data
    useEffect(() => {
        // params.current.loading = false;
        // If params have changed since this fetch started, invalidate it and refetch
        function toCompareShape(obj: any) {
            return {
                advancedSearchParams: obj.advancedSearchParams,
                findManyEndpoint: obj.findManyEndpoint,
                searchString: obj.searchString,
                sortBy: obj.sortBy,
                timeFrame: obj.timeFrame,
                where: obj.where,
            };
        }
        const last = toCompareShape(lastParams.current);
        const current = toCompareShape(params.current);
        if (JSON.stringify(last) !== JSON.stringify(current)) {
            setAllData([]);
            getData();
            return;
        }
        // Parse data
        const parsedData = parseData(pageData, stableResolve);
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
    }, [getData, pageData, stableResolve]);

    const autocompleteOptions: AutocompleteOption[] = useMemo(() => listToAutocomplete(allData as unknown as ListObject[], getUserLanguages(session)), [allData, session]);

    const setSortBy = useCallback((sortBy: string) => {
        const newSortBy = updateSortBy(params.current, sortBy);
        if (newSortBy === params.current.sortBy) return;
        params.current.sortBy = newSortBy;
        setAllData([]);
        updateSearchUrl(controlsUrlRef.current, params.current, setLocation);
        getData();
    }, [getData, setLocation]);

    const setSearchString = useCallback((searchString: string) => {
        if (searchString === params.current.searchString) return;
        params.current.searchString = searchString;
        setAllData([]);
        updateSearchUrl(controlsUrlRef.current, params.current, setLocation);
        getData();
    }, [getData, setLocation]);

    const setTimeFrame = useCallback((timeFrame: TimeFrame | undefined) => {
        if (timeFrame === params.current.timeFrame) return;
        params.current.timeFrame = timeFrame;
        setAllData([]);
        getData();
    }, [getData]);

    return {
        addItem,
        advancedSearchParams,
        advancedSearchSchema: params?.current?.advancedSearchSchema,
        allData,
        autocompleteOptions,
        defaultSortBy: params?.current?.defaultSortBy,
        loading,
        loadMore,
        removeItem,
        searchString: params?.current?.searchString,
        searchType,
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
}
