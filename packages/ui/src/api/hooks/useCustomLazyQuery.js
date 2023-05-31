import { useLazyQuery } from "@apollo/client";
import { useMemo } from "react";
/**
 * A wapper for useLazyQuery that:
 * 1. Allows undefined data, so it can be lazy loaded
 * 2. Wrap variables in "input" object. Our entire API has inputs wrapped this way to
 * avoid having to specify endpoints in every query, so not having to specify this when
 * using the hook is a nice convenience
 * 3. Removes top-level object from result. Since each result is always wrapped in an object with the endpoint name,
 * this removes that top-level object so the result is just the data
 */
export function useCustomLazyQuery(query, options) {
    // Create useLazyQuery hook with blank query (if none defined) and modified options
    const [execute, { data, error, loading, refetch }] = useLazyQuery(query !== null && query !== void 0 ? query : {
        kind: "Document", definitions: [{
                kind: "OperationDefinition",
                operation: "query",
                name: { kind: "Name", value: "EmptyQuery" },
                variableDefinitions: [],
                directives: [],
                selectionSet: { kind: "SelectionSet", selections: [] },
            }],
    }, Object.assign(Object.assign({}, options), { variables: (options === null || options === void 0 ? void 0 : options.variables) ? { input: options.variables } : undefined }));
    // When data is received, remove the top-level object
    const modifiedData = data && Object.values(data)[0];
    // If variables passed into execute, make sure they are wrapped in "input" object
    const executeWithVariables = useMemo(() => {
        return (props) => {
            if (props === null || props === void 0 ? void 0 : props.variables) {
                return execute(Object.assign(Object.assign({}, props), { variables: { input: props.variables } }));
            }
            return execute(props);
        };
    }, [execute]);
    // Return the modified execute function and modified data
    return [executeWithVariables, { data: modifiedData, error, loading, refetch }];
}
