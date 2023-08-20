import { exists, FindByIdInput, FindByIdOrHandleInput, FindVersionInput, GqlModelType } from "@local/shared";
import { Dispatch, SetStateAction, useEffect, useMemo, useState } from "react";
import { ParseSearchParamsResult } from "route";
import { PartialWithType } from "types";
import { getCookiePartialData, setCookiePartialData } from "utils/cookies";
import { defaultYou, getYou, YouInflated } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
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
    endpoint,
    objectType,
    onInvalidUrlParams,
    overrideObject,
    transform,
}: {
    endpoint: string,
    objectType: PData["__typename"],
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => void,
    /** If passed, uses this instead of fetching from server or cache */
    overrideObject?: Partial<PData>,
    /** Used by forms to properly shape the data for formik */
    transform?: TFunc,
}): UseObjectFromUrlReturn<TData, TFunc> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl({}), []);

    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);
    const stableTransform = useStableCallback(transform);

    // Fetch data
    const [getData, { data, loading: isLoading, errors }] = useLazyFetch<FindByIdInput | FindVersionInput | FindByIdOrHandleInput, PData>({ endpoint });
    const [object, setObject] = useState<ObjectReturnType<TData, TFunc>>(() => {
        // If overrideObject provided, use it. Also use transform if provided
        if (typeof overrideObject === "object") return (typeof stableTransform === "function" ? stableTransform(overrideObject) : overrideObject) as ObjectReturnType<TData, TFunc>;
        // Try to find object in cache
        const storedData = getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        // If transform provided, use it
        const data = (typeof stableTransform === "function" ? stableTransform(storedData) : storedData) as ObjectReturnType<TData, TFunc>;
        // Return data
        return data;
    });
    useDisplayServerError(errors);
    useEffect(() => {
        // If overrideObject provided, don't fetch
        if (typeof overrideObject === "object") return;
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        // If transform provided, ignore bad URL params. This is because we only use the transform for 
        // upsert forms, which don't have a valid URL if the object doesn't exist yet (i.e. when creating)
        else if (typeof stableTransform === "function") return;
        // Else if onInvalidUrlParams provided, call it
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        // Else, show error
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, objectType, overrideObject, stableOnInvalidUrlParams, stableTransform, urlParams]);
    useEffect(() => {
        // If overrideObject provided, use it
        if (typeof overrideObject === "object") {
            setObject((typeof stableTransform === "function" ? stableTransform(overrideObject) : overrideObject) as ObjectReturnType<TData, TFunc>);
            return;
        }
        // If data was queried (i.e. object exists), store it in local state
        if (data) setCookiePartialData(data, "full");
        const knownData = data ?? getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        // If transform provided, use it
        const changedData = (typeof stableTransform === "function" ? stableTransform(knownData) : knownData) as ObjectReturnType<TData, TFunc>;
        // Set object
        setObject(changedData as ObjectReturnType<TData, TFunc>);
    }, [data, objectType, overrideObject, stableTransform, urlParams]);


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