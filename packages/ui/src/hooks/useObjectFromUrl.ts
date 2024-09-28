import { DUMMY_ID, FindByIdInput, FindByIdOrHandleInput, FindVersionInput, GqlModelType, ParseSearchParamsResult, YouInflated, exists, parseSearchParams, uuidValidate } from "@local/shared";
import { FetchInputOptions } from "api/types";
import { Dispatch, SetStateAction, useEffect, useMemo, useRef, useState } from "react";
import { useLocation } from "route";
import { PartialWithType } from "types";
import { defaultYou, getYou } from "utils/display/listTools";
import { getCookieFormData, getCookiePartialData, removeCookiePartialData, setCookiePartialData } from "utils/localStorage";
import { UrlInfo, parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { useLazyFetch } from "./useLazyFetch";

type UrlObject = { __typename: GqlModelType | `${GqlModelType}`, id?: string };

type FetchInput = FindByIdInput | FindVersionInput | FindByIdOrHandleInput;

/** 
 * When transform is provided, we know that all fields of the object must exist. 
 * When it is not provided, we can only guarantee that the __typename field exists.
 */
type ObjectReturnType<TData extends UrlObject, TFunc> = TFunc extends (data: never) => infer R ? R : PartialWithType<TData>;

export type UseObjectFromUrlReturn<TData extends UrlObject, TFunc> = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
    isLoading: boolean;
    object: ObjectReturnType<TData, TFunc>;
    permissions: YouInflated;
    setObject: Dispatch<SetStateAction<ObjectReturnType<TData, TFunc>>>;
}

/** Either applies the transform or returns the input data directly */
export function applyDataTransform<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>
>(
    data: Partial<PData>,
    transform: ((data: Partial<PData>) => TData) | undefined,
): any {//ObjectReturnType<TData, (data: Partial<PData>) => TData> => {
    // If data is malformed, return null and pretend it's the correct type
    if (!data) return null as unknown as ObjectReturnType<TData, (data: Partial<PData>) => TData>;
    // Otherwise, apply the transform if it exists
    return (typeof transform === "function" ? transform(data) : data) as ObjectReturnType<TData, (data: Partial<PData>) => TData>;
}

/**
 * Fetches data using identifiers from the url, if they exist.
 * @param params Item information parsed from URL
 * @param getData Function for fetching data
 * @param onError Function for handling errors
 * @param displayError Boolean to indicate if error snack should be displayed
 * @returns True if "getData" was called, false otherwise
 */
export function fetchDataUsingUrl(
    params: UrlInfo,
    getData: ((input: FetchInput, inputOptions?: FetchInputOptions | undefined) => unknown),
    onError?: FetchInputOptions["onError"],
    displayError?: boolean | undefined,
) {
    const inputOptions = { onError, displayError };
    if (exists(params.handle)) getData({ handle: params.handle }, inputOptions);
    else if (exists(params.handleRoot)) getData({ handleRoot: params.handleRoot }, inputOptions);
    else if (exists(params.id)) getData({ id: params.id }, inputOptions);
    else if (exists(params.idRoot)) getData({ idRoot: params.idRoot }, inputOptions);
    else return false;
    return true;
}

/**
 * Hook for finding an object from the URL and providing relevant properties and functions
 */
export function useObjectFromUrl<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>({
    // componentId,
    disabled,
    displayError,
    endpoint,
    isCreate,
    objectType,
    onError,
    onInvalidUrlParams,
    overrideObject,
    transform,
}: {
    // /** The ID for the component/page. Used to get and set cache data */
    // componentId: string, TODO
    disabled?: boolean,
    /** If true, shows error snack when fetching fails */
    displayError?: boolean,
    endpoint: string,
    /** If passed, tries to find existing form data in cache */
    isCreate?: boolean,
    /** Typically the type of the object being used, but is also sometimes the parent object (comments do this, for example) */
    objectType: GqlModelType | `${GqlModelType}`,
    onError?: FetchInputOptions["onError"],
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => unknown,
    /** If passed, uses this instead of fetching from server or cache */
    overrideObject?: Partial<PData>,
    /** Used by forms to properly shape the data for formik */
    transform?: TFunc,
}): UseObjectFromUrlReturn<TData, TFunc> {
    // Get URL params for querying and default values
    const [{ pathname }] = useLocation();
    const urlParams = useMemo(() => parseSingleItemUrl({ pathname }), [pathname]);

    const onErrorRef = useRef(onError);
    onErrorRef.current = onError;
    const onInvalidUrlParamsRef = useRef(onInvalidUrlParams);
    onInvalidUrlParamsRef.current = onInvalidUrlParams;
    const transformRef = useRef(transform);
    transformRef.current = transform;

    // Fetch data
    const [getData, { data: fetchedData, errors: fetchedErrors, loading: isLoading }] = useLazyFetch<FetchInput, PData>({ endpoint });
    const [object, setObject] = useState<ObjectReturnType<TData, TFunc>>(() => {
        // If overrideObject provided, use it. Also use transform if provided
        if (typeof overrideObject === "object") return applyDataTransform(overrideObject, transformRef.current);
        // If disabled, don't try anything else
        if (disabled) return applyDataTransform({}, transformRef.current);
        // Try to find object in cache
        const storedData = getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        // If transform provided, use it
        const data = applyDataTransform(storedData, transformRef.current);
        // Try to find form data in cache
        const storedForm = getCookieFormData(`${objectType}-${isCreate ? DUMMY_ID : urlParams.id}`) as Partial<PData> | undefined;
        if (storedForm) {
            if (isCreate) return applyDataTransform(storedForm, transformRef.current);
            else if (JSON.stringify(storedForm) === JSON.stringify(data)) return data;
            PubSub.get().publish("snack", {
                autoHideDuration: "persist",
                messageKey: "FormDataFound",
                buttonKey: "Yes",
                buttonClicked: () => {
                    setObject(applyDataTransform(storedForm, transformRef.current));
                },
                severity: "Warning",
            });
        }
        // If no form data found but we're creating, we can try to find initial values in URL.
        else if (isCreate) {
            const searchParams = parseSearchParams();
            if (Object.keys(searchParams).length) {
                return applyDataTransform(searchParams as Partial<PData>, transformRef.current);
            }
        }
        // Return data
        return data;
    });
    useEffect(() => {
        // If overrideObject provided or disabled, don't fetch
        if (typeof overrideObject === "object" || disabled) return;
        // Try to fetch data using URL params
        const fetched = fetchDataUsingUrl(urlParams, getData, onErrorRef.current, displayError);
        if (fetched) return;
        // If transform provided, ignore bad URL params. This is because we only use the transform for 
        // upsert forms, which don't have a valid URL if the object doesn't exist yet (i.e. when creating)
        if (typeof transformRef.current === "function") return;
        // Else if onInvalidUrlParams provided, call it
        else if (exists(onInvalidUrlParamsRef.current)) onInvalidUrlParamsRef.current(urlParams);
        // Else, show error
        else PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, objectType, overrideObject, displayError, urlParams, disabled]);
    useEffect(() => {
        // If overrideObject provided, use it
        if (typeof overrideObject === "object") {
            setObject(applyDataTransform(overrideObject, transformRef.current));
            return;
        }
        // If disabled, don't try anything else
        if (disabled) return;
        // If data was queried (i.e. object exists), store it in local state
        if (fetchedData) setCookiePartialData(fetchedData, "full");
        // If we didn't receive fetched data, and we received an "Unauthorized" error, 
        // we should clear the cookie data and set the object to its default value
        else if (fetchedErrors?.some(e => e.code === "Unauthorized")) {
            removeCookiePartialData({ __typename: objectType, ...urlParams });
            setObject(applyDataTransform({}, transformRef.current));
            return;
        }
        // If we have fetched data or cached data, set the object   
        const knownData = fetchedData ?? getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        if (knownData && typeof knownData === "object" && uuidValidate(knownData.id)) {
            setObject(applyDataTransform(knownData, transformRef.current));
        }
    }, [disabled, fetchedData, fetchedErrors, objectType, overrideObject, urlParams]);

    // If object found, get permissions
    const permissions = useMemo(() => object ? getYou(object) : defaultYou, [object]);

    return {
        id: object?.id ?? urlParams.id,
        isLoading,
        object,
        permissions,
        setObject,
    };
}
