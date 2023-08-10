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
 * When upsertTransform is provided, we know that all fields of the object must exist. 
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
    upsertTransform,
}: {
    endpoint: string,
    objectType: PData["__typename"],
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => void,
    /** Used by forms to properly shape the data for formik */
    upsertTransform?: TFunc,
}): UseObjectFromUrlReturn<TData, TFunc> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl({}), []);

    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);

    // Fetch data
    const [getData, { data, loading: isLoading, errors }] = useLazyFetch<FindByIdInput | FindVersionInput | FindByIdOrHandleInput, PData>({ endpoint });
    const [object, setObject] = useState<ObjectReturnType<TData, TFunc>>(() => {
        const storedData = getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        if (typeof upsertTransform === "function") return upsertTransform(storedData) as ObjectReturnType<TData, TFunc>;
        return storedData as ObjectReturnType<TData, TFunc>;
    });
    useDisplayServerError(errors);
    useEffect(() => {
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        // If upsertTransform provided, ignore bad URL params. This is because we only use the transform for 
        // upsert forms, which don't have a valid URL if the object doesn't exist yet
        else if (typeof upsertTransform === "function") return;
        // Else if onInvalidUrlParams provided, call it
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        // Else, show error
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, objectType, stableOnInvalidUrlParams, upsertTransform, urlParams]);
    useEffect(() => {
        // If data was queried (i.e. object exists), store it in local state
        if (data) setCookiePartialData(data);
        const knownData = data ?? getCookiePartialData<PartialWithType<PData>>({ __typename: objectType, ...urlParams });
        if (typeof upsertTransform === "function") setObject(upsertTransform(knownData) as ObjectReturnType<TData, TFunc>);
        else setObject(knownData as ObjectReturnType<TData, TFunc>);
    }, [data, objectType, upsertTransform, urlParams]);


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
