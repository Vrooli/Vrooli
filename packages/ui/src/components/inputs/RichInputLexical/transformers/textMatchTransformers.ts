import { $createLinkNode, $isLinkNode, LinkNode } from "../nodes/LinkNode";
import { $createCodeBlockNode, CodeBlockNode } from "../plugins/CodePlugin";
import { TextMatchTransformer } from "../types";
import { $createTextNode, $isTextNode } from "../utils";

export const CODE_BLOCK: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        if (node instanceof CodeBlockNode) {
            const codeText = exportChildren(node);
            return `\`\`\`\n${codeText}\n\`\`\``;
        }
        return null;
    },
    importRegExp: /```(.*?)```/s,  // 's' flag for multiline support
    regExp: /```(.*?)```$/s,
    replace: (textNode, match) => {
        const codeContent = match[0].slice(3, -3); // Remove the surrounding backticks
        // Assuming $createCodeBlockNode is a function to create a code block node
        const codeBlockNode = $createCodeBlockNode(codeContent);
        textNode.replace(codeBlockNode);
    },
    trigger: "```",
    type: "text-match",
};

export const LINK: TextMatchTransformer = {
    dependencies: [LinkNode],
    export: (node, exportChildren, exportFormat) => {
        if (!$isLinkNode(node)) {
            return null;
        }
        const title = node.getTitle();
        const linkContent = title
            ? `[${node.getTextContent()}](${node.getURL()} "${title}")`
            : `[${node.getTextContent()}](${node.getURL()})`;
        const firstChild = node.getFirstChild();
        // Add text styles only if link has single text node inside. If it's more
        // then one we ignore it as markdown does not support nested styles for links
        if (node.getChildrenSize() === 1 && $isTextNode(firstChild)) {
            return exportFormat(firstChild, linkContent);
        } else {
            return linkContent;
        }
    },
    importRegExp:
        /(?:\[([^[]+)\])(?:\((?:([^()\s]+)(?:\s"((?:[^"]*\\")*[^"]*)"\s*)?)\))/,
    regExp:
        /(?:\[([^[]+)\])(?:\((?:([^()\s]+)(?:\s"((?:[^"]*\\")*[^"]*)"\s*)?)\))$/,
    replace: (textNode, match) => {
        const [, linkText, linkUrl, linkTitle] = match;
        const linkNode = $createLinkNode(linkUrl, { title: linkTitle });
        const linkTextNode = $createTextNode(linkText);
        linkTextNode.setFormat(textNode.getFormat());
        linkNode.append(linkTextNode);
        textNode.replace(linkNode);
    },
    trigger: ")",
    type: "text-match",
};

export const TEXT_MATCH_TRANSFORMERS = [CODE_BLOCK, LINK];
