import { exists, ParseSearchParamsResult } from "@local/shared";
import { Dispatch, SetStateAction, useEffect, useMemo, useState } from "react";
import { defaultYou, getYou, ListObjectType, YouInflated } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { useDisplayServerError } from "./useDisplayServerError";
import { useLazyFetch } from "./useLazyFetch";
import { useStableCallback } from "./useStableCallback";

type UseObjectFromUrlProps<TData extends ListObjectType> = {
    endpoint: string,
    partialData?: Partial<TData>,
    idFallback?: string | null | undefined,
    onInvalidUrlParams?: (params: ParseSearchParamsResult) => void,
}

export type UseObjectFromUrlReturn<TData extends object> = {
    handleRoot?: string,
    handle?: string,
    idRoot?: string,
    id?: string,
    isLoading: boolean;
    object: TData | null | undefined;
    permissions: YouInflated;
    setObject: Dispatch<SetStateAction<any>>;
}

/**
 * Hook for finding an object from the URL and providing relevant properties and functions
 */
export function useObjectFromUrl<TData extends ListObjectType>({
    endpoint,
    onInvalidUrlParams,
    partialData,
    idFallback,
}: UseObjectFromUrlProps<TData>): UseObjectFromUrlReturn<TData> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl(), []);

    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);

    // Fetch data
    const [getData, { data, loading: isLoading, error }] = useLazyFetch<any, TData>(endpoint);
    const [object, setObject] = useState<TData | null | undefined>(null);
    useDisplayServerError(error);
    useEffect(() => {
        console.log("parseSingleItemUrl", urlParams);
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ handle: urlParams.handle });
        else if (exists(urlParams.handleRoot)) getData({ handleRoot: urlParams.handleRoot });
        else if (exists(urlParams.id)) getData({ id: urlParams.id });
        else if (exists(urlParams.idRoot)) getData({ idRoot: urlParams.idRoot });
        else if (exists(idFallback)) getData({ id: idFallback });
        // If no valid identifier found, show error or call onInvalidUrlParams
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, idFallback, stableOnInvalidUrlParams, urlParams]);
    useEffect(() => {
        setObject(data ?? partialData as TData);
    }, [data, partialData]);


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
