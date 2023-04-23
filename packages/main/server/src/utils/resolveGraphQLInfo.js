export function parseFieldNode(node, fragments) {
    if (node.selectionSet) {
        let results = {};
        node.selectionSet.selections.forEach((selection) => {
            results = parseSelectionNode(results, selection, fragments);
        });
        return results;
    }
    else
        return true;
}
export function parseFragmentSpreadNode(node, fragments) {
    const fragment = fragments[node.name.value];
    let result = {};
    fragment.selectionSet.selections.forEach((selection) => {
        result = parseSelectionNode(result, selection, fragments);
    });
    result.__typename = fragment.typeCondition.name.value;
    return result;
}
export function parseInlineFragmentNode(node, fragments) {
    let result = {};
    node.selectionSet.selections.forEach((selection) => {
        result = parseSelectionNode(result, selection, fragments);
    });
    return result;
}
export function parseSelectionNode(parsed, node, fragments) {
    const result = parsed;
    switch (node.kind) {
        case "Field":
            const inlineFragments = node.selectionSet?.selections.filter((selection) => selection.kind === "InlineFragment");
            const isUnion = inlineFragments?.length !== undefined ? inlineFragments.length === node.selectionSet.selections.length - 1 : false;
            if (isUnion) {
                const union = {};
                for (const selection of inlineFragments) {
                    union[`${selection.typeCondition?.name.value}`] = parseInlineFragmentNode(selection, fragments);
                }
                result[node.name.value] = union;
            }
            else {
                result[node.name.value] = parseFieldNode(node, fragments);
            }
            break;
        case "FragmentSpread":
            const spread = parseFragmentSpreadNode(node, fragments);
            for (const key in spread) {
                result[key] = spread[key];
            }
            break;
        case "InlineFragment":
            result[`${node.typeCondition?.name.value}`] = parseInlineFragmentNode(node, fragments);
            break;
    }
    return result;
}
export const resolveGraphQLInfo = (info) => {
    const selectionNodes = info.fieldNodes[0].selectionSet?.selections;
    if (!selectionNodes)
        return {};
    let result = {};
    selectionNodes.forEach((selectionNode) => {
        result = parseSelectionNode(result, selectionNode, info.fragments);
    });
    return result;
};
//# sourceMappingURL=resolveGraphQLInfo.js.map