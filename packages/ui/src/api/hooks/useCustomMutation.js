import { useMutation } from "@apollo/client";
/**
 * A wrapper for useMutation that:
 * 1. Allows undefined data, so it can be lazy loaded
 * 2. Wrap variables in "input" object. Our entire API has inputs wrapped this way to
 * avoid having to specify endpoints in every query, so not having to specify this when
 * using the hook is a nice convenience
 * 3. Removes top-level object from result. Since each result is always wrapped in an object with the endpoint name,
 * this removes that top-level object so the result is just the data
 */
export function useCustomMutation(mutation, options) {
    // Create useMutation hook with blank mutation (if none defined) and modified options
    const [execute, { data, error, loading }] = useMutation(mutation !== null && mutation !== void 0 ? mutation : {
        kind: "Document", definitions: [{
                kind: "OperationDefinition",
                operation: "mutation",
                name: { kind: "Name", value: "EmptyMutation" },
                variableDefinitions: [],
                directives: [],
                selectionSet: { kind: "SelectionSet", selections: [] },
            }]
    }, Object.assign(Object.assign({}, options), { variables: (options === null || options === void 0 ? void 0 : options.variables) ? { input: options.variables } : undefined }));
    // When data is received, remove the top-level object
    const modifiedData = data && Object.values(data)[0];
    // Return the modified execute function and modified data
    return [execute, { data: modifiedData, error, loading }];
}
