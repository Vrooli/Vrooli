import { Headers } from "../../utils.js";
import { $createCodeNode } from "../nodes/CodeNode.js";
import { type ElementNode } from "../nodes/ElementNode.js";
import { type LexicalNode } from "../nodes/LexicalNode.js";
import { listExport, listReplace } from "../nodes/ListNode.js";
import { ElementTransformer } from "../types.js";
import { $createNode, $isNode, getPreviousSibling } from "../utils.js";

function createBlockNode(
    createNode: (match: Array<string>) => ElementNode,
): ElementTransformer["replace"] {
    return (parentNode, children, match) => {
        const node = createNode(match);
        node.append(...children);
        parentNode.replace(node);
        node.select(0, 0);
    };
}

export const HEADING: ElementTransformer = {
    export: (node, exportChildren) => {
        if (!$isNode("Heading", node)) {
            return null;
        }
        const level = Number(node.getTag().slice(1));
        return "#".repeat(level) + " " + exportChildren(node);
    },
    regExp: /^(#{1,6})\s/,
    replace: createBlockNode((match) => {
        const tag = ("h" + match[1].length) as Headers;
        return $createNode("Heading", { tag });
    }),
    type: "element",
};

export const QUOTE: ElementTransformer = {
    export: (node, exportChildren) => {
        if (!$isNode("Quote", node)) {
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
            const previousNode = getPreviousSibling(parentNode);
            if ($isNode("Quote", previousNode)) {
                previousNode.splice(previousNode.getChildrenSize(), 0, [
                    $createNode("LineBreak", {}),
                    ...children,
                ]);
                previousNode.select(0, 0);
                parentNode.remove();
                return;
            }
        }

        const node = $createNode("Quote", {});
        node.append(...children);
        parentNode.replace(node);
        node.select(0, 0);
    },
    type: "element",
};

export const CODE: ElementTransformer = {
    export: (node: LexicalNode) => {
        if (!$isNode("Code", node)) {
            return null;
        }
        const textContent = node.getTextContent();
        return (
            "```" +
            (node.__language || "") +
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
    export: (node, exportChildren) => {
        return $isNode("List", node) ? listExport(node, exportChildren, 0) : null;
    },
    regExp: /^(\s*)[-*+]\s/,
    replace: listReplace("bullet"),
    type: "element",
};

export const CHECK_LIST: ElementTransformer = {
    export: (node, exportChildren) => {
        return $isNode("List", node) ? listExport(node, exportChildren, 0) : null;
    },
    regExp: /^(\s*)(?:-\s)?\s?(\[(\s|x)?\])\s/i,
    replace: listReplace("check"),
    type: "element",
};

export const ORDERED_LIST: ElementTransformer = {
    export: (node, exportChildren) => {
        return $isNode("List", node) ? listExport(node, exportChildren, 0) : null;
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
