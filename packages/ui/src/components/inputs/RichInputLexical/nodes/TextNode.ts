/* eslint-disable @typescript-eslint/ban-ts-comment */
import { COMPOSITION_SUFFIX, DETAIL_TYPE_TO_DETAIL, DOM_ELEMENT_TYPE, DOM_TEXT_TYPE, IS_DIRECTIONLESS, IS_FIREFOX, IS_SEGMENTED, IS_TOKEN, IS_UNMERGEABLE, TEXT_FLAGS, TEXT_MODE_TO_TYPE, TEXT_TYPE_TO_MODE } from "../consts";
import { $updateElementSelectionOnCreateDeleteNode, RangeSelection, adjustPointOffsetForMergedSibling, internalMakeRangeSelection } from "../selection";
import { hasFormat, hasTextFormat, toggleTextFormatType } from "../transformers/textFormatTransformers";
import { BaseSelection, DOMConversionMap, DOMConversionOutput, DOMExportOutput, EditorConfig, NodeConstructorPayloads, NodeType, SerializedTextNode, TextDetailType, TextFormatType, TextModeType, TextNodeThemeClasses } from "../types";
import { errorOnReadOnly } from "../updates";
import { $createNode, $getCompositionKey, $getSelection, $isNode, $isRangeSelection, $setCompositionKey, getCachedClassNameArray, getIndexWithinParent, getNextSibling, getParentOrThrow, getPreviousSibling, internalMarkSiblingsAsDirty, isHTMLElement } from "../utils";
import { LexicalNode } from "./LexicalNode";

const OUTER_TAGS = {
    [TEXT_FLAGS.CODE_INLINE]: "code",
    [TEXT_FLAGS.HIGHLIGHT]: "mark",
    [TEXT_FLAGS.SUBSCRIPT]: "sub",
    [TEXT_FLAGS.SUPERSCRIPT]: "sup",
};

const getElementOuterTag = (format: number): string | null => {
    return OUTER_TAGS[format] || null;
};

const INNER_TAGS = {
    [TEXT_FLAGS.BOLD]: "strong",
    [TEXT_FLAGS.ITALIC]: "em",
    [TEXT_FLAGS.UNDERLINE_LINES]: "u",
    [TEXT_FLAGS.UNDERLINE_TAGS]: "u",
    [TEXT_FLAGS.STRIKETHROUGH]: "s",
    // [TEXT_FLAGS.SPOILER_LINES]: "span",
};

const getElementInnerTag = (format: number): string => {
    return INNER_TAGS[format] || "span";
};

const setTextThemeClassNames = (
    tag: string,
    prevFormat: number,
    nextFormat: number,
    dom: HTMLElement,
    textClassNames: TextNodeThemeClasses,
) => {
    const domClassList = dom.classList;
    // Firstly we handle the base theme.
    let classNames = getCachedClassNameArray(textClassNames, "base");
    if (classNames !== undefined) {
        domClassList.add(...classNames);
    }
    // Secondly we handle the special case: underline + strikethrough.
    // We have to do this as we need a way to compose the fact that
    // the same CSS property will need to be used: text-decoration.
    // In an ideal world we shouldn't have to do this, but there's no
    // easy workaround for many atomic CSS systems today.
    classNames = getCachedClassNameArray(
        textClassNames,
        "underlineStrikethrough",
    );
    let hasUnderlineStrikethrough = false;
    // Check if previous was both underlined and strikethrough
    const prevUnderlineStrikethrough =
        hasTextFormat(prevFormat, "UNDERLINE_LINES", "UNDERLINE_TAGS") &&
        hasTextFormat(prevFormat, "STRIKETHROUGH");
    // Check if next is both underlined and strikethrough
    const nextUnderlineStrikethrough =
        hasTextFormat(nextFormat, "UNDERLINE_LINES", "UNDERLINE_TAGS") &&
        hasTextFormat(nextFormat, "STRIKETHROUGH");

    if (classNames !== undefined) {
        if (nextUnderlineStrikethrough) {
            hasUnderlineStrikethrough = true;
            if (!prevUnderlineStrikethrough) {
                domClassList.add(...classNames);
            }
        } else if (prevUnderlineStrikethrough) {
            domClassList.remove(...classNames);
        }
    }

    for (const flag in TEXT_FLAGS) {
        classNames = getCachedClassNameArray(textClassNames, flag);
        if (classNames !== undefined) {
            // Add new styles to DOM
            if (hasTextFormat(nextFormat, flag as TextFormatType)) {
                if (
                    hasUnderlineStrikethrough &&
                    (flag === "UNDERLINE_LINES" || flag === "UNDERLINE_TAGS" || flag === "STRIKETHROUGH")
                ) {
                    if (hasTextFormat(prevFormat, flag as TextFormatType)) {
                        domClassList.remove(...classNames);
                    }
                    continue;
                }
                if (
                    !hasTextFormat(prevFormat, flag as TextFormatType) ||
                    (prevUnderlineStrikethrough && (flag === "UNDERLINE_LINES" || flag === "UNDERLINE_TAGS")) ||
                    flag === "STRIKETHROUGH"
                ) {
                    domClassList.add(...classNames);
                }
            }
            // Remove old styles from DOM
            else if (hasTextFormat(prevFormat, flag as TextFormatType)) {
                domClassList.remove(...classNames);
            }
        }
    }
};

const diffComposedText = (a: string, b: string): [number, number, string] => {
    const aLength = a.length;
    const bLength = b.length;
    let left = 0;
    let right = 0;

    while (left < aLength && left < bLength && a[left] === b[left]) {
        left++;
    }
    while (
        right + left < aLength &&
        right + left < bLength &&
        a[aLength - right - 1] === b[bLength - right - 1]
    ) {
        right++;
    }

    return [left, aLength - left - right, b.slice(left, bLength - right)];
};

const setTextContent = (
    nextText: string,
    dom: HTMLElement,
    node: TextNode,
) => {
    const firstChild = dom.firstChild;
    const isComposing = node.isComposing();
    // Always add a suffix if we're composing a node
    const suffix = isComposing ? COMPOSITION_SUFFIX : "";
    const text: string = nextText + suffix;

    if (firstChild === null) {
        dom.textContent = text;
    } else {
        const nodeValue = firstChild.nodeValue;
        if (nodeValue !== text) {
            if (isComposing || IS_FIREFOX) {
                // We also use the diff composed text for general text in FF to avoid
                // the spellcheck red line from flickering.
                const [index, remove, insert] = diffComposedText(
                    nodeValue as string,
                    text,
                );
                if (remove !== 0) {
                    // @ts-expect-error
                    firstChild.deleteData(index, remove);
                }
                // @ts-expect-error
                firstChild.insertData(index, insert);
            } else {
                firstChild.nodeValue = text;
            }
        }
    }
};

const createTextInnerDOM = (
    innerDOM: HTMLElement,
    node: TextNode,
    innerTag: string,
    format: number,
    text: string,
) => {
    setTextContent(text, innerDOM, node);
};

const wrapElementWith = (
    element: HTMLElement | Text,
    tag: string,
): HTMLElement => {
    const el = document.createElement(tag);
    el.appendChild(element);
    return el;
};

/** @noInheritDoc */
export class TextNode extends LexicalNode {
    static __type: NodeType = "Text";
    __text: string;
    __format: number;
    __style: string;
    __mode: 0 | 1 | 2 | 3;
    __detail: number;

    static clone(node: TextNode): TextNode {
        const { __text, __key } = node;
        return $createNode("Text", { text: __text, key: __key });
    }

    constructor({ text, key }: NodeConstructorPayloads["Text"]) {
        super(key);
        this.__text = text;
        this.__format = 0;
        this.__style = "";
        this.__mode = 0;
        this.__detail = 0;
    }

    /**
     * Returns a 32-bit integer that represents the TextFormatTypes currently applied to the
     * TextNode. You probably don't want to use this method directly - consider using TextNode.hasFormat instead.
     *
     * @returns a number representing the format of the text node.
     */
    getFormat(): number {
        const self = this.getLatest();
        return self.__format;
    }

    /**
     * Returns a 32-bit integer that represents the TextDetailTypes currently applied to the
     * TextNode. You probably don't want to use this method directly - consider using TextNode.isDirectionless
     * or TextNode.isUnmergeable instead.
     *
     * @returns a number representing the detail of the text node.
     */
    getDetail(): number {
        const self = this.getLatest();
        return self.__detail;
    }

    /**
     * Returns the mode (TextModeType) of the TextNode, which may be "normal", "token", or "segmented"
     *
     * @returns TextModeType.
     */
    getMode(): TextModeType {
        const self = this.getLatest();
        return TEXT_TYPE_TO_MODE[self.__mode];
    }

    /**
     * Returns the styles currently applied to the node. This is analogous to CSSText in the DOM.
     *
     * @returns CSSText-like string of styles applied to the underlying DOM node.
     */
    getStyle(): string {
        const self = this.getLatest();
        return self.__style;
    }

    /**
     * Returns whether or not the node is in "token" mode. TextNodes in token mode can be navigated through character-by-character
     * with a RangeSelection, but are deleted as a single entity (not invdividually by character).
     *
     * @returns true if the node is in token mode, false otherwise.
     */
    isToken(): boolean {
        const self = this.getLatest();
        return self.__mode === IS_TOKEN;
    }

    /**
     *
     * @returns true if Lexical detects that an IME or other 3rd-party script is attempting to
     * mutate the TextNode, false otherwise.
     */
    isComposing(): boolean {
        return this.__key === $getCompositionKey();
    }

    /**
     * Returns whether or not the node is in "segemented" mode. TextNodes in segemented mode can be navigated through character-by-character
     * with a RangeSelection, but are deleted in space-delimited "segments".
     *
     * @returns true if the node is in segmented mode, false otherwise.
     */
    isSegmented(): boolean {
        const self = this.getLatest();
        return self.__mode === IS_SEGMENTED;
    }
    /**
     * Returns whether or not the node is "directionless". Directionless nodes don't respect changes between RTL and LTR modes.
     *
     * @returns true if the node is directionless, false otherwise.
     */
    isDirectionless(): boolean {
        const self = this.getLatest();
        return (self.__detail & IS_DIRECTIONLESS) !== 0;
    }
    /**
     * Returns whether or not the node is unmergeable. In some scenarios, Lexical tries to merge
     * adjacent TextNodes into a single TextNode. If a TextNode is unmergeable, this won't happen.
     *
     * @returns true if the node is unmergeable, false otherwise.
     */
    isUnmergeable(): boolean {
        const self = this.getLatest();
        return (self.__detail & IS_UNMERGEABLE) !== 0;
    }

    /**
     * Returns whether or not the node is simple text. Simple text is defined as a TextNode that has the string type "text"
     * (i.e., not a subclass) and has no mode applied to it (i.e., not segmented or token).
     *
     * @returns true if the node is simple text, false otherwise.
     */
    isSimpleText(): boolean {
        return this.__mode === 0;
    }

    /**
     * Returns the text content of the node as a string.
     *
     * @returns a string representing the text content of the node.
     */
    getTextContent(): string {
        const self = this.getLatest();
        return self.__text;
    }

    /**
     * Returns the format flags applied to the node as a 32-bit integer.
     *
     * @returns a number representing the TextFormatTypes applied to the node.
     */
    getFormatFlags(type: TextFormatType, alignWithFormat: null | number): number {
        const self = this.getLatest();
        const format = self.__format;
        return toggleTextFormatType(format, type, alignWithFormat);
    }

    /**
     *
     * @returns true if the text node supports font styling, false otherwise.
     */
    canHaveFormat(): boolean {
        return true;
    }

    // View

    createDOM(): HTMLElement {
        const format = this.__format;
        const outerTag = getElementOuterTag(format);
        const innerTag = getElementInnerTag(format);
        const tag = outerTag === null ? innerTag : outerTag;
        const dom = document.createElement(tag);
        let innerDOM = dom;
        if (hasFormat(this, "CODE_INLINE")) {
            dom.setAttribute("spellcheck", "false");
        }
        if (outerTag !== null) {
            innerDOM = document.createElement(innerTag);
            dom.appendChild(innerDOM);
        }
        const text = this.__text;
        createTextInnerDOM(innerDOM, this, innerTag, format, text);
        const style = this.__style;
        if (style !== "") {
            dom.style.cssText = style;
        }
        return dom;
    }

    updateDOM(
        prevNode: TextNode,
        dom: HTMLElement,
        config: EditorConfig,
    ): boolean {
        const nextText = this.__text;
        const prevFormat = prevNode.__format;
        const nextFormat = this.__format;
        const prevOuterTag = getElementOuterTag(prevFormat);
        const nextOuterTag = getElementOuterTag(nextFormat);
        const prevInnerTag = getElementInnerTag(prevFormat);
        const nextInnerTag = getElementInnerTag(nextFormat);
        const prevTag = prevOuterTag === null ? prevInnerTag : prevOuterTag;
        const nextTag = nextOuterTag === null ? nextInnerTag : nextOuterTag;

        if (prevTag !== nextTag) {
            return true;
        }
        if (prevOuterTag === nextOuterTag && prevInnerTag !== nextInnerTag) {
            // should always be an element
            const prevInnerDOM: HTMLElement = dom.firstChild as HTMLElement;
            if (prevInnerDOM === null) {
                throw new Error("updateDOM: prevInnerDOM is null or undefined");
            }
            const nextInnerDOM = document.createElement(nextInnerTag);
            createTextInnerDOM(
                nextInnerDOM,
                this,
                nextInnerTag,
                nextFormat,
                nextText,
            );
            dom.replaceChild(nextInnerDOM, prevInnerDOM);
            return false;
        }
        let innerDOM = dom;
        if (nextOuterTag !== null) {
            if (prevOuterTag !== null) {
                innerDOM = dom.firstChild as HTMLElement;
                if (innerDOM === null) {
                    throw new Error("updateDOM: innerDOM is null or undefined");
                }
            }
        }
        setTextContent(nextText, innerDOM, this);
        const prevStyle = prevNode.__style;
        const nextStyle = this.__style;
        if (prevStyle !== nextStyle) {
            dom.style.cssText = nextStyle;
        }
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            "#text": () => ({
                conversion: convertTextDOMNode,
                priority: 0,
            }),
            b: () => ({
                conversion: convertBringAttentionToElement,
                priority: 0,
            }),
            code: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            em: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            i: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            s: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            span: () => ({
                conversion: convertSpanElement,
                priority: 0,
            }),
            strong: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            sub: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            sup: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
            u: () => ({
                conversion: convertTextFormatElement,
                priority: 0,
            }),
        };
    }

    static importJSON(serializedNode: SerializedTextNode): TextNode {
        const node = $createNode("Text", { text: serializedNode.text });
        node.setFormat(serializedNode.format);
        node.setDetail(serializedNode.detail);
        node.setMode(serializedNode.mode);
        node.setStyle(serializedNode.style);
        return node;
    }

    // This improves Lexical's basic text output in copy+paste plus
    // for headless mode where people might use Lexical to generate
    // HTML content and not have the ability to use CSS classes.
    exportDOM(): DOMExportOutput {
        let { element } = super.exportDOM();
        if (element === null || !isHTMLElement(element)) {
            throw new Error("Expected TextNode createDOM to always return a HTMLElement");
        }
        element.style.whiteSpace = "pre-wrap";
        // This is the only way to properly add support for most clients,
        // even if it's semantically incorrect to have to resort to using
        // <b>, <u>, <s>, <i> elements.
        if (hasFormat(this, "BOLD")) {
            element = wrapElementWith(element, "b");
        }
        if (hasFormat(this, "ITALIC")) {
            element = wrapElementWith(element, "i");
        }
        if (hasFormat(this, "STRIKETHROUGH")) {
            element = wrapElementWith(element, "s");
        }
        if (hasFormat(this, "UNDERLINE_LINES") || hasFormat(this, "UNDERLINE_TAGS")) {
            element = wrapElementWith(element, "u");
        }

        return {
            element,
        };
    }

    exportJSON(): SerializedTextNode {
        return {
            __type: "Text",
            detail: this.getDetail(),
            format: this.getFormat(),
            mode: this.getMode(),
            style: this.getStyle(),
            text: this.getTextContent(),
            version: 1,
        };
    }

    // Mutators
    selectionTransform(
        prevSelection: BaseSelection | null | undefined,
        nextSelection: RangeSelection,
    ): void {
        return;
    }

    /**
     * Sets the node format to the provided TextFormatType or 32-bit integer. Note that the TextFormatType
     * version of the argument can only specify one format and doing so will remove all other formats that
     * may be applied to the node. For toggling behavior, consider using {@link TextNode.toggleFormat}
     *
     * @param format - TextFormatType or 32-bit integer representing the node format.
     *
     * @returns this TextNode.
     */
    setFormat(format: TextFormatType | number): this {
        const self = this.getWritable();
        self.__format = typeof format === "string" ? TEXT_FLAGS[format] : format;
        return self;
    }

    /**
     * Sets the node detail to the provided TextDetailType or 32-bit integer. Note that the TextDetailType
     * version of the argument can only specify one detail value and doing so will remove all other detail values that
     * may be applied to the node. For toggling behavior, consider using {@link TextNode.toggleDirectionless}
     * or {@link TextNode.togglerUnmergeable}
     *
     * @param detail - TextDetailType or 32-bit integer representing the node detail.
     *
     * @returns this TextNode.
     */
    setDetail(detail: TextDetailType | number): this {
        const self = this.getWritable();
        self.__detail =
            typeof detail === "string" ? DETAIL_TYPE_TO_DETAIL[detail] : detail;
        return self;
    }

    /**
     * Sets the node style to the provided CSSText-like string. Set this property as you
     * would an HTMLElement style attribute to apply inline styles to the underlying DOM Element.
     *
     * @param style - CSSText to be applied to the underlying HTMLElement.
     *
     * @returns this TextNode.
     */
    setStyle(style: string): this {
        const self = this.getWritable();
        self.__style = style;
        return self;
    }

    /**
     * Applies the provided format to this TextNode if it's not present. Removes it if it's present.
     * The subscript and superscript formats are mutually exclusive.
     * Prefer using this method to turn specific formats on and off.
     *
     * @param type - TextFormatType to toggle.
     *
     * @returns this TextNode.
     */
    toggleFormat(type: TextFormatType): this {
        const format = this.getFormat();
        const newFormat = toggleTextFormatType(format, type);
        return this.setFormat(newFormat);
    }

    /**
     * Toggles the directionless detail value of the node. Prefer using this method over setDetail.
     *
     * @returns this TextNode.
     */
    toggleDirectionless(): this {
        const self = this.getWritable();
        self.__detail ^= IS_DIRECTIONLESS;
        return self;
    }

    /**
     * Toggles the unmergeable detail value of the node. Prefer using this method over setDetail.
     *
     * @returns this TextNode.
     */
    toggleUnmergeable(): this {
        const self = this.getWritable();
        self.__detail ^= IS_UNMERGEABLE;
        return self;
    }

    /**
     * Sets the mode of the node.
     *
     * @returns this TextNode.
     */
    setMode(type: TextModeType): this {
        const mode = TEXT_MODE_TO_TYPE[type];
        if (this.__mode === mode) {
            return this;
        }
        const self = this.getWritable();
        self.__mode = mode;
        return self;
    }

    /**
     * Sets the text content of the node.
     *
     * @param text - the string to set as the text value of the node.
     *
     * @returns this TextNode.
     */
    setTextContent(text: string): this {
        if (this.__text === text) {
            return this;
        }
        const self = this.getWritable();
        self.__text = text;
        return self;
    }

    /**
     * Sets the current Lexical selection to be a RangeSelection with anchor and focus on this TextNode at the provided offsets.
     *
     * @param _anchorOffset - the offset at which the Selection anchor will be placed.
     * @param _focusOffset - the offset at which the Selection focus will be placed.
     *
     * @returns the new RangeSelection.
     */
    select(_anchorOffset?: number, _focusOffset?: number): RangeSelection {
        errorOnReadOnly();
        let anchorOffset = _anchorOffset;
        let focusOffset = _focusOffset;
        const selection = $getSelection();
        const text = this.getTextContent();
        const key = this.__key;
        if (typeof text === "string") {
            const lastOffset = text.length;
            if (anchorOffset === undefined) {
                anchorOffset = lastOffset;
            }
            if (focusOffset === undefined) {
                focusOffset = lastOffset;
            }
        } else {
            anchorOffset = 0;
            focusOffset = 0;
        }
        if (!$isRangeSelection(selection)) {
            return internalMakeRangeSelection(
                key,
                anchorOffset,
                key,
                focusOffset,
                "text",
                "text",
            );
        } else {
            const compositionKey = $getCompositionKey();
            if (
                compositionKey === selection.anchor.key ||
                compositionKey === selection.focus.key
            ) {
                $setCompositionKey(key);
            }
            selection.setTextNodeRange(this, anchorOffset, this, focusOffset);
        }
        return selection;
    }

    selectStart(): RangeSelection {
        return this.select(0, 0);
    }

    selectEnd(): RangeSelection {
        const size = this.getTextContentSize();
        return this.select(size, size);
    }

    /**
     * Inserts the provided text into this TextNode at the provided offset, deleting the number of characters
     * specified. Can optionally calculate a new selection after the operation is complete.
     *
     * @param offset - the offset at which the splice operation should begin.
     * @param delCount - the number of characters to delete, starting from the offset.
     * @param newText - the text to insert into the TextNode at the offset.
     * @param moveSelection - optional, whether or not to move selection to the end of the inserted substring.
     *
     * @returns this TextNode.
     */
    spliceText(
        offset: number,
        delCount: number,
        newText: string,
        moveSelection?: boolean,
    ): TextNode {
        const writableSelf = this.getWritable();
        const text = writableSelf.__text;
        const handledTextLength = newText.length;
        let index = offset;
        if (index < 0) {
            index = handledTextLength + index;
            if (index < 0) {
                index = 0;
            }
        }
        const selection = $getSelection();
        if (moveSelection && $isRangeSelection(selection)) {
            const newOffset = offset + handledTextLength;
            selection.setTextNodeRange(
                writableSelf,
                newOffset,
                writableSelf,
                newOffset,
            );
        }

        const updatedText =
            text.slice(0, index) + newText + text.slice(index + delCount);

        writableSelf.__text = updatedText;
        return writableSelf;
    }

    /**
     * This method is meant to be overriden by TextNode subclasses to control the behavior of those nodes
     * when a user event would cause text to be inserted before them in the editor. If true, Lexical will attempt
     * to insert text into this node. If false, it will insert the text in a new sibling node.
     *
     * @returns true if text can be inserted before the node, false otherwise.
     */
    canInsertTextBefore(): boolean {
        return true;
    }

    /**
     * This method is meant to be overriden by TextNode subclasses to control the behavior of those nodes
     * when a user event would cause text to be inserted after them in the editor. If true, Lexical will attempt
     * to insert text into this node. If false, it will insert the text in a new sibling node.
     *
     * @returns true if text can be inserted after the node, false otherwise.
     */
    canInsertTextAfter(): boolean {
        return true;
    }

    /**
     * Splits this TextNode at the provided character offsets, forming new TextNodes from the substrings
     * formed by the split, and inserting those new TextNodes into the editor, replacing the one that was split.
     *
     * @param splitOffsets - rest param of the text content character offsets at which this node should be split.
     *
     * @returns an Array containing the newly-created TextNodes.
     */
    splitText(...splitOffsets: Array<number>): Array<TextNode> {
        errorOnReadOnly();
        const self = this.getLatest();
        const textContent = self.getTextContent();
        const key = self.__key;
        const compositionKey = $getCompositionKey();
        const offsetsSet = new Set(splitOffsets);
        const parts: string[] = [];
        const textLength = textContent.length;
        let string = "";
        for (let i = 0; i < textLength; i++) {
            if (string !== "" && offsetsSet.has(i)) {
                parts.push(string);
                string = "";
            }
            string += textContent[i];
        }
        if (string !== "") {
            parts.push(string);
        }
        const partsLength = parts.length;
        if (partsLength === 0) {
            return [];
        } else if (parts[0] === textContent) {
            return [self];
        }
        const firstPart = parts[0];
        const parent = getParentOrThrow(self);
        let writableNode;
        const format = self.getFormat();
        const style = self.getStyle();
        const detail = self.__detail;
        let hasReplacedSelf = false;

        if (self.isSegmented()) {
            // Create a new TextNode
            writableNode = $createNode("Text", { text: firstPart });
            writableNode.__format = format;
            writableNode.__style = style;
            writableNode.__detail = detail;
            hasReplacedSelf = true;
        } else {
            // For the first part, update the existing node
            writableNode = self.getWritable();
            writableNode.__text = firstPart;
        }

        // Handle selection
        const selection = $getSelection();

        // Then handle all other parts
        const splitNodes: TextNode[] = [writableNode];
        let textSize = firstPart.length;

        for (let i = 1; i < partsLength; i++) {
            const part = parts[i];
            const partSize = part.length;
            const sibling = $createNode("Text", { text: part }).getWritable();
            sibling.__format = format;
            sibling.__style = style;
            sibling.__detail = detail;
            const siblingKey = sibling.__key;
            const nextTextSize = textSize + partSize;

            if ($isRangeSelection(selection)) {
                const anchor = selection.anchor;
                const focus = selection.focus;

                if (
                    anchor.key === key &&
                    anchor.type === "text" &&
                    anchor.offset > textSize &&
                    anchor.offset <= nextTextSize
                ) {
                    anchor.key = siblingKey;
                    anchor.offset -= textSize;
                    selection.dirty = true;
                }
                if (
                    focus.key === key &&
                    focus.type === "text" &&
                    focus.offset > textSize &&
                    focus.offset <= nextTextSize
                ) {
                    focus.key = siblingKey;
                    focus.offset -= textSize;
                    selection.dirty = true;
                }
            }
            if (compositionKey === key) {
                $setCompositionKey(siblingKey);
            }
            textSize = nextTextSize;
            splitNodes.push(sibling);
        }

        // Insert the nodes into the parent's children
        internalMarkSiblingsAsDirty(this);
        const writableParent = parent.getWritable();
        const insertionIndex = getIndexWithinParent(this);
        if (hasReplacedSelf) {
            writableParent.splice(insertionIndex, 0, splitNodes);
            this.remove();
        } else {
            writableParent.splice(insertionIndex, 1, splitNodes);
        }

        if ($isRangeSelection(selection)) {
            $updateElementSelectionOnCreateDeleteNode(
                selection,
                parent,
                insertionIndex,
                partsLength - 1,
            );
        }

        return splitNodes;
    }

    /**
     * Merges the target TextNode into this TextNode, removing the target node.
     *
     * @param target - the TextNode to merge into this one.
     *
     * @returns this TextNode.
     */
    mergeWithSibling(target: TextNode): TextNode {
        const isBefore = target === getPreviousSibling(this);
        if (!isBefore && target !== getNextSibling(this)) {
            throw new Error("mergeWithSibling: sibling must be a previous or next sibling");
        }
        const key = this.__key;
        const targetKey = target.__key;
        const text = this.__text;
        const textLength = text.length;
        const compositionKey = $getCompositionKey();

        if (compositionKey === targetKey) {
            $setCompositionKey(key);
        }
        const selection = $getSelection();
        if ($isRangeSelection(selection)) {
            const anchor = selection.anchor;
            const focus = selection.focus;
            if (anchor !== null && anchor.key === targetKey) {
                adjustPointOffsetForMergedSibling(
                    anchor,
                    isBefore,
                    key,
                    target,
                    textLength,
                );
                selection.dirty = true;
            }
            if (focus !== null && focus.key === targetKey) {
                adjustPointOffsetForMergedSibling(
                    focus,
                    isBefore,
                    key,
                    target,
                    textLength,
                );
                selection.dirty = true;
            }
        }
        const targetText = target.__text;
        const newText = isBefore ? targetText + text : text + targetText;
        this.setTextContent(newText);
        const writableSelf = this.getWritable();
        target.remove();
        return writableSelf;
    }

    /**
     * This method is meant to be overriden by TextNode subclasses to control the behavior of those nodes
     * when used with the registerLexicalTextEntity function. If you're using registerLexicalTextEntity, the
     * node class that you create and replace matched text with should return true from this method.
     *
     * @returns true if the node is to be treated as a "text entity", false otherwise.
     */
    isTextEntity(): boolean {
        return false;
    }
}

const convertSpanElement = (domNode: Node): DOMConversionOutput => {
    // domNode is a <span> since we matched it by nodeName
    const span = domNode as HTMLSpanElement;
    const style = span.style;
    const fontWeight = style.fontWeight;
    // Google Docs uses span tags + font-weight for bold text
    const hasBoldFontWeight = fontWeight === "700" || fontWeight === "bold";
    // Google Docs uses span tags + text-decoration: line-through for strikethrough text
    const hasLinethroughTextDecoration = style.textDecoration === "line-through";
    // Google Docs uses span tags + font-style for italic text
    const hasItalicFontStyle = style.fontStyle === "italic";
    // Google Docs uses span tags + text-decoration: underline for underline text
    const hasUnderlineTextDecoration = style.textDecoration === "underline";
    // Google Docs uses span tags + vertical-align to specify subscript and superscript
    const verticalAlign = style.verticalAlign;

    return {
        forChild: (lexicalNode) => {
            if (!$isNode("Text", lexicalNode)) {
                return lexicalNode;
            }
            if (hasBoldFontWeight) {
                lexicalNode.toggleFormat("BOLD");
            }
            if (hasLinethroughTextDecoration) {
                lexicalNode.toggleFormat("STRIKETHROUGH");
            }
            if (hasItalicFontStyle) {
                lexicalNode.toggleFormat("ITALIC");
            }
            if (hasUnderlineTextDecoration) {
                lexicalNode.toggleFormat("UNDERLINE_TAGS");
            }
            if (verticalAlign === "sub") {
                lexicalNode.toggleFormat("SUBSCRIPT");
            }
            if (verticalAlign === "super") {
                lexicalNode.toggleFormat("SUPERSCRIPT");
            }

            return lexicalNode;
        },
        node: null,
    };
};

const convertBringAttentionToElement = (domNode: Node): DOMConversionOutput => {
    // domNode is a <b> since we matched it by nodeName
    const b = domNode as HTMLElement;
    // Google Docs wraps all copied HTML in a <b> with font-weight normal
    const hasNormalFontWeight = b.style.fontWeight === "normal";
    return {
        forChild: (lexicalNode) => {
            if ($isNode("Text", lexicalNode) && !hasNormalFontWeight) {
                lexicalNode.toggleFormat("BOLD");
            }

            return lexicalNode;
        },
        node: null,
    };
};

const preParentCache = new WeakMap<Node, null | Node>();

const isNodePre = (node: Node): boolean => {
    return (
        node.nodeName === "PRE" ||
        (node.nodeType === DOM_ELEMENT_TYPE &&
            (node as HTMLElement).style !== undefined &&
            (node as HTMLElement).style.whiteSpace !== undefined &&
            (node as HTMLElement).style.whiteSpace.startsWith("pre"))
    );
};

export const findParentPreDOMNode = (node: Node) => {
    let cached;
    let parent = node.parentNode;
    const visited = [node];
    while (
        parent !== null &&
        (cached = preParentCache.get(parent)) === undefined &&
        !isNodePre(parent)
    ) {
        visited.push(parent);
        parent = parent.parentNode;
    }
    const resultNode = cached === undefined ? parent : cached;
    for (let i = 0; i < visited.length; i++) {
        preParentCache.set(visited[i], resultNode);
    }
    return resultNode;
};

const convertTextDOMNode = (domNode: Node): DOMConversionOutput => {
    const domNode_ = domNode as Text;
    const parentDom = domNode.parentElement;
    if (parentDom === null) {
        throw new Error("Expected parentElement of Text not to be null");
    }
    let textContent = domNode_.textContent || "";
    // No collapse and preserve segment break for pre, pre-wrap and pre-line
    if (findParentPreDOMNode(domNode_) !== null) {
        const parts = textContent.split(/(\r?\n|\t)/);
        const nodes: Array<LexicalNode> = [];
        const length = parts.length;
        for (let i = 0; i < length; i++) {
            const part = parts[i];
            if (part === "\n" || part === "\r\n") {
                nodes.push($createNode("LineBreak", {}));
            } else if (part === "\t") {
                nodes.push($createNode("Tab", {}));
            } else if (part !== "") {
                nodes.push($createNode("Text", { text: part }));
            }
        }
        return { node: nodes };
    }
    textContent = textContent.replace(/\r/g, "").replace(/[ \t\n]+/g, " ");
    if (textContent === "") {
        return { node: null };
    }
    if (textContent[0] === " ") {
        // Traverse backward while in the same line. If content contains new line or tab -> pontential
        // delete, other elements can borrow from this one. Deletion depends on whether it's also the
        // last space (see next condition: textContent[textContent.length - 1] === ' '))
        let previousText: null | Text = domNode_;
        let isStartOfLine = true;
        while (
            previousText !== null &&
            (previousText = findTextInLine(previousText, false)) !== null
        ) {
            const previousTextContent = previousText.textContent || "";
            if (previousTextContent.length > 0) {
                if (/[ \t\n]$/.test(previousTextContent)) {
                    textContent = textContent.slice(1);
                }
                isStartOfLine = false;
                break;
            }
        }
        if (isStartOfLine) {
            textContent = textContent.slice(1);
        }
    }
    if (textContent[textContent.length - 1] === " ") {
        // Traverse forward while in the same line, preserve if next inline will require a space
        let nextText: null | Text = domNode_;
        let isEndOfLine = true;
        while (
            nextText !== null &&
            (nextText = findTextInLine(nextText, true)) !== null
        ) {
            const nextTextContent = (nextText.textContent || "").replace(
                /^( |\t|\r?\n)+/,
                "",
            );
            if (nextTextContent.length > 0) {
                isEndOfLine = false;
                break;
            }
        }
        if (isEndOfLine) {
            textContent = textContent.slice(0, textContent.length - 1);
        }
    }
    if (textContent === "") {
        return { node: null };
    }
    return { node: $createNode("Text", { text: textContent }) };
};

const inlineParents = new RegExp(
    /^(a|abbr|acronym|b|cite|code|del|em|i|ins|kbd|label|output|q|ruby|s|samp|span|strong|sub|sup|time|u|tt|var)$/,
    "i",
);

const findTextInLine = (text: Text, forward: boolean): null | Text => {
    let node: Node = text;
    // eslint-disable-next-line no-constant-condition
    while (true) {
        let sibling: null | Node;
        while (
            (sibling = forward ? node.nextSibling : node.previousSibling) === null
        ) {
            const parentElement = node.parentElement;
            if (parentElement === null) {
                return null;
            }
            node = parentElement;
        }
        node = sibling;
        if (node.nodeType === DOM_ELEMENT_TYPE) {
            const display = (node as HTMLElement).style.display;
            if (
                (display === "" && node.nodeName.match(inlineParents) === null) ||
                (display !== "" && !display.startsWith("inline"))
            ) {
                return null;
            }
        }
        let descendant: null | Node = node;
        while ((descendant = forward ? node.firstChild : node.lastChild) !== null) {
            node = descendant;
        }
        if (node.nodeType === DOM_TEXT_TYPE) {
            return node as Text;
        } else if (node.nodeName === "BR") {
            return null;
        }
    }
};

const nodeNameToTextFormat: Record<string, TextFormatType> = {
    code: "CODE_INLINE",
    em: "ITALIC",
    i: "ITALIC",
    s: "STRIKETHROUGH",
    strong: "BOLD",
    sub: "SUBSCRIPT",
    sup: "SUPERSCRIPT",
    u: "UNDERLINE_TAGS",
};

const convertTextFormatElement = (domNode: Node): DOMConversionOutput => {
    const format = nodeNameToTextFormat[domNode.nodeName.toLowerCase()];
    if (format === undefined) {
        return { node: null };
    }
    return {
        forChild: (lexicalNode) => {
            if (hasFormat(lexicalNode, format)) {
                (lexicalNode as TextNode).toggleFormat(format);
            }

            return lexicalNode;
        },
        node: null,
    };
};
