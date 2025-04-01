
// We have copied the code here for customization purposes, such as replacing the default code block component
import { LexicalEditor } from "./editor.js";
import { type ElementNode } from "./nodes/ElementNode.js";
import { type LexicalNode } from "./nodes/LexicalNode.js";
import { type TextNode } from "./nodes/TextNode.js";
import { type CodeBlockNode } from "./plugins/CodePlugin.js";
import { $createRangeSelection } from "./selection.js";
import { hasFormat } from "./transformers/textFormatTransformers.js";
import { ElementTransformer, LexicalTransformer, TextFormatTransformer, TextMatchTransformer } from "./types.js";
import { $createNode, $findMatchingParent, $getRoot, $getSelection, $isNode, $isRangeSelection, $isRootOrShadowRoot, $setSelection, getNextSibling, getNextSiblings, getParent, getPreviousSibling, isAttachedToRoot } from "./utils.js";

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
 * Escapes special characters in a string to prevent them from being interpreted as regex symbols.
 */
function escapeSpecialCharacters(text: string) { return text.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&"); }

/**
 * Finds first "<tag>content<tag>" match that is not nested into another tag
 * @param textContent - text content to search for tag match
 * @param textTransformersIndex - Information about available text transformers
 * @returns Tuple of start tag and content match or null if no match found
 */
function findOutermostMatch(
    textContent: string,
    textTransformersIndex: TextMatchTransformersIndex,
): [string, RegExpMatchArray] | null {
    const openTagsMatch = textContent.match(textTransformersIndex.openTagsRegExp);
    if (!openTagsMatch) {
        return null;
    }
    for (const match of openTagsMatch) {
        // Open tags reg exp might capture leading space so removing it
        // before using match to find transformer
        const tag = match.trim();
        const fullMatchRegExp = textTransformersIndex.fullMatchRegExpByTag[tag];
        if (!fullMatchRegExp) {
            continue;
        }
        const fullMatch = textContent.match(fullMatchRegExp);
        console.log("here is fullMatch", match, fullMatch, fullMatchRegExp, textContent);
        if (!fullMatch) {
            continue;
        }
        const transformer = textTransformersIndex.transformersByTag[tag];
        if (fullMatch !== null && transformer !== null) {
            return [tag, fullMatch];
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
function importTextFormatTransformers(
    textNode: TextNode,
    textFormatTransformersIndex: TextMatchTransformersIndex,
    textMatchTransformers: TextMatchTransformer[],
) {
    let textContent = textNode.getTextContent();
    let outerMatch: [string, RegExpMatchArray] | null;

    // Use a stack to handle nodes instead of recursion
    const stack = [textNode];

    while (stack.length) {
        let currentNode = stack.pop();
        console.log("in importTextFormatTransformers stack loop", currentNode, currentNode?.getTextContent());
        if (!currentNode) {
            continue;
        }
        textContent = currentNode.getTextContent();
        console.log("im importtextformatransformers. got textcontent", currentNode, textContent);
        outerMatch = findOutermostMatch(textContent, textFormatTransformersIndex);

        if (!outerMatch) {
            importTextMatchTransformers(currentNode, textMatchTransformers);
            continue;
        }
        const [startTag, match] = outerMatch;
        console.log("textContent and match", textContent, match, startTag);

        const [startIndex, endIndex] = [match.index || 0, (match.index || 0) + match[0].length];
        let remainderNode: TextNode, leadingNode: TextNode;

        if (match[0] === textContent) {
            currentNode.setTextContent(match[2]); // Update the whole node if it matches completely
        } else {
            if (startIndex > 0) {
                [leadingNode, currentNode] = currentNode.splitText(startIndex);
                stack.push(leadingNode);
            }
            if (endIndex < textContent.length) {
                [currentNode, remainderNode] = currentNode.splitText(endIndex - startIndex);
                stack.push(remainderNode);
            }
        }

        // Apply formats
        const transformer = textFormatTransformersIndex.transformersByTag[startTag];
        console.log("apply formats transformer", transformer, startTag, currentNode);
        if (transformer) {
            // Handle built-in format
            if (!hasFormat(currentNode, transformer.format)) {
                console.log("applying format", transformer);
                currentNode.toggleFormat(transformer.format);
                currentNode.setTextContent(match[1]); // Remove tags from the content
            }
            // Handle custom formats TODO
        }

        // Skip code blocks
        if (!hasFormat(currentNode, "CODE_BLOCK") && !hasFormat(currentNode, "CODE_INLINE")) {
            stack.push(currentNode);
        }
    }
}

function importTextMatchTransformers(textNode_, textMatchTransformers) {
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

/**
 * Assembles the elements of a lexical document for a specific line of markdown content.
 * @param lineText The markdown content of a single line.
 * @param rootNode The root node of the lexical document.
 * @param elementTransformers An array of element transformers, which are typically used 
 * for block-level elements like headers, lists, and quotes.
 * @param textFormatTransformersIndex An index of text format transformers, which are used 
 * for inline text formatting like bold, italic, and code.
 * @param textMatchTransformers An array of text match transformers, which are used for
 * matching and transforming specific text patterns in the markdown content.
 */
function importBlocks(
    lineText: string,
    rootNode: ElementNode,
    elementTransformers: ElementTransformer[],
    textFormatTransformersIndex: TextMatchTransformersIndex,
    textMatchTransformers: TextMatchTransformer[],
) {
    console.time("importBlocks beginning");
    const lineTextTrimmed = lineText.trim();
    console.log("importBlocks beginning", lineText, lineTextTrimmed);
    const textNode = $createNode("Text", { text: lineTextTrimmed });
    console.log("created text node", textNode, lineTextTrimmed);
    const elementNode = $createNode("Paragraph", {});
    elementNode.append(textNode);
    rootNode.append(elementNode);
    for (const {
        regExp,
        replace,
    } of elementTransformers) {
        const match = lineText.match(regExp);
        if (match) {
            textNode.setTextContent(lineText.slice(match[0].length));
            replace(elementNode, [textNode], match, true);
            break;
        }
    }
    console.timeEnd("importBlocks beginning");
    console.time("importTextFormatTransformers");
    importTextFormatTransformers(textNode, textFormatTransformersIndex, textMatchTransformers);
    console.timeEnd("importTextFormatTransformers");

    // If no transformer found and we left with original paragraph node
    // can check if its content can be appended to the previous node
    // if it's a paragraph, quote or list
    console.time("importBlocks end");
    if (isAttachedToRoot(elementNode) && lineTextTrimmed.length > 0) {
        const previousNode = getPreviousSibling(elementNode);
        if ($isNode("Paragraph", previousNode) || $isNode("Quote", previousNode) || $isNode("List", previousNode)) {
            let targetNode: ElementNode | null = previousNode;
            if ($isNode("List", previousNode)) {
                const lastDescendant = previousNode.getLastDescendant();
                if (!lastDescendant) {
                    targetNode = null;
                } else {
                    targetNode = $findMatchingParent(lastDescendant, (node) => $isNode("ListItem", node)) as ElementNode;
                }
            }
            if (targetNode !== null && targetNode.getTextContentSize() > 0) {
                targetNode.splice(targetNode.getChildrenSize(), 0, [$createNode("LineBreak", {}), ...elementNode.getChildren()]);
                elementNode.remove();
            }
        }
    }
    console.log("end of importBlocks", rootNode.getTextContent());
    console.timeEnd("importBlocks end");
}

/**
 * Imports a code block from markdown content into a Lexical document.
 * 
 * Identifies code blocks within an array of markdown lines starting from a given index.
 * It looks for lines enclosed within code block delimiters (e.g., "```") and creates a `CodeBlockNode`
 * containing the text within these delimiters. The created `CodeBlockNode`, along with its content, is
 * then appended to the specified root node in the Lexical document.
 *
 * @param lines - An array of strings, each representing a line of the markdown content.
 * @param startLineIndex - The index of the line from which to start searching for a code block.
 * @param rootNode - The Lexical ElementNode to which the created `CodeBlockNode` should be appended.
 * @returns A tuple where the first element is the created `CodeBlockNode` or `null`
 *                                           if no code block was found, and the second element is the index of the line
 *                                           where the code block ends or the start line index if no code block was found.
 */
function importCodeBlock(
    lines: string[],
    startLineIndex: number,
    rootNode: ElementNode,
): [CodeBlockNode | null, number] {
    const openMatch = lines[startLineIndex].match(CODE_BLOCK_REG_EXP);
    if (openMatch) {
        let endLineIndex = startLineIndex;
        const linesLength = lines.length;
        while (++endLineIndex < linesLength) {
            const closeMatch = lines[endLineIndex].match(CODE_BLOCK_REG_EXP);
            if (closeMatch) {
                const codeBlockNode = $createNode("Code", { language: openMatch[1] });
                const textNode = $createNode("Text", { text: lines.slice(startLineIndex + 1, endLineIndex).join("\n") });
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
function indexBy<T, K extends string | number | symbol>(list: T[], callback: (item: T) => K): Record<K, T[]> {
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
}

function transformersByType(transformers: LexicalTransformer[]) {
    const byType = indexBy(transformers, t => t.type);
    return {
        element: byType.element || [],
        textFormat: byType["text-format"] || [],
        textMatch: byType["text-match"] || [],
    } as {
        element: ElementTransformer[];
        textFormat: TextFormatTransformer[];
        textMatch: TextMatchTransformer[];
    };
}

type TextMatchTransformersIndex = {
    /** Regex to match each text format tag (e.g. **, ~~) */
    fullMatchRegExpByTag: Record<string, RegExp>;
    /** Regex to match non-escaped text format tags */
    openTagsRegExp: RegExp;
    /** LexicalTransformer objects for each taxt format tag */
    transformersByTag: Record<string, TextFormatTransformer>;
};

/**
 * Creates an index of text transformers based on the provided array of text transformers.
 * 
 * Text transformers are one of the three types of transformers - the others being "element" and "textMatch".
 * 
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
function createTextFormatTransformersIndex(textTransformers: TextFormatTransformer[]): TextMatchTransformersIndex {
    console.log("creating text format transformers index - which is currently missing custom transformers", textTransformers);
    const transformersByTag = {};
    const fullMatchRegExpByTag = {};
    // Stores escaped start tags to build a regex later for matching opening tags
    const openTags: string[] = [];
    // Loop through each transformer to build regex patterns
    for (const transformer of textTransformers) {
        const [startTag, endTag] = transformer.tags;

        // Use tags to build regex patterns
        const escapedStartTag = escapeSpecialCharacters(startTag);
        const escapedEndTag = escapeSpecialCharacters(endTag);
        const fullRegex = new RegExp(`${escapedStartTag}(.*?)${escapedEndTag}`, "s");

        transformersByTag[startTag] = transformer;
        openTags.push(escapedStartTag);

        fullMatchRegExpByTag[startTag] = fullRegex;
    }
    // Combine all open tags into a single regex pattern
    const openTagsRegExp = new RegExp(`(${openTags.join("|")})`, "g");
    return {
        fullMatchRegExpByTag,
        openTagsRegExp,
        transformersByTag,
    };
}

const MAX_LINES_TO_PROCESS = 2048;

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
 * @param transformers - An array of LexicalTransformer objects that define the rules for
 *                                       converting markdown syntax into Lexical nodes. These transformers
 *                                       cover various markdown elements, such as headers, lists, and text formatting.
 * @returns A function that takes a markdown string and an optional Lexical ElementNode. The function
 *                     processes the markdown string using the provided transformers and inserts the resulting
 *                     nodes into the Lexical document, either under the specified node or at the root level.
 */
function createMarkdownImport(
    transformers: LexicalTransformer[],
): ((markdownString: string, node?: ElementNode) => void) {
    console.log("creating markdown input - transformers", transformers);
    const byType = transformersByType(transformers);
    console.log("creating markdown import - transformers by type", byType);
    const textFormatTransformersIndex = createTextFormatTransformersIndex(byType.textFormat);
    return (markdownString, node) => {
        const lines = markdownString.split("\n", MAX_LINES_TO_PROCESS);
        const linesLength = lines.length;
        const root = node || $getRoot();
        root.clear();
        for (let i = 0; i < linesLength; i++) {
            console.log("processing import line", i, lines[i]);
            const lineText = lines[i];
            // Codeblocks are processed first as anything inside such block
            // is ignored for further processing
            console.time("importCodeBlock");
            const [codeBlockNode, shiftedIndex] = importCodeBlock(lines, i, root);
            console.timeEnd("importCodeBlock");
            if (codeBlockNode !== null) {
                console.log("skipping because code block found");
                i = shiftedIndex;
                continue;
            }
            console.time("importBlocks");
            importBlocks(lineText, root, byType.element, textFormatTransformersIndex, byType.textMatch);
            console.timeEnd("importBlocks");
        }

        if ($getSelection() !== null) {
            root.selectEnd();
        }
        console.log("end of createMarkdownImport", root.getTextContent());
        console.log("createMarkdownImport end");
    };
}

/**
 * Get next or previous text sibling a text node, including cases 
 * when it's a child of inline element (e.g. link)
 */
function getTextSibling(node: LexicalNode, backward: boolean) {
    let sibling = backward ? getPreviousSibling(node) : getNextSibling(node);
    if (!sibling) {
        const parent = getParent(node, { throwIfNull: true });
        if (parent.isInline()) {
            sibling = backward ? getPreviousSibling(parent) : getNextSibling(parent);
        }
    }
    while (sibling) {
        if ($isNode("Element", sibling)) {
            if (!sibling.isInline()) {
                break;
            }
            const descendant = backward ? sibling.getLastDescendant() : sibling.getFirstDescendant();
            if ($isNode("Text", descendant)) {
                return descendant;
            } else {
                sibling = backward ? getPreviousSibling(sibling) : getNextSibling(sibling);
            }
        }
        if ($isNode("Text", sibling)) {
            return sibling;
        }
        if (!$isNode("Element", sibling)) {
            return null;
        }
    }
    return null;
}

function exportTextFormat(node: LexicalNode, textContent: string, textFormatTransformers: TextFormatTransformer[]) {
    // This function handles the case of a string looking like this: "   foo   "
    // Where it would be invalid markdown to generate: "**   foo   **"
    // We instead want to trim the whitespace out, apply formatting, and then
    // bring the whitespace back. So our returned string looks like this: "   **foo**   "
    const frozenString = textContent.trim();
    let output = frozenString;
    const applied = new Set();
    for (const transformer of textFormatTransformers) {
        const format = transformer.format;
        const [startTag, endTag] = transformer.tags;
        if (hasFormat(node, format) && !applied.has(format)) {
            // Multiple tags might be used for the same format (*, _)
            applied.add(format);
            // Prevent adding opening tag is already opened by the previous sibling
            const previousNode = getTextSibling(node, true);
            if (!hasFormat(previousNode, format)) {
                output = startTag + output;
            }

            // Prevent adding closing tag if next sibling will do it
            const nextNode = getTextSibling(node, false);
            if (!hasFormat(nextNode, format)) {
                output += endTag;
            }
        }
    }

    // Replace trimmed version of textContent ensuring surrounding whitespace is not modified
    return textContent.replace(frozenString, output);
}

/**
 * Converts a markdown string to Lexical nodes using the specified transformers.
 * This function is a higher-order function that utilizes a markdown import function
 * created by `createMarkdownImport`. It allows for the markdown string to be parsed
 * and transformed into a Lexical-compatible document structure, optionally within a specified
 * parent node.
 *
 * @param markdown - The markdown string to be converted into Lexical nodes.
 * @param transformers - An optional array of LexicalTransformer objects that define how
 *                                         markdown syntax is converted to Lexical nodes. Each transformer
 *                                         should handle a specific markdown pattern or syntax.
 * @param node - An optional Lexical ElementNode within which the converted nodes will
 *                               be inserted. If not provided, the nodes will be created at the root level.
 */
export function $convertFromMarkdownString(markdown: string, transformers: LexicalTransformer[], node?: ElementNode) {
    if (typeof markdown !== "string") {
        console.error("$convertFromMarkdownString: markdown is not a string", markdown);
        return;
    }
    console.log("in $convertFromMarkdownString", markdown);
    const importMarkdown = createMarkdownImport(transformers);
    importMarkdown(markdown, node);
}

/**
 * Checks if a substring of `stringA` starting from index `aStart` is equal to a substring of `stringB` starting from index `bStart`.
 * 
 * @param stringA - The first string.
 * @param aStart - The starting index of the substring in `stringA`.
 * @param stringB - The second string.
 * @param bStart - The starting index of the substring in `stringB`.
 * @param length - The length of the substrings to compare.
 * @returns `true` if the substrings are equal, `false` otherwise.
 */
function isEqualSubString(
    stringA: string,
    aStart: number,
    stringB: string,
    bStart: number,
    length: number,
): boolean {
    for (let i = 0; i < length; i++) {
        if (stringA[aStart + i] !== stringB[bStart + i]) {
            return false;
        }
    }

    return true;
}

function getOpenTagStartIndex(
    string: string,
    maxIndex: number,
    tag: string,
): number {
    const tagLength = tag.length;

    for (let i = maxIndex; i >= tagLength; i--) {
        const startIndex = i - tagLength;

        if (
            isEqualSubString(string, startIndex, tag, 0, tagLength) && // Space after opening tag cancels transformation
            string[startIndex + tagLength] !== " "
        ) {
            return startIndex;
        }
    }

    return -1;
}

function runElementTransformers(
    parentNode: ElementNode,
    anchorNode: TextNode,
    anchorOffset: number,
    elementTransformers: ReadonlyArray<ElementTransformer>,
): boolean {
    const grandParentNode = getParent(parentNode);

    if (
        !$isRootOrShadowRoot(grandParentNode) ||
        parentNode.getFirstChild() !== anchorNode
    ) {
        return false;
    }

    const textContent = anchorNode.getTextContent();

    // Checking for anchorOffset position to prevent any checks for cases when caret is too far
    // from a line start to be a part of block-level markdown trigger.
    //
    // TODO:
    // Can have a quick check if caret is close enough to the beginning of the string (e.g. offset less than 10-20)
    // since otherwise it won't be a markdown shortcut, but tables are exception
    if (textContent[anchorOffset - 1] !== " ") {
        return false;
    }

    for (const { regExp, replace } of elementTransformers) {
        const match = textContent.match(regExp);

        if (match && match[0].length === anchorOffset) {
            const nextSiblings = getNextSiblings(anchorNode);
            const [leadingNode, remainderNode] = anchorNode.splitText(anchorOffset);
            leadingNode.remove();
            const siblings = remainderNode
                ? [remainderNode, ...nextSiblings]
                : nextSiblings;
            replace(parentNode, siblings, match, false);
            return true;
        }
    }

    return false;
}

function runTextMatchTransformers(
    anchorNode: TextNode,
    anchorOffset: number,
    transformersByTrigger: Readonly<Record<string, TextMatchTransformer[]>>,
): boolean {
    let textContent = anchorNode.getTextContent();
    const lastChar = textContent[anchorOffset - 1];
    const transformers = transformersByTrigger[lastChar];

    if (!transformers) {
        return false;
    }

    // If typing in the middle of content, remove the tail to do
    // reg exp match up to a string end (caret position)
    if (anchorOffset < textContent.length) {
        textContent = textContent.slice(0, anchorOffset);
    }

    for (const transformer of transformers) {
        const match = textContent.match(transformer.regExp);

        if (!match) {
            continue;
        }

        const startIndex = match.index || 0;
        const endIndex = startIndex + match[0].length;
        let replaceNode;

        if (startIndex === 0) {
            [replaceNode] = anchorNode.splitText(endIndex);
        } else {
            [, replaceNode] = anchorNode.splitText(startIndex, endIndex);
        }

        replaceNode.selectNext(0, 0);
        transformer.replace(replaceNode, match);
        return true;
    }

    return false;
}

function runTextFormatTransformers(
    anchorNode: TextNode,
    anchorOffset: number,
    textFormatTransformers: Readonly<
        Record<string, ReadonlyArray<TextFormatTransformer>>
    >,
): boolean {
    const textContent = anchorNode.getTextContent();
    const closeTagEndIndex = anchorOffset - 1;
    const closeChar = textContent[closeTagEndIndex];
    // Quick check if we're possibly at the end of inline markdown style
    const matchers = textFormatTransformers[closeChar];

    if (!matchers) {
        return false;
    }

    for (const matcher of matchers) {
        const [startTag, endTag] = matcher.tags;
        const startTagLength = startTag.length;
        const endTagLength = endTag.length;
        const endTagStartIndex = closeTagEndIndex - endTagLength + 1;

        // If end tag is not single char check if rest of it matches with text content
        if (endTagLength > 1) {
            if (
                !isEqualSubString(textContent, endTagStartIndex, endTag, 0, endTagLength)
            ) {
                continue;
            }
        }

        // Space before closing tag cancels inline markdown
        if (textContent[endTagStartIndex - 1] === " ") {
            continue;
        }

        const closeNode = anchorNode;
        let openNode = closeNode;
        let openTagStartIndex = getOpenTagStartIndex(
            textContent,
            endTagStartIndex,
            startTag,
        );

        // Go through text node siblings and search for opening tag
        // if haven't found it within the same text node as closing tag
        let sibling: TextNode | null = openNode;

        while (
            openTagStartIndex < 0 &&
            (sibling = getPreviousSibling<TextNode>(sibling))
        ) {
            if ($isNode("LineBreak", sibling)) {
                break;
            }

            if ($isNode("Text", sibling)) {
                const siblingTextContent = sibling.getTextContent();
                openNode = sibling;
                openTagStartIndex = getOpenTagStartIndex(
                    siblingTextContent,
                    siblingTextContent.length,
                    startTag,
                );
            }
        }

        // Opening tag is not found
        if (openTagStartIndex < 0) {
            continue;
        }

        // No content between opening and closing tag
        if (
            openNode === closeNode &&
            openTagStartIndex + startTagLength === endTagStartIndex
        ) {
            continue;
        }

        // Checking longer tags for repeating chars (e.g. *** vs **)
        const prevOpenNodeText = openNode.getTextContent();

        if (
            openTagStartIndex > 0 &&
            prevOpenNodeText[openTagStartIndex - 1] === closeChar
        ) {
            continue;
        }

        // Clean text from opening and closing tags (starting from closing tag
        // to prevent any offset shifts if we start from opening one)
        const prevCloseNodeText = closeNode.getTextContent();
        const closeNodeText =
            prevCloseNodeText.slice(0, endTagStartIndex) +
            prevCloseNodeText.slice(closeTagEndIndex + 1);
        closeNode.setTextContent(closeNodeText);
        const openNodeText =
            openNode === closeNode ? closeNodeText : prevOpenNodeText;
        openNode.setTextContent(
            openNodeText.slice(0, openTagStartIndex) +
            openNodeText.slice(openTagStartIndex + startTagLength),
        );
        const selection = $getSelection();
        const nextSelection = $createRangeSelection();
        $setSelection(nextSelection);
        // Adjust offset based on deleted chars
        const newOffset =
            closeTagEndIndex - endTagLength * (openNode === closeNode ? 2 : 1) + 1;
        nextSelection.anchor.set(openNode.__key, openTagStartIndex, "text");
        nextSelection.focus.set(closeNode.__key, newOffset, "text");

        // Apply formatting to selected text
        if (!nextSelection.hasFormat(matcher.format)) {
            nextSelection.formatText(matcher.format);
        }

        // Collapse selection up to the focus point
        nextSelection.anchor.set(
            nextSelection.focus.key,
            nextSelection.focus.offset,
            nextSelection.focus.type,
        );

        // Remove formatting from collapsed selection
        if (nextSelection.hasFormat(matcher.format)) {
            nextSelection.toggleFormat(matcher.format);
        }

        if ($isRangeSelection(selection)) {
            nextSelection.format = selection.format;
        }

        return true;
    }

    return false;
}

export function registerMarkdownShortcuts(
    editor: LexicalEditor,
    transformers: LexicalTransformer[],
): (() => void) {
    const byType = transformersByType(transformers);
    const textFormatTransformersIndex = indexBy(
        byType.textFormat,
        ({ tags }) => tags[0],
    );
    const textMatchTransformersIndex = indexBy(
        byType.textMatch,
        ({ trigger }) => trigger,
    );

    function transform(
        parentNode: ElementNode,
        anchorNode: TextNode,
        anchorOffset: number,
    ) {
        if (
            runElementTransformers(
                parentNode,
                anchorNode,
                anchorOffset,
                byType.element,
            )
        ) {
            return;
        }

        if (
            runTextMatchTransformers(
                anchorNode,
                anchorOffset,
                textMatchTransformersIndex,
            )
        ) {
            return;
        }

        runTextFormatTransformers(
            anchorNode,
            anchorOffset,
            textFormatTransformersIndex,
        );
    }

    return editor.registerListener(
        "update",
        ({ tags, dirtyLeaves, editorState, prevEditorState }) => {
            // Ignore updates from collaboration and undo/redo (as changes already calculated)
            if (tags.has("collaboration") || tags.has("historic")) {
                return;
            }

            // If editor is still composing (i.e. backticks) we must wait before the user confirms the key
            if (editor.isComposing()) {
                return;
            }

            const selection = editorState.read($getSelection);
            const prevSelection = prevEditorState.read($getSelection);

            if (
                !$isRangeSelection(prevSelection) ||
                !$isRangeSelection(selection) ||
                !selection.isCollapsed()
            ) {
                return;
            }

            const anchorKey = selection.anchor.key;
            const anchorOffset = selection.anchor.offset;

            const anchorNode = editorState._nodeMap.get(anchorKey);

            if (
                !$isNode("Text", anchorNode) ||
                !dirtyLeaves.has(anchorKey) ||
                (anchorOffset !== 1 && anchorOffset > prevSelection.anchor.offset + 1)
            ) {
                return;
            }

            editor.update(() => {
                // Markdown is not available inside code
                if (hasFormat(anchorNode, "CODE_BLOCK")) {
                    return;
                }

                const parentNode = getParent(anchorNode);

                if (!parentNode || $isNode("Code", parentNode)) {
                    return;
                }

                transform(parentNode, anchorNode, selection.anchor.offset);
            });
        },
    );
}
