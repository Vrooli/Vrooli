import { useLazyQuery } from "@apollo/client";
import { useMemo } from "react";
export function useCustomLazyQuery(query, options) {
    const [execute, { data, error, loading, refetch }] = useLazyQuery(query ?? {
        kind: "Document", definitions: [{
                kind: "OperationDefinition",
                operation: "query",
                name: { kind: "Name", value: "EmptyQuery" },
                variableDefinitions: [],
                directives: [],
                selectionSet: { kind: "SelectionSet", selections: [] },
            }],
    }, {
        ...options,
        variables: options?.variables ? { input: options.variables } : undefined,
    });
    const modifiedData = data && Object.values(data)[0];
    const executeWithVariables = useMemo(() => {
        return (props) => {
            if (props?.variables) {
                return execute({ ...props, variables: { input: props.variables } });
            }
            return execute(props);
        };
    }, [execute]);
    return [executeWithVariables, { data: modifiedData, error, loading, refetch }];
}
//# sourceMappingURL=useCustomLazyQuery.js.map