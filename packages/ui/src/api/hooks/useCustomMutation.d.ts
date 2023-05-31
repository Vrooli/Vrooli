import { DocumentNode, MutationHookOptions, OperationVariables, TypedDocumentNode } from "@apollo/client";
type TDataDefined = object | boolean | number | string;
/**
 * A wrapper for useMutation that:
 * 1. Allows undefined data, so it can be lazy loaded
 * 2. Wrap variables in "input" object. Our entire API has inputs wrapped this way to
 * avoid having to specify endpoints in every query, so not having to specify this when
 * using the hook is a nice convenience
 * 3. Removes top-level object from result. Since each result is always wrapped in an object with the endpoint name,
 * this removes that top-level object so the result is just the data
 */
export declare function useCustomMutation<TData extends TDataDefined | undefined, TVariables extends OperationVariables | undefined>(mutation: DocumentNode | TypedDocumentNode<TData, TVariables> | undefined, options?: (TData extends TDataDefined ? MutationHookOptions<TData, TVariables> : MutationHookOptions<undefined, TVariables>) & {
    refetchQueries?: any;
}): readonly [(options?: import("@apollo/client").MutationFunctionOptions<any, OperationVariables, import("@apollo/client").DefaultContext, import("@apollo/client").ApolloCache<any>> | undefined) => Promise<import("@apollo/client").FetchResult<any, Record<string, any>, Record<string, any>>>, {
    readonly data: TData | undefined;
    readonly error: import("@apollo/client").ApolloError | undefined;
    readonly loading: boolean;
}];
export {};
