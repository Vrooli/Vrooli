
import { useEffect } from "react";
import { CONTROLLED_TEXT_INSERTION_COMMAND, DELETE_CHARACTER_COMMAND, DELETE_LINE_COMMAND, DELETE_WORD_COMMAND, FOCUS_COMMAND, FORMAT_ELEMENT_COMMAND, FORMAT_TEXT_COMMAND, INSERT_PARAGRAPH_COMMAND, INSERT_TABLE_COMMAND, KEY_ARROW_DOWN_COMMAND, KEY_ARROW_LEFT_COMMAND, KEY_ARROW_RIGHT_COMMAND, KEY_ARROW_UP_COMMAND, KEY_BACKSPACE_COMMAND, KEY_DELETE_COMMAND, KEY_ESCAPE_COMMAND, KEY_TAB_COMMAND, SELECTION_CHANGE_COMMAND, SELECTION_INSERT_CLIPBOARD_NODES_COMMAND } from "../commands.js";
import { COMMAND_PRIORITY_CRITICAL, COMMAND_PRIORITY_EDITOR, COMMAND_PRIORITY_HIGH } from "../consts.js";
import { useLexicalComposerContext } from "../context.js";
import { LexicalEditor } from "../editor.js";
import { ElementNode } from "../nodes/ElementNode.js";
import { type LexicalNode } from "../nodes/LexicalNode.js";
import { type TableCellNode } from "../nodes/TableCellNode.js";
import { $createTableNodeWithDimensions, TableNode } from "../nodes/TableNode.js";
import { type TableRowNode } from "../nodes/TableRowNode.js";
import { $createPoint, $createRangeSelection, $createRangeSelectionFromDom, $getPreviousSelection, RangeSelection, getTable } from "../selection.js";
import { BaseSelection, CustomDomElement, ElementFormatType, InsertTableCommandPayload, LexicalCommand, NodeKey, PointType, TableDOMCell, TableDOMTable, TableMapType, TableMapValueType, TableSelectionShape, TextFormatType } from "../types.js";
import { isCurrentlyReadOnlyMode } from "../updates.js";
import { $computeTableMap, $createNode, $findMatchingParent, $getNearestNodeFromDOMNode, $getNodeByKey, $getRoot, $getSelection, $insertNodeToNearestRoot, $isNode, $isRangeSelection, $nodesOfType, $normalizeSelection, $setSelection, getDOMSelection, getIndexWithinParent, getNextSibling, getParent, getParents, getPreviousSibling, isSelected } from "../utils.js";

export function TablePlugin({
    hasCellMerge = true,
    hasCellBackgroundColor = true,
    hasTabHandler = true,
}: {
    hasCellMerge?: boolean;
    hasCellBackgroundColor?: boolean;
    hasTabHandler?: boolean;
}): JSX.Element | null {
    const editor = useLexicalComposerContext();

    useEffect(() => {
        if (!editor) return;
        return editor.registerCommand<InsertTableCommandPayload>(
            INSERT_TABLE_COMMAND,
            ({ columns, rows, includeHeaders }) => {
                const tableNode = $createTableNodeWithDimensions(
                    Number(rows),
                    Number(columns),
                    includeHeaders,
                );
                $insertNodeToNearestRoot(tableNode);

                const firstDescendant = tableNode.getFirstDescendant();
                if ($isNode("Text", firstDescendant)) {
                    firstDescendant.select();
                }

                return true;
            },
            COMMAND_PRIORITY_EDITOR,
        );
    }, [editor]);

    useEffect(() => {
        if (!editor) return;
        const tableSelections = new Map<NodeKey, TableObserver>();

        const initializeTableNode = (tableNode: TableNode) => {
            const nodeKey = tableNode.__key;
            const tableElement = editor.getElementByKey(nodeKey);
            if (tableElement && !tableSelections.has(nodeKey)) {
                const tableSelection = applyTableHandlers(
                    tableNode,
                    tableElement as CustomDomElement<HTMLTableElement>,
                    editor,
                    hasTabHandler,
                );
                tableSelections.set(nodeKey, tableSelection);
            }
        };

        // Plugins might be loaded _after_ initial content is set, hence existing table nodes
        // won't be initialized from mutation[create] listener. Instead doing it here,
        editor.getEditorState()?.read(() => {
            const tableNodes = $nodesOfType("Table");
            for (const tableNode of tableNodes) {
                if ($isNode("Table", tableNode)) {
                    initializeTableNode(tableNode);
                }
            }
        });

        const unregisterMutationListener = editor.registerListener(
            "Table",
            (nodeMutations) => {
                for (const [nodeKey, mutation] of Object.entries(nodeMutations)) {
                    if (mutation === "created") {
                        editor.getEditorState()?.read(() => {
                            const tableNode = $getNodeByKey<TableNode>(nodeKey);
                            if ($isNode("Table", tableNode)) {
                                initializeTableNode(tableNode);
                            }
                        });
                    } else if (mutation === "destroyed") {
                        const tableSelection = tableSelections.get(nodeKey);

                        if (tableSelection !== undefined) {
                            tableSelection.removeListeners();
                            tableSelections.delete(nodeKey);
                        }
                    }
                }
            },
        );

        return () => {
            unregisterMutationListener();
            // Hook might be called multiple times so cleaning up tables listeners as well,
            // as it'll be reinitialized during recurring call
            for (const [, tableSelection] of tableSelections) {
                tableSelection.removeListeners();
            }
        };
    }, [editor, hasTabHandler]);

    // Unmerge cells when the feature isn't enabled
    useEffect(() => {
        if (!editor || hasCellMerge) return;
        return editor.registerNodeTransform<TableCellNode>("TableCell", (node) => {
            if (node.getColSpan() > 1 || node.getRowSpan() > 1) {
                // When we have rowSpan we have to map the entire Table to understand where the new Cells
                // fit best; let's analyze all Cells at once to save us from further transform iterations
                const [, , gridNode] = $getNodeTriplet(node);
                const [gridMap] = $computeTableMap(gridNode, node, node);
                // TODO this function expects Tables to be normalized. Look into this once it exists
                const rowsCount = gridMap.length;
                const columnsCount = gridMap[0].length;
                let row = gridNode.getFirstChild();
                if (!$isNode("TableRow", row)) {
                    throw new Error("Expected TableNode first child to be a RowNode");
                }
                const unmerged: TableCellNode[] = [];
                for (let i = 0; i < rowsCount; i++) {
                    if (i !== 0) {
                        row = getNextSibling(row);
                        if (!$isNode("TableRow", row)) {
                            throw new Error("Expected TableNode first child to be a RowNode");
                        }
                    }
                    let lastRowCell: null | TableCellNode = null;
                    for (let j = 0; j < columnsCount; j++) {
                        const cellMap = gridMap[i][j];
                        const cell = cellMap.cell;
                        if (cellMap.startRow === i && cellMap.startColumn === j) {
                            lastRowCell = cell;
                            unmerged.push(cell);
                        } else if (cell.getColSpan() > 1 || cell.getRowSpan() > 1) {
                            if (!$isNode("TableCell", cell)) {
                                throw new Error("Expected TableNode cell to be a TableCellNode");
                            }
                            const newCell = $createNode("TableCell", { headerState: cell.__headerState });
                            if (lastRowCell !== null) {
                                lastRowCell.insertAfter(newCell);
                            } else {
                                $insertFirst(row, newCell);
                            }
                        }
                    }
                }
                for (const cell of unmerged) {
                    cell.setColSpan(1);
                    cell.setRowSpan(1);
                }
            }
        });
    }, [editor, hasCellMerge]);

    // Remove cell background color when feature is disabled
    useEffect(() => {
        if (!editor || hasCellBackgroundColor) return;
        return editor.registerNodeTransform<TableCellNode>("TableCell", (node) => {
            if (node.getBackgroundColor() !== null) {
                node.setBackgroundColor(null);
            }
        });
    }, [editor, hasCellBackgroundColor, hasCellMerge]);

    return null;
}

const $insertFirst = (parent: ElementNode, node: LexicalNode): void => {
    const firstChild = parent.getFirstChild();
    if (firstChild !== null) {
        firstChild.insertBefore(node);
    } else {
        parent.append(node);
    }
};

export const $getNodeTriplet = (
    source: PointType | LexicalNode | TableCellNode,
): [TableCellNode, TableRowNode, TableNode] => {
    let cell: TableCellNode;
    if ($isNode("TableCell", source as LexicalNode)) {
        cell = source as TableCellNode;
    }
    // Check if it's a Lexical by checking for a `getType` method 
    else if (Object.prototype.hasOwnProperty.call(source, "getType")) {
        const cell_ = $findMatchingParent(source as LexicalNode, (node): node is TableCellNode => $isNode("TableCell", node));
        if (!$isNode("TableCell", cell_)) {
            throw new Error("Expected to find a parent TableCellNode");
        }
        cell = cell_;
    } else {
        const cell_ = $findMatchingParent((source as PointType).getNode(), (node): node is TableCellNode => $isNode("TableCell", node));
        if (!$isNode("TableCell", cell_)) {
            throw new Error("Expected to find a parent TableCellNode");
        }
        cell = cell_;
    }
    const row = getParent(cell);
    if (!$isNode("TableRow", row)) {
        throw new Error("Expected TableCellNode to have a parent TableRowNode");
    }
    const grid = getParent(row);
    if (!$isNode("Table", grid)) {
        throw new Error("Expected TableRowNode to have a parent GridNode");
    }
    return [cell, row, grid];
};

export class TableObserver {
    focusX: number;
    focusY: number;
    listenersToRemove: Set<() => void>;
    table: TableDOMTable;
    isHighlightingCells: boolean;
    anchorX: number;
    anchorY: number;
    tableNodeKey: NodeKey;
    anchorCell: TableDOMCell | null;
    focusCell: TableDOMCell | null;
    anchorCellNodeKey: NodeKey | null;
    focusCellNodeKey: NodeKey | null;
    editor: LexicalEditor;
    tableSelection: TableSelection | null;
    hasHijackedSelectionStyles: boolean;
    isSelecting: boolean;

    constructor(editor: LexicalEditor, tableNodeKey: string) {
        this.isHighlightingCells = false;
        this.anchorX = -1;
        this.anchorY = -1;
        this.focusX = -1;
        this.focusY = -1;
        this.listenersToRemove = new Set();
        this.tableNodeKey = tableNodeKey;
        this.editor = editor;
        this.table = {
            columns: 0,
            domRows: [],
            rows: 0,
        };
        this.tableSelection = null;
        this.anchorCellNodeKey = null;
        this.focusCellNodeKey = null;
        this.anchorCell = null;
        this.focusCell = null;
        this.hasHijackedSelectionStyles = false;
        this.trackTable();
        this.isSelecting = false;
    }

    getTable(): TableDOMTable {
        return this.table;
    }

    removeListeners() {
        Array.from(this.listenersToRemove).forEach((removeListener) =>
            removeListener(),
        );
    }

    trackTable() {
        const observer = new MutationObserver((records) => {
            this.editor.update(() => {
                let gridNeedsRedraw = false;

                for (let i = 0; i < records.length; i++) {
                    const record = records[i];
                    const target = record.target;
                    const nodeName = target.nodeName;

                    if (nodeName === "TABLE" || nodeName === "TR") {
                        gridNeedsRedraw = true;
                        break;
                    }
                }

                if (!gridNeedsRedraw) {
                    return;
                }

                const tableElement = this.editor.getElementByKey(this.tableNodeKey);

                if (!tableElement) {
                    throw new Error("Expected to find TableElement in DOM");
                }

                this.table = getTable(tableElement);
            });
        });
        this.editor.update(() => {
            const tableElement = this.editor.getElementByKey(this.tableNodeKey);

            if (!tableElement) {
                throw new Error("Expected to find TableElement in DOM");
            }

            this.table = getTable(tableElement);
            observer.observe(tableElement, {
                childList: true,
                subtree: true,
            });
        });
    }

    clearHighlight() {
        const editor = this.editor;
        this.isHighlightingCells = false;
        this.anchorX = -1;
        this.anchorY = -1;
        this.focusX = -1;
        this.focusY = -1;
        this.tableSelection = null;
        this.anchorCellNodeKey = null;
        this.focusCellNodeKey = null;
        this.anchorCell = null;
        this.focusCell = null;
        this.hasHijackedSelectionStyles = false;

        this.enableHighlightStyle();

        editor.update(() => {
            const tableNode = $getNodeByKey(this.tableNodeKey);

            if (!$isNode("Table", tableNode)) {
                throw new Error("Expected TableNode.");
            }

            const tableElement = editor.getElementByKey(this.tableNodeKey);

            if (!tableElement) {
                throw new Error("Expected to find TableElement in DOM");
            }

            const grid = getTable(tableElement);
            $updateDOMForSelection(editor, grid, null);
            $setSelection(null);
            editor.dispatchCommand(SELECTION_CHANGE_COMMAND, undefined);
        });
    }

    enableHighlightStyle() {
        const editor = this.editor;
        editor.update(() => {
            const tableElement = editor.getElementByKey(this.tableNodeKey);

            if (!tableElement) {
                throw new Error("Expected to find TableElement in DOM");
            }

            tableElement.classList.remove("disable-selection");
            this.hasHijackedSelectionStyles = false;
        });
    }

    disableHighlightStyle() {
        const editor = this.editor;
        editor.update(() => {
            const tableElement = editor.getElementByKey(this.tableNodeKey);

            if (!tableElement) {
                throw new Error("Expected to find TableElement in DOM");
            }

            this.hasHijackedSelectionStyles = true;
        });
    }

    updateTableTableSelection(selection: TableSelection | null): void {
        if (selection !== null && selection.tableKey === this.tableNodeKey) {
            const editor = this.editor;
            this.tableSelection = selection;
            this.isHighlightingCells = true;
            this.disableHighlightStyle();
            $updateDOMForSelection(editor, this.table, this.tableSelection);
        } else if (selection === null) {
            this.clearHighlight();
        } else {
            this.tableNodeKey = selection.tableKey;
            this.updateTableTableSelection(selection);
        }
    }

    setFocusCellForSelection(cell: TableDOMCell, ignoreStart = false) {
        const editor = this.editor;
        editor.update(() => {
            const tableNode = $getNodeByKey(this.tableNodeKey);

            if (!$isNode("Table", tableNode)) {
                throw new Error("Expected TableNode.");
            }

            const tableElement = editor.getElementByKey(this.tableNodeKey);

            if (!tableElement) {
                throw new Error("Expected to find TableElement in DOM");
            }

            const cellX = cell.x;
            const cellY = cell.y;
            this.focusCell = cell;

            if (this.anchorCell !== null) {
                const domSelection = getDOMSelection(editor._window);
                // Collapse the selection
                if (domSelection) {
                    domSelection.setBaseAndExtent(
                        this.anchorCell.elem,
                        0,
                        this.focusCell.elem,
                        0,
                    );
                }
            }

            if (
                !this.isHighlightingCells &&
                (this.anchorX !== cellX || this.anchorY !== cellY || ignoreStart)
            ) {
                this.isHighlightingCells = true;
                this.disableHighlightStyle();
            } else if (cellX === this.focusX && cellY === this.focusY) {
                return;
            }

            this.focusX = cellX;
            this.focusY = cellY;

            if (this.isHighlightingCells) {
                const focusTableCellNode = $getNearestNodeFromDOMNode(cell.elem);

                if (
                    this.tableSelection !== null &&
                    this.anchorCellNodeKey !== null &&
                    $isNode("TableCell", focusTableCellNode) &&
                    tableNode.__key !== $findTableNode(focusTableCellNode)?.__key
                ) {
                    const focusNodeKey = focusTableCellNode.__key;

                    this.tableSelection =
                        this.tableSelection.clone() || $createTableSelection();

                    this.focusCellNodeKey = focusNodeKey;
                    this.tableSelection.set(
                        this.tableNodeKey,
                        this.anchorCellNodeKey,
                        this.focusCellNodeKey,
                    );

                    $setSelection(this.tableSelection);

                    editor.dispatchCommand(SELECTION_CHANGE_COMMAND, undefined);

                    $updateDOMForSelection(editor, this.table, this.tableSelection);
                }
            }
        });
    }

    setAnchorCellForSelection(cell: TableDOMCell) {
        this.isHighlightingCells = false;
        this.anchorCell = cell;
        this.anchorX = cell.x;
        this.anchorY = cell.y;

        this.editor.update(() => {
            const anchorTableCellNode = $getNearestNodeFromDOMNode(cell.elem);

            if ($isNode("TableCell", anchorTableCellNode)) {
                const anchorNodeKey = anchorTableCellNode.__key;
                this.tableSelection =
                    this.tableSelection !== null
                        ? this.tableSelection.clone()
                        : $createTableSelection();
                this.anchorCellNodeKey = anchorNodeKey;
            }
        });
    }

    formatCells(type: TextFormatType) {
        this.editor.update(() => {
            const selection = $getSelection();

            if (!$isTableSelection(selection)) {
                throw new Error("Expected grid selection");
            }

            const formatSelection = $createRangeSelection();

            const anchor = formatSelection.anchor;
            const focus = formatSelection.focus;

            selection.getNodes().forEach((cellNode) => {
                if ($isNode("TableCell", cellNode) && cellNode.getTextContentSize() !== 0) {
                    anchor.set(cellNode.__key, 0, "element");
                    focus.set(cellNode.__key, cellNode.getChildrenSize(), "element");
                    formatSelection.formatText(type);
                }
            });

            $setSelection(selection);

            this.editor.dispatchCommand(SELECTION_CHANGE_COMMAND, undefined);
        });
    }

    clearText() {
        const editor = this.editor;
        editor.update(() => {
            const tableNode = $getNodeByKey(this.tableNodeKey);

            if (!$isNode("Table", tableNode)) {
                throw new Error("Expected TableNode.");
            }

            const selection = $getSelection();

            if (!$isTableSelection(selection)) {
                throw new Error("Expected grid selection");
            }

            const selectedNodes = selection.getNodes().filter((node): node is TableCellNode => $isNode("TableCell", node));

            if (selectedNodes.length === this.table.columns * this.table.rows) {
                tableNode.selectPrevious();
                // Delete entire table
                tableNode.remove();
                const rootNode = $getRoot();
                rootNode.selectStart();
                return;
            }

            selectedNodes.forEach((cellNode) => {
                if ($isNode("Element", cellNode)) {
                    const paragraphNode = $createNode("Paragraph", {});
                    const textNode = $createNode("Text", { text: "" });
                    paragraphNode.append(textNode);
                    cellNode.append(paragraphNode);
                    cellNode.getChildren().forEach((child) => {
                        if (child !== paragraphNode) {
                            child.remove();
                        }
                    });
                }
            });

            $updateDOMForSelection(editor, this.table, null);

            $setSelection(null);

            editor.dispatchCommand(SELECTION_CHANGE_COMMAND, undefined);
        });
    }
}

export class TableSelection implements BaseSelection {
    tableKey: NodeKey;
    anchor: PointType;
    focus: PointType;
    _cachedNodes: Array<LexicalNode> | null;
    dirty: boolean;

    constructor(tableKey: NodeKey, anchor: PointType, focus: PointType) {
        this.anchor = anchor;
        this.focus = focus;
        anchor._selection = this;
        focus._selection = this;
        this._cachedNodes = null;
        this.dirty = false;
        this.tableKey = tableKey;
    }

    getStartEndPoints(): [PointType, PointType] {
        return [this.anchor, this.focus];
    }

    /**
     * Returns whether the Selection is "backwards", meaning the focus
     * logically precedes the anchor in the EditorState.
     * @returns true if the Selection is backwards, false otherwise.
     */
    isBackward(): boolean {
        return this.focus.isBefore(this.anchor);
    }

    getCachedNodes(): LexicalNode[] | null {
        return this._cachedNodes;
    }

    setCachedNodes(nodes: LexicalNode[] | null): void {
        this._cachedNodes = nodes;
    }

    is(selection: null | BaseSelection): boolean {
        if (!$isTableSelection(selection)) {
            return false;
        }
        return (
            this.tableKey === selection.tableKey &&
            this.anchor.is(selection.anchor) &&
            this.focus.is(selection.focus)
        );
    }

    set(tableKey: NodeKey, anchorCellKey: NodeKey, focusCellKey: NodeKey): void {
        this.dirty = true;
        this.tableKey = tableKey;
        this.anchor.key = anchorCellKey;
        this.focus.key = focusCellKey;
        this._cachedNodes = null;
    }

    clone(): TableSelection {
        return new TableSelection(this.tableKey, this.anchor, this.focus);
    }

    isCollapsed(): boolean {
        return false;
    }

    extract(): Array<LexicalNode> {
        return this.getNodes();
    }

    insertRawText(text: string): void {
        // Do nothing?
    }

    insertText(): void {
        // Do nothing?
    }

    insertNodes(nodes: Array<LexicalNode>) {
        const focusNode = this.focus.getNode();
        if (!$isNode("Element", focusNode)) {
            throw new Error("Expected TableSelection focus to be an ElementNode");
        }
        const selection = $normalizeSelection(
            focusNode.select(0, focusNode.getChildrenSize()),
        );
        selection.insertNodes(nodes);
    }

    // TODO Deprecate this method. It's confusing when used with colspan|rowspan
    getShape(): TableSelectionShape {
        const anchorCellNode = $getNodeByKey(this.anchor.key);
        if (!$isNode("TableCell", anchorCellNode)) {
            throw new Error("Expected TableSelection anchor to be (or a child of) TableCellNode");
        }
        const anchorCellNodeRect = $getTableCellNodeRect(anchorCellNode);
        if (anchorCellNodeRect === null) {
            throw new Error("getCellRect: expected to find AnchorNode");
        }

        const focusCellNode = $getNodeByKey(this.focus.key);
        if (!$isNode("TableCell", focusCellNode)) {
            throw new Error("Expected TableSelection focus to be (or a child of) TableCellNode");
        }
        const focusCellNodeRect = $getTableCellNodeRect(focusCellNode);
        if (focusCellNodeRect === null) {
            throw new Error("getCellRect: expected to find FocusNode");
        }

        const startX = Math.min(
            anchorCellNodeRect.columnIndex,
            focusCellNodeRect.columnIndex,
        );
        const stopX = Math.max(
            anchorCellNodeRect.columnIndex,
            focusCellNodeRect.columnIndex,
        );

        const startY = Math.min(
            anchorCellNodeRect.rowIndex,
            focusCellNodeRect.rowIndex,
        );
        const stopY = Math.max(
            anchorCellNodeRect.rowIndex,
            focusCellNodeRect.rowIndex,
        );

        return {
            fromX: Math.min(startX, stopX),
            fromY: Math.min(startY, stopY),
            toX: Math.max(startX, stopX),
            toY: Math.max(startY, stopY),
        };
    }

    getNodes(): Array<LexicalNode> {
        const cachedNodes = this._cachedNodes;
        if (cachedNodes !== null) {
            return cachedNodes;
        }

        const anchorNode = this.anchor.getNode();
        const focusNode = this.focus.getNode();
        const anchorCell = $findMatchingParent(anchorNode, (node): node is TableCellNode => $isNode("TableCell", node));
        // todo replace with triplet
        const focusCell = $findMatchingParent(focusNode, (node): node is TableCellNode => $isNode("TableCell", node));
        if (!$isNode("TableCell", anchorCell)) {
            throw new Error("Expected TableSelection anchor to be (or a child of) TableCellNode");
        }
        if (!$isNode("TableCell", focusCell)) {
            throw new Error("Expected TableSelection focus to be (or a child of) TableCellNode");
        }
        const anchorRow = getParent(anchorCell);
        if (!$isNode("TableRow", anchorRow)) {
            throw new Error("Expected anchorCell to have a parent TableRowNode");
        }
        const tableNode = getParent(anchorRow);
        if (!$isNode("Table", tableNode)) {
            throw new Error("Expected anchorRow to have a parent TableNode");
        }

        const focusCellGrid = getParents(focusCell)[1];
        if (focusCellGrid !== tableNode) {
            if (!tableNode.isParentOf(focusCell)) {
                // focus is on higher Grid level than anchor
                const gridParent = getParent(tableNode);
                if (gridParent === null) {
                    throw new Error("Expected gridParent to have a parent");
                }
                this.set(this.tableKey, gridParent.__key, focusCell.__key);
            } else {
                // anchor is on higher Grid level than focus
                const focusCellParent = getParent(focusCellGrid);
                if (focusCellParent === null) {
                    throw new Error("Expected focusCellParent to have a parent");
                }
                this.set(this.tableKey, focusCell.__key, focusCellParent.__key);
            }
            return this.getNodes();
        }

        // TODO Mapping the whole Grid every time not efficient. We need to compute the entire state only
        // once (on load) and iterate on it as updates occur. However, to do this we need to have the
        // ability to store a state. Killing TableSelection and moving the logic to the plugin would make
        // this possible.
        const [map, cellAMap, cellBMap] = $computeTableMap(
            tableNode,
            anchorCell,
            focusCell,
        );

        let minColumn = Math.min(cellAMap.startColumn, cellBMap.startColumn);
        let minRow = Math.min(cellAMap.startRow, cellBMap.startRow);
        let maxColumn = Math.max(
            cellAMap.startColumn + cellAMap.cell.__colSpan - 1,
            cellBMap.startColumn + cellBMap.cell.__colSpan - 1,
        );
        let maxRow = Math.max(
            cellAMap.startRow + cellAMap.cell.__rowSpan - 1,
            cellBMap.startRow + cellBMap.cell.__rowSpan - 1,
        );
        let exploredMinColumn = minColumn;
        let exploredMinRow = minRow;
        let exploredMaxColumn = minColumn;
        let exploredMaxRow = minRow;
        function expandBoundary(mapValue: TableMapValueType): void {
            const {
                cell,
                startColumn: cellStartColumn,
                startRow: cellStartRow,
            } = mapValue;
            minColumn = Math.min(minColumn, cellStartColumn);
            minRow = Math.min(minRow, cellStartRow);
            maxColumn = Math.max(maxColumn, cellStartColumn + cell.__colSpan - 1);
            maxRow = Math.max(maxRow, cellStartRow + cell.__rowSpan - 1);
        }
        while (
            minColumn < exploredMinColumn ||
            minRow < exploredMinRow ||
            maxColumn > exploredMaxColumn ||
            maxRow > exploredMaxRow
        ) {
            if (minColumn < exploredMinColumn) {
                // Expand on the left
                const rowDiff = exploredMaxRow - exploredMinRow;
                const previousColumn = exploredMinColumn - 1;
                for (let i = 0; i <= rowDiff; i++) {
                    expandBoundary(map[exploredMinRow + i][previousColumn]);
                }
                exploredMinColumn = previousColumn;
            }
            if (minRow < exploredMinRow) {
                // Expand on top
                const columnDiff = exploredMaxColumn - exploredMinColumn;
                const previousRow = exploredMinRow - 1;
                for (let i = 0; i <= columnDiff; i++) {
                    expandBoundary(map[previousRow][exploredMinColumn + i]);
                }
                exploredMinRow = previousRow;
            }
            if (maxColumn > exploredMaxColumn) {
                // Expand on the right
                const rowDiff = exploredMaxRow - exploredMinRow;
                const nextColumn = exploredMaxColumn + 1;
                for (let i = 0; i <= rowDiff; i++) {
                    expandBoundary(map[exploredMinRow + i][nextColumn]);
                }
                exploredMaxColumn = nextColumn;
            }
            if (maxRow > exploredMaxRow) {
                // Expand on the bottom
                const columnDiff = exploredMaxColumn - exploredMinColumn;
                const nextRow = exploredMaxRow + 1;
                for (let i = 0; i <= columnDiff; i++) {
                    expandBoundary(map[nextRow][exploredMinColumn + i]);
                }
                exploredMaxRow = nextRow;
            }
        }

        const nodes: Array<LexicalNode> = [tableNode];
        let lastRow: TableRowNode | null = null;
        for (let i = minRow; i <= maxRow; i++) {
            for (let j = minColumn; j <= maxColumn; j++) {
                const { cell } = map[i][j];
                const currentRow = getParent(cell);
                if (!$isNode("TableRow", currentRow)) {
                    throw new Error("Expected TableCellNode parent to be a TableRowNode");
                }
                if (currentRow !== lastRow) {
                    nodes.push(currentRow);
                }
                nodes.push(cell, ...$getChildrenRecursively(cell));
                lastRow = currentRow;
            }
        }

        if (!isCurrentlyReadOnlyMode()) {
            this._cachedNodes = nodes;
        }
        return nodes;
    }

    getMarkdownContent() {
        //TODO need way to generate markdown table
        return this.getTextContent();
    }

    getTextContent() {
        return this.getNodes().map((node) => node.getTextContent()).join("");
    }

    getTextContentSize() {
        return this.getNodes().reduce((acc, node) => acc + node.getTextContentSize(), 0);
    }
}

export function $isTableSelection(x: unknown): x is TableSelection {
    return x instanceof TableSelection;
}

export function $createTableSelection(): TableSelection {
    const anchor = $createPoint("root", 0, "element");
    const focus = $createPoint("root", 0, "element");
    return new TableSelection("root", anchor, focus);
}

export function $getChildrenRecursively(node: LexicalNode): Array<LexicalNode> {
    const nodes: LexicalNode[] = [];
    const stack = [node];
    while (stack.length > 0) {
        const currentNode = stack.pop();
        if (currentNode === undefined) {
            throw new Error("Stack.length > 0; can't be undefined");
        }
        if ($isNode("Element", currentNode)) {
            stack.unshift(...currentNode.getChildren());
        }
        if (currentNode !== node) {
            nodes.push(currentNode);
        }
    }
    return nodes;
}

export const applyTableHandlers = (
    tableNode: TableNode,
    tableElement: CustomDomElement<HTMLTableElement>,
    editor: LexicalEditor,
    hasTabHandler: boolean,
): TableObserver => {
    const rootElement = editor.getRootElement();

    if (rootElement === null) {
        throw new Error("No root element.");
    }

    const tableObserver = new TableObserver(editor, tableNode.__key);
    const editorWindow = editor._window || window;

    // Attach table observer to table element
    tableElement.__lexicalTableSelection = tableObserver;

    const createMouseHandlers = () => {
        const onMouseUp = () => {
            tableObserver.isSelecting = false;
            editorWindow.removeEventListener("mouseup", onMouseUp);
            editorWindow.removeEventListener("mousemove", onMouseMove);
        };

        const onMouseMove = (moveEvent: MouseEvent) => {
            const focusCell = getDOMCellFromTarget(moveEvent.target as Node);
            if (
                focusCell !== null &&
                (tableObserver.anchorX !== focusCell.x ||
                    tableObserver.anchorY !== focusCell.y)
            ) {
                moveEvent.preventDefault();
                tableObserver.setFocusCellForSelection(focusCell);
            }
        };
        return { onMouseMove, onMouseUp };
    };

    tableElement.addEventListener("mousedown", (event: MouseEvent) => {
        setTimeout(() => {
            if (event.button !== 0) {
                return;
            }

            if (!editorWindow) {
                return;
            }

            const anchorCell = getDOMCellFromTarget(event.target as Node);
            if (anchorCell !== null) {
                stopEvent(event);
                tableObserver.setAnchorCellForSelection(anchorCell);
            }

            const { onMouseUp, onMouseMove } = createMouseHandlers();
            tableObserver.isSelecting = true;
            editorWindow.addEventListener("mouseup", onMouseUp);
            editorWindow.addEventListener("mousemove", onMouseMove);
        }, 0);
    });

    // Clear selection when clicking outside of dom.
    const mouseDownCallback = (event: MouseEvent) => {
        if (event.button !== 0) {
            return;
        }

        editor.update(() => {
            const selection = $getSelection();
            const target = event.target as Node;
            if (
                $isTableSelection(selection) &&
                selection.tableKey === tableObserver.tableNodeKey &&
                rootElement.contains(target)
            ) {
                tableObserver.clearHighlight();
            }
        });
    };

    editorWindow.addEventListener("mousedown", mouseDownCallback);

    tableObserver.listenersToRemove.add(() =>
        editorWindow.removeEventListener("mousedown", mouseDownCallback),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_DOWN_COMMAND,
            (event) =>
                $handleArrowKey(editor, event, "down", tableNode, tableObserver),
            COMMAND_PRIORITY_HIGH,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_UP_COMMAND,
            (event) => $handleArrowKey(editor, event, "up", tableNode, tableObserver),
            COMMAND_PRIORITY_HIGH,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_LEFT_COMMAND,
            (event) =>
                $handleArrowKey(editor, event, "backward", tableNode, tableObserver),
            COMMAND_PRIORITY_HIGH,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_ARROW_RIGHT_COMMAND,
            (event) =>
                $handleArrowKey(editor, event, "forward", tableNode, tableObserver),
            COMMAND_PRIORITY_HIGH,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_ESCAPE_COMMAND,
            (event) => {
                const selection = $getSelection();
                if ($isTableSelection(selection)) {
                    const focusCellNode = $findMatchingParent(
                        selection.focus.getNode(),
                        (node): node is TableCellNode => $isNode("TableCell", node),
                    );
                    if ($isNode("TableCell", focusCellNode)) {
                        stopEvent(event);
                        focusCellNode.selectEnd();
                        return true;
                    }
                }

                return false;
            },
            COMMAND_PRIORITY_HIGH,
        ),
    );

    const deleteTextHandler = (command: LexicalCommand<boolean>) => () => {
        const selection = $getSelection();

        if (!$isSelectionInTable(selection, tableNode)) {
            return false;
        }

        if ($isTableSelection(selection)) {
            tableObserver.clearText();

            return true;
        } else if ($isRangeSelection(selection)) {
            const tableCellNode = $findMatchingParent(
                selection.anchor.getNode(),
                (n) => $isNode("TableCell", n),
            );

            if (!$isNode("TableCell", tableCellNode)) {
                return false;
            }

            const anchorNode = selection.anchor.getNode();
            const focusNode = selection.focus.getNode();
            const isAnchorInside = tableNode.isParentOf(anchorNode);
            const isFocusInside = tableNode.isParentOf(focusNode);

            const selectionContainsPartialTable =
                (isAnchorInside && !isFocusInside) ||
                (isFocusInside && !isAnchorInside);

            if (selectionContainsPartialTable) {
                tableObserver.clearText();
                return true;
            }

            const nearestElementNode = $findMatchingParent(
                selection.anchor.getNode(),
                (n) => $isNode("Element", n),
            );

            const topLevelCellElementNode =
                nearestElementNode &&
                $findMatchingParent(
                    nearestElementNode,
                    (n) => $isNode("Element", n) && $isNode("TableCell", getParent(n)),
                );

            if (
                !$isNode("Element", topLevelCellElementNode) ||
                !$isNode("Element", nearestElementNode)
            ) {
                return false;
            }

            if (
                command === DELETE_LINE_COMMAND &&
                getPreviousSibling(topLevelCellElementNode) === null
            ) {
                // TODO: Fix Delete Line in Table Cells.
                return true;
            }

            if (
                command === DELETE_CHARACTER_COMMAND ||
                command === DELETE_WORD_COMMAND
            ) {
                if (selection.isCollapsed() && selection.anchor.offset === 0) {
                    if (nearestElementNode !== topLevelCellElementNode) {
                        const children = nearestElementNode.getChildren();
                        const newParagraphNode = $createNode("Paragraph", {});
                        children.forEach((child) => newParagraphNode.append(child));
                        nearestElementNode.replace(newParagraphNode);
                        nearestElementNode.getWritable().__parent = tableCellNode.__key;
                        return true;
                    }
                }
            }
        }

        return false;
    };

    [DELETE_WORD_COMMAND, DELETE_LINE_COMMAND, DELETE_CHARACTER_COMMAND].forEach(
        (command) => {
            tableObserver.listenersToRemove.add(
                editor.registerCommand(
                    command,
                    deleteTextHandler(command),
                    COMMAND_PRIORITY_CRITICAL,
                ),
            );
        },
    );

    const deleteCellHandler = (event: KeyboardEvent): boolean => {
        const selection = $getSelection();

        if (!$isSelectionInTable(selection, tableNode)) {
            return false;
        }

        if ($isTableSelection(selection)) {
            event.preventDefault();
            event.stopPropagation();
            tableObserver.clearText();

            return true;
        } else if ($isRangeSelection(selection)) {
            const tableCellNode = $findMatchingParent(
                selection.anchor.getNode(),
                (n) => $isNode("TableCell", n),
            );

            if (!$isNode("TableCell", tableCellNode)) {
                return false;
            }
        }

        return false;
    };

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_BACKSPACE_COMMAND,
            deleteCellHandler,
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<KeyboardEvent>(
            KEY_DELETE_COMMAND,
            deleteCellHandler,
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<TextFormatType>(
            FORMAT_TEXT_COMMAND,
            (payload) => {
                const selection = $getSelection();

                if (!$isSelectionInTable(selection, tableNode)) {
                    return false;
                }

                if ($isTableSelection(selection)) {
                    tableObserver.formatCells(payload);

                    return true;
                } else if ($isRangeSelection(selection)) {
                    const tableCellNode = $findMatchingParent(
                        selection.anchor.getNode(),
                        (n) => $isNode("TableCell", n),
                    );

                    if (!$isNode("TableCell", tableCellNode)) {
                        return false;
                    }
                }

                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand<ElementFormatType>(
            FORMAT_ELEMENT_COMMAND,
            (formatType) => {
                const selection = $getSelection();
                if (
                    !$isTableSelection(selection) ||
                    !$isSelectionInTable(selection, tableNode)
                ) {
                    return false;
                }

                const anchorNode = selection.anchor.getNode();
                const focusNode = selection.focus.getNode();
                if (!$isNode("TableCell", anchorNode) || !$isNode("TableCell", focusNode)) {
                    return false;
                }

                const [tableMap, anchorCell, focusCell] = $computeTableMap(
                    tableNode,
                    anchorNode,
                    focusNode,
                );
                const maxRow = Math.max(anchorCell.startRow, focusCell.startRow);
                const maxColumn = Math.max(
                    anchorCell.startColumn,
                    focusCell.startColumn,
                );
                const minRow = Math.min(anchorCell.startRow, focusCell.startRow);
                const minColumn = Math.min(
                    anchorCell.startColumn,
                    focusCell.startColumn,
                );
                for (let i = minRow; i <= maxRow; i++) {
                    for (let j = minColumn; j <= maxColumn; j++) {
                        const cell = tableMap[i][j].cell;
                        cell.setFormat(formatType);

                        const cellChildren = cell.getChildren();
                        for (let k = 0; k < cellChildren.length; k++) {
                            const child = cellChildren[k];
                            if ($isNode("Element", child) && !child.isInline()) {
                                child.setFormat(formatType);
                            }
                        }
                    }
                }
                return true;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand(
            CONTROLLED_TEXT_INSERTION_COMMAND,
            (payload) => {
                const selection = $getSelection();

                if (!$isSelectionInTable(selection, tableNode)) {
                    return false;
                }

                if ($isTableSelection(selection)) {
                    tableObserver.clearHighlight();

                    return false;
                } else if ($isRangeSelection(selection)) {
                    const tableCellNode = $findMatchingParent(
                        selection.anchor.getNode(),
                        (n) => $isNode("TableCell", n),
                    );

                    if (!$isNode("TableCell", tableCellNode)) {
                        return false;
                    }

                    if (typeof payload === "string") {
                        const edgePosition = $getTableEdgeCursorPosition(
                            editor,
                            selection,
                            tableNode,
                        );
                        if (edgePosition) {
                            $insertParagraphAtTableEdge(edgePosition, tableNode, [
                                $createNode("Text", { text: payload }),
                            ]);
                            return true;
                        }
                    }
                }

                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    if (hasTabHandler) {
        tableObserver.listenersToRemove.add(
            editor.registerCommand<KeyboardEvent>(
                KEY_TAB_COMMAND,
                (event) => {
                    const selection = $getSelection();
                    if (
                        !$isRangeSelection(selection) ||
                        !selection.isCollapsed() ||
                        !$isSelectionInTable(selection, tableNode)
                    ) {
                        return false;
                    }

                    const tableCellNode = $findCellNode(selection.anchor.getNode());
                    if (tableCellNode === null) {
                        return false;
                    }

                    stopEvent(event);

                    const currentCords = tableNode.getCordsFromCellNode(
                        tableCellNode,
                        tableObserver.table,
                    );

                    selectTableNodeInDirection(
                        tableObserver,
                        tableNode,
                        currentCords.x,
                        currentCords.y,
                        !event.shiftKey ? "forward" : "backward",
                    );

                    return true;
                },
                COMMAND_PRIORITY_CRITICAL,
            ),
        );
    }

    tableObserver.listenersToRemove.add(
        editor.registerCommand(
            FOCUS_COMMAND,
            (payload) => {
                return isSelected(tableNode);
            },
            COMMAND_PRIORITY_HIGH,
        ),
    );

    function getObserverCellFromCellNode(
        tableCellNode: TableCellNode,
    ): TableDOMCell {
        const currentCords = tableNode.getCordsFromCellNode(
            tableCellNode,
            tableObserver.table,
        );
        return tableNode.getDOMCellFromCordsOrThrow(
            currentCords.x,
            currentCords.y,
            tableObserver.table,
        );
    }

    tableObserver.listenersToRemove.add(
        editor.registerCommand(
            SELECTION_INSERT_CLIPBOARD_NODES_COMMAND,
            (selectionPayload) => {
                const { nodes, selection } = selectionPayload;
                const anchorAndFocus = selection.getStartEndPoints();
                const isTableSelection = $isTableSelection(selection);
                const isRangeSelection = $isRangeSelection(selection);
                const isSelectionInsideOfGrid =
                    (isRangeSelection &&
                        $findMatchingParent(selection.anchor.getNode(), (n) =>
                            $isNode("TableCell", n),
                        ) !== null &&
                        $findMatchingParent(selection.focus.getNode(), (n) =>
                            $isNode("TableCell", n),
                        ) !== null) ||
                    isTableSelection;

                if (
                    nodes.length !== 1 ||
                    !$isNode("Table", nodes[0]) ||
                    !isSelectionInsideOfGrid ||
                    anchorAndFocus === null
                ) {
                    return false;
                }
                const [anchor] = anchorAndFocus;

                const newGrid = nodes[0];
                const newGridRows = newGrid.getChildren();
                const newColumnCount = newGrid
                    .getFirstChildOrThrow<TableNode>()
                    .getChildrenSize();
                const newRowCount = newGrid.getChildrenSize();
                const gridCellNode = $findMatchingParent(anchor.getNode(), (n) =>
                    $isNode("TableCell", n),
                );
                const gridRowNode =
                    gridCellNode &&
                    $findMatchingParent(gridCellNode, (n) => $isNode("TableRow", n));
                const gridNode =
                    gridRowNode &&
                    $findMatchingParent(gridRowNode, (n) => $isNode("Table", n));

                if (
                    !$isNode("TableCell", gridCellNode) ||
                    !$isNode("TableRow", gridRowNode) ||
                    !$isNode("Table", gridNode)
                ) {
                    return false;
                }

                const startY = getIndexWithinParent(gridRowNode);
                const stopY = Math.min(
                    gridNode.getChildrenSize() - 1,
                    startY + newRowCount - 1,
                );
                const startX = getIndexWithinParent(gridCellNode);
                const stopX = Math.min(
                    gridRowNode.getChildrenSize() - 1,
                    startX + newColumnCount - 1,
                );
                const fromX = Math.min(startX, stopX);
                const fromY = Math.min(startY, stopY);
                const toX = Math.max(startX, stopX);
                const toY = Math.max(startY, stopY);
                const gridRowNodes = gridNode.getChildren();
                let newRowIdx = 0;
                let newAnchorCellKey;
                let newFocusCellKey;

                for (let r = fromY; r <= toY; r++) {
                    const currentGridRowNode = gridRowNodes[r];

                    if (!$isNode("TableRow", currentGridRowNode)) {
                        return false;
                    }

                    const newGridRowNode = newGridRows[newRowIdx];

                    if (!$isNode("TableRow", newGridRowNode)) {
                        return false;
                    }

                    const gridCellNodes = currentGridRowNode.getChildren();
                    const newGridCellNodes = newGridRowNode.getChildren();
                    let newColumnIdx = 0;

                    for (let c = fromX; c <= toX; c++) {
                        const currentGridCellNode = gridCellNodes[c];

                        if (!$isNode("TableCell", currentGridCellNode)) {
                            return false;
                        }

                        const newGridCellNode = newGridCellNodes[newColumnIdx];

                        if (!$isNode("TableCell", newGridCellNode)) {
                            return false;
                        }

                        if (r === fromY && c === fromX) {
                            newAnchorCellKey = currentGridCellNode.__key;
                        } else if (r === toY && c === toX) {
                            newFocusCellKey = currentGridCellNode.__key;
                        }

                        const originalChildren = currentGridCellNode.getChildren();
                        newGridCellNode.getChildren().forEach((child) => {
                            if ($isNode("Text", child)) {
                                const paragraphNode = $createNode("Paragraph", {});
                                paragraphNode.append(child);
                                currentGridCellNode.append(child);
                            } else {
                                currentGridCellNode.append(child);
                            }
                        });
                        originalChildren.forEach((n) => n.remove());
                        newColumnIdx++;
                    }

                    newRowIdx++;
                }
                if (newAnchorCellKey && newFocusCellKey) {
                    const newTableSelection = $createTableSelection();
                    newTableSelection.set(
                        nodes[0].__key,
                        newAnchorCellKey,
                        newFocusCellKey,
                    );
                    $setSelection(newTableSelection);
                }
                return true;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand(
            SELECTION_CHANGE_COMMAND,
            () => {
                const selection = $getSelection();
                const prevSelection = $getPreviousSelection();

                if ($isRangeSelection(selection)) {
                    const { anchor, focus } = selection;
                    const anchorNode = anchor.getNode();
                    const focusNode = focus.getNode();
                    // Using explicit comparison with table node to ensure it's not a nested table
                    // as in that case we'll leave selection resolving to that table
                    const anchorCellNode = $findCellNode(anchorNode);
                    const focusCellNode = $findCellNode(focusNode);
                    const isAnchorInside = !!(
                        anchorCellNode && tableNode.__key !== $findTableNode(anchorCellNode)?.__key
                    );
                    const isFocusInside = !!(
                        focusCellNode && tableNode.__key !== $findTableNode(focusCellNode)?.__key
                    );
                    const isPartialyWithinTable = isAnchorInside !== isFocusInside;
                    const isWithinTable = isAnchorInside && isFocusInside;
                    const isBackward = selection.isBackward();

                    if (isPartialyWithinTable) {
                        const newSelection = selection.clone();
                        if (isFocusInside) {
                            newSelection.focus.set(
                                getParent(tableNode, { throwIfNull: true }).__key,
                                isBackward
                                    ? getIndexWithinParent(tableNode)
                                    : getIndexWithinParent(tableNode) + 1,
                                "element",
                            );
                        } else {
                            newSelection.anchor.set(
                                getParent(tableNode, { throwIfNull: true }).__key,
                                isBackward
                                    ? getIndexWithinParent(tableNode) + 1
                                    : getIndexWithinParent(tableNode),
                                "element",
                            );
                        }
                        $setSelection(newSelection);
                        $addHighlightStyleToTable(editor, tableObserver);
                    } else if (isWithinTable) {
                        // Handle case when selection spans across multiple cells but still
                        // has range selection, then we convert it into grid selection
                        if (anchorCellNode.__key !== focusCellNode.__key) {
                            tableObserver.setAnchorCellForSelection(
                                getObserverCellFromCellNode(anchorCellNode),
                            );
                            tableObserver.setFocusCellForSelection(
                                getObserverCellFromCellNode(focusCellNode),
                                true,
                            );
                            if (!tableObserver.isSelecting) {
                                setTimeout(() => {
                                    const { onMouseUp, onMouseMove } = createMouseHandlers();
                                    tableObserver.isSelecting = true;
                                    editorWindow.addEventListener("mouseup", onMouseUp);
                                    editorWindow.addEventListener("mousemove", onMouseMove);
                                }, 0);
                            }
                        }
                    }
                } else if (
                    selection &&
                    $isTableSelection(selection) &&
                    selection.is(prevSelection) &&
                    selection.tableKey === tableNode.__key
                ) {
                    // if selection goes outside of the table we need to change it to Range selection
                    const domSelection = getDOMSelection(editor._window);
                    if (
                        domSelection &&
                        domSelection.anchorNode &&
                        domSelection.focusNode
                    ) {
                        const focusNode = $getNearestNodeFromDOMNode(
                            domSelection.focusNode,
                        );
                        const isFocusOutside =
                            focusNode && !tableNode.is($findTableNode(focusNode));

                        const anchorNode = $getNearestNodeFromDOMNode(
                            domSelection.anchorNode,
                        );
                        const isAnchorInside =
                            anchorNode && tableNode.is($findTableNode(anchorNode));

                        if (
                            isFocusOutside &&
                            isAnchorInside &&
                            domSelection.rangeCount > 0
                        ) {
                            const newSelection = $createRangeSelectionFromDom(
                                domSelection,
                                editor,
                            );
                            if (newSelection) {
                                newSelection.anchor.set(
                                    tableNode.__key,
                                    selection.isBackward() ? tableNode.getChildrenSize() : 0,
                                    "element",
                                );
                                domSelection.removeAllRanges();
                                $setSelection(newSelection);
                            }
                        }
                    }
                }

                if (
                    selection &&
                    !selection.is(prevSelection) &&
                    ($isTableSelection(selection) || $isTableSelection(prevSelection)) &&
                    tableObserver.tableSelection &&
                    !tableObserver.tableSelection.is(prevSelection)
                ) {
                    if (
                        $isTableSelection(selection) &&
                        selection.tableKey === tableObserver.tableNodeKey
                    ) {
                        tableObserver.updateTableTableSelection(selection);
                    } else if (
                        !$isTableSelection(selection) &&
                        $isTableSelection(prevSelection) &&
                        prevSelection.tableKey === tableObserver.tableNodeKey
                    ) {
                        tableObserver.updateTableTableSelection(null);
                    }
                    return false;
                }

                if (
                    tableObserver.hasHijackedSelectionStyles &&
                    !isSelected(tableNode)
                ) {
                    $removeHighlightStyleToTable(editor, tableObserver);
                } else if (
                    !tableObserver.hasHijackedSelectionStyles &&
                    isSelected(tableNode)
                ) {
                    $addHighlightStyleToTable(editor, tableObserver);
                }

                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    tableObserver.listenersToRemove.add(
        editor.registerCommand(
            INSERT_PARAGRAPH_COMMAND,
            () => {
                const selection = $getSelection();
                if (
                    !$isRangeSelection(selection) ||
                    !selection.isCollapsed() ||
                    !$isSelectionInTable(selection, tableNode)
                ) {
                    return false;
                }
                const edgePosition = $getTableEdgeCursorPosition(
                    editor,
                    selection,
                    tableNode,
                );
                if (edgePosition) {
                    $insertParagraphAtTableEdge(edgePosition, tableNode);
                    return true;
                }
                return false;
            },
            COMMAND_PRIORITY_CRITICAL,
        ),
    );

    return tableObserver;
};

function $findCellNode(node: LexicalNode): null | TableCellNode {
    const cellNode = $findMatchingParent(node, (node): node is TableCellNode => $isNode("TableCell", node));
    return $isNode("TableCell", cellNode) ? cellNode : null;
}

export function $findTableNode(node: LexicalNode): null | TableNode {
    const tableNode = $findMatchingParent(node, (node): node is TableNode => $isNode("Table", node));
    return $isNode("Table", tableNode) ? tableNode : null;
}

function $isSelectionInTable(
    selection: null | BaseSelection,
    tableNode: TableNode,
): boolean {
    if ($isRangeSelection(selection) || $isTableSelection(selection)) {
        const isAnchorInside = tableNode.isParentOf(selection.anchor.getNode());
        const isFocusInside = tableNode.isParentOf(selection.focus.getNode());

        return isAnchorInside && isFocusInside;
    }

    return false;
}

function $insertParagraphAtTableEdge(
    edgePosition: "first" | "last",
    tableNode: TableNode,
    children?: LexicalNode[],
) {
    const paragraphNode = $createNode("Paragraph", {});
    if (edgePosition === "first") {
        tableNode.insertBefore(paragraphNode);
    } else {
        tableNode.insertAfter(paragraphNode);
    }
    paragraphNode.append(...(children || []));
    paragraphNode.selectEnd();
}

function $getTableEdgeCursorPosition(
    editor: LexicalEditor,
    selection: RangeSelection,
    tableNode: TableNode,
) {
    // TODO: Add support for nested tables
    const domSelection = window.getSelection();
    if (!domSelection || domSelection.anchorNode !== editor.getRootElement()) {
        return undefined;
    }

    const anchorCellNode = $findMatchingParent(selection.anchor.getNode(), (n) =>
        $isNode("TableCell", n),
    ) as TableCellNode | null;
    if (!anchorCellNode) {
        return undefined;
    }

    const parentTable = $findMatchingParent(anchorCellNode, (n) =>
        $isNode("Table", n),
    );
    if (!$isNode("Table", parentTable) || !parentTable.is(tableNode)) {
        return undefined;
    }

    const [tableMap, cellValue] = $computeTableMap(
        tableNode,
        anchorCellNode,
        anchorCellNode,
    );
    const firstCell = tableMap[0][0];
    const lastCell = tableMap[tableMap.length - 1][tableMap[0].length - 1];
    const { startRow, startColumn } = cellValue;

    const isAtFirstCell =
        startRow === firstCell.startRow && startColumn === firstCell.startColumn;
    const isAtLastCell =
        startRow === lastCell.startRow && startColumn === lastCell.startColumn;

    if (isAtFirstCell) {
        return "first";
    } else if (isAtLastCell) {
        return "last";
    } else {
        return undefined;
    }
}

export function $addHighlightStyleToTable(
    editor: LexicalEditor,
    tableSelection: TableObserver,
) {
    tableSelection.disableHighlightStyle();
    $forEachTableCell(tableSelection.table, (cell) => {
        cell.highlighted = true;
        $addHighlightToDOM(editor, cell);
    });
}

export function $removeHighlightStyleToTable(
    editor: LexicalEditor,
    tableObserver: TableObserver,
) {
    tableObserver.enableHighlightStyle();
    $forEachTableCell(tableObserver.table, (cell) => {
        const elem = cell.elem;
        cell.highlighted = false;
        $removeHighlightFromDOM(editor, cell);

        if (!elem.getAttribute("style")) {
            elem.removeAttribute("style");
        }
    });
}

export function $forEachTableCell(
    grid: TableDOMTable,
    cb: (
        cell: TableDOMCell,
        lexicalNode: LexicalNode,
        cords: {
            x: number;
            y: number;
        },
    ) => void,
) {
    const { domRows } = grid;

    for (let y = 0; y < domRows.length; y++) {
        const row = domRows[y];
        if (!row) {
            continue;
        }

        for (let x = 0; x < row.length; x++) {
            const cell = row[x];
            if (!cell) {
                continue;
            }
            const lexicalNode = $getNearestNodeFromDOMNode(cell.elem);

            if (lexicalNode !== null) {
                cb(cell, lexicalNode, {
                    x,
                    y,
                });
            }
        }
    }
}

const BROWSER_BLUE_RGB = "172,206,247";
function $addHighlightToDOM(editor: LexicalEditor, cell: TableDOMCell): void {
    const element = cell.elem;
    const node = $getNearestNodeFromDOMNode(element);
    if (!$isNode("TableCell", node)) {
        throw new Error("Expected to find LexicalNode from Table Cell DOMNode");
    }
    const backgroundColor = node.getBackgroundColor();
    if (backgroundColor === null) {
        element.style.setProperty("background-color", `rgb(${BROWSER_BLUE_RGB})`);
    } else {
        element.style.setProperty(
            "background-image",
            `linear-gradient(to right, rgba(${BROWSER_BLUE_RGB},0.85), rgba(${BROWSER_BLUE_RGB},0.85))`,
        );
    }
    element.style.setProperty("caret-color", "transparent");
}

function $removeHighlightFromDOM(
    editor: LexicalEditor,
    cell: TableDOMCell,
): void {
    const element = cell.elem;
    const node = $getNearestNodeFromDOMNode(element);
    if (!$isNode("TableCell", node)) {
        throw new Error("Expected to find LexicalNode from Table Cell DOMNode");
    }
    const backgroundColor = node.getBackgroundColor();
    if (backgroundColor === null) {
        element.style.removeProperty("background-color");
    }
    element.style.removeProperty("background-image");
    element.style.removeProperty("caret-color");
}

type Direction = "backward" | "forward" | "up" | "down";

const selectTableNodeInDirection = (
    tableObserver: TableObserver,
    tableNode: TableNode,
    x: number,
    y: number,
    direction: Direction,
): boolean => {
    const isForward = direction === "forward";

    switch (direction) {
        case "backward":
        case "forward":
            if (x !== (isForward ? tableObserver.table.columns - 1 : 0)) {
                selectTableCellNode(
                    tableNode.getCellNodeFromCordsOrThrow(
                        x + (isForward ? 1 : -1),
                        y,
                        tableObserver.table,
                    ),
                    isForward,
                );
            } else {
                if (y !== (isForward ? tableObserver.table.rows - 1 : 0)) {
                    selectTableCellNode(
                        tableNode.getCellNodeFromCordsOrThrow(
                            isForward ? 0 : tableObserver.table.columns - 1,
                            y + (isForward ? 1 : -1),
                            tableObserver.table,
                        ),
                        isForward,
                    );
                } else if (!isForward) {
                    tableNode.selectPrevious();
                } else {
                    tableNode.selectNext();
                }
            }

            return true;

        case "up":
            if (y !== 0) {
                selectTableCellNode(
                    tableNode.getCellNodeFromCordsOrThrow(x, y - 1, tableObserver.table),
                    false,
                );
            } else {
                tableNode.selectPrevious();
            }

            return true;

        case "down":
            if (y !== tableObserver.table.rows - 1) {
                selectTableCellNode(
                    tableNode.getCellNodeFromCordsOrThrow(x, y + 1, tableObserver.table),
                    true,
                );
            } else {
                tableNode.selectNext();
            }

            return true;
        default:
            return false;
    }
};

function selectTableCellNode(tableCell: TableCellNode, fromStart: boolean) {
    if (fromStart) {
        tableCell.selectStart();
    } else {
        tableCell.selectEnd();
    }
}

const adjustFocusNodeInDirection = (
    tableObserver: TableObserver,
    tableNode: TableNode,
    x: number,
    y: number,
    direction: Direction,
): boolean => {
    const isForward = direction === "forward";

    switch (direction) {
        case "backward":
        case "forward":
            if (x !== (isForward ? tableObserver.table.columns - 1 : 0)) {
                tableObserver.setFocusCellForSelection(
                    tableNode.getDOMCellFromCordsOrThrow(
                        x + (isForward ? 1 : -1),
                        y,
                        tableObserver.table,
                    ),
                );
            }

            return true;
        case "up":
            if (y !== 0) {
                tableObserver.setFocusCellForSelection(
                    tableNode.getDOMCellFromCordsOrThrow(x, y - 1, tableObserver.table),
                );

                return true;
            } else {
                return false;
            }
        case "down":
            if (y !== tableObserver.table.rows - 1) {
                tableObserver.setFocusCellForSelection(
                    tableNode.getDOMCellFromCordsOrThrow(x, y + 1, tableObserver.table),
                );

                return true;
            } else {
                return false;
            }
        default:
            return false;
    }
};

function $handleArrowKey(
    editor: LexicalEditor,
    event: KeyboardEvent,
    direction: Direction,
    tableNode: TableNode,
    tableObserver: TableObserver,
): boolean {
    const selection = $getSelection();

    if (!$isSelectionInTable(selection, tableNode)) {
        if (
            direction === "backward" &&
            $isRangeSelection(selection) &&
            selection.isCollapsed()
        ) {
            const anchorType = selection.anchor.type;
            const anchorOffset = selection.anchor.offset;
            if (
                anchorType !== "element" &&
                !(anchorType === "text" && anchorOffset === 0)
            ) {
                return false;
            }
            const anchorNode = selection.anchor.getNode();
            if (!anchorNode) {
                return false;
            }
            const parentNode = $findMatchingParent(
                anchorNode,
                (n) => $isNode("Element", n) && !n.isInline(),
            );
            if (!parentNode) {
                return false;
            }
            const siblingNode = getPreviousSibling(parentNode);
            if (!siblingNode || !$isNode("Table", siblingNode)) {
                return false;
            }
            stopEvent(event);
            siblingNode.selectEnd();
            return true;
        }
        return false;
    }

    if ($isRangeSelection(selection) && selection.isCollapsed()) {
        const { anchor, focus } = selection;
        const anchorCellNode = $findMatchingParent(
            anchor.getNode(),
            (node): node is TableCellNode => $isNode("TableCell", node),
        );
        const focusCellNode = $findMatchingParent(
            focus.getNode(),
            (node): node is TableCellNode => $isNode("TableCell", node),
        );
        if (
            !$isNode("TableCell", anchorCellNode) ||
            !focusCellNode ||
            anchorCellNode.__key !== focusCellNode.__key
        ) {
            return false;
        }
        const anchorCellTable = $findTableNode(anchorCellNode);
        if (anchorCellTable !== tableNode && anchorCellTable !== null) {
            const anchorCellTableElement = editor.getElementByKey(
                anchorCellTable.__key,
            );
            if (anchorCellTableElement !== null) {
                tableObserver.table = getTable(anchorCellTableElement);
                return $handleArrowKey(
                    editor,
                    event,
                    direction,
                    anchorCellTable,
                    tableObserver,
                );
            }
        }

        if (direction === "backward" || direction === "forward") {
            const anchorType = anchor.type;
            const anchorOffset = anchor.offset;
            const anchorNode = anchor.getNode();
            if (!anchorNode) {
                return false;
            }

            if (
                isExitingTableAnchor(anchorType, anchorOffset, anchorNode, direction)
            ) {
                return $handleTableExit(event, anchorNode, tableNode, direction);
            }

            return false;
        }

        const anchorCellDom = editor.getElementByKey(anchorCellNode.__key);
        const anchorDOM = editor.getElementByKey(anchor.key);
        if (anchorDOM === null || anchorCellDom === null) {
            return false;
        }

        let edgeSelectionRect;
        if (anchor.type === "element") {
            edgeSelectionRect = anchorDOM.getBoundingClientRect();
        } else {
            const domSelection = window.getSelection();
            if (domSelection === null || domSelection.rangeCount === 0) {
                return false;
            }

            const range = domSelection.getRangeAt(0);
            edgeSelectionRect = range.getBoundingClientRect();
        }

        const edgeChild =
            direction === "up"
                ? anchorCellNode.getFirstChild()
                : anchorCellNode.getLastChild();
        if (edgeChild === null) {
            return false;
        }

        const edgeChildDOM = editor.getElementByKey(edgeChild.__key);

        if (edgeChildDOM === null) {
            return false;
        }

        const edgeRect = edgeChildDOM.getBoundingClientRect();
        const isExiting =
            direction === "up"
                ? edgeRect.top > edgeSelectionRect.top - edgeSelectionRect.height
                : edgeSelectionRect.bottom + edgeSelectionRect.height > edgeRect.bottom;

        if (isExiting) {
            stopEvent(event);

            const cords = tableNode.getCordsFromCellNode(
                anchorCellNode,
                tableObserver.table,
            );

            if (event.shiftKey) {
                const cell = tableNode.getDOMCellFromCordsOrThrow(
                    cords.x,
                    cords.y,
                    tableObserver.table,
                );
                tableObserver.setAnchorCellForSelection(cell);
                tableObserver.setFocusCellForSelection(cell, true);
            } else {
                return selectTableNodeInDirection(
                    tableObserver,
                    tableNode,
                    cords.x,
                    cords.y,
                    direction,
                );
            }

            return true;
        }
    } else if ($isTableSelection(selection)) {
        const { anchor, focus } = selection;
        const anchorCellNode = $findMatchingParent(
            anchor.getNode(),
            (node): node is TableCellNode => $isNode("TableCell", node),
        );
        const focusCellNode = $findMatchingParent(
            focus.getNode(),
            (node): node is TableCellNode => $isNode("TableCell", node),
        );

        const [tableNodeFromSelection] = selection.getNodes();
        const tableElement = editor.getElementByKey(
            tableNodeFromSelection.__key,
        );
        if (
            !$isNode("TableCell", anchorCellNode) ||
            !$isNode("TableCell", focusCellNode) ||
            !$isNode("Table", tableNodeFromSelection) ||
            tableElement === null
        ) {
            return false;
        }
        tableObserver.updateTableTableSelection(selection);

        const grid = getTable(tableElement);
        const cordsAnchor = tableNode.getCordsFromCellNode(anchorCellNode, grid);
        const anchorCell = tableNode.getDOMCellFromCordsOrThrow(
            cordsAnchor.x,
            cordsAnchor.y,
            grid,
        );
        tableObserver.setAnchorCellForSelection(anchorCell);

        stopEvent(event);

        if (event.shiftKey) {
            const cords = tableNode.getCordsFromCellNode(focusCellNode, grid);
            return adjustFocusNodeInDirection(
                tableObserver,
                tableNodeFromSelection,
                cords.x,
                cords.y,
                direction,
            );
        } else {
            focusCellNode.selectEnd();
        }

        return true;
    }

    return false;
}

function stopEvent(event: Event) {
    event.preventDefault();
    event.stopImmediatePropagation();
    event.stopPropagation();
}

function isExitingTableAnchor(
    type: string,
    offset: number,
    anchorNode: LexicalNode,
    direction: "backward" | "forward",
) {
    return (
        isExitingTableElementAnchor(type, anchorNode, direction) ||
        isExitingTableTextAnchor(type, offset, anchorNode, direction)
    );
}

function $handleTableExit(
    event: KeyboardEvent,
    anchorNode: LexicalNode,
    tableNode: TableNode,
    direction: "backward" | "forward",
) {
    const anchorCellNode = $findMatchingParent(anchorNode, (node): node is TableCellNode => $isNode("TableCell", node));
    if (!$isNode("TableCell", anchorCellNode)) {
        return false;
    }
    const [tableMap, cellValue] = $computeTableMap(
        tableNode,
        anchorCellNode,
        anchorCellNode,
    );
    if (!isExitingCell(tableMap, cellValue, direction)) {
        return false;
    }

    const toNode = getExitingToNode(anchorNode, direction, tableNode);
    if (!toNode || $isNode("Table", toNode)) {
        return false;
    }

    stopEvent(event);
    if (direction === "backward") {
        toNode.selectEnd();
    } else {
        toNode.selectStart();
    }
    return true;
}

function isExitingTableElementAnchor(
    type: string,
    anchorNode: LexicalNode,
    direction: "backward" | "forward",
) {
    return (
        type === "element" &&
        (direction === "backward"
            ? getPreviousSibling(anchorNode) === null
            : getNextSibling(anchorNode) === null)
    );
}

function isExitingTableTextAnchor(
    type: string,
    offset: number,
    anchorNode: LexicalNode,
    direction: "backward" | "forward",
) {
    const parentNode = $findMatchingParent(
        anchorNode,
        (n) => $isNode("Element", n) && !n.isInline(),
    );
    if (!parentNode) {
        return false;
    }
    const hasValidOffset =
        direction === "backward"
            ? offset === 0
            : offset === anchorNode.getTextContentSize();
    return (
        type === "text" &&
        hasValidOffset &&
        (direction === "backward"
            ? getPreviousSibling(parentNode) === null
            : getNextSibling(parentNode) === null)
    );
}

function getExitingToNode(
    anchorNode: LexicalNode,
    direction: "backward" | "forward",
    tableNode: TableNode,
) {
    const parentNode = $findMatchingParent(
        anchorNode,
        (n) => $isNode("Element", n) && !n.isInline(),
    );
    if (!parentNode) {
        return undefined;
    }
    const anchorSibling =
        direction === "backward"
            ? getPreviousSibling(parentNode)
            : getNextSibling(parentNode);
    return anchorSibling && $isNode("Table", anchorSibling)
        ? anchorSibling
        : direction === "backward"
            ? getPreviousSibling(tableNode)
            : getNextSibling(tableNode);
}

function isExitingCell(
    tableMap: TableMapType,
    cellValue: TableMapValueType,
    direction: "backward" | "forward",
) {
    const firstCell = tableMap[0][0];
    const lastCell = tableMap[tableMap.length - 1][tableMap[0].length - 1];
    const { startColumn, startRow } = cellValue;
    return direction === "backward"
        ? startColumn === firstCell.startColumn && startRow === firstCell.startRow
        : startColumn === lastCell.startColumn && startRow === lastCell.startRow;
}

export function getDOMCellFromTarget(node: Node): TableDOMCell | null {
    let currentNode: ParentNode | Node | null = node;

    while (currentNode !== null) {
        const nodeName = currentNode.nodeName;

        if (nodeName === "TD" || nodeName === "TH") {
            // @ts-expect-error: internal field
            const cell = currentNode._cell;

            if (cell === undefined) {
                return null;
            }

            return cell;
        }

        currentNode = currentNode.parentNode;
    }

    return null;
}

export function $updateDOMForSelection(
    editor: LexicalEditor,
    table: TableDOMTable,
    selection: TableSelection | RangeSelection | null,
) {
    const selectedCellNodes = new Set(selection ? selection.getNodes() : []);
    $forEachTableCell(table, (cell, lexicalNode) => {
        const elem = cell.elem;

        if (selectedCellNodes.has(lexicalNode)) {
            cell.highlighted = true;
            $addHighlightToDOM(editor, cell);
        } else {
            cell.highlighted = false;
            $removeHighlightFromDOM(editor, cell);
            if (!elem.getAttribute("style")) {
                elem.removeAttribute("style");
            }
        }
    });
}

export function $getTableCellNodeRect(tableCellNode: TableCellNode): {
    rowIndex: number;
    columnIndex: number;
    rowSpan: number;
    colSpan: number;
} | null {
    const [cellNode, , gridNode] = $getNodeTriplet(tableCellNode);
    const rows = gridNode.getChildren<TableRowNode>();
    const rowCount = rows.length;
    const columnCount = rows[0].getChildren().length;

    // Create a matrix of the same size as the table to track the position of each cell
    const cellMatrix = new Array(rowCount);
    for (let i = 0; i < rowCount; i++) {
        cellMatrix[i] = new Array(columnCount);
    }

    for (let rowIndex = 0; rowIndex < rowCount; rowIndex++) {
        const row = rows[rowIndex];
        const cells = row.getChildren<TableCellNode>();
        let columnIndex = 0;

        for (let cellIndex = 0; cellIndex < cells.length; cellIndex++) {
            // Find the next available position in the matrix, skip the position of merged cells
            while (cellMatrix[rowIndex][columnIndex]) {
                columnIndex++;
            }

            const cell = cells[cellIndex];
            const rowSpan = cell.__rowSpan || 1;
            const colSpan = cell.__colSpan || 1;

            // Put the cell into the corresponding position in the matrix
            for (let i = 0; i < rowSpan; i++) {
                for (let j = 0; j < colSpan; j++) {
                    cellMatrix[rowIndex + i][columnIndex + j] = cell;
                }
            }

            // Return to the original index, row span and column span of the cell.
            if (cellNode === cell) {
                return {
                    colSpan,
                    columnIndex,
                    rowIndex,
                    rowSpan,
                };
            }

            columnIndex += colSpan;
        }
    }

    return null;
}
