import { DocumentNode, MutationHookOptions, OperationVariables, TypedDocumentNode, useMutation } from "@apollo/client";

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
export function useCustomMutation<
    TData extends TDataDefined | undefined,
    TVariables extends OperationVariables | undefined
>(
    mutation: DocumentNode | TypedDocumentNode<TData, TVariables> | undefined,
    options?: (TData extends TDataDefined
        ? MutationHookOptions<TData, TVariables>
        : MutationHookOptions<undefined, TVariables>) &
    { refetchQueries?: any },
) {
    // Create useMutation hook with blank mutation (if none defined) and modified options
    const [execute, { data, error, loading }] = useMutation(
        mutation ?? ({
            kind: "Document", definitions: [{
                kind: "OperationDefinition",
                operation: "mutation",
                name: { kind: "Name", value: "EmptyMutation" },
                variableDefinitions: [],
                directives: [],
                selectionSet: { kind: "SelectionSet", selections: [] },
            }]
        } as DocumentNode),
        {
            ...options,
            variables: options?.variables ? { input: options.variables } : undefined,
        } as any,
    );

    // When data is received, remove the top-level object
    const modifiedData: TData | undefined = data && Object.values(data)[0];

    // Return the modified execute function and modified data
    return [execute, { data: modifiedData, error, loading }] as const;
}
