import { exists } from "@local/utils";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useCustomLazyQuery } from "../../api";
import { listToAutocomplete } from "../display/listTools";
import { getUserLanguages } from "../display/translationTools";
import { addSearchParams, parseSearchParams, useLocation } from "../route";
import { searchTypeToParams } from "../search/objectToSearch";
import { SessionContext } from "../SessionContext";
import { useDisplayApolloError } from "./useDisplayApolloError";
import { useStableObject } from "./useStableObject";
export const parseData = (data, resolve) => {
    if (!data)
        return [];
    const queryData = (Object.keys(data).length === 1 && !["success", "count"].includes(Object.keys(data)[0])) ? Object.values(data)[0] : data;
    if (resolve)
        return resolve(queryData);
    if (!queryData || !queryData.edges)
        return [];
    return queryData.edges.map((edge, index) => edge.node);
};
export const useFindMany = ({ canSearch, searchType, resolve, take = 20, where, }) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const stableResolve = useStableObject(resolve);
    const stableWhere = useStableObject(where);
    const [params, setParams] = useState({});
    useEffect(() => {
        const fetchParams = async () => {
            const newParams = searchTypeToParams[searchType];
            if (!newParams)
                return;
            const resolvedParams = await newParams();
            setParams({ ...resolvedParams, where: stableWhere });
        };
        fetchParams();
    }, [searchType, stableWhere]);
    const [sortBy, setSortBy] = useState(params?.defaultSortBy ?? "");
    const [searchString, setSearchString] = useState("");
    const [timeFrame, setTimeFrame] = useState(undefined);
    useEffect(() => {
        const searchParams = parseSearchParams();
        if (typeof searchParams.search === "string")
            setSearchString(searchParams.search);
        if (typeof searchParams.sort === "string") {
            if (exists(params?.sortByOptions) && searchParams.sort in params.sortByOptions) {
                setSortBy(searchParams.sort);
            }
            else {
                setSortBy(params?.defaultSortBy ?? "");
            }
        }
        if (typeof searchParams.time === "object" &&
            !Array.isArray(searchParams.time) &&
            searchParams.time.hasOwnProperty("after") &&
            searchParams.time.hasOwnProperty("before")) {
            setTimeFrame({
                after: new Date(searchParams.time.after),
                before: new Date(searchParams.time.before),
            });
        }
    }, [params?.defaultSortBy, params?.sortByOptions]);
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
    const after = useRef(undefined);
    const [advancedSearchParams, setAdvancedSearchParams] = useState(null);
    const [getPageData, { data: pageData, loading, error }] = useCustomLazyQuery(params.query, {
        variables: {
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
        },
        errorPolicy: "all",
    });
    useDisplayApolloError(error);
    const [allData, setAllData] = useState(() => {
        const lastPath = sessionStorage.getItem("lastPath");
        const lastSearchParams = sessionStorage.getItem("lastSearchParams");
        console.log("lastPath", lastPath);
        console.log("lastSearchParams", lastSearchParams);
        return [];
    });
    const [hasMore, setHasMore] = useState(true);
    useEffect(() => {
        console.log("resetting hasMore");
        setHasMore(true);
    }, [advancedSearchParams, searchString, sortBy, timeFrame]);
    useEffect(() => {
        after.current = undefined;
        if (canSearch && sortBy?.length > 0)
            getPageData();
    }, [advancedSearchParams, canSearch, searchString, sortBy, timeFrame, getPageData]);
    const loadMore = useCallback(() => {
        if (!pageData || !canSearch)
            return;
        if (!pageData.pageInfo)
            return [];
        if (pageData.pageInfo?.hasNextPage) {
            const { endCursor } = pageData.pageInfo;
            if (endCursor) {
                after.current = endCursor;
                getPageData();
            }
        }
        else {
            setHasMore(false);
        }
    }, [canSearch, getPageData, pageData]);
    useEffect(() => {
        const parsedData = parseData(pageData, stableResolve);
        console.log("got parsed data", parsedData);
        if (!parsedData) {
            setAllData([]);
            return;
        }
        if (after.current) {
            setAllData(curr => [...curr, ...parsedData]);
        }
        else {
            setAllData(parsedData);
        }
    }, [pageData, stableResolve]);
    const autocompleteOptions = useMemo(() => {
        console.log("LISTTOAUTOCOMPLETE allData", allData);
        return listToAutocomplete(allData, getUserLanguages(session)).sort((a, b) => {
            return b.bookmarks - a.bookmarks;
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
//# sourceMappingURL=useFindMany.js.map