import { type TextMatchTransformer } from "../types.js";
import { $createNode, $isNode } from "../utils.js";

export const CODE_BLOCK: TextMatchTransformer = {
    export: (node, exportChildren, exportFormat) => {
        if ($isNode("Code", node)) {
            const codeText = exportChildren(node);
            return `\`\`\`\n${codeText}\n\`\`\``;
        }
        return null;
    },
    importRegExp: /```(.*?)```/s,  // 's' flag for multiline support
    regExp: /```(.*?)```$/s,
    replace: (textNode, match) => {
        const textContent = match[0].slice(3, -3); // Remove the surrounding backticks
        textNode.setTextContent(textContent);
        const codeNode = $createNode("Code", {});
        textNode.replace(codeNode);
    },
    trigger: "```",
    type: "text-match",
};

export const LINK: TextMatchTransformer = {
    export: (node, exportChildren, exportFormat) => {
        if (!$isNode("Link", node)) {
            return null;
        }
        const title = node.getTitle();
        const linkContent = title
            ? `[${node.getTextContent()}](${node.getURL()} "${title}")`
            : `[${node.getTextContent()}](${node.getURL()})`;
        const firstChild = node.getFirstChild();
        // Add text styles only if link has single text node inside. If it's more
        // then one we ignore it as markdown does not support nested styles for links
        if (node.getChildrenSize() === 1 && $isNode("Text", firstChild)) {
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
        const linkNode = $createNode("Link", { title: linkTitle, url: linkUrl });
        const linkTextNode = $createNode("Text", { text: linkText });
        linkTextNode.setFormat(textNode.getFormat());
        linkNode.append(linkTextNode);
        textNode.replace(linkNode);
    },
    trigger: ")",
    type: "text-match",
};

export const TEXT_MATCH_TRANSFORMERS = [CODE_BLOCK, LINK];
