import { useMutation } from "@apollo/client";
export function useCustomMutation(mutation, options) {
    const [execute, { data, error, loading }] = useMutation(mutation ?? {
        kind: "Document", definitions: [{
                kind: "OperationDefinition",
                operation: "mutation",
                name: { kind: "Name", value: "EmptyMutation" },
                variableDefinitions: [],
                directives: [],
                selectionSet: { kind: "SelectionSet", selections: [] },
            }],
    }, {
        ...options,
        variables: options?.variables ? { input: options.variables } : undefined,
    });
    const modifiedData = data && Object.values(data)[0];
    return [execute, { data: modifiedData, error, loading }];
}
//# sourceMappingURL=useCustomMutation.js.map