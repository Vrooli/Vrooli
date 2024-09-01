import { LexicalEditor } from "../editor";
import { getTable } from "../selection";
import { DOMConversionMap, DOMConversionOutput, DOMExportOutput, InsertTableCommandPayloadHeaders, NodeType, SerializedElementNode, SerializedTableNode, TableCellHeaderStates, TableDOMCell, TableDOMTable } from "../types";
import { $createNode, $getNearestNodeFromDOMNode, $isNode, isHTMLElement } from "../utils";
import { ElementNode } from "./ElementNode";
import { type TableCellNode } from "./TableCellNode";
import { type TableRowNode } from "./TableRowNode";

export class TableNode extends ElementNode {
    static __type: NodeType = "Table";

    static clone(node: TableNode): TableNode {
        return new TableNode({ key: node.__key });
    }

    static importDOM(): DOMConversionMap {
        return {
            table: (_node: Node) => ({
                conversion: convertTableElement,
                priority: 1,
            }),
        };
    }

    static importJSON(_serializedNode: SerializedTableNode): TableNode {
        return $createNode("Table", {});
    }

    exportJSON(): SerializedElementNode {
        return {
            ...super.exportJSON(),
            __type: "Table",
            version: 1,
        };
    }

    createDOM(): HTMLElement {
        const tableElement = document.createElement("table");

        return tableElement;
    }

    updateDOM(): boolean {
        return false;
    }

    exportDOM(): DOMExportOutput {
        return {
            ...super.exportDOM(),
            after: (tableElement) => {
                if (tableElement) {
                    const newElement = tableElement.cloneNode() as ParentNode;
                    const colGroup = document.createElement("colgroup");
                    const tBody = document.createElement("tbody");
                    if (isHTMLElement(tableElement)) {
                        tBody.append(...tableElement.children);
                    }
                    const firstRow = this.getFirstChildOrThrow<TableRowNode>();

                    if (!$isNode("TableRow", firstRow)) {
                        throw new Error("Expected to find row node.");
                    }

                    const colCount = firstRow.getChildrenSize();

                    for (let i = 0; i < colCount; i++) {
                        const col = document.createElement("col");
                        colGroup.append(col);
                    }

                    newElement.replaceChildren(colGroup, tBody);

                    return newElement as HTMLElement;
                }
            },
        };
    }

    canBeEmpty(): false {
        return false;
    }

    isShadowRoot(): boolean {
        return true;
    }

    getCordsFromCellNode(
        tableCellNode: TableCellNode,
        table: TableDOMTable,
    ): { x: number; y: number } {
        const { rows, domRows } = table;

        for (let y = 0; y < rows; y++) {
            const row = domRows[y];

            if (row === null || row == undefined) {
                continue;
            }

            const x = row.findIndex((cell) => {
                if (!cell) {
                    return false;
                }
                const { elem } = cell;
                const cellNode = $getNearestNodeFromDOMNode(elem);
                return cellNode === tableCellNode;
            });

            if (x !== -1) {
                return { x, y };
            }
        }

        throw new Error("Cell not found in table.");
    }

    getDOMCellFromCords(
        x: number,
        y: number,
        table: TableDOMTable,
    ): null | TableDOMCell {
        const { domRows } = table;

        const row = domRows[y];

        if (row === null || row === undefined) {
            return null;
        }

        const cell = row[x];

        if (cell === null || cell === undefined) {
            return null;
        }

        return cell;
    }

    getDOMCellFromCordsOrThrow(
        x: number,
        y: number,
        table: TableDOMTable,
    ): TableDOMCell {
        const cell = this.getDOMCellFromCords(x, y, table);

        if (!cell) {
            throw new Error("Cell not found at cords.");
        }

        return cell;
    }

    getCellNodeFromCords(
        x: number,
        y: number,
        table: TableDOMTable,
    ): null | TableCellNode {
        const cell = this.getDOMCellFromCords(x, y, table);

        if (cell === null) {
            return null;
        }

        const node = $getNearestNodeFromDOMNode(cell.elem);

        if ($isNode("TableCell", node)) {
            return node;
        }

        return null;
    }

    getCellNodeFromCordsOrThrow(
        x: number,
        y: number,
        table: TableDOMTable,
    ): TableCellNode {
        const node = this.getCellNodeFromCords(x, y, table);

        if (!node) {
            throw new Error("Node at cords not TableCellNode.");
        }

        return node;
    }

    canSelectBefore(): true {
        return true;
    }

    canIndent(): false {
        return false;
    }
}

export function $getElementForTableNode(
    editor: LexicalEditor,
    tableNode: TableNode,
): TableDOMTable {
    const tableElement = editor.getElementByKey(tableNode.__key);

    if (tableElement === null) {
        throw new Error("Table Element Not Found");
    }

    return getTable(tableElement);
}

export function convertTableElement(_domNode: Node): DOMConversionOutput {
    return { node: $createNode("Table", {}) };
}

export function $createTableNodeWithDimensions(
    rowCount: number,
    columnCount: number,
    includeHeaders: InsertTableCommandPayloadHeaders = true,
): TableNode {
    const tableNode = $createNode("Table", {});

    for (let iRow = 0; iRow < rowCount; iRow++) {
        const tableRowNode = $createNode("TableRow", {});

        for (let iColumn = 0; iColumn < columnCount; iColumn++) {
            let headerState = TableCellHeaderStates.NO_STATUS;

            if (typeof includeHeaders === "object") {
                if (iRow === 0 && includeHeaders.rows) {
                    headerState |= TableCellHeaderStates.ROW;
                }
                if (iColumn === 0 && includeHeaders.columns) {
                    headerState |= TableCellHeaderStates.COLUMN;
                }
            } else if (includeHeaders) {
                if (iRow === 0) {
                    headerState |= TableCellHeaderStates.ROW;
                }
                if (iColumn === 0) {
                    headerState |= TableCellHeaderStates.COLUMN;
                }
            }

            const tableCellNode = $createNode("TableCell", { headerState });
            const paragraphNode = $createNode("Paragraph", {});
            paragraphNode.append($createNode("Text", { text: "" }));
            tableCellNode.append(paragraphNode);
            tableRowNode.append(tableCellNode);
        }

        tableNode.append(tableRowNode);
    }

    return tableNode;
}
