/* eslint-disable @typescript-eslint/ban-ts-comment */
//TODO replace this with custom styling in CodePlugin, then remove CodePlugin

import { LexicalEditor } from "../editor";
import { RangeSelection } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, EditorThemeClasses, NodeKey, SerializedCodeHighlightNode, SerializedCodeNode } from "../types";
import { $applyNodeReplacement, $isTextNode, addClassNamesToElement, isHTMLElement, removeClassNamesFromElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { LexicalNode } from "./LexicalNode";
import { $createLineBreakNode, LineBreakNode } from "./LineBreakNode";
import { $createParagraphNode, ParagraphNode } from "./ParagraphNode";
import { $createTabNode, $isTabNode, TabNode } from "./TabNode";
import { TextNode } from "./TextNode";

/**
 * Copyright (c) Meta Platforms, Inc. and affiliates.
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 *
 */

const mapToPrismLanguage = (
    language: string | null | undefined,
): string | null | undefined => {
    // return language != null && window.Prism.languages.hasOwnProperty(language)
    //     ? language
    //     : undefined;
    return undefined;
};

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
    /** @internal */
    __language: string | null | undefined;

    static getType(): string {
        return "code";
    }

    static clone(node: CodeNode): CodeNode {
        return new CodeNode(node.__language, node.__key);
    }

    constructor(language?: string | null | undefined, key?: NodeKey) {
        super(key);
        this.__language = mapToPrismLanguage(language);
    }

    // View
    createDOM(config: EditorConfig): HTMLElement {
        const element = document.createElement("code");
        addClassNamesToElement(element, config.theme.code);
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
        config: EditorConfig,
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

    exportDOM(editor: LexicalEditor): DOMExportOutput {
        const element = document.createElement("pre");
        addClassNamesToElement(element, editor._config.theme.code);
        element.setAttribute("spellcheck", "false");
        const language = this.getLanguage();
        if (language) {
            element.setAttribute(LANGUAGE_DATA_ATTRIBUTE, language);
        }
        return { element };
    }

    static importDOM(): DOMConversionMap | null {
        return {
            // Typically <pre> is used for code blocks, and <code> for inline code styles
            // but if it's a multi line <code> we'll create a block. Pass through to
            // inline format handled by TextNode otherwise.
            code: (node: Node) => {
                const isMultiLine =
                    node.textContent != null &&
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
            language: this.getLanguage(),
            type: "code",
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
            const newElement = $createParagraphNode();
            this.insertAfter(newElement, restoreSelection);
            return newElement;
        }

        // If the selection is within the codeblock, find all leading tabs and
        // spaces of the current line. Create a new line that has all those
        // tabs and spaces, such that leading indentation is preserved.
        const { anchor, focus } = selection;
        const firstPoint = anchor.isBefore(focus) ? anchor : focus;
        const firstSelectionNode = firstPoint.getNode();
        if ($isTextNode(firstSelectionNode)) {
            let node = getFirstCodeNodeOfLine(firstSelectionNode);
            const insertNodes: LexicalNode[] = [];
            // eslint-disable-next-line no-constant-condition
            while (true) {
                if ($isTabNode(node)) {
                    insertNodes.push($createTabNode());
                    node = node.getNextSibling();
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
                    node = node.getNextSibling();
                } else {
                    break;
                }
            }
            const split = firstSelectionNode.splitText(anchor.offset)[0];
            const x = anchor.offset === 0 ? 0 : 1;
            const index = split.getIndexWithinParent() + x;
            const codeNode = firstSelectionNode.getParentOrThrow();
            const nodesToInsert = [$createLineBreakNode(), ...insertNodes];
            codeNode.splice(index, 0, nodesToInsert);
            const last = insertNodes[insertNodes.length - 1];
            if (last) {
                // @ts-ignore TODO
                last.select();
            } else if (anchor.offset === 0) {
                split.selectPrevious();
            } else {
                split.getNextSibling()!.selectNext(0, 0);
            }
        }
        if ($isCodeNode(firstSelectionNode)) {
            const { offset } = selection.anchor;
            firstSelectionNode.splice(offset, 0, [$createLineBreakNode()]);
            firstSelectionNode.select(offset + 1, offset + 1);
        }

        return null;
    }

    canIndent(): false {
        return false;
    }

    collapseAtStart(): boolean {
        const paragraph = $createParagraphNode();
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
        return this.getLatest().__language;
    }
}

export class CodeHighlightNode extends TextNode {
    /** @internal */
    __highlightType: string | null | undefined;

    constructor(
        text: string,
        highlightType?: string | null | undefined,
        key?: NodeKey,
    ) {
        super(text, key);
        this.__highlightType = highlightType;
    }

    static getType(): string {
        return "code-highlight";
    }

    static clone(node: CodeHighlightNode): CodeHighlightNode {
        return new CodeHighlightNode(
            node.__text,
            node.__highlightType || undefined,
            node.__key,
        );
    }

    getHighlightType(): string | null | undefined {
        const self = this.getLatest();
        return self.__highlightType;
    }

    canHaveFormat(): boolean {
        return false;
    }

    createDOM(config: EditorConfig): HTMLElement {
        const element = super.createDOM(config);
        const className = getHighlightThemeClass(
            config.theme,
            this.__highlightType,
        );
        addClassNamesToElement(element, className);
        return element;
    }

    updateDOM(
        prevNode: CodeHighlightNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        const update = super.updateDOM(prevNode, dom, config);
        const prevClassName = getHighlightThemeClass(
            config.theme,
            prevNode.__highlightType,
        );
        const nextClassName = getHighlightThemeClass(
            config.theme,
            this.__highlightType,
        );
        if (prevClassName !== nextClassName) {
            if (prevClassName) {
                removeClassNamesFromElement(dom, prevClassName);
            }
            if (nextClassName) {
                addClassNamesToElement(dom, nextClassName);
            }
        }
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
            highlightType: this.getHighlightType(),
            type: "code-highlight",
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
    return $applyNodeReplacement(new CodeNode(language));
}

export function $isCodeNode(
    node: LexicalNode | null | undefined,
): node is CodeNode {
    return node instanceof CodeNode;
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
            if (domParent != null && domNode !== domParent.lastChild) {
                childLexicalNodes.push($createLineBreakNode());
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
                childLexicalNodes.push($createLineBreakNode());
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
    while ($isCodeHighlightNode(node) || $isTabNode(node)) {
        previousNode = node;
        node = node.getPreviousSibling();
    }
    return previousNode;
}

export function getLastCodeNodeOfLine(
    anchor: CodeHighlightNode | TabNode | LineBreakNode,
): CodeHighlightNode | TabNode | LineBreakNode {
    let nextNode = anchor;
    let node: null | LexicalNode = anchor;
    while ($isCodeHighlightNode(node) || $isTabNode(node)) {
        nextNode = node;
        node = node.getNextSibling();
    }
    return nextNode;
}

function getHighlightThemeClass(
    theme: EditorThemeClasses,
    highlightType: string | null | undefined,
): string | null | undefined {
    return (
        highlightType &&
        theme &&
        theme.codeHighlight &&
        theme.codeHighlight[highlightType]
    );
}
