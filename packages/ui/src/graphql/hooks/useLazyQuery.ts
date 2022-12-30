/**
 * Custom useLazyQuery hook to simplify usage
 */
import { DocumentNode } from 'graphql';
import { TypedDocumentNode } from '@graphql-typed-document-node/core';
import { useCallback, useMemo, useRef } from 'react';
import { LazyQueryHookOptions, LazyQueryResultTuple, mergeOptions, OperationVariables, QueryResult, useApolloClient } from '@apollo/client';
import { useInternalState } from '@apollo/client/react/hooks/useQuery';
import { IWrap, Wrap } from 'types';

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

export function useLazyQuery<
    TData extends object | boolean | number | string,
    TVariables extends OperationVariables | null,
    Endpoint extends string = string
>(
    query: DocumentNode | TypedDocumentNode<TData, TVariables>,
    endpoint: Endpoint, // Only used for type inference
    options?: LazyQueryHookOptions<TData, TVariables>
): LazyQueryResultTuple<Wrap<TData, Endpoint>, TVariables> {
    const { variables, ...rest } = options || {};
    // If variables are provided (i.e. not undefined), wrap them in "input" to match the schema. We use this 
    // to avoid having to define the input fields in the DocumentNode definitions
    const opts = useMemo(() => ({ ...rest, variables: variables ? { input: variables } : undefined }), [rest, variables]);
    const internalState = useInternalState<Wrap<TData, Endpoint>, IWrap<TVariables>>(
        useApolloClient(opts && opts.client),
        query,
    );

    const execOptionsRef = useRef<Partial<LazyQueryHookOptions<Wrap<TData, Endpoint>, IWrap<TVariables>>>>();
    const merged = execOptionsRef.current
        ? mergeOptions(opts, execOptionsRef.current)
        : opts;

    const useQueryResult = internalState.useQuery({
        ...merged,
        skip: !execOptionsRef.current,
    } as any);

    const initialFetchPolicy =
        useQueryResult.observable.options.initialFetchPolicy ||
        internalState.getDefaultFetchPolicy();

    const result: QueryResult<Wrap<TData, Endpoint>, IWrap<TVariables>> =
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
    }, []);

    Object.assign(result, eagerMethods);

    const execute = useCallback<
        LazyQueryResultTuple<Wrap<TData, Endpoint>, TVariables>[0]
    >(executeOptions => {
        execOptionsRef.current = (executeOptions ? {
            ...executeOptions,
            // Make sure variables are wrapped in "input" to match the schema
            variables: executeOptions.variables ? { input: executeOptions.variables } : undefined,
            fetchPolicy: executeOptions.fetchPolicy || initialFetchPolicy,
        } : {
            fetchPolicy: initialFetchPolicy,
        }) as any;

        const promise = internalState
            .asyncUpdate() // Like internalState.forceUpdate, but returns a Promise.
            .then(queryResult => Object.assign(queryResult, eagerMethods));

        // Because the return value of `useLazyQuery` is usually floated, we need
        // to catch the promise to prevent unhandled rejections.
        promise.catch(() => { });

        return promise as any;
    }, []);

    return [execute, result] as any;
}