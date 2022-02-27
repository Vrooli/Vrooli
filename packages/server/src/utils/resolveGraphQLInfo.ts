import { FieldNode, FragmentDefinitionNode, FragmentSpreadNode, GraphQLResolveInfo, InlineFragmentNode, SelectionNode } from 'graphql';

/**
 * Finds the name of a SelectionNode
 * @param node SelectionNode to find name of
 * @returns Name of SelectionNode
 */
export function getSelectionNodeName(node: SelectionNode): string {
    switch (node.kind) {
        case 'Field':
        case 'FragmentSpread':
            return node.name.value;
        case 'InlineFragment':
            return `...${node.typeCondition?.name.value}`; // Ellipsis added to distinguish that it's a union type
    }
}

/**
 * Parses a GraphQL FieldNode type
 * @param node Field node
 * @param fragments All fragments in GraphQL info
 * @returns Either true, or a select object with fields as keys and "true" as values
 */
export function parseFieldNode(node: FieldNode, fragments: { [x: string]: FragmentDefinitionNode }): any {
    // Check if it's a primitive or object
    // If it has the "selectionSet" property, it's an object
    if (node.selectionSet) {
        // Call parseSelectionNode for each item in selection set
        let results: { [x: string]: any } = {};
        node.selectionSet.selections.forEach((selection: SelectionNode) => {
            const selectionName = getSelectionNodeName(selection);
            // If __typename, skip. These must be injected later
            if (selectionName === '__typename') return;
            results[selectionName] = parseSelectionNode(selection, fragments)
        });
        return results;
    }
    // If it doesn't have the "selectionSet" property, it's a primitive.
    else return true;
}

/**
 * Parses a GraphQL FragmentSpreadNode type
 * @param node FragmentSpread node
 * @param fragments All fragments in GraphQL info
 * @returns Select object with fields as keys and "true" as values
 */
export function parseFragmentSpreadNode(node: FragmentSpreadNode, fragments: { [x: string]: FragmentDefinitionNode }): { [x: string]: any } {
    // Get fragment
    const fragment: FragmentDefinitionNode = fragments[node.name.value];
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each selection
    fragment.selectionSet.selections.forEach((selection: SelectionNode) => {
        // Get name
        const selectionName = getSelectionNodeName(selection);
        // If __typename, skip. These must be injected later
        if (selectionName === '__typename') return;
        // Parse selection
        result[selectionName] = parseSelectionNode(selection, fragments);
    });
    // Find __typename
    result.__typename = fragment.typeCondition.name.value;
    // Return result
    return result;
}

/**
 * Parses a GraphQL InlineFragmentNode (union) type. Each type is its own field
 * @param node InlineFragment node
 * @param fragments All fragments in GraphQL info
 * @returns Select object with fields as keys and "true" as values
 */
export function parseInlineFragmentNode(node: InlineFragmentNode, fragments: { [x: string]: FragmentDefinitionNode }): { [x: string]: any } {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each selection (deconstructed union type)
    node.selectionSet.selections.forEach((selection: SelectionNode) => {
        // Get name
        const selectionName = getSelectionNodeName(selection);
        // Parse selection
        result[selectionName] = parseSelectionNode(selection, fragments);
    });
    // Return result
    return result;
}

/**
 * Parses any GraphQL SelectionNode type and returns an object with 
 * the formatted select fields as keys and "true" as values
 * @param node Current selection node
 * @param fragments All fragments in GraphQL info
 * @param typename If this is a root query, the typename is the name of the query. This cannot be found in 
 * @returns Select object with fields as keys and "true" as values
 */
export function parseSelectionNode(node: SelectionNode, fragments: { [x: string]: FragmentDefinitionNode }): { [key: string]: any } {
    // Determine which helper function to use
    switch (node.kind) {
        case 'Field':
            return parseFieldNode(node, fragments);
        case 'FragmentSpread':
            return parseFragmentSpreadNode(node, fragments);
        case 'InlineFragment': //TODO inline fragments are unions. 
            return parseInlineFragmentNode(node, fragments);
    }
}

/**
 * Converts a GraphQL resolve info object into an object that:
 * - Is in the shape of the GraphQL response object
 * - Has "true" for every key's value, except for the __typename key
 * @param info - GraphQL resolve info object
 */
export const resolveGraphQLInfo = (info: GraphQLResolveInfo): { [x: string]: any } => {
    //console.log('Q STRINGIFIED', JSON.stringify(info));
    // Get return type. This will be converted to __typename later
    let returnType: string = info.returnType.toString()
    console.log('Q RETURN TYPE', returnType);
    // Get selected nodes
    const selectionNodes = info.fieldNodes[0].selectionSet?.selections;
    console.log('Q SELECTION NODES', selectionNodes);
    if (!selectionNodes) return {};
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each selection node
    selectionNodes.forEach((selectionNode: SelectionNode) => {
        // Get name
        const selectionName = getSelectionNodeName(selectionNode);
        // If __typename, skip. These must be injected later
        if (selectionName === '__typename') return;
        // Parse selection
        result[selectionName] = parseSelectionNode(selectionNode, info.fragments);
    });
    return result;
}