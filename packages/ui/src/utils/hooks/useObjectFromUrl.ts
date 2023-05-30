import { OperationVariables, TypedDocumentNode } from "@apollo/client";
import { exists, ParseSearchParamsResult } from "@local/shared";
import { useCustomLazyQuery } from "api";
import { DocumentNode } from "graphql";
import { Dispatch, SetStateAction, useEffect, useMemo, useState } from "react";
import { defaultYou, getYou, ListObjectType, YouInflated } from "utils/display/listTools";
import { parseSingleItemUrl } from "utils/navigation/urlTools";
import { PubSub } from "utils/pubsub";
import { useDisplayApolloError } from "./useDisplayApolloError";
import { useStableCallback } from "./useStableCallback";

type UseObjectFromUrlProps<
    TData extends ListObjectType,
    TVariables extends OperationVariables | undefined,
> = {
    query: DocumentNode | TypedDocumentNode<TData, TVariables>,
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
export function useObjectFromUrl<
    TData extends ListObjectType,
    TVariables extends OperationVariables | undefined,
>({
    query,
    onInvalidUrlParams,
    partialData,
    idFallback,
}: UseObjectFromUrlProps<TData, TVariables>): UseObjectFromUrlReturn<TData> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl(), []);

    const stableOnInvalidUrlParams = useStableCallback(onInvalidUrlParams);

    // Fetch data
    const [getData, { data, error, loading: isLoading }] = useCustomLazyQuery<TData, TVariables>(query, { errorPolicy: "all" } as any);
    const [object, setObject] = useState<TData | null | undefined>(null);
    useDisplayApolloError(error);
    useEffect(() => {
        console.log("parseSingleItemUrl", urlParams);
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ variables: { handle: urlParams.handle } as any });
        else if (exists(urlParams.handleRoot)) getData({ variables: { handleRoot: urlParams.handleRoot } as any });
        else if (exists(urlParams.id)) getData({ variables: { id: urlParams.id } as any });
        else if (exists(urlParams.idRoot)) getData({ variables: { idRoot: urlParams.idRoot } as any });
        else if (exists(idFallback)) getData({ variables: { id: idFallback } as any });
        // If no valid identifier found, show error or call onInvalidUrlParams
        else if (exists(stableOnInvalidUrlParams)) stableOnInvalidUrlParams(urlParams);
        else PubSub.get().publishSnack({ messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, idFallback, stableOnInvalidUrlParams, urlParams]);
    useEffect(() => {
        setObject(data ?? partialData as any);
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
