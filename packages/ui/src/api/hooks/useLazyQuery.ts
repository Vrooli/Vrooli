/**
 * Custom useLazyQuery hook to simplify usage
 */
import { TypedDocumentNode } from '@graphql-typed-document-node/core';
import { useCallback, useMemo, useRef } from 'react';
import { DocumentNode, LazyQueryHookOptions, LazyQueryResultTuple, mergeOptions, Operation, OperationVariables, QueryResult, useApolloClient } from '@apollo/client';
import { useInternalState } from '@apollo/client/react/hooks/useQuery';
import { IWrap, Wrap } from 'types';
import { exists } from '@shared/utils';

// The following methods, when called will execute the query, regardless of
// whether the useLazyQuery execute function was called before.
const EAGER_METHODS = [
    'refetch',
    'reobserve',
    'fetchMore',
    'updateQuery',
    'startPolling',
    'subscribeToMore',
] as const;

type UseLazyQueryResult<TData extends object | boolean | number | string | undefined, TVariables extends OperationVariables | undefined, Endpoint extends string = string> =
    TData extends undefined ?
    [() => {}, { data: Record<string, any>, loading: false }] : // Dummy data
    TVariables extends OperationVariables ?
    [Wrap<TData, Endpoint>, QueryResult<Wrap<TData, Endpoint>, TVariables>] :
    [Wrap<TData, Endpoint>, QueryResult<Wrap<TData, Endpoint>>];

type UseLazyQueryParams<TData extends object | boolean | number | string | undefined, TVariables extends OperationVariables | undefined, Endpoint extends string = string> = {
    query: DocumentNode | TypedDocumentNode<TData, TVariables> | undefined;
    endpoint: Endpoint;
    options?: TData extends undefined ? {} : TVariables extends OperationVariables ? LazyQueryHookOptions<TData, TVariables> : LazyQueryHookOptions<TData>;
}

export function useLazyQuery<
    TData extends object | boolean | number | string,
    TVariables extends OperationVariables | undefined,
    Endpoint extends string = string
>({
    query,
    endpoint, // Only used for type inference
    options,
}: UseLazyQueryParams<TData, TVariables, Endpoint>): UseLazyQueryResult<TData, TVariables, Endpoint> {
    const { variables, ...rest } = options || {};
    // If variables are provided (i.e. not undefined), wrap them in "input" to match the schema. We use this 
    // to avoid having to define the input fields in the DocumentNode definitions
    const opts = useMemo(() => ({ ...rest, variables: variables ? { input: variables } : undefined }), [rest, variables]);
    const internalState = useInternalState<Wrap<TData, Endpoint>, IWrap<TVariables>>(
        useApolloClient(opts && opts.client),
        query ?? ({ kind: 'Document', definitions: [] } as DocumentNode),
    );

    const execOptionsRef = useRef<Partial<LazyQueryHookOptions<Wrap<TData, Endpoint>, IWrap<TVariables>>>>();
    const merged = execOptionsRef.current
        ? mergeOptions(opts, execOptionsRef.current as OperationVariables)
        : opts;

    const useQueryResult = internalState.useQuery({
        ...merged,
        skip: !execOptionsRef.current,
    } as any);

    const initialFetchPolicy =
        useQueryResult.observable.options.initialFetchPolicy ||
        internalState.getDefaultFetchPolicy();

    const result =
        Object.assign(useQueryResult, {
            called: !!execOptionsRef.current,
        });

    // We use useMemo here to make sure the eager methods have a stable identity.
    const eagerMethods = useMemo(() => {
        const eagerMethods: Record<string, any> = {};
        for (const key of EAGER_METHODS) {
            const method: any = result[key];
            eagerMethods[key] = function () {
                if (!execOptionsRef.current) {
                    execOptionsRef.current = Object.create(null);
                    // Only the first time populating execOptionsRef.current matters here.
                    internalState.forceUpdate();
                }
                return method.apply(this, arguments);
            };
        }

        return eagerMethods;
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    Object.assign(result, eagerMethods);

    const execute = useCallback<
        LazyQueryResultTuple<Wrap<TData, Endpoint>, OperationVariables>[0]
    >(executeOptions => {
        execOptionsRef.current = (executeOptions ? {
            ...executeOptions,
            // Make sure variables are wrapped in "input" to match the schema
            variables: executeOptions.variables ? { input: executeOptions.variables } : undefined,
            fetchPolicy: executeOptions.fetchPolicy || initialFetchPolicy,
        } : {
            fetchPolicy: initialFetchPolicy,
        }) as any;

        const promise = (internalState as any)
            .asyncUpdate() // Like internalState.forceUpdate, but returns a Promise.
            .then(queryResult => Object.assign(queryResult, eagerMethods));

        // Because the return value of `useLazyQuery` is usually floated, we need
        // to catch the promise to prevent unhandled rejections.
        promise.catch(() => { });

        return promise as any;
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    // If the query is undefined, return a dummy function and dummy data
    if (!exists(query)) {
        return [() => { }, { data: {}, loading: false }] as any;
    }
    // Otherwise, return the actual data
    return [execute, result] as any;
}