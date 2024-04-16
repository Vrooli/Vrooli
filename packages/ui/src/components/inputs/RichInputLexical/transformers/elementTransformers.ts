import { $createCodeNode, $isCodeNode, CodeNode } from "../nodes/CodeNode";
import { ElementNode } from "../nodes/ElementNode";
import { $createHeadingNode, $isHeadingNode, HeadingNode } from "../nodes/HeadingNode";
import { LexicalNode } from "../nodes/LexicalNode";
import { $createLineBreakNode } from "../nodes/LineBreakNode";
import { ListItemNode } from "../nodes/ListItemNode";
import { $isListNode, ListNode, listExport, listReplace } from "../nodes/ListNode";
import { $createQuoteNode, $isQuoteNode, QuoteNode } from "../nodes/QuoteNode";
import { ElementTransformer, HeadingTagType } from "../types";

const createBlockNode = (
    createNode: (match: Array<string>) => ElementNode,
): ElementTransformer["replace"] => {
    return (parentNode, children, match) => {
        const node = createNode(match);
        node.append(...children);
        parentNode.replace(node);
        node.select(0, 0);
    };
};

export const HEADING: ElementTransformer = {
    dependencies: [HeadingNode],
    export: (node, exportChildren) => {
        if (!$isHeadingNode(node)) {
            return null;
        }
        const level = Number(node.getTag().slice(1));
        return "#".repeat(level) + " " + exportChildren(node);
    },
    regExp: /^(#{1,6})\s/,
    replace: createBlockNode((match) => {
        const tag = ("h" + match[1].length) as HeadingTagType;
        return $createHeadingNode(tag);
    }),
    type: "element",
};

export const QUOTE: ElementTransformer = {
    dependencies: [QuoteNode],
    export: (node, exportChildren) => {
        if (!$isQuoteNode(node)) {
            return null;
        }

        const lines = exportChildren(node).split("\n");
        const output: string[] = [];
        for (const line of lines) {
            output.push("> " + line);
        }
        return output.join("\n");
    },
    regExp: /^>\s/,
    replace: (parentNode, children, _match, isImport) => {
        if (isImport) {
            const previousNode = parentNode.getPreviousSibling();
            if ($isQuoteNode(previousNode)) {
                previousNode.splice(previousNode.getChildrenSize(), 0, [
                    $createLineBreakNode(),
                    ...children,
                ]);
                previousNode.select(0, 0);
                parentNode.remove();
                return;
            }
        }

        const node = $createQuoteNode();
        node.append(...children);
        parentNode.replace(node);
        node.select(0, 0);
    },
    type: "element",
};

export const CODE: ElementTransformer = {
    dependencies: [CodeNode],
    export: (node: LexicalNode) => {
        if (!$isCodeNode(node)) {
            return null;
        }
        const textContent = node.getTextContent();
        return (
            "```" +
            (node.getLanguage() || "") +
            (textContent ? "\n" + textContent : "") +
            "\n" +
            "```"
        );
    },
    regExp: /^```(\w{1,10})?\s/,
    replace: createBlockNode((match) => {
        return $createCodeNode(match ? match[1] : undefined);
    }),
    type: "element",
};

export const UNORDERED_LIST: ElementTransformer = {
    dependencies: [ListNode, ListItemNode],
    export: (node, exportChildren) => {
        return $isListNode(node) ? listExport(node, exportChildren, 0) : null;
    },
    regExp: /^(\s*)[-*+]\s/,
    replace: listReplace("bullet"),
    type: "element",
};

export const CHECK_LIST: ElementTransformer = {
    dependencies: [ListNode, ListItemNode],
    export: (node, exportChildren) => {
        return $isListNode(node) ? listExport(node, exportChildren, 0) : null;
    },
    regExp: /^(\s*)(?:-\s)?\s?(\[(\s|x)?\])\s/i,
    replace: listReplace("check"),
    type: "element",
};

export const ORDERED_LIST: ElementTransformer = {
    dependencies: [ListNode, ListItemNode],
    export: (node, exportChildren) => {
        return $isListNode(node) ? listExport(node, exportChildren, 0) : null;
    },
    regExp: /^(\s*)(\d{1,})\.\s/,
    replace: listReplace("number"),
    type: "element",
};

export const ELEMENT_TRANSFORMERS = [
    HEADING,
    QUOTE,
    CODE,
    UNORDERED_LIST,
    CHECK_LIST,
    ORDERED_LIST,
];
