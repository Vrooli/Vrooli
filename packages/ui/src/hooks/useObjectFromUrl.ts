import { DUMMY_ID, exists, FindByIdInput, FindByIdOrHandleInput, FindVersionInput, GqlModelType, uuidValidate } from "@local/shared";
import { Dispatch, SetStateAction, useCallback, useEffect, useMemo, useState } from "react";
import { parseSearchParams, ParseSearchParamsResult, useLocation } from "route";
import { PartialWithType } from "types";
import { getCookieFormData, getCookiePartialData, setCookiePartialData } from "utils/cookies";
import { defaultYou, getYou, YouInflated } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { FetchInputOptions, useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";

type UrlObject = { __typename: GqlModelType | `${GqlModelType}`, id?: string };

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

/**
 * Hook for finding an object from the URL and providing relevant properties and functions
 */
export function useObjectFromUrl<
    PData extends UrlObject,
    TData extends UrlObject = PartialWithType<PData>,
    TFunc extends (data: Partial<PData>) => TData = (data: Partial<PData>) => TData
>({
    displayError,
    endpoint,
    isCreate,
    objectType,
    onError,
    onInvalidUrlParams,
    overrideObject,
    transform,
}: {
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

    const stableOnError = useStableCallback(onError);
    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);
    const stableTransform = useStableCallback(transform);

    /** Either applies the transform or returns the input data directly */
    const applyTransform = useCallback((data: Partial<PData>) => {
        return typeof stableTransform === "function" ? stableTransform(data) : data;
    }, [stableTransform]);

    // Fetch data
    const [getData, { data, loading: isLoading }] = useLazyFetch<FindByIdInput | FindVersionInput | FindByIdOrHandleInput, PData>({ endpoint });
    const [object, setObject] = useState<ObjectReturnType<TData, TFunc>>(() => {
        // If overrideObject provided, use it. Also use transform if provided
        if (typeof overrideObject === "object") return applyTransform(overrideObject) as ObjectReturnType<TData, TFunc>;
        // Try to find object in cache
        const storedData = getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        // If transform provided, use it
        const data = applyTransform(storedData) as ObjectReturnType<TData, TFunc>;
        // Try to find form data in cache
        const storedForm = getCookieFormData(`${objectType}-${isCreate ? DUMMY_ID : urlParams.id}`);
        if (storedForm) {
            if (isCreate) return applyTransform(storedForm as Partial<PData>) as ObjectReturnType<TData, TFunc>;
            else if (JSON.stringify(storedForm) === JSON.stringify(data)) return data;
            PubSub.get().publish("snack", {
                autoHideDuration: "persist",
                messageKey: "FormDataFound",
                buttonKey: "Yes",
                buttonClicked: () => {
                    setObject(applyTransform(storedForm as Partial<PData>) as ObjectReturnType<TData, TFunc>);
                },
                severity: "Warning",
            });
        }
        // If no form data found but we're creating, we can try to find initial values in URL.
        else if (isCreate) {
            const searchParams = parseSearchParams();
            if (Object.keys(searchParams).length) {
                return applyTransform(searchParams as Partial<PData>) as ObjectReturnType<TData, TFunc>;
            }
        }
        // Return data
        return data;
    });
    useEffect(() => {
        // If overrideObject provided, don't fetch
        if (typeof overrideObject === "object") return;
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle }, { onError: stableOnError, displayError });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot }, { onError: stableOnError, displayError });
        else if (exists(urlParams.id)) getData({ id: urlParams.id }, { onError: stableOnError, displayError });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot }, { onError: stableOnError, displayError });
        // If transform provided, ignore bad URL params. This is because we only use the transform for 
        // upsert forms, which don't have a valid URL if the object doesn't exist yet (i.e. when creating)
        else if (typeof stableTransform === "function") return;
        // Else if onInvalidUrlParams provided, call it
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        // Else, show error
        else PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
    }, [stableOnError, getData, objectType, overrideObject, displayError, stableOnInvalidUrlParams, stableTransform, urlParams]);
    useEffect(() => {
        // If overrideObject provided, use it
        if (typeof overrideObject === "object") {
            setObject(applyTransform(overrideObject) as ObjectReturnType<TData, TFunc>);
            return;
        }
        // If data was queried (i.e. object exists), store it in local state
        if (data) setCookiePartialData(data, "full");
        const knownData = data ?? getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        if (knownData && typeof knownData === "object" && uuidValidate(knownData.id)) {
            // If transform provided, use it
            const changedData = applyTransform(knownData) as ObjectReturnType<TData, TFunc>;
            // Set object
            setObject(changedData as ObjectReturnType<TData, TFunc>);
        }
    }, [applyTransform, data, objectType, overrideObject, urlParams]);

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
