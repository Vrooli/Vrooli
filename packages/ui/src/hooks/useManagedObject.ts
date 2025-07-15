import { isEqual, parseSearchParams, type ModelType, type ParseSearchParamsResult, type YouInflated } from "@vrooli/shared";
import { useCallback, useEffect, useMemo, useRef, useState, type Dispatch, type SetStateAction } from "react";
import { ServerResponseParser } from "../api/responseParser.js";
import { type FetchInputOptions } from "../api/types.js";
import { type PartialWithType } from "../types.js";
import { defaultYou, getYou } from "../utils/display/listTools.js";
import { getCookiePartialData, removeCookiePartialData, setCookiePartialData } from "../utils/localStorage.js";
import { PubSub } from "../utils/pubsub.js";
import { useFormCacheStore } from "./forms.js";
import { uiPathToApi, useLazyFetch } from "./useFetch.js";
import { useStableObject } from "./useStableObject.js";

type UrlObject = { __typename: ModelType | `${ModelType}`; id?: string };

/**
 * Maximum number of retries for fetching an object before giving up
 */
const MAX_FETCH_RETRIES = 3;

/**
 * Determines the return type based on whether a transform function is provided.
 */
type ObjectReturnType<TData extends UrlObject, TFunc> = TFunc extends (data: unknown) => infer R
    ? R
    : PartialWithType<TData>;

export type UseManagedObjectReturn<TData extends UrlObject, TFunc> = {
    id?: string;
    isLoading: boolean;
    object: ObjectReturnType<TData, TFunc>;
    permissions: YouInflated;
    setObject: Dispatch<SetStateAction<ObjectReturnType<TData, TFunc>>>;
};

interface UseManagedObjectParams<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
> {
    /** The pathname to fetch data from */
    pathname: string;
    /** If true, the hook will not attempt to fetch or load data */
    disabled?: boolean;
    /** If true, shows error snack when fetching fails */
    displayError?: boolean;
    /** If true, attempts to find existing form data in cache */
    isCreate?: boolean;
    /** Function to handle errors during fetching */
    onError?: FetchInputOptions["onError"];
    /** Function to handle invalid URL parameters */
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => unknown;
    /** If provided, uses this object instead of fetching from server or cache */
    overrideObject?: Partial<PData>;
    /** Function to transform the data, typically used for forms */
    transform?: TFunc;
}

/**
 * Hook for fetching object data from server endpoints
 */
export function useObjectData<TData extends UrlObject>(
    shouldFetch: boolean,
    pathname: string,
    onError: FetchInputOptions["onError"],
    displayError: boolean,
) {
    const [fetchData, fetchResult] = useLazyFetch<undefined, TData>({ endpoint: undefined, method: "GET" });
    const retryCountRef = useRef(0);
    const [fetchFailed, setFetchFailed] = useState(false);
    // Track if initial fetch has been done
    const initialFetchDoneRef = useRef(false);

    const fetchObjectData = useCallback(function fetchObjectDataCallback() {
        if (!shouldFetch) {
            if (process.env.NODE_ENV === "development") {
                console.info("[useObjectData] Fetch skipped: shouldFetch is false");
            }
            return false;
        }

        // Don't retry if we've already hit the maximum number of retries
        if (retryCountRef.current >= MAX_FETCH_RETRIES) {
            if (process.env.NODE_ENV === "development") {
                console.warn(`[useObjectData] Max retry count (${MAX_FETCH_RETRIES}) reached, marking as failed`);
            }
            setFetchFailed(true);
            return false;
        }

        fetchData(undefined, {
            endpointOverride: uiPathToApi(pathname),
            onError,
            displayError,
        });
        retryCountRef.current++;
        initialFetchDoneRef.current = true;
        return true;
    }, [fetchData, onError, displayError, pathname, shouldFetch]);

    useEffect(function refetchOnPathnameChangeEffect() {
        if (process.env.NODE_ENV === "development") {
            console.info("[useObjectData] Pathname changed, resetting retry count");
        }
        retryCountRef.current = 0;
        setFetchFailed(false);
        initialFetchDoneRef.current = false;
    }, [pathname]);

    // Mark as failed when errors occur
    useEffect(function trackErrorsEffect() {
        if (fetchResult.errors && fetchResult.errors.length > 0) {
            if (process.env.NODE_ENV === "development") {
                console.error("[useObjectData] Fetch errors:", fetchResult.errors);
            }
            // If we have errors and hit max retries, mark as failed
            if (retryCountRef.current >= MAX_FETCH_RETRIES) {
                if (process.env.NODE_ENV === "development") {
                    console.warn("[useObjectData] Max retries reached with errors, marking as failed");
                }
                setFetchFailed(true);
            }
        }
    }, [fetchResult.errors]);

    return {
        data: fetchResult.data,
        isLoading: fetchResult.loading,
        errors: fetchResult.errors,
        fetchObjectData,
        hasUnauthorizedError: ServerResponseParser.hasErrorCode(fetchResult, "Unauthorized"),
        fetchFailed,
    };
}

/**
 * Hook for managing object caching operations
 */
export function useObjectCache<TData extends UrlObject>(
    pathname: string,
) {
    const getCachedData = useCallback(() => {
        return getCookiePartialData<PartialWithType<TData>>(pathname);
    }, [pathname]);

    const setCachedData = useCallback((data: TData) => {
        setCookiePartialData(data, "full");
    }, [pathname]);

    const clearCache = useCallback(() => {
        removeCookiePartialData(null, pathname);
    }, [pathname]);

    return { getCachedData, setCachedData, clearCache };
}

/**
 * Helper function to apply a transform function to data if provided
 */
export function applyDataTransform<PData extends UrlObject, TData extends UrlObject>(
    data: Partial<PData> | null | undefined,
    transform?: (data: Partial<PData>) => TData,
): TData {
    if (!data) {
        return {} as TData;
    }
    
    if (transform) {
        try {
            return transform(data);
        } catch (error) {
            if (process.env.NODE_ENV === "development") {
                console.error("[applyDataTransform] Transform function threw an error:", error);
            }
            // Return the untransformed data as fallback
            return data as TData;
        }
    }
    
    return data as TData;
}

/**
 * Hook for managing form state for an object with conflict detection
 */
export function useObjectForm<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>(
    pathname: string,
    isCreate = false,
    initialData?: Partial<PData>,
    transform?: TFunc,
) {
    const getFormCacheData = useFormCacheStore(state => state.getCacheData);

    // Get stored form data if it exists (memoized)
    const storedFormData = useMemo(() => pathname && pathname.length > 1 ? getFormCacheData(pathname) as Partial<PData> : undefined, [pathname, getFormCacheData]);

    // Memoize transform function to prevent unnecessary recalculations
    const stableTransform = useCallback(transform || ((data: Partial<PData>) => data as TData), [transform]);

    // Determine initial data with priorities
    const initialObject = useMemo(() => {
        // Priority 1: Provided initial data
        if (initialData) {
            return applyDataTransform(initialData, stableTransform);
        }

        // Priority 2: Stored form data for create mode
        if (isCreate && storedFormData) {
            return applyDataTransform(storedFormData, stableTransform);
        }

        // Priority 3: URL search params for creation
        if (isCreate) {
            const searchParams = parseSearchParams();
            if (Object.keys(searchParams).length) {
                return applyDataTransform(searchParams as Partial<PData>, stableTransform);
            }
        }

        // Default: Empty object
        return applyDataTransform({} as Partial<PData>, stableTransform);
    }, [initialData, isCreate, storedFormData, stableTransform]);

    const [formData, setFormData] = useState<ObjectReturnType<TData, TFunc>>(
        initialObject as ObjectReturnType<TData, TFunc>,
    );

    // Check for data conflict - only log once
    const hasLoggedConflictRef = useRef(false);

    // Check for data conflict 
    const hasDataConflict = useMemo(() => {
        const conflict = !isCreate && storedFormData && initialData && !isEqual(storedFormData, initialData);

        if (conflict && !hasLoggedConflictRef.current) {
            console.warn("Data conflict detected", { stored: storedFormData, initial: initialData });
            hasLoggedConflictRef.current = true;
        }

        return conflict;
    }, [isCreate, storedFormData, initialData]);

    // Function to use stored data
    const useStoredData = useCallback(() => {
        if (storedFormData) {
            setFormData(applyDataTransform(storedFormData, stableTransform) as ObjectReturnType<TData, TFunc>);
        }
    }, [storedFormData, stableTransform]);

    // Prompt for conflict resolution
    useEffect(() => {
        if (hasDataConflict && storedFormData) {
            PubSub.get().publish("snack", {
                autoHideDuration: "persist",
                messageKey: "FormDataFound",
                buttonKey: "Yes",
                buttonClicked: useStoredData,
                severity: "Warning",
            });
        }
    }, [hasDataConflict, storedFormData, useStoredData]);

    return {
        formData,
        setFormData,
        hasStoredData: Boolean(storedFormData),
        hasDataConflict,
        useStoredData,
    };
}

/**
 * Hook that manages the state of an object, handling data fetching, caching, and state updates.
 * Composed using more focused hooks for better maintainability.
 */
export function useManagedObject<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>({
    pathname,
    disabled = false,
    displayError = true,
    isCreate = false,
    onError,
    onInvalidUrlParams,
    overrideObject: unstableOverrideObject,
    transform,
}: UseManagedObjectParams<PData, TData, TFunc>): UseManagedObjectReturn<TData, TFunc> {

    // Create stable callbacks/objects
    const overrideObject = useStableObject(unstableOverrideObject);

    // Initialize data fetching if not disabled and no override object
    const shouldFetch = !disabled && !overrideObject;

    // Cache some state to prevent excessive dependency churn
    const stateRef = useRef({
        hasAlreadyShownFetchFailedError: false,
        hasAlreadyShownInvalidUrlError: false,
        hasAttemptedInitialFetch: false,
        // Ensure we only reset formData once on fetch failure
        hasAlreadyResolvedFetchFailed: false,
    });

    // Reset error state when pathname changes
    useEffect(() => {
        stateRef.current = {
            hasAlreadyShownFetchFailedError: false,
            hasAlreadyShownInvalidUrlError: false,
            hasAttemptedInitialFetch: false,
            hasAlreadyResolvedFetchFailed: false,
        };
    }, [pathname]);

    // Get object data from server
    const {
        data: fetchedData,
        isLoading,
        hasUnauthorizedError,
        fetchObjectData,
        fetchFailed,
    } = useObjectData<PData>(shouldFetch, pathname, onError, displayError);

    const {
        getCachedData,
        setCachedData,
        clearCache,
    } = useObjectCache<PData>(pathname);

    // Get cached data once to avoid recreating it on each render
    const cachedData = useMemo(() => {
        const data = getCachedData();
        return data;
    }, [getCachedData]);

    // Initial data to use (either override or cached)
    const initialDataToUse = useMemo(() => {
        if (overrideObject) {
            return overrideObject;
        }
        if (cachedData) {
            return cachedData;
        }
        return undefined;
    }, [overrideObject, cachedData]);

    const {
        formData,
        setFormData,
    } = useObjectForm<PData, TData, TFunc>(
        pathname,
        isCreate,
        initialDataToUse,
        transform,
    );

    // Simplified effect for applying data according to priority
    // We've reorganized to minimize dependencies
    useEffect(() => {
        if (disabled) {
            if (process.env.NODE_ENV === "development") {
                console.info("Hook is disabled, skipping data update");
            }
            return;
        }

        // PRIORITY LOGIC - Apply data from various sources in priority order
        // All early returns to avoid multiple state updates

        // Priority 1: Override object
        if (overrideObject) {
            setFormData(applyDataTransform(overrideObject, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 2: Fetched data
        if (fetchedData) {
            setCachedData(fetchedData);
            setFormData(applyDataTransform(fetchedData, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 3: Handle unauthorized errors
        if (hasUnauthorizedError) {
            clearCache();
            setFormData(applyDataTransform({}, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 4: Handle fetch failures
        if (fetchFailed) {
            // Show error only once
            if (!stateRef.current.hasAlreadyShownFetchFailedError && displayError) {
                stateRef.current.hasAlreadyShownFetchFailedError = true;
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            }
            // Only reset formData on first failure to avoid infinite loop
            if (!stateRef.current.hasAlreadyResolvedFetchFailed) {
                stateRef.current.hasAlreadyResolvedFetchFailed = true;
                setFormData(applyDataTransform({}, transform) as ObjectReturnType<TData, TFunc>);
            }
            return;
        }


        // Priority 7: Initiate fetch when needed
        if (shouldFetch && pathname && pathname.length > 1 && !fetchedData && !isLoading && !stateRef.current.hasAttemptedInitialFetch) {
            stateRef.current.hasAttemptedInitialFetch = true;
            fetchObjectData();
            return;
        }
    }, [
        // Core dependencies that should trigger reevaluation
        disabled,
        overrideObject,
        fetchedData,
        hasUnauthorizedError,
        fetchFailed,
        isLoading,
        pathname,
        shouldFetch,
        onInvalidUrlParams,
    ]);

    // Determine permissions based on the object
    const permissions = useMemo(() =>
        formData ? getYou(formData as PData) : defaultYou,
        [formData],
    );

    return {
        id: formData?.id ?? fetchedData?.id ?? cachedData?.id,
        isLoading,
        object: formData,
        permissions,
        setObject: setFormData,
    };
}
