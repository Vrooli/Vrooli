import { DUMMY_ID, FindByIdInput, FindByIdOrHandleInput, FindVersionInput, ModelType, ParseSearchParamsResult, YouInflated, exists, isEqual, parseSearchParams } from "@local/shared";
import { ServerResponseParser } from "api/responseParser.js";
import { FetchInputOptions } from "api/types.js";
import { Dispatch, SetStateAction, useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "route";
import { PartialWithType } from "types";
import { defaultYou, getYou } from "utils/display/listTools.js";
import { getCookiePartialData, removeCookiePartialData, setCookiePartialData } from "utils/localStorage.js";
import { UrlInfo, parseSingleItemUrl } from "utils/navigation/urlTools.js";
import { PubSub } from "utils/pubsub.js";
import { useFormCacheStore } from "./forms.js";
import { useLazyFetch } from "./useLazyFetch.js";

type UrlObject = { __typename: ModelType | `${ModelType}`; id?: string };

type FetchInput = FindByIdInput | FindVersionInput | FindByIdOrHandleInput;

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
 * Hook that manages the state of an object, handling data fetching, caching, and state updates.
 */
export function useManagedObject<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>(params: UseManagedObjectParams<PData, TData, TFunc>): UseManagedObjectReturn<TData, TFunc> {
    const {
        endpoint,
        objectType,
        disabled = false,
        displayError = true,
        isCreate = false,
        onError,
        onInvalidUrlParams,
        overrideObject,
        transform,
    } = params;

    const [{ pathname }] = useLocation();
    const urlParams = useMemo(() => parseSingleItemUrl({ pathname }), [pathname]);

    // Use refs to avoid unnecessary re-renders
    const onErrorRef = useRef(onError);
    const onInvalidUrlParamsRef = useRef(onInvalidUrlParams);
    const transformRef = useRef(transform);

    const getFormCacheData = useFormCacheStore(state => state.getCacheData);

    // Initialize fetch function
    const [fetchData, fetchResult] = useLazyFetch<FetchInput, PData>({ endpoint });

    // Initialize object state
    const {
        initialObject,
        hasStoredFormDataConflict,
        storedFormData,
    } = initializeObjectState<PData, TData, TFunc>({
        disabled,
        getFormCacheData,
        isCreate,
        objectType,
        overrideObject,
        transform: transformRef.current,
        urlParams,
    });
    const [object, setObject] = useState<ObjectReturnType<TData, TFunc>>(initialObject as ObjectReturnType<TData, TFunc>);

    useEffect(function promptOnFormConflict() {
        if (hasStoredFormDataConflict && storedFormData) {
            promptToUseStoredFormData(() => {
                setObject(applyDataTransform(storedFormData, transformRef.current) as ObjectReturnType<TData, TFunc>);
            });
        }
    }, [hasStoredFormDataConflict, storedFormData, transformRef]);

    useEffect(function handleFetchData() {
        if (shouldFetchData({ disabled, overrideObject })) {
            const fetched = fetchDataUsingUrl(urlParams, fetchData, onErrorRef.current, displayError);

            if (!fetched) {
                handleInvalidUrlParams({
                    transform: transformRef.current,
                    onInvalidUrlParams: onInvalidUrlParamsRef.current,
                    urlParams,
                });
            }
        }
    }, [fetchData, objectType, overrideObject, displayError, urlParams, disabled]);

    useEffect(function handleUpdateOnFetchResult() {
        console.log("in handleUpdateOnFetchResult", fetchResult, objectType);
        if (disabled || fetchResult.loading) {
            return;
        }

        if (overrideObject) {
            // Prioritize overrideObject over other data sources
            setObject(applyDataTransform(overrideObject, transformRef.current) as ObjectReturnType<TData, TFunc>);
            return;
        }

        if (fetchResult.data) {
            // If data was fetched, cache it and update the object
            setCookiePartialData(fetchResult.data, "full");
            setObject(applyDataTransform(fetchResult.data, transformRef.current) as ObjectReturnType<TData, TFunc>);
        } else if (ServerResponseParser.hasErrorCode(fetchResult, "Unauthorized")) {
            // If unauthorized error, clear cache and reset object
            removeCookiePartialData({ __typename: objectType, ...urlParams });
            setObject(applyDataTransform({}, transformRef.current) as ObjectReturnType<TData, TFunc>);
        } else {
            // If no fetched data, attempt to load from cache
            const cachedData = getCachedData<PData>({ objectType, urlParams });
            if (cachedData) {
                setObject(applyDataTransform(cachedData, transformRef.current) as ObjectReturnType<TData, TFunc>);
            }
        }
    }, [disabled, fetchResult, objectType, overrideObject, urlParams]);

    // Determine permissions based on the object
    const permissions = useMemo(() => (object ? getYou(object) : defaultYou), [object]);

    return {
        id: object?.id ?? urlParams.id,
        isLoading: fetchResult.loading,
        object,
        permissions,
        setObject,
    };
}

/**
 * Initializes the object state based on various sources: overrideObject, cache, or defaults.
 */
export function initializeObjectState<
    PData extends UrlObject,
    TData extends UrlObject,
    TFunc extends (data: Partial<PData>) => TData
>({
    disabled,
    getFormCacheData,
    isCreate,
    objectType,
    overrideObject,
    transform,
    urlParams,
}: {
    disabled: boolean;
    getFormCacheData: ((objectType: ModelType | `${ModelType}`, id: string) => object | null),
    isCreate: boolean;
    objectType: ModelType | `${ModelType}`;
    overrideObject?: Partial<PData>;
    transform?: TFunc;
    urlParams: UrlInfo;
}): {
    initialObject: TData;
    hasStoredFormDataConflict: boolean;
    storedFormData?: Partial<PData>;
} {
    // **Priority 1: overrideObject**
    if (overrideObject) {
        return { initialObject: applyDataTransform(overrideObject, transform), hasStoredFormDataConflict: false };
    }

    // **Priority 2: Disabled State**
    if (disabled) {
        return { initialObject: applyDataTransform({}, transform), hasStoredFormDataConflict: false };
    }

    // **Priority 3: Cached Data**
    const cachedData = getCachedData<PData>({ objectType, urlParams });
    const transformedData = applyDataTransform(cachedData, transform);

    // **Priority 4: Stored Form Data**
    const objectId = isCreate ? DUMMY_ID : urlParams.id;
    const storedFormData = objectId ? getFormCacheData(objectType, objectId) as PData : undefined;
    if (storedFormData) {
        if (isCreate) {
            return { initialObject: applyDataTransform(storedFormData, transform), hasStoredFormDataConflict: false };
        } else if (!isEqual(storedFormData, transformedData)) {
            // Indicate that there's a conflict and provide the stored form data
            return { initialObject: transformedData, hasStoredFormDataConflict: true, storedFormData };
        }
    }

    // **Priority 5: URL Search Params for Creation**
    if (isCreate) {
        const searchParams = parseSearchParams();
        if (Object.keys(searchParams).length) {
            return {
                initialObject: applyDataTransform(searchParams as Partial<PData>, transform),
                hasStoredFormDataConflict: false,
            };
        }
    }

    console.log("returning transformed data", transformedData);
    // **Default: Return transformed cached data or empty object**
    return { initialObject: transformedData, hasStoredFormDataConflict: false };
}

/**
 * Determines whether data fetching should occur.
 */
export function shouldFetchData({ disabled, overrideObject }: { disabled: boolean; overrideObject?: any }): boolean {
    console.log("in shouldFetchData", disabled, overrideObject, disabled !== true && (typeof overrideObject !== "object" || overrideObject === null));
    // Fetch data only if not disabled and no overrideObject is provided.
    return disabled !== true && (typeof overrideObject !== "object" || overrideObject === null);
}

/**
 * Handles the scenario where URL parameters are invalid.
 */
export function handleInvalidUrlParams({
    transform,
    onInvalidUrlParams,
    urlParams,
}: {
    transform?: any;
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => unknown;
    urlParams: UrlInfo;
}): void {
    if (transform) {
        // If a transform function is provided (e.g., in forms), invalid URL params can be ignored.
        // This is common in creation forms where the object doesn't exist yet.
        return;
    } else if (onInvalidUrlParams) {
        // If a custom handler is provided, use it.
        onInvalidUrlParams(urlParams);
    } else {
        // Otherwise, display a default error message.
        PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
    }
}

/**
 * Applies the transform function to the data if provided, or returns the data as is.
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
 * Fetches data using identifiers from the URL, if they exist.
 */
export function fetchDataUsingUrl(
    params: UrlInfo,
    fetchData: (input: FetchInput, options?: FetchInputOptions) => unknown,
    onError?: FetchInputOptions["onError"],
    displayError?: boolean,
): boolean {
    const options = { onError, displayError };
    console.log("might fetch data", params);
    if (exists(params.handle)) {
        fetchData({ handle: params.handle }, options);
        return true;
    }
    if (exists(params.handleRoot)) {
        fetchData({ handleRoot: params.handleRoot }, options);
        return true;
    }
    if (exists(params.id)) {
        fetchData({ id: params.id }, options);
        return true;
    }
    if (exists(params.idRoot)) {
        fetchData({ idRoot: params.idRoot }, options);
        return true;
    }
    return false;
}

type GetCachedDataProps = {
    objectType: ModelType | `${ModelType}`;
    urlParams: UrlInfo;
};
/**
 * Retrieves cached data for the object if available.
 */
export function getCachedData<PData extends UrlObject>({ objectType, urlParams }: GetCachedDataProps): Partial<PData> | null {
    return getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
}

/**
 * Prompts the user to use stored form data.
 */
export function promptToUseStoredFormData(onConfirm: () => void): void {
    PubSub.get().publish("snack", {
        autoHideDuration: "persist",
        messageKey: "FormDataFound",
        buttonKey: "Yes",
        buttonClicked: onConfirm,
        severity: "Warning",
    });
}
