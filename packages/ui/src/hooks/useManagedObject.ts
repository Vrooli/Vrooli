import { DUMMY_ID, FindByIdInput, FindByIdOrHandleInput, FindVersionInput, ModelType, ParseSearchParamsResult, YouInflated, exists, isEqual, parseSearchParams } from "@local/shared";
import { Dispatch, SetStateAction, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { ServerResponseParser } from "../api/responseParser.js";
import { FetchInputOptions } from "../api/types.js";
import { useLocation } from "../route/router.js";
import { PartialWithType } from "../types.js";
import { defaultYou, getYou } from "../utils/display/listTools.js";
import { getCookiePartialData, removeCookiePartialData, setCookiePartialData } from "../utils/localStorage.js";
import { UrlInfo, parseSingleItemUrl } from "../utils/navigation/urlTools.js";
import { PubSub } from "../utils/pubsub.js";
import { useFormCacheStore } from "./forms.js";
import { useLazyFetch } from "./useLazyFetch.js";
import { useStableObject } from "./useStableObject.js";

type UrlObject = { __typename: ModelType | `${ModelType}`; id?: string };

type FetchInput = FindByIdInput | FindVersionInput | FindByIdOrHandleInput;

/**
 * Maximum number of retries for fetching an object before giving up
 */
const MAX_FETCH_RETRIES = 3;

/**
 * Determines the return type based on whether a transform function is provided.
 */
type ObjectReturnType<TData extends UrlObject, TFunc> = TFunc extends (data: any) => infer R
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
    /** The endpoint to fetch data from */
    endpoint: string;
    /** The GraphQL object type (e.g., 'BookmarkList') */
    objectType: ModelType | `${ModelType}`;
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
 * Hook for parsing and accessing URL parameters relevant to object identification
 */
export function useObjectUrl() {
    const [{ pathname }] = useLocation();
    const urlParams = useMemo(function parseUrlParamsMemo() {
        return parseSingleItemUrl({ pathname });
    }, [pathname]);

    const result = {
        params: urlParams,
        id: urlParams.id,
        handle: urlParams.handle,
        idRoot: urlParams.idRoot,
        handleRoot: urlParams.handleRoot,
        hasIdentifier: Boolean(urlParams.id || urlParams.handle || urlParams.idRoot || urlParams.handleRoot),
    };

    return result;
}

/**
 * Hook for fetching object data from server endpoints
 */
export function useObjectData<TData extends UrlObject>(
    endpoint: string,
    shouldFetch: boolean,
    identifier: { id?: string; handle?: string; idRoot?: string; handleRoot?: string },
    onError: FetchInputOptions["onError"],
    displayError: boolean,
) {
    const [fetchData, fetchResult] = useLazyFetch<FetchInput, TData>({ endpoint });
    const retryCountRef = useRef(0);
    const [fetchFailed, setFetchFailed] = useState(false);
    // Track if initial fetch has been done
    const initialFetchDoneRef = useRef(false);

    // Create a stable version of the identifier to prevent infinite loops
    const stableIdentifier = useMemo(() => ({
        id: identifier.id,
        handle: identifier.handle,
        idRoot: identifier.idRoot,
        handleRoot: identifier.handleRoot
    }), [identifier.id, identifier.handle, identifier.idRoot, identifier.handleRoot]);

    // Track the previous identifier to detect real changes
    const prevIdentifierRef = useRef(stableIdentifier);

    // Check if identifier actually changed (values, not just object reference)
    const identifierChanged = useMemo(() => {
        const changed = !isEqual(prevIdentifierRef.current, stableIdentifier);
        if (changed) {
            prevIdentifierRef.current = stableIdentifier;
        }
        return changed;
    }, [stableIdentifier]);

    const fetchObjectData = useCallback(function fetchObjectDataCallback() {
        if (!shouldFetch) {
            console.info("[useObjectData] Fetch skipped: shouldFetch is false");
            return false;
        }

        // Don't retry if we've already hit the maximum number of retries
        if (retryCountRef.current >= MAX_FETCH_RETRIES) {
            console.warn(`[useObjectData] Max retry count (${MAX_FETCH_RETRIES}) reached, marking as failed`);
            setFetchFailed(true);
            return false;
        }

        // Determine which identifier to use for fetching
        let identifierType: string | null = null;
        let identifierValue: string | null = null;
        let fetchInput: FetchInput | null = null;

        if (exists(stableIdentifier.handle)) {
            identifierType = "handle";
            identifierValue = stableIdentifier.handle;
            fetchInput = { handle: stableIdentifier.handle };
        } else if (exists(stableIdentifier.handleRoot)) {
            identifierType = "handleRoot";
            identifierValue = stableIdentifier.handleRoot;
            fetchInput = { handleRoot: stableIdentifier.handleRoot };
        } else if (exists(stableIdentifier.id)) {
            identifierType = "id";
            identifierValue = stableIdentifier.id;
            fetchInput = { id: stableIdentifier.id };
        } else if (exists(stableIdentifier.idRoot)) {
            identifierType = "idRoot";
            identifierValue = stableIdentifier.idRoot;
            fetchInput = { idRoot: stableIdentifier.idRoot };
        }

        if (fetchInput) {
            console.info(`[useObjectData] Fetching with ${identifierType}=${identifierValue}, attempt ${retryCountRef.current + 1}/${MAX_FETCH_RETRIES}`);
            fetchData(fetchInput, { onError, displayError });
            retryCountRef.current++;
            initialFetchDoneRef.current = true;
            return true;
        } else {
            console.warn("[useObjectData] No valid identifier found for fetching");
            return false;
        }
    }, [stableIdentifier, fetchData, onError, displayError, shouldFetch]);

    // Reset state when identifier *values* change (not just object reference)
    useEffect(function refetchOnIdentifierChange() {
        if (identifierChanged) {
            console.info("[useObjectData] Identifier values changed, resetting retry count");
            retryCountRef.current = 0;
            setFetchFailed(false);
            initialFetchDoneRef.current = false;
        }
    }, [identifierChanged]);

    // Mark as failed when errors occur
    useEffect(() => {
        if (fetchResult.errors && fetchResult.errors.length > 0) {
            console.error("[useObjectData] Fetch errors:", fetchResult.errors);
            // If we have errors and hit max retries, mark as failed
            if (retryCountRef.current >= MAX_FETCH_RETRIES) {
                console.warn("[useObjectData] Max retries reached with errors, marking as failed");
                setFetchFailed(true);
            }
        }
    }, [fetchResult.errors]);

    // IMPORTANT: Initial fetch if we have an identifier and haven't fetched yet
    // This ensures the fetch is triggered automatically when the hook is used
    useEffect(() => {
        const hasIdentifier = exists(stableIdentifier.id) ||
            exists(stableIdentifier.handle) ||
            exists(stableIdentifier.idRoot) ||
            exists(stableIdentifier.handleRoot);

        if (shouldFetch && !initialFetchDoneRef.current && hasIdentifier) {
            console.info("[useObjectData] Triggering initial fetch");
            fetchObjectData();
        }
    }, [shouldFetch, stableIdentifier, fetchObjectData]);

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
    objectType: ModelType | `${ModelType}`,
    identifier: { id?: string; handle?: string; idRoot?: string; handleRoot?: string },
) {
    // Create a shallow copy of the identifier to ensure we only use the values we need
    const safeIdentifier = useMemo(() => ({
        id: identifier.id,
        handle: identifier.handle,
        idRoot: identifier.idRoot,
        handleRoot: identifier.handleRoot
    }), [identifier.id, identifier.handle, identifier.idRoot, identifier.handleRoot]);

    // Only use the actual specified values when creating the cache object
    const cacheRefObject = useMemo(() => {
        const obj: any = { __typename: objectType };
        if (safeIdentifier.id) obj.id = safeIdentifier.id;
        if (safeIdentifier.handle) obj.handle = safeIdentifier.handle;
        if (safeIdentifier.idRoot) obj.idRoot = safeIdentifier.idRoot;
        if (safeIdentifier.handleRoot) obj.handleRoot = safeIdentifier.handleRoot;
        return obj;
    }, [objectType, safeIdentifier]);

    const getCachedData = useCallback(() => {
        return getCookiePartialData<PartialWithType<TData>>(cacheRefObject);
    }, [cacheRefObject]);

    const setCachedData = useCallback((data: TData) => {
        setCookiePartialData(data, "full");
    }, []);

    const clearCache = useCallback(() => {
        removeCookiePartialData(cacheRefObject);
    }, [cacheRefObject]);

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
    return transform ? transform(data) : (data as unknown as TData);
}

/**
 * Hook for managing form state for an object with conflict detection
 */
export function useObjectForm<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>(
    objectType: ModelType | `${ModelType}`,
    urlParams: UrlInfo,
    isCreate = false,
    initialData?: Partial<PData>,
    transform?: TFunc,
) {
    console.debug("useObjectForm called with:", { objectType, urlParams, isCreate });

    const getFormCacheData = useFormCacheStore(state => state.getCacheData);
    const objectId = isCreate ? DUMMY_ID : urlParams.id;

    // Memoize parameters to prevent unnecessary recalculations
    const paramsMemo = useMemo(() => ({
        objectType,
        objectId,
        isCreate
    }), [objectType, objectId, isCreate]);

    // Get stored form data if it exists (memoized)
    const storedFormData = useMemo(() =>
        paramsMemo.objectId
            ? getFormCacheData(paramsMemo.objectType, paramsMemo.objectId) as PData
            : undefined
        , [paramsMemo, getFormCacheData]);

    // Memoize transform function to prevent unnecessary recalculations
    const stableTransform = useCallback(transform || ((data: Partial<PData>) => data as unknown as TData),
        [transform]);

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
            console.warn(`Data conflict detected for ${objectType}`, {
                stored: storedFormData,
                initial: initialData
            });
            hasLoggedConflictRef.current = true;
        }

        return conflict;
    }, [isCreate, storedFormData, initialData, objectType]);

    // Function to use stored data
    const useStoredData = useCallback(() => {
        if (storedFormData) {
            console.info(`Using stored form data for ${objectType}`);
            setFormData(applyDataTransform(storedFormData, stableTransform) as ObjectReturnType<TData, TFunc>);
        }
    }, [storedFormData, stableTransform, objectType]);

    // Prompt for conflict resolution
    useEffect(() => {
        if (hasDataConflict && storedFormData) {
            console.info(`Showing form data conflict resolution for ${objectType}`);
            PubSub.get().publish("snack", {
                autoHideDuration: "persist",
                messageKey: "FormDataFound",
                buttonKey: "Yes",
                buttonClicked: useStoredData,
                severity: "Warning",
            });
        }
    }, [hasDataConflict, storedFormData, useStoredData, objectType]);

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
    endpoint,
    objectType,
    disabled = false,
    displayError = true,
    isCreate = false,
    onError,
    onInvalidUrlParams,
    overrideObject: unstableOverrideObject,
    transform,
}: UseManagedObjectParams<PData, TData, TFunc>): UseManagedObjectReturn<TData, TFunc> {
    // Get URL parameters
    const urlInfo = useObjectUrl();

    // Create stable callbacks/objects
    const overrideObject = useStableObject(unstableOverrideObject as any);

    // Initialize data fetching if not disabled and no override object
    const shouldFetch = !disabled && !overrideObject;

    // Cache some state to prevent excessive dependency churn
    const stateRef = useRef({
        hasAlreadyShownFetchFailedError: false,
        hasAlreadyShownInvalidUrlError: false,
        hasAttemptedInitialFetch: false
    });

    console.info(`Initializing for ${objectType} with params:`, {
        disabled,
        shouldFetch,
        hasOverride: !!overrideObject,
        urlInfo: {
            id: urlInfo.id,
            handle: urlInfo.handle,
            idRoot: urlInfo.idRoot,
            handleRoot: urlInfo.handleRoot,
            hasIdentifier: urlInfo.hasIdentifier
        }
    });

    // Handle caching - use stable identifier reference
    const stableIdentifier = useMemo(() => ({
        id: urlInfo.id,
        handle: urlInfo.handle,
        idRoot: urlInfo.idRoot,
        handleRoot: urlInfo.handleRoot
    }), [urlInfo.id, urlInfo.handle, urlInfo.idRoot, urlInfo.handleRoot]);

    // Reset error state when identifier changes
    useEffect(() => {
        stateRef.current = {
            hasAlreadyShownFetchFailedError: false,
            hasAlreadyShownInvalidUrlError: false,
            hasAttemptedInitialFetch: false
        };
    }, [stableIdentifier]);

    // Get object data from server
    const {
        data: fetchedData,
        isLoading,
        hasUnauthorizedError,
        fetchObjectData,
        fetchFailed,
    } = useObjectData<PData>(endpoint, shouldFetch, stableIdentifier, onError, displayError);

    const { getCachedData, setCachedData, clearCache } = useObjectCache<PData>(
        objectType,
        stableIdentifier
    );

    // Get cached data once to avoid recreating it on each render
    const cachedData = useMemo(() => {
        const data = getCachedData();
        if (data) {
            console.info(`Found cached data for ${objectType}`);
        }
        return data;
    }, [getCachedData, objectType]);

    // Initial data to use (either override or cached)
    const initialDataToUse = useMemo(() => {
        if (overrideObject) {
            console.info(`Using override object for ${objectType}`);
            return overrideObject;
        }
        if (cachedData) {
            console.info(`Using cached data for ${objectType}`);
            return cachedData;
        }
        console.info(`No initial data available for ${objectType}`);
        return undefined;
    }, [overrideObject, cachedData, objectType]);

    // Handle form state with conflict resolution - use stable parameters
    const stableUrlParams = useMemo(() => urlInfo.params, [urlInfo.params.id]);

    const {
        formData,
        setFormData,
        hasDataConflict,
    } = useObjectForm<PData, TData, TFunc>(
        objectType,
        stableUrlParams,
        isCreate,
        initialDataToUse,
        transform,
    );

    // Simplified effect for applying data according to priority
    // We've reorganized to minimize dependencies
    useEffect(() => {
        if (disabled) {
            console.info(`Hook is disabled for ${objectType}, skipping data update`);
            return;
        }

        // PRIORITY LOGIC - Apply data from various sources in priority order
        // All early returns to avoid multiple state updates

        // Priority 1: Override object
        if (overrideObject) {
            console.info(`Priority 1: Using override object for ${objectType}`);
            setFormData(applyDataTransform(overrideObject, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 2: Fetched data
        if (fetchedData) {
            console.info(`Priority 2: Using fetched data for ${objectType}`);
            setCachedData(fetchedData);
            setFormData(applyDataTransform(fetchedData, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 3: Handle unauthorized errors
        if (hasUnauthorizedError) {
            console.warn(`Priority 3: Unauthorized error for ${objectType}, clearing cache`);
            clearCache();
            setFormData(applyDataTransform({}, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 4: Handle fetch failures
        if (fetchFailed) {
            console.error(`Priority 4: Fetch failed for ${objectType} after max retries`);
            if (!stateRef.current.hasAlreadyShownFetchFailedError && displayError) {
                stateRef.current.hasAlreadyShownFetchFailedError = true;
                PubSub.get().publish("snack", { messageKey: "ErrorUnknown", severity: "Error" });
            }
            setFormData(applyDataTransform({}, transform) as ObjectReturnType<TData, TFunc>);
            return;
        }

        // Priority 5: Invalid URL with handler
        if (!isLoading && !urlInfo.hasIdentifier && onInvalidUrlParams) {
            console.warn(`Priority 5: Invalid URL for ${objectType}, using onInvalidUrlParams`);
            onInvalidUrlParams(urlInfo.params);
            return;
        }

        // Priority 6: Invalid URL without handler
        if (!isLoading && !urlInfo.hasIdentifier) {
            console.error(`Priority 6: Invalid URL for ${objectType} with no handler`);
            if (!stateRef.current.hasAlreadyShownInvalidUrlError && displayError) {
                stateRef.current.hasAlreadyShownInvalidUrlError = true;
                PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
            }
            return;
        }

        // Priority 7: Initiate fetch when needed
        if (shouldFetch && urlInfo.hasIdentifier && !fetchedData && !isLoading && !stateRef.current.hasAttemptedInitialFetch) {
            console.info(`Priority 7: Initiating first fetch for ${objectType}`);
            stateRef.current.hasAttemptedInitialFetch = true;
            fetchObjectData();
            return;
        }

        // Priority 8: Still loading
        if (isLoading) {
            console.info(`Data is still loading for ${objectType}`);
        }
    }, [
        // Core dependencies that should trigger reevaluation
        objectType,
        disabled,
        overrideObject,
        fetchedData,
        hasUnauthorizedError,
        fetchFailed,
        isLoading,
        // Functions and identifiers needed for the effect
        transform,
        setFormData,
        setCachedData,
        clearCache,
        fetchObjectData,
        // URL info but only the essential parts
        urlInfo.hasIdentifier,
        // Other flags and handlers
        shouldFetch,
        displayError,
        onInvalidUrlParams,
    ]);

    // Determine permissions based on the object
    const permissions = useMemo(() =>
        formData ? getYou(formData as unknown as PData) : defaultYou,
        [formData],
    );

    return {
        id: formData?.id ?? urlInfo.id,
        isLoading,
        object: formData,
        permissions,
        setObject: setFormData,
    };
}
