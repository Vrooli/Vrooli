/**
 * Custom useMutation hook to simplify types
 */
import { useCallback, useEffect, useRef, useState } from 'react';
import { DocumentNode } from 'graphql';
import { TypedDocumentNode } from '@graphql-typed-document-node/core';
import { equal } from '@wry/equality';
import { ApolloCache, ApolloError, DefaultContext, mergeOptions, MutationFunctionOptions, MutationHookOptions, MutationResult, MutationTuple, OperationVariables, useApolloClient } from '@apollo/client';
import { DocumentType, verifyDocumentType } from '@apollo/client/react/parser';

export function useMutation<
    TData extends object | boolean | number | string,
    TVariables extends OperationVariables | null,
    Endpoint extends string = string,
    TContext = DefaultContext,
    TCache extends ApolloCache<any> = ApolloCache<any>,
>(
    mutation: DocumentNode | TypedDocumentNode<TData, TVariables>,
    endpoint: Endpoint, // Only used for type inference
    options?: MutationHookOptions<TData, TVariables, TContext>,
): MutationTuple<TData, TVariables, TContext, TCache> {
    const client = useApolloClient(options?.client);
    verifyDocumentType(mutation, DocumentType.Mutation);
    const [result, setResult] = useState<Omit<MutationResult, 'reset'>>({
        called: false,
        loading: false,
        client,
    });

    const ref = useRef({
        result,
        mutationId: 0,
        isMounted: true,
        client,
        mutation,
        options,
    });

    // TODO: Trying to assign these in a useEffect or useLayoutEffect breaks
    // higher-order components.
    // eslint-disable-next-line no-lone-blocks
    {
        Object.assign(ref.current, { client, options, mutation });
    }

    const execute = useCallback((
        executeOptions: MutationFunctionOptions<
            TData,
            TVariables,
            TContext,
            TCache
        > = {}
    ) => {
        const { client, options, mutation } = ref.current;
        const baseOptions = { ...options, mutation };
        if (!ref.current.result.loading && !baseOptions.ignoreResults) {
            setResult(ref.current.result = {
                loading: true,
                error: void 0,
                data: void 0,
                called: true,
                client,
            });
        }

        const mutationId = ++ref.current.mutationId;
        const clientOptions = mergeOptions(
            baseOptions,
            executeOptions as any,
        );

        return client.mutate(clientOptions).then((response) => {
            const { data, errors } = response;
            const error =
                errors && errors.length > 0
                    ? new ApolloError({ graphQLErrors: errors })
                    : void 0;

            if (
                mutationId === ref.current.mutationId &&
                !clientOptions.ignoreResults
            ) {
                const result = {
                    called: true,
                    loading: false,
                    data,
                    error,
                    client,
                };

                if (ref.current.isMounted && !equal(ref.current.result, result)) {
                    setResult(ref.current.result = result);
                }
            }

            ref.current.options?.onCompleted?.(response.data!);
            executeOptions.onCompleted?.(response.data!);
            return response;
        }).catch((error) => {
            if (
                mutationId === ref.current.mutationId &&
                ref.current.isMounted
            ) {
                const result = {
                    loading: false,
                    error,
                    data: void 0,
                    called: true,
                    client,
                };

                if (!equal(ref.current.result, result)) {
                    setResult(ref.current.result = result);
                }
            }

            if (ref.current.options?.onError || clientOptions.onError) {
                ref.current.options?.onError?.(error);
                executeOptions.onError?.(error);
                // TODO(brian): why are we returning this here???
                return { data: void 0, errors: error };
            }

            throw error;
        });
    }, []);

    const reset = useCallback(() => {
        setResult({ called: false, loading: false, client });
    }, []);

    useEffect(() => {
        ref.current.isMounted = true;

        return () => {
            ref.current.isMounted = false;
        };
    }, []);

    return [execute, { reset, ...result }];
}