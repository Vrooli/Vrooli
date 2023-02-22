import { OperationVariables, TypedDocumentNode } from "@apollo/client";
import { Session } from "@shared/consts";
import { ParseSearchParamsResult } from "@shared/route";
import { exists } from "@shared/utils";
import { useLazyQuery } from "api";
import { DocumentNode } from "graphql";
import { Dispatch, SetStateAction, useEffect, useMemo, useState } from "react";
import { defaultYou, getYou, ListObjectType, YouInflated } from "utils/display";
import { parseSingleItemUrl } from "utils/navigation";
import { PubSub } from "utils/pubsub";
import { useDisplayApolloError } from "./useDisplayApolloError";

type UseObjectFromUrlProps<
    TData extends ListObjectType,
    TVariables extends OperationVariables | null,
    Endpoint extends string = string
> = {
    query: DocumentNode | TypedDocumentNode<TData, TVariables>,
    endpoint: Endpoint,
    partialData?: Partial<TData>,
    session: Session,
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
    TVariables extends OperationVariables | null,
    Endpoint extends string = string
>({
    query,
    endpoint,
    onInvalidUrlParams,
    partialData,
    session,
    idFallback,
}: UseObjectFromUrlProps<TData, TVariables, Endpoint>): UseObjectFromUrlReturn<TData> {
    // Get URL params
    const urlParams = useMemo(() => parseSingleItemUrl(), []);

    // Fetch data
    const [getData, { data, error, loading: isLoading }] = useLazyQuery<TData, TVariables>(query, endpoint, { errorPolicy: 'all' });
    const [object, setObject] = useState<TData | null | undefined>(null);
    useDisplayApolloError(error);
    useEffect(() => {
        // Objects can be found using a few different unique identifiers
        if (exists(urlParams.handle)) getData({ variables: { handle: urlParams.handle } as any })
        else if (exists(urlParams.handleRoot)) getData({ variables: { handleRoot: urlParams.handleRoot } as any })
        else if (exists(urlParams.id)) getData({ variables: { id: urlParams.id } as any })
        else if (exists(urlParams.idRoot)) getData({ variables: { idRoot: urlParams.idRoot } as any })
        else if (exists(idFallback)) getData({ variables: { id: idFallback } as any })
        // If no valid identifier found, show error or call onInvalidUrlParams
        else if (exists(onInvalidUrlParams)) onInvalidUrlParams(urlParams);
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, idFallback, onInvalidUrlParams, urlParams]);
    useEffect(() => {
        setObject(data?.[endpoint] ?? partialData as any);
    }, [data, endpoint, partialData]);


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