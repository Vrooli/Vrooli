import { GraphQLError, Kind } from "graphql";
import { logger } from "../events/logger";
export const depthLimit = (maxDepth, options = {}, callback = () => { }) => (validationContext) => {
    try {
        const { definitions } = validationContext.getDocument();
        const fragments = getFragments(definitions);
        const queries = getQueriesAndMutations(definitions);
        const queryDepths = {};
        for (const name in queries) {
            queryDepths[name] = determineDepth(queries[name], fragments, 0, maxDepth, validationContext, name, options);
        }
        callback(queryDepths);
        return validationContext;
    }
    catch (error) {
        logger.error("Caught error finding depthLimit", { trace: "0001", error });
        throw error;
    }
};
function getFragments(definitions) {
    return definitions.reduce((map, definition) => {
        if (definition.kind === Kind.FRAGMENT_DEFINITION) {
            map[definition.name.value] = definition;
        }
        return map;
    }, {});
}
function getQueriesAndMutations(definitions) {
    return definitions.reduce((map, definition) => {
        if (definition.kind === Kind.OPERATION_DEFINITION) {
            map[definition.name ? definition.name.value : ""] = definition;
        }
        return map;
    }, {});
}
const determineDepth = (node, fragments, depthSoFar, maxDepth, context, operationName, options) => {
    if (node === undefined) {
        throw new Error("Node is undefined in depth limit. This usually means that your query is invalid. Please check if your query fragments are spelled correctly.");
    }
    if (depthSoFar > maxDepth) {
        return context.reportError(new GraphQLError(`'${operationName}' exceeds maximum operation depth of ${maxDepth}`, [node]));
    }
    switch (node.kind) {
        case Kind.FIELD:
            const shouldIgnore = /^__/.test(node.name.value) || seeIfIgnored(node, options.ignoreTypenames);
            if (shouldIgnore || !node.selectionSet) {
                return 0;
            }
            return 1 + Math.max(...node.selectionSet.selections.map((selection) => determineDepth(selection, fragments, depthSoFar + 1, maxDepth, context, operationName, options)));
        case Kind.FRAGMENT_SPREAD:
            return determineDepth(fragments[node.name.value], fragments, depthSoFar, maxDepth, context, operationName, options);
        case Kind.INLINE_FRAGMENT:
        case Kind.FRAGMENT_DEFINITION:
        case Kind.OPERATION_DEFINITION:
            return Math.max(...node.selectionSet.selections.map((selection) => determineDepth(selection, fragments, depthSoFar, maxDepth, context, operationName, options)));
        default:
            throw new Error("uh oh! depth crawler cannot handle: " + node.kind);
    }
};
function seeIfIgnored(node, ignore) {
    if (!ignore)
        return false;
    const ignoreArray = Array.isArray(ignore) ? ignore : [ignore];
    for (const rule of ignoreArray) {
        const fieldName = node.name.value;
        switch (rule.constructor) {
            case Function:
                if (rule(fieldName)) {
                    return true;
                }
                break;
            case String:
            case RegExp:
                if (fieldName.match(rule)) {
                    return true;
                }
                break;
            default:
                throw new Error(`Invalid ignore option: ${rule}`);
        }
    }
    return false;
}
//# sourceMappingURL=depthLimit.js.map