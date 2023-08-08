import { exists, GqlModelType } from "@local/shared";
import { Dispatch, SetStateAction, useEffect, useMemo, useState } from "react";
import { ParseSearchParamsResult } from "route";
import { PartialWithType } from "types";
import { getCookiePartialData } from "utils/cookies";
import { defaultYou, getYou, YouInflated } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";

type UrlObject = { __typename: GqlModelType | `${GqlModelType}`, id?: string };

type UseObjectFromUrlProps<TData extends UrlObject> = {
    endpoint: string,
    objectType: TData["__typename"],
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => void,
}

export type UseObjectFromUrlReturn<TData extends UrlObject> = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
    isLoading: boolean;
    object: PartialWithType<TData>;
    permissions: YouInflated;
    setObject: Dispatch<SetStateAction<any>>;
}

/**
 * Hook for finding an object from the URL and providing relevant properties and functions
 */
export function useObjectFromUrl<TData extends UrlObject>({
    endpoint,
    objectType,
    onInvalidUrlParams,
}: UseObjectFromUrlProps<TData>): UseObjectFromUrlReturn<TData> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl({}), []);

    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);

    // Fetch data
    const [getData, { data, loading: isLoading, errors }] = useLazyFetch<any, TData>({ endpoint });
    const [object, setObject] = useState<PartialWithType<TData>>(getCookiePartialData({ __typename: objectType, id: urlParams.id, handle: urlParams.handle } as unknown as PartialWithType<TData>));
    useDisplayServerError(errors);
    useEffect(() => {
        console.log("parseSingleItemUrl", urlParams);
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        // If no valid identifier found, show error or call onInvalidUrlParams
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, objectType, stableOnInvalidUrlParams, urlParams]);
    useEffect(() => {
        setObject(data ?? getCookiePartialData({ __typename: objectType, id: urlParams.id, handle: urlParams.handle } as unknown as PartialWithType<TData>));
    }, [data, objectType, urlParams]);


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
