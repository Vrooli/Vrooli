/* eslint-disable @typescript-eslint/ban-ts-comment */
//TODO replace this with custom styling in CodePlugin, then remove CodePlugin

import { RangeSelection } from "../selection.js";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, NodeConstructorPayloads, NodeKey, NodeType, SerializedCodeHighlightNode, SerializedCodeNode } from "../types.js";
import { $applyNodeReplacement, $createNode, $isNode, getIndexWithinParent, getNextSibling, getParent, getPreviousSibling, isHTMLElement } from "../utils.js";
import { ElementNode } from "./ElementNode.js";
import { type LexicalNode } from "./LexicalNode.js";
import { type LineBreakNode } from "./LineBreakNode.js";
import { type ParagraphNode } from "./ParagraphNode.js";
import { type TabNode } from "./TabNode.js";
import { TextNode } from "./TextNode.js";

/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 */

function mapToPrismLanguage(
    language: string | null | undefined,
): string | null | undefined {
    //TODO
    // return language !== null && window.Prism.languages.hasOwnProperty(language)
    //     ? language
    //     : undefined;
    return undefined;
}

function hasChildDOMNodeTag(node: Node, tagName: string) {
    for (const child of node.childNodes) {
        if (isHTMLElement(child) && child.tagName === tagName) {
            return true;
        }
        hasChildDOMNodeTag(child, tagName);
    }
    return false;
}

const LANGUAGE_DATA_ATTRIBUTE = "data-highlight-language";

export class CodeNode extends ElementNode {
    static __type: NodeType = "Code";
    __language: string | null | undefined;

    static clone(node: CodeNode): CodeNode {
        const { __language, __key } = node;
        return new CodeNode({ language: __language ?? undefined, key: __key });
    }

    constructor({ language, key }: NodeConstructorPayloads["Code"]) {
        super({ key });
        this.__language = mapToPrismLanguage(language);
    }

    // View
    createDOM(): HTMLElement {
        const element = document.createElement("code");
        element.setAttribute("spellcheck", "false");
        const language = this.getLanguage();
        if (language) {
            element.setAttribute(LANGUAGE_DATA_ATTRIBUTE, language);
        }
        return element;
    }
    updateDOM(
        prevNode: CodeNode,
        dom: HTMLElement,
    ): boolean {
        const language = this.__language;
        const prevLanguage = prevNode.__language;

        if (language) {
            if (language !== prevLanguage) {
                dom.setAttribute(LANGUAGE_DATA_ATTRIBUTE, language);
            }
        } else if (prevLanguage) {
            dom.removeAttribute(LANGUAGE_DATA_ATTRIBUTE);
        }
        return false;
    }

    exportDOM(): DOMExportOutput {
        const element = document.createElement("pre");
        element.setAttribute("spellcheck", "false");
        const language = this.getLanguage();
        if (language) {
            element.setAttribute(LANGUAGE_DATA_ATTRIBUTE, language);
        }
        return { element };
    }

    static importDOM(): DOMConversionMap {
        return {
            // Typically <pre> is used for code blocks, and <code> for inline code styles
            // but if it's a multi line <code> we'll create a block. Pass through to
            // inline format handled by TextNode otherwise.
            code: (node: Node) => {
                const isMultiLine =
                    node.textContent !== null &&
                    (/\r?\n/.test(node.textContent) || hasChildDOMNodeTag(node, "BR"));

                return isMultiLine
                    ? {
                        conversion: convertPreElement,
                        priority: 1,
                    }
                    : null;
            },
            div: (node: Node) => ({
                conversion: convertDivElement,
                priority: 1,
            }),
            pre: (node: Node) => ({
                conversion: convertPreElement,
                priority: 0,
            }),
            table: (node: Node) => {
                const table = node;
                // domNode is a <table> since we matched it by nodeName
                if (isGitHubCodeTable(table as HTMLTableElement)) {
                    return {
                        conversion: convertTableElement,
                        priority: 3,
                    };
                }
                return null;
            },
            td: (node: Node) => {
                // element is a <td> since we matched it by nodeName
                const td = node as HTMLTableCellElement;
                const table: HTMLTableElement | null = td.closest("table");

                if (isGitHubCodeCell(td)) {
                    return {
                        conversion: convertTableCellElement,
                        priority: 3,
                    };
                }
                if (table && isGitHubCodeTable(table)) {
                    // Return a no-op if it's a table cell in a code table, but not a code line.
                    // Otherwise it'll fall back to the T
                    return {
                        conversion: convertCodeNoop,
                        priority: 3,
                    };
                }

                return null;
            },
            tr: (node: Node) => {
                // element is a <tr> since we matched it by nodeName
                const tr = node as HTMLTableCellElement;
                const table: HTMLTableElement | null = tr.closest("table");
                if (table && isGitHubCodeTable(table)) {
                    return {
                        conversion: convertCodeNoop,
                        priority: 3,
                    };
                }
                return null;
            },
        };
    }

    static importJSON(serializedNode: SerializedCodeNode): CodeNode {
        const node = $createCodeNode(serializedNode.language);
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    exportJSON(): SerializedCodeNode {
        return {
            ...super.exportJSON(),
            __type: "Code",
            language: this.getLanguage(),
            version: 1,
        };
    }

    // Mutation
    insertNewAfter(
        selection: RangeSelection,
        restoreSelection = true,
    ): null | ParagraphNode | CodeHighlightNode | TabNode {
        const children = this.getChildren();
        const childrenLength = children.length;

        if (
            childrenLength >= 2 &&
            children[childrenLength - 1].getTextContent() === "\n" &&
            children[childrenLength - 2].getTextContent() === "\n" &&
            selection.isCollapsed() &&
            selection.anchor.key === this.__key &&
            selection.anchor.offset === childrenLength
        ) {
            children[childrenLength - 1].remove();
            children[childrenLength - 2].remove();
            const newElement = $createNode("Paragraph", {});
            this.insertAfter(newElement, restoreSelection);
            return newElement;
        }

        // If the selection is within the codeblock, find all leading tabs and
        // spaces of the current line. Create a new line that has all those
        // tabs and spaces, such that leading indentation is preserved.
        const { anchor, focus } = selection;
        const firstPoint = anchor.isBefore(focus) ? anchor : focus;
        const firstSelectionNode = firstPoint.getNode();
        if ($isNode("Text", firstSelectionNode)) {
            let node = getFirstCodeNodeOfLine(firstSelectionNode);
            const insertNodes: LexicalNode[] = [];
            // eslint-disable-next-line no-constant-condition
            while (true) {
                if ($isNode("Tab", node)) {
                    insertNodes.push($createNode("Tab", {}));
                    node = getNextSibling(node);
                } else if ($isCodeHighlightNode(node)) {
                    let spaces = 0;
                    const text = node.getTextContent();
                    const textSize = node.getTextContentSize();
                    while (spaces < textSize && text[spaces] === " ") {
                        spaces++;
                    }
                    if (spaces !== 0) {
                        insertNodes.push($createCodeHighlightNode(" ".repeat(spaces)));
                    }
                    if (spaces !== textSize) {
                        break;
                    }
                    node = getNextSibling(node);
                } else {
                    break;
                }
            }
            const split = firstSelectionNode.splitText(anchor.offset)[0];
            const x = anchor.offset === 0 ? 0 : 1;
            const index = getIndexWithinParent(split) + x;
            const codeNode = getParent(firstSelectionNode, { throwIfNull: true });
            const nodesToInsert = [$createNode("LineBreak", {}), ...insertNodes];
            codeNode.splice(index, 0, nodesToInsert);
            const last = insertNodes[insertNodes.length - 1];
            if (last) {
                // @ts-ignore TODO
                last.select();
            } else if (anchor.offset === 0) {
                split.selectPrevious();
            } else {
                getNextSibling(split)!.selectNext(0, 0);
            }
        }
        if ($isNode("Code", firstSelectionNode)) {
            const { offset } = selection.anchor;
            firstSelectionNode.splice(offset, 0, [$createNode("LineBreak", {})]);
            firstSelectionNode.select(offset + 1, offset + 1);
        }

        return null;
    }

    canIndent(): false {
        return false;
    }

    collapseAtStart(): boolean {
        const paragraph = $createNode("Paragraph", {});
        const children = this.getChildren();
        children.forEach((child) => paragraph.append(child));
        this.replace(paragraph);
        return true;
    }

    setLanguage(language: string): void {
        const writable = this.getWritable();
        writable.__language = mapToPrismLanguage(language);
    }

    getLanguage(): string | null | undefined {
        return this.__language;
    }
}

export class CodeHighlightNode extends TextNode {
    static type: NodeType = "Code";
    __highlightType: string | null | undefined;

    constructor(
        text: string,
        highlightType?: string | null | undefined,
        key?: NodeKey,
    ) {
        super({ text, key });
        this.__highlightType = highlightType;
    }

    static clone(node: CodeHighlightNode): CodeHighlightNode {
        return new CodeHighlightNode(
            node.__text,
            node.__highlightType || undefined,
            node.__key,
        );
    }

    getHighlightType(): string | null | undefined {
        return this.getLatest().__highlightType;
    }

    createDOM(): HTMLElement {
        const element = super.createDOM();
        return element;
    }

    updateDOM(
        prevNode: CodeHighlightNode,
        dom: HTMLElement,
    ): boolean {
        const update = super.updateDOM(prevNode, dom);
        return update;
    }

    static importJSON(
        serializedNode: SerializedCodeHighlightNode,
    ): CodeHighlightNode {
        const node = $createCodeHighlightNode(
            serializedNode.text,
            serializedNode.highlightType,
        );
        node.setFormat(serializedNode.format);
        node.setDetail(serializedNode.detail);
        node.setMode(serializedNode.mode);
        node.setStyle(serializedNode.style);
        return node;
    }

    exportJSON(): SerializedCodeHighlightNode {
        return {
            ...super.exportJSON(),
            __type: "Code",
            highlightType: this.getLatest().__highlightType,
            version: 1,
        };
    }

    // Prevent formatting (bold, underline, etc)
    setFormat(format: number): this {
        return this;
    }

    isParentRequired(): true {
        return true;
    }

    createParentElementNode(): ElementNode {
        return $createCodeNode();
    }
}

export function $createCodeNode(
    language?: string | null | undefined,
): CodeNode {
    return $applyNodeReplacement(new CodeNode({ language: language ?? undefined }));
}

function convertPreElement(domNode: Node): DOMConversionOutput {
    let language;
    if (isHTMLElement(domNode)) {
        language = domNode.getAttribute(LANGUAGE_DATA_ATTRIBUTE);
    }
    return { node: $createCodeNode(language) };
}

function convertDivElement(domNode: Node): DOMConversionOutput {
    // domNode is a <div> since we matched it by nodeName
    const div = domNode as HTMLDivElement;
    const isCode = isCodeElement(div);
    if (!isCode && !isCodeChildElement(div)) {
        return {
            node: null,
        };
    }
    return {
        after: (childLexicalNodes) => {
            const domParent = domNode.parentNode;
            if (domParent !== null && domNode !== domParent.lastChild) {
                childLexicalNodes.push($createNode("LineBreak", {}));
            }
            return childLexicalNodes;
        },
        node: isCode ? $createCodeNode() : null,
    };
}

function convertTableElement(): DOMConversionOutput {
    return { node: $createCodeNode() };
}

function convertCodeNoop(): DOMConversionOutput {
    return { node: null };
}

function convertTableCellElement(domNode: Node): DOMConversionOutput {
    // domNode is a <td> since we matched it by nodeName
    const cell = domNode as HTMLTableCellElement;

    return {
        after: (childLexicalNodes) => {
            if (cell.parentNode && cell.parentNode.nextSibling) {
                // Append newline between code lines
                childLexicalNodes.push($createNode("LineBreak", {}));
            }
            return childLexicalNodes;
        },
        node: null,
    };
}

function isCodeElement(div: HTMLElement): boolean {
    return div.style.fontFamily.match("monospace") !== null;
}

function isCodeChildElement(node: HTMLElement): boolean {
    let parent = node.parentElement;
    while (parent !== null) {
        if (isCodeElement(parent)) {
            return true;
        }
        parent = parent.parentElement;
    }
    return false;
}

function isGitHubCodeCell(
    cell: HTMLTableCellElement,
): cell is HTMLTableCellElement {
    return cell.classList.contains("js-file-line");
}

function isGitHubCodeTable(table: HTMLTableElement): table is HTMLTableElement {
    return table.classList.contains("js-file-line-container");
}

export function $createCodeHighlightNode(
    text: string,
    highlightType?: string | null | undefined,
): CodeHighlightNode {
    return $applyNodeReplacement(new CodeHighlightNode(text, highlightType));
}

export function $isCodeHighlightNode(
    node: LexicalNode | CodeHighlightNode | null | undefined,
): node is CodeHighlightNode {
    return node instanceof CodeHighlightNode;
}

export function getFirstCodeNodeOfLine(
    anchor: CodeHighlightNode | TabNode | LineBreakNode,
): null | CodeHighlightNode | TabNode | LineBreakNode {
    let previousNode = anchor;
    let node: null | LexicalNode = anchor;
    while ($isCodeHighlightNode(node) || $isNode("Tab", node)) {
        previousNode = node;
        node = getPreviousSibling(node);
    }
    return previousNode;
}

export function getLastCodeNodeOfLine(
    anchor: CodeHighlightNode | TabNode | LineBreakNode,
): CodeHighlightNode | TabNode | LineBreakNode {
    let nextNode = anchor;
    let node: null | LexicalNode = anchor;
    while ($isCodeHighlightNode(node) || $isNode("Tab", node)) {
        nextNode = node;
        node = getNextSibling(node);
    }
    return nextNode;
}
