// NOTE: Much of this file is taken from the lexical package, which is licensed under the MIT license. 
// We have copied the code here for customization purposes, such as replacing the default code block component
import { $isListItemNode, $isListNode } from "@lexical/list";
import { $isQuoteNode } from "@lexical/rich-text";
import { $findMatchingParent } from "@lexical/utils";
import "highlight.js/styles/monokai-sublime.css";
import { $createLineBreakNode, $createParagraphNode, $createTextNode, $getRoot, $getSelection, $isDecoratorNode, $isElementNode, $isLineBreakNode, $isParagraphNode, $isTextNode, ElementNode, LexicalNode } from "lexical";
import { DeviceOS, getDeviceInfo } from "utils/display/device";
import { $createCodeBlockNode, CodeBlockNode } from "./plugins/code/CodePlugin";
import { ElementTransformer, TextFormatTransformer, TextMatchTransformer, Transformer } from "./types";

/**
 * Matches any punctuation or whitespace character.
 * This regex is designed to identify punctuation symbols and spaces in a string. It includes a wide range
 * of punctuation marks (from '!' to '/', ':', to '@', '[', to '`', and '{' to '~') as well as any whitespace
 * character (denoted by \s). It's useful for parsing tasks where punctuation or spaces need to be identified
 * or separated from other text content.
 */
const PUNCTUATION_OR_SPACE = /[!-/:-@[-`{-~\s]/;
/**
 * Matches an empty line or a line with up to 3 leading spaces in markdown content.
 * This regex is used to identify empty lines within markdown content, which are important for recognizing
 * block elements or separating paragraphs. The regex allows for up to 3 spaces at the beginning of the line,
 * accommodating markdown syntax that considers lines with four or more leading spaces as code blocks.
 */
const MARKDOWN_EMPTY_LINE_REG_EXP = /^\s{0,3}$/;
/**
 * Matches the opening or closing line of a code block in markdown content.
 * This regex is used to detect lines that signify the start or end of a fenced code block in markdown,
 * which is indicated by three backticks (```). The regex optionally captures a language identifier
 * following the backticks, allowing for syntax highlighting hints. The language identifier is expected
 * to be an alphanumeric string up to 10 characters long. The regex also allows for an optional space
 * following the backticks but before the language identifier.
 */
const CODE_BLOCK_REG_EXP = /^```(\w{1,10})?\s?$/;

/**
 * Finds first "<tag>content<tag>" match that is not nested into another tag
 */
function findOutermostMatch(textContent, textTransformersIndex) {
    const openTagsMatch = textContent.match(textTransformersIndex.openTagsRegExp);
    if (openTagsMatch == null) {
        return null;
    }
    for (const match of openTagsMatch) {
        // Open tags reg exp might capture leading space so removing it
        // before using match to find transformer
        const tag = match.replace(/^\s/, '');
        const fullMatchRegExp = textTransformersIndex.fullMatchRegExpByTag[tag];
        if (fullMatchRegExp == null) {
            continue;
        }
        const fullMatch = textContent.match(fullMatchRegExp);
        const transformer = textTransformersIndex.transformersByTag[tag];
        if (fullMatch != null && transformer != null) {
            if (transformer.intraword !== false) {
                return fullMatch;
            }

            // For non-intraword transformers checking if it's within a word
            // or surrounded with space/punctuation/newline
            const {
                index = 0
            } = fullMatch;
            const beforeChar = textContent[index - 1];
            const afterChar = textContent[index + fullMatch[0].length];
            if ((!beforeChar || PUNCTUATION_OR_SPACE.test(beforeChar)) && (!afterChar || PUNCTUATION_OR_SPACE.test(afterChar))) {
                return fullMatch;
            }
        }
    }
    return null;
}

/**
 * Processes text content and replaces text format tags.
 * It takes outermost tag match and its content, creates text node with
 * format based on tag and then recursively executed over node's content
 * 
 * E.g. for "*Hello **world**!*" string it will create text node with
 * "Hello **world**!" content and italic format and run recursively over
 * its content to transform "**world**" part
 */
const importTextFormatTransformers = (textNode, textFormatTransformersIndex, textMatchTransformers) => {
    const textContent = textNode.getTextContent();
    const match = findOutermostMatch(textContent, textFormatTransformersIndex);
    if (!match) {
        // Once text format processing is done run text match transformers, as it
        // only can span within single text node (unline formats that can cover multiple nodes)
        importTextMatchTransformers(textNode, textMatchTransformers);
        return;
    }
    let currentNode, remainderNode, leadingNode;

    // If matching full content there's no need to run splitText and can reuse existing textNode
    // to update its content and apply format. E.g. for **_Hello_** string after applying bold
    // format (**) it will reuse the same text node to apply italic (_)
    if (match[0] === textContent) {
        currentNode = textNode;
    } else {
        const startIndex = match.index || 0;
        const endIndex = startIndex + match[0].length;
        if (startIndex === 0) {
            [currentNode, remainderNode] = textNode.splitText(endIndex);
        } else {
            [leadingNode, currentNode, remainderNode] = textNode.splitText(startIndex, endIndex);
        }
    }
    currentNode.setTextContent(match[2]);
    const transformer = textFormatTransformersIndex.transformersByTag[match[1]];
    if (transformer) {
        for (const format of transformer.format) {
            if (!currentNode.hasFormat(format)) {
                currentNode.toggleFormat(format);
            }
        }
    }

    // Recursively run over inner text if it's not inline code
    if (!currentNode.hasFormat('code')) {
        importTextFormatTransformers(currentNode, textFormatTransformersIndex, textMatchTransformers);
    }

    // Run over leading/remaining text if any
    if (leadingNode) {
        importTextFormatTransformers(leadingNode, textFormatTransformersIndex, textMatchTransformers);
    }
    if (remainderNode) {
        importTextFormatTransformers(remainderNode, textFormatTransformersIndex, textMatchTransformers);
    }
}

const importTextMatchTransformers = (textNode_, textMatchTransformers) => {
    let textNode = textNode_;
    mainLoop: while (textNode) {
        for (const transformer of textMatchTransformers) {
            const match = textNode.getTextContent().match(transformer.importRegExp);
            if (!match) {
                continue;
            }
            const startIndex = match.index || 0;
            const endIndex = startIndex + match[0].length;
            let replaceNode, newTextNode;
            if (startIndex === 0) {
                [replaceNode, textNode] = textNode.splitText(endIndex);
            } else {
                [, replaceNode, newTextNode] = textNode.splitText(startIndex, endIndex);
            }
            if (newTextNode) {
                importTextMatchTransformers(newTextNode, textMatchTransformers);
            }
            transformer.replace(replaceNode, match);
            continue mainLoop;
        }
        break;
    }
}

const importBlocks = (lineText, rootNode, elementTransformers, textFormatTransformersIndex, textMatchTransformers) => {
    const lineTextTrimmed = lineText.trim();
    const textNode = $createTextNode(lineTextTrimmed);
    const elementNode = $createParagraphNode();
    elementNode.append(textNode);
    rootNode.append(elementNode);
    for (const {
        regExp,
        replace
    } of elementTransformers) {
        const match = lineText.match(regExp);
        if (match) {
            textNode.setTextContent(lineText.slice(match[0].length));
            replace(elementNode, [textNode], match, true);
            break;
        }
    }
    importTextFormatTransformers(textNode, textFormatTransformersIndex, textMatchTransformers);

    // If no transformer found and we left with original paragraph node
    // can check if its content can be appended to the previous node
    // if it's a paragraph, quote or list
    if (elementNode.isAttached() && lineTextTrimmed.length > 0) {
        const previousNode = elementNode.getPreviousSibling();
        if ($isParagraphNode(previousNode) || $isQuoteNode(previousNode) || $isListNode(previousNode)) {
            let targetNode: ElementNode | null = previousNode;
            if ($isListNode(previousNode)) {
                const lastDescendant = previousNode.getLastDescendant();
                if (lastDescendant == null) {
                    targetNode = null;
                } else {
                    targetNode = $findMatchingParent(lastDescendant, $isListItemNode);
                }
            }
            if (targetNode != null && targetNode.getTextContentSize() > 0) {
                targetNode.splice(targetNode.getChildrenSize(), 0, [$createLineBreakNode(), ...elementNode.getChildren()]);
                elementNode.remove();
            }
        }
    }
}

/**
 * Imports a code block from markdown content into a Lexical document.
 * This function identifies code blocks within an array of markdown lines starting from a given index.
 * It looks for lines enclosed within code block delimiters (e.g., "```") and creates a `CodeBlockNode`
 * containing the text within these delimiters. The created `CodeBlockNode`, along with its content, is
 * then appended to the specified root node in the Lexical document.
 *
 * The function returns a tuple containing the created `CodeBlockNode` (or `null` if no code block is found)
 * and the index of the line where the closing delimiter of the code block was found. This index helps to skip
 * the lines of the code block in further processing.
 *
 * @param lines - An array of strings, each representing a line of the markdown content.
 * @param startLineIndex - The index of the line from which to start searching for a code block.
 * @param rootNode - The Lexical ElementNode to which the created `CodeBlockNode` should be appended.
 * @returns A tuple where the first element is the created `CodeBlockNode` or `null`
 *                                           if no code block was found, and the second element is the index of the line
 *                                           where the code block ends or the start line index if no code block was found.
 */
const importCodeBlock = (lines: string[], startLineIndex: number, rootNode: ElementNode): [CodeBlockNode | null, number] => {
    const openMatch = lines[startLineIndex].match(CODE_BLOCK_REG_EXP);
    if (openMatch) {
        let endLineIndex = startLineIndex;
        const linesLength = lines.length;
        while (++endLineIndex < linesLength) {
            const closeMatch = lines[endLineIndex].match(CODE_BLOCK_REG_EXP);
            if (closeMatch) {
                const codeBlockNode = $createCodeBlockNode(openMatch[1]);
                const textNode = $createTextNode(lines.slice(startLineIndex + 1, endLineIndex).join('\n'));
                codeBlockNode.append(textNode);
                rootNode.append(codeBlockNode);
                return [codeBlockNode, endLineIndex];
            }
        }
    }
    return [null, startLineIndex];
}

/**
 * Organizes items from a list into an index (a Record) based on a key generated by a callback function.
 * This utility function is generic and can work with any list of items, provided a callback function
 * that extracts a key from each item. Items with the same key will be grouped together in an array.
 *
 * @param list - An array of items to be indexed. Each item is of type T.
 * @param callback - A function that takes an item of type T from the list and returns a key of type K. This key is used to index the items.
 * 
 * @returns An object where each key (of type K) maps to an array of items (of type T) that correspond to that key. This creates a grouped structure where items are indexed by the key derived from them through the callback function.
 */
const indexBy = <T, K extends string | number | symbol>(list: T[], callback: (item: T) => K): Record<K, T[]> => {
    const index: Record<K, T[]> = {} as Record<K, T[]>;
    for (const item of list) {
        const key = callback(item);
        if (index[key]) {
            index[key].push(item);
        } else {
            index[key] = [item];
        }
    }
    return index;
};

const transformersByType = (transformers: Transformer[]) => {
    const byType = indexBy(transformers, t => t.type);
    return {
        element: byType.element || [],
        textFormat: byType['text-format'] || [],
        textMatch: byType['text-match'] || []
    } as {
        element: ElementTransformer[];
        textFormat: TextFormatTransformer[];
        textMatch: TextMatchTransformer[];
    };
}

const isEmptyParagraph = (node) => {
    if (!$isParagraphNode(node)) {
        return false;
    }
    const firstChild = node.getFirstChild();
    return firstChild == null || node.getChildrenSize() === 1 && $isTextNode(firstChild) && MARKDOWN_EMPTY_LINE_REG_EXP.test(firstChild.getTextContent());
}

/**
 * Creates an index of text transformers based on the provided array of text transformers.
 * Each transformer is associated with a specific tag (like '*', '_', etc.) used for text formatting.
 * This function generates regex patterns for matching these tags and their enclosed content.
 * 
 * Due to variations in regex support across different browsers, particularly the support for
 * lookbehind assertions, this function uses different regex patterns for iOS devices (including Safari)
 * and other platforms. Lookbehind assertions allow for matching a pattern only if it's preceded
 * (or not preceded) by another pattern. However, Safari and iOS devices historically had limited
 * support for these assertions, necessitating a simplified regex pattern that does not rely on them.
 * 
 * For non-iOS devices, the function employs more complex regex patterns with negative and positive
 * lookbehind assertions for precise matching, ensuring that tags are not part of escaped sequences or
 * within invalid contexts. For iOS devices and Safari, it falls back to a more straightforward pattern
 * to ensure compatibility, albeit with potentially less precision in matching.
 *
 * @param textTransformers - An array of text transformer objects, each associated with a specific tag for formatting.
 * @returns  An object containing three properties:
 *   - fullMatchRegExpByTag: An object mapping each tag to its full matching regex pattern.
 *   - openTagsRegExp: A regex pattern for matching opening tags.
 *   - transformersByTag: An object mapping each tag to its corresponding transformer.
 */
const createTextFormatTransformersIndex = (textTransformers: TextFormatTransformer[]) => {
    const transformersByTag = {};
    const fullMatchRegExpByTag = {};
    const openTagsRegExp: string[] = [];
    const escapeRegExp = `(?<![\\\\])`;
    const { deviceOS } = getDeviceInfo();
    const isApple = deviceOS === DeviceOS.IOS || deviceOS === DeviceOS.MacOS;
    for (const transformer of textTransformers) {
        const { tag } = transformer;
        transformersByTag[tag] = transformer;
        const tagRegExp = tag.replace(/(\*|\^|\+)/g, '\\$1');
        openTagsRegExp.push(tagRegExp);
        if (isApple) {
            fullMatchRegExpByTag[tag] = new RegExp(`(${tagRegExp})(?![${tagRegExp}\\s])(.*?[^${tagRegExp}\\s])${tagRegExp}(?!${tagRegExp})`);
        } else {
            fullMatchRegExpByTag[tag] = new RegExp(`(?<![\\\\${tagRegExp}])(${tagRegExp})((\\\\${tagRegExp})?.*?[^${tagRegExp}\\s](\\\\${tagRegExp})?)((?<!\\\\)|(?<=\\\\\\\\))(${tagRegExp})(?![\\\\${tagRegExp}])`);
        }
    }
    return {
        // Reg exp to find open tag + content + close tag
        fullMatchRegExpByTag,
        // Reg exp to find opening tags
        openTagsRegExp: new RegExp((isApple ? '' : `${escapeRegExp}`) + '(' + openTagsRegExp.join('|') + ')', 'g'),
        transformersByTag
    };
}

/**
 * Creates a function that imports markdown content into a Lexical document.
 * This higher-order function leverages an array of transformers to parse and convert
 * markdown strings into a structured format compatible with Lexical nodes. The resulting
 * function takes a markdown string and an optional Lexical ElementNode as arguments,
 * inserting the parsed content into the specified node or the root node by default.
 *
 * The conversion process involves splitting the markdown content into lines and processing
 * each line individually, taking into consideration special markdown elements like code blocks.
 * Code blocks are processed first to ensure their content is not further parsed.
 *
 * If there is an active selection, the cursor is moved to the end of the imported content to facilitate
 * further user interactions.
 *
 * @param transformers - An array of Transformer objects that define the rules for
 *                                       converting markdown syntax into Lexical nodes. These transformers
 *                                       cover various markdown elements, such as headers, lists, and text formatting.
 * @returns A function that takes a markdown string and an optional Lexical ElementNode. The function
 *                     processes the markdown string using the provided transformers and inserts the resulting
 *                     nodes into the Lexical document, either under the specified node or at the root level.
 */
const createMarkdownImport = (transformers: Transformer[]): ((markdownString: string, node?: ElementNode) => void) => {
    const byType = transformersByType(transformers);
    const textFormatTransformersIndex = createTextFormatTransformersIndex(byType.textFormat);
    return (markdownString, node) => {
        const lines = markdownString.split('\n');
        const linesLength = lines.length;
        const root = node || $getRoot();
        root.clear();
        for (let i = 0; i < linesLength; i++) {
            const lineText = lines[i];
            // Codeblocks are processed first as anything inside such block
            // is ignored for further processing
            const [codeBlockNode, shiftedIndex] = importCodeBlock(lines, i, root);
            if (codeBlockNode != null) {
                i = shiftedIndex;
                continue;
            }
            importBlocks(lineText, root, byType.element, textFormatTransformersIndex, byType.textMatch);
        }

        if ($getSelection() !== null) {
            root.selectEnd();
        }
    };
}

/**
 * Get next or previous text sibling a text node, including cases 
 * when it's a child of inline element (e.g. link)
 */
const getTextSibling = (node, backward) => {
    let sibling = backward ? node.getPreviousSibling() : node.getNextSibling();
    if (!sibling) {
        const parent = node.getParentOrThrow();
        if (parent.isInline()) {
            sibling = backward ? parent.getPreviousSibling() : parent.getNextSibling();
        }
    }
    while (sibling) {
        if ($isElementNode(sibling)) {
            if (!sibling.isInline()) {
                break;
            }
            const descendant = backward ? sibling.getLastDescendant() : sibling.getFirstDescendant();
            if ($isTextNode(descendant)) {
                return descendant;
            } else {
                sibling = backward ? sibling.getPreviousSibling() : sibling.getNextSibling();
            }
        }
        if ($isTextNode(sibling)) {
            return sibling;
        }
        if (!$isElementNode(sibling)) {
            return null;
        }
    }
    return null;
}

const hasFormat = (node, format) => {
    return $isTextNode(node) && node.hasFormat(format);
}

const exportTextFormat = (node: LexicalNode, textContent: string, textFormatTransformers: TextFormatTransformer[]) => {
    // This function handles the case of a string looking like this: "   foo   "
    // Where it would be invalid markdown to generate: "**   foo   **"
    // We instead want to trim the whitespace out, apply formatting, and then
    // bring the whitespace back. So our returned string looks like this: "   **foo**   "
    const frozenString = textContent.trim();
    let output = frozenString;
    const applied = new Set();
    for (const transformer of textFormatTransformers) {
        const format = transformer.format[0];
        const tag = transformer.tag;
        if (hasFormat(node, format) && !applied.has(format)) {
            // Multiple tags might be used for the same format (*, _)
            applied.add(format);
            // Prevent adding opening tag is already opened by the previous sibling
            const previousNode = getTextSibling(node, true);
            if (!hasFormat(previousNode, format)) {
                output = tag + output;
            }

            // Prevent adding closing tag if next sibling will do it
            const nextNode = getTextSibling(node, false);
            if (!hasFormat(nextNode, format)) {
                output += tag;
            }
        }
    }

    // Replace trimmed version of textContent ensuring surrounding whitespace is not modified
    return textContent.replace(frozenString, output);
}

const exportChildren = (node: ElementNode, textFormatTransformers: TextFormatTransformer[], textMatchTransformers: TextMatchTransformer[]) => {
    const output: string[] = [];
    const children = node.getChildren();
    mainLoop: for (const child of children) {
        for (const transformer of textMatchTransformers) {
            const result = transformer.export(child, parentNode => exportChildren(parentNode, textFormatTransformers, textMatchTransformers), (textNode, textContent) => exportTextFormat(textNode, textContent, textFormatTransformers));
            if (result != null) {
                output.push(result);
                continue mainLoop;
            }
        }
        if ($isLineBreakNode(child)) {
            output.push("\n");
        } else if ($isTextNode(child)) {
            output.push(exportTextFormat(child, child.getTextContent(), textFormatTransformers));
        } else if ($isElementNode(child)) {
            output.push(exportChildren(child, textFormatTransformers, textMatchTransformers));
        } else if ($isDecoratorNode(child)) {
            output.push(child.getTextContent());
        }
    }
    return output.join("");
}

const exportTopLevelElements = (
    node: LexicalNode,
    elementTransformers: ElementTransformer[],
    textFormatTransformers: TextFormatTransformer[],
    textMatchTransformers: TextMatchTransformer[],
): string | null => {
    for (const transformer of elementTransformers) {
        const result = transformer.export(node, _node => exportChildren(_node, textFormatTransformers, textMatchTransformers));
        if (result != null) {
            return result;
        }
    }
    if ($isElementNode(node)) {
        return exportChildren(node, textFormatTransformers, textMatchTransformers);
    } else if ($isDecoratorNode(node)) {
        return node.getTextContent();
    } else {
        return null;
    }
}

/**
 * Creates a function that exports Lexical document content or content within a specified node to markdown format.
 * This higher-order function uses an array of transformers to serialize Lexical nodes into markdown syntax. The
 * resulting function can take an optional Lexical ElementNode as an argument, exporting the content from this node
 * or from the root node if none is specified.
 *
 * The export process involves iterating over the children of the specified or root node, converting each to its
 * markdown representation according to the applicable transformers. Text formatting transformers are filtered to
 * use only those responsible for single formats (e.g., separate transformers for bold and italic, rather than a
 * combined bold-italic transformer) to ensure clarity and simplicity in the markdown output.
 *
 * The function aggregates the markdown strings of all processed nodes, joining them with double newline characters
 * to preserve the document structure and readability in the markdown output.
 *
 * @param transformers - An array of Transformer objects that define how Lexical nodes are serialized into markdown
 *                       syntax. These transformers handle various Lexical elements and text formatting, ensuring a
 *                       comprehensive and accurate representation of the document in markdown format.
 * @returns A function that optionally takes a Lexical ElementNode and returns a string. This string is the markdown
 *          serialization of the content within the specified node or the entire document if no node is specified.
 *          The serialized content is structured to reflect the original document structure, making it suitable for
 *          export and use in markdown-compatible environments.
 */
const createMarkdownExport = (transformers: Transformer[]): ((node?: ElementNode) => string) => {
    const byType = transformersByType(transformers);

    // Export only uses text formats that are responsible for single format
    // e.g. it will filter out *** (bold, italic) and instead use separate ** and *
    const textFormatTransformers = byType.textFormat.filter(transformer => transformer.format.length === 1);
    return node => {
        const output: string[] = [];
        const children = (node || $getRoot()).getChildren();
        for (const child of children) {
            const result = exportTopLevelElements(child, byType.element, textFormatTransformers, byType.textMatch);
            if (result != null) {
                output.push(result);
            }
        }
        return output.join('\n\n');
    };
}

/**
 * Converts a markdown string to Lexical nodes using the specified transformers.
 * This function is a higher-order function that utilizes a markdown import function
 * created by `createMarkdownImport`. It allows for the markdown string to be parsed
 * and transformed into a Lexical-compatible document structure, optionally within a specified
 * parent node.
 *
 * @param markdown - The markdown string to be converted into Lexical nodes.
 * @param transformers - An optional array of Transformer objects that define how
 *                                         markdown syntax is converted to Lexical nodes. Each transformer
 *                                         should handle a specific markdown pattern or syntax.
 * @param node - An optional Lexical ElementNode within which the converted nodes will
 *                               be inserted. If not provided, the nodes will be created at the root level.
 */
export const $convertFromMarkdownString = (markdown: string, transformers: Transformer[], node?: ElementNode) => {
    const importMarkdown = createMarkdownImport(transformers);
    console.log('got importMarkdown');
    return importMarkdown(markdown, node); //TODO might not need a return
}

/**
 * Converts Lexical nodes within a specified parent node or the entire document into a markdown string
 * using the specified transformers. This function is a higher-order function that utilizes a markdown export function
 * created by `createMarkdownExport`. It allows for the Lexical nodes to be serialized into a markdown string format,
 * which can be useful for exporting the document content to markdown-compatible platforms or for storage purposes.
 *
 * @param transformers - An array of Transformer objects that define how Lexical nodes are converted to markdown syntax.
 *                       Each transformer should handle a specific Lexical node type or pattern and serialize it to its
 *                       corresponding markdown representation.
 * @param node - An optional Lexical ElementNode from which the conversion to markdown will start. If not provided,
 *               the conversion process will include all nodes in the document, effectively serializing the entire
 *               document content to markdown. This parameter can be used to selectively export a portion of the document.
 * @returns A string representing the markdown serialization of the Lexical nodes, starting from the specified node or
 *          encompassing the entire document if no node is specified.
 */

export const $convertToMarkdownString = (transformers: Transformer[], node?: ElementNode) => {
    const exportMarkdown = createMarkdownExport(transformers);
    return exportMarkdown(node);
}