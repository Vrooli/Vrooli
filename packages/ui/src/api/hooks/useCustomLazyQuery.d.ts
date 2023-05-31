import { DocumentNode, LazyQueryHookOptions, OperationVariables, TypedDocumentNode } from "@apollo/client";
type TDataDefined = object | boolean | number | string;
/**
 * A wapper for useLazyQuery that:
 * 1. Allows undefined data, so it can be lazy loaded
 * 2. Wrap variables in "input" object. Our entire API has inputs wrapped this way to
 * avoid having to specify endpoints in every query, so not having to specify this when
 * using the hook is a nice convenience
 * 3. Removes top-level object from result. Since each result is always wrapped in an object with the endpoint name,
 * this removes that top-level object so the result is just the data
 */
export declare function useCustomLazyQuery<TData extends TDataDefined | undefined, TVariables extends OperationVariables | undefined>(query: DocumentNode | TypedDocumentNode<TData, TVariables> | undefined, options?: (TData extends TDataDefined ? TVariables extends OperationVariables ? LazyQueryHookOptions<TData, TVariables> : TData extends TDataDefined ? LazyQueryHookOptions<TData, {}> : Record<string, any> : Record<string, any>)): readonly [(props?: any) => Promise<import("@apollo/client").QueryResult<any, OperationVariables>>, {
    readonly data: TData | undefined;
    readonly error: import("@apollo/client").ApolloError | undefined;
    readonly loading: boolean;
    readonly refetch: (variables?: Partial<OperationVariables> | undefined) => Promise<import("@apollo/client").ApolloQueryResult<any>>;
}];
export {};
