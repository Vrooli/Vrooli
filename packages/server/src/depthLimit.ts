import { GraphQLError, Kind } from 'graphql';
import isArray from 'lodash/isArray';
import { genErrorCode, logger, LogLevel } from './logger';

interface Options {
    ignoreTypenames?: any
}

/**
 * Creates a validator for the GraphQL query depth. 
 * Setting a depth limit is useful for preventing client attacks.
 * @param {Number} maxDepth - The maximum allowed depth for any operation in a GraphQL document.
 * @param {Object} [options]
 * @param {Options} options - Options for changing the behavior of the validator.
 * @param {Function} [callback] - Called each time validation runs. Receives an Object which is a map of the depths for each operation. 
 * @returns {Function} The validator function for GraphQL validation phase.
 */
export const depthLimit = (maxDepth: number, options: Options = {}, callback: any = () => { }) => (validationContext: any) => {
    try {
        const { definitions } = validationContext.getDocument()
        const fragments = getFragments(definitions)
        const queries = getQueriesAndMutations(definitions)
        const queryDepths: any = {}
        for (let name in queries) {
            queryDepths[name] = determineDepth(queries[name], fragments, 0, maxDepth, validationContext, name, options)
        }
        callback(queryDepths)
        return validationContext
    } catch (error) {
        logger.log(LogLevel.error, 'Caught error finding depthLimit', { code: genErrorCode('0001'), error });
        throw error
    }
}

function getFragments(definitions: any) {
    return definitions.reduce((map: any, definition: any) => {
        if (definition.kind === Kind.FRAGMENT_DEFINITION) {
            map[definition.name.value] = definition
        }
        return map
    }, {})
}

// this will actually get both queries and mutations. we can basically treat those the same
function getQueriesAndMutations(definitions: any) {
    return definitions.reduce((map: any, definition: any) => {
        if (definition.kind === Kind.OPERATION_DEFINITION) {
            map[definition.name ? definition.name.value : ''] = definition
        }
        return map
    }, {})
}
/**
 * Recursively determines depth of a query node
 * @param node - The query node to determine depth for
 * @param fragments - The fragments in the query
 * @param {number} depthSoFar - The current depth of the query
 * @param {number} maxDepth - The maximum allowed depth allowed. Throws error if exceeded.
 * @param context - The GraphQL validation context
 * @param operationName - The name of the operation 
 * @param options - The options for the depth limit
 * @returns {Number} The depth of the query node
 */
const determineDepth: any = (node: any, fragments: any, depthSoFar: number, maxDepth: number, context: any, operationName: any, options: Options) => {
    if (node === undefined) {
        throw new Error('Node is undefined in depth limit. This usually means that your query is invalid. Please check if your query fragments are spelled correctly.')
    }
    if (depthSoFar > maxDepth) {
        return context.reportError(
            new GraphQLError(`'${operationName}' exceeds maximum operation depth of ${maxDepth}`, [node])
        )
    }

    switch (node.kind) {
        case Kind.FIELD:
            // by default, ignore the introspection fields which begin with double underscores
            const shouldIgnore = /^__/.test(node.name.value) || seeIfIgnored(node, options.ignoreTypenames)

            if (shouldIgnore || !node.selectionSet) {
                return 0
            }
            return 1 + Math.max(...node.selectionSet.selections.map((selection: any) =>
                determineDepth(selection, fragments, depthSoFar + 1, maxDepth, context, operationName, options)
            ))
        case Kind.FRAGMENT_SPREAD:
            return determineDepth(fragments[node.name.value], fragments, depthSoFar, maxDepth, context, operationName, options)
        case Kind.INLINE_FRAGMENT:
        case Kind.FRAGMENT_DEFINITION:
        case Kind.OPERATION_DEFINITION:
            return Math.max(...node.selectionSet.selections.map((selection: any) =>
                determineDepth(selection, fragments, depthSoFar, maxDepth, context, operationName, options)
            ))
        /* istanbul ignore next */
        default:
            throw new Error('uh oh! depth crawler cannot handle: ' + node.kind)
    }
}

function seeIfIgnored(node: any, ignore: any) {
    if (!ignore) return false;
    const ignoreArray = isArray(ignore) ? ignore : [ignore];
    for (let rule of ignoreArray) {
        const fieldName = node.name.value
        switch (rule.constructor) {
            case Function:
                if (rule(fieldName)) {
                    return true
                }
                break
            case String:
            case RegExp:
                if (fieldName.match(rule)) {
                    return true
                }
                break
            /* istanbul ignore next */
            default:
                throw new Error(`Invalid ignore option: ${rule}`)
        }
    }
    return false
}

