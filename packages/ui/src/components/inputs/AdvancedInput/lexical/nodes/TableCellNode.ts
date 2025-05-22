import { PIXEL_VALUE_REG_EXP } from "../consts.js";
import { type DOMConversionMap, type DOMConversionOutput, type DOMExportOutput, type NodeConstructorPayloads, type NodeType, type SerializedTableCellNode, type TableCellHeaderState, TableCellHeaderStates } from "../types.js";
import { $createNode, $isNode, getParent } from "../utils.js";
import { ElementNode } from "./ElementNode.js";

export class TableCellNode extends ElementNode {
    static __type: NodeType = "TableCell";
    __colSpan: number;
    __rowSpan: number;
    __headerState: TableCellHeaderState;
    __width?: number;
    __backgroundColor: null | string;

    constructor({ colSpan, headerState, width, ...rest }: NodeConstructorPayloads["TableCell"]) {
        super(rest);
        this.__colSpan = colSpan !== undefined ? colSpan : 1;
        this.__rowSpan = 1;
        this.__headerState = headerState || TableCellHeaderStates.NO_STATUS;
        this.__width = width;
        this.__backgroundColor = null;
    }

    static clone(node: TableCellNode): TableCellNode {
        const { __colSpan, __headerState, __width, __key } = node;
        const cellNode = $createNode("TableCell", { colSpan: __colSpan, headerState: __headerState, width: __width, key: __key });
        cellNode.__rowSpan = node.__rowSpan;
        cellNode.__backgroundColor = node.__backgroundColor;
        return cellNode;
    }

    static importDOM(): DOMConversionMap {
        return {
            td: (node: Node) => ({
                conversion: convertTableCellNodeElement,
                priority: 0,
            }),
            th: (node: Node) => ({
                conversion: convertTableCellNodeElement,
                priority: 0,
            }),
        };
    }

    static importJSON({ backgroundColor, colSpan, headerState, rowSpan, width }: SerializedTableCellNode): TableCellNode {
        const cellNode = $createNode("TableCell", {
            colSpan: colSpan !== undefined ? colSpan : 1,
            headerState,
            width,
        });
        cellNode.__rowSpan = rowSpan !== undefined ? rowSpan : 1;
        cellNode.__backgroundColor = backgroundColor || null;
        return cellNode;
    }

    createDOM(): HTMLElement {
        const element = document.createElement(
            this.getTag(),
        ) as HTMLTableCellElement;

        if (this.__width) {
            element.style.width = `${this.__width}px`;
        }
        if (this.__colSpan > 1) {
            element.colSpan = this.__colSpan;
        }
        if (this.__rowSpan > 1) {
            element.rowSpan = this.__rowSpan;
        }
        if (this.__backgroundColor !== null) {
            element.style.backgroundColor = this.__backgroundColor;
        }
        return element;
    }

    exportDOM(): DOMExportOutput {
        const { element } = super.exportDOM();

        if (element) {
            const element_ = element as HTMLTableCellElement;
            const maxWidth = 700;
            const colCount = getParent(this, { throwIfNull: true }).getChildrenSize();
            element_.style.border = "1px solid black";
            if (this.__colSpan > 1) {
                element_.colSpan = this.__colSpan;
            }
            if (this.__rowSpan > 1) {
                element_.rowSpan = this.__rowSpan;
            }
            element_.style.width = `${this.getWidth() || Math.max(90, maxWidth / colCount)
                }px`;

            element_.style.verticalAlign = "top";
            element_.style.textAlign = "start";

            const backgroundColor = this.getBackgroundColor();
            if (backgroundColor !== null) {
                element_.style.backgroundColor = backgroundColor;
            } else if (this.hasHeader()) {
                element_.style.backgroundColor = "#f2f3f5";
            }
        }

        return {
            element,
        };
    }

    exportJSON(): SerializedTableCellNode {
        return {
            ...super.exportJSON(),
            __type: "ListItem",
            backgroundColor: this.getBackgroundColor(),
            colSpan: this.__colSpan,
            headerState: this.__headerState,
            rowSpan: this.__rowSpan,
            width: this.getWidth(),
        };
    }

    getColSpan(): number {
        return this.__colSpan;
    }

    setColSpan(colSpan: number): this {
        this.getWritable().__colSpan = colSpan;
        return this;
    }

    getRowSpan(): number {
        return this.__rowSpan;
    }

    setRowSpan(rowSpan: number): this {
        this.getWritable().__rowSpan = rowSpan;
        return this;
    }

    getTag(): string {
        return this.hasHeader() ? "th" : "td";
    }

    setHeaderStyles(headerState: TableCellHeaderState): TableCellHeaderState {
        const self = this.getWritable();
        self.__headerState = headerState;
        return this.__headerState;
    }

    getHeaderStyles(): TableCellHeaderState {
        return this.getLatest().__headerState;
    }

    setWidth(width: number): number | null | undefined {
        const self = this.getWritable();
        self.__width = width;
        return this.__width;
    }

    getWidth(): number | undefined {
        return this.getLatest().__width;
    }

    getBackgroundColor(): null | string {
        return this.getLatest().__backgroundColor;
    }

    setBackgroundColor(newBackgroundColor: null | string): void {
        this.getWritable().__backgroundColor = newBackgroundColor;
    }

    toggleHeaderStyle(headerStateToToggle: TableCellHeaderState): TableCellNode {
        const self = this.getWritable();

        if ((self.__headerState & headerStateToToggle) === headerStateToToggle) {
            self.__headerState -= headerStateToToggle;
        } else {
            self.__headerState += headerStateToToggle;
        }

        return self;
    }

    hasHeaderState(headerState: TableCellHeaderState): boolean {
        return (this.getHeaderStyles() & headerState) === headerState;
    }

    hasHeader(): boolean {
        return this.getLatest().__headerState !== TableCellHeaderStates.NO_STATUS;
    }

    updateDOM(prevNode: TableCellNode): boolean {
        return (
            prevNode.__headerState !== this.__headerState ||
            prevNode.__width !== this.__width ||
            prevNode.__colSpan !== this.__colSpan ||
            prevNode.__rowSpan !== this.__rowSpan ||
            prevNode.__backgroundColor !== this.__backgroundColor
        );
    }

    isShadowRoot(): boolean {
        return true;
    }

    collapseAtStart(): true {
        return true;
    }

    canBeEmpty(): false {
        return false;
    }

    canIndent(): false {
        return false;
    }
}

export function convertTableCellNodeElement(
    domNode: Node,
): DOMConversionOutput {
    const domNode_ = domNode as HTMLTableCellElement;
    const nodeName = domNode.nodeName.toLowerCase();

    let width: number | undefined = undefined;

    if (PIXEL_VALUE_REG_EXP.test(domNode_.style.width)) {
        width = parseFloat(domNode_.style.width);
    }

    const tableCellNode = $createNode("TableCell", {
        colSpan: domNode_.colSpan,
        headerState: nodeName === "th"
            ? TableCellHeaderStates.ROW
            : TableCellHeaderStates.NO_STATUS,
        width,
    });

    tableCellNode.__rowSpan = domNode_.rowSpan;
    const backgroundColor = domNode_.style.backgroundColor;
    if (backgroundColor !== "") {
        tableCellNode.__backgroundColor = backgroundColor;
    }

    const style = domNode_.style;
    const hasBoldFontWeight =
        style.fontWeight === "700" || style.fontWeight === "bold";
    const hasLinethroughTextDecoration = style.textDecoration === "line-through";
    const hasItalicFontStyle = style.fontStyle === "italic";
    const hasUnderlineTextDecoration = style.textDecoration === "underline";

    return {
        after: (childLexicalNodes) => {
            if (childLexicalNodes.length === 0) {
                childLexicalNodes.push($createNode("Paragraph", {}));
            }
            return childLexicalNodes;
        },
        forChild: (lexicalNode, parentLexicalNode) => {
            if ($isNode("TableCell", parentLexicalNode) && !$isNode("Element", lexicalNode)) {
                const paragraphNode = $createNode("Paragraph", {});
                if (
                    $isNode("LineBreak", lexicalNode) &&
                    lexicalNode.getTextContent() === "\n"
                ) {
                    return null;
                }
                if ($isNode("Text", lexicalNode)) {
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
                }
                paragraphNode.append(lexicalNode);
                return paragraphNode;
            }

            return lexicalNode;
        },
        node: tableCellNode,
    };
}
