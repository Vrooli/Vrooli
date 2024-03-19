import { noop, OrArray } from "@local/shared";
import { Method } from "api";
import { ListObject } from "utils/display/listTools";
import { MakeLazyRequest, useLazyFetch, UseLazyFetchProps } from "./useLazyFetch";

type CommonProps<IsMutate extends boolean> = {
    isCreate: boolean,
    isMutate: IsMutate,
};
type Endpoint = { endpoint: string, method: Method };
type Endpoints = {
    endpointCreate: Endpoint,
    endpointUpdate: Endpoint,
};
type UseUpsertFetchParams<IsMutate extends boolean> = IsMutate extends false ?
    Partial<Endpoints> & CommonProps<IsMutate> :
    Endpoints & CommonProps<IsMutate>;

/**
 * Creates fetch logic for creating and updating an object. 
 * If `isMutate` is false, it returns noops instead of fetch functions.
 */
export const useUpsertFetch = <
    T extends OrArray<{ __typename: ListObject["__typename"], id: string }>,
    TCreateInput extends Record<string, any>,
    TUpdateInput extends Record<string, any>,
>({
    endpointCreate,
    endpointUpdate,
    isCreate,
    isMutate = true,
}: UseUpsertFetchParams<typeof isMutate>) => {
    const [fetchCreate, { loading: isCreateLoading }] = useLazyFetch<TCreateInput, T>(isMutate === false ? {} : endpointCreate as UseLazyFetchProps<TCreateInput>);
    const [fetchUpdate, { loading: isUpdateLoading }] = useLazyFetch<TUpdateInput, T>(isMutate === false ? {} : endpointUpdate as UseLazyFetchProps<TUpdateInput>);
    const fetch = (isMutate === false ? noop : isCreate ? fetchCreate : fetchUpdate) as MakeLazyRequest<TCreateInput | TUpdateInput, T>;

    return {
        fetch,
        fetchCreate,
        fetchUpdate,
        isCreateLoading,
        isUpdateLoading,
    };
};
