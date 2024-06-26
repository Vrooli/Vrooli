import { Headers } from "utils/display/stringTools";
import { DECORATOR_NODES, ELEMENT_NODES, TEXT_NODES } from "./consts";
import { type EditorState, type LexicalEditor } from "./editor";
import { type DecoratorNode } from "./nodes/DecoratorNode";
import { type ElementNode } from "./nodes/ElementNode";
import { type HashtagNode } from "./nodes/HashtagNode";
import { type HeadingNode } from "./nodes/HeadingNode";
import { type LexicalNode } from "./nodes/LexicalNode";
import { type LineBreakNode } from "./nodes/LineBreakNode";
import { type LinkNode } from "./nodes/LinkNode";
import { type ListItemNode } from "./nodes/ListItemNode";
import { type ListNode } from "./nodes/ListNode";
import { type ParagraphNode } from "./nodes/ParagraphNode";
import { type QuoteNode } from "./nodes/QuoteNode";
import { type RootNode } from "./nodes/RootNode";
import { type TabNode } from "./nodes/TabNode";
import { type TableCellNode } from "./nodes/TableCellNode";
import { type TableNode } from "./nodes/TableNode";
import { type TableRowNode } from "./nodes/TableRowNode";
import { type TextNode } from "./nodes/TextNode";
import { type CodeBlockNode } from "./plugins/CodePlugin";
import { type TableObserver } from "./plugins/TablePlugin";

export abstract class LexicalNodeBase {
    static clone: (node: any) => any;
    static getType: () => NodeType;
    static importDOM: () => DOMConversionMap;
    static importJSON: (serializedNode: any) => any;
}

/** A generic non-instance class */
export type ObjectClass<T = object> = new (...args: any[]) => T;

/**
 * A non-instance (i.e. didn't use "new") class that defines the type of a LexicalNode.
 */
export type LexicalNodeClass<T extends LexicalNode = LexicalNode> = ObjectClass<T> & Omit<typeof LexicalNodeBase, "new">;

export type RegisteredNode = {
    klass: LexicalNodeClass;
    transforms: Set<Transform<LexicalNode>>;
};
export type RegisteredNodes = Record<string, RegisteredNode>

export type ElementTransformer = {
    export: (node: LexicalNode, traverseChildren: (node: ElementNode) => string) => string | null;
    regExp: RegExp;
    replace: (parentNode: ElementNode, children: LexicalNode[], match: string[], isImport: boolean) => void;
    type: "element";
};
export type TextMatchTransformer = Readonly<{
    export: (node: LexicalNode, exportChildren: (node: ElementNode) => string, exportFormat: (node: TextNode, textContent: string) => string) => string | null;
    importRegExp: RegExp;
    regExp: RegExp;
    replace: (node: TextNode, match: RegExpMatchArray) => void;
    trigger: string;
    type: "text-match";
}>;
export type LexicalTransformer = ElementTransformer | TextFormatTransformer | TextMatchTransformer;

export type NodeConstructors = {
    Code: typeof CodeBlockNode;
    Decorator: typeof DecoratorNode;
    Element: typeof ElementNode;
    Hashtag: typeof HashtagNode;
    Heading: typeof HeadingNode;
    LineBreak: typeof LineBreakNode;
    Link: typeof LinkNode;
    List: typeof ListNode;
    ListItem: typeof ListItemNode;
    Paragraph: typeof ParagraphNode;
    Quote: typeof QuoteNode;
    Root: typeof RootNode;
    Tab: typeof TabNode;
    Table: typeof TableNode;
    TableCell: typeof TableCellNode;
    TableRow: typeof TableRowNode;
    Text: typeof TextNode;
};

export type NodeConstructorBase = { key?: NodeKey };
export type NodeConstructorPayloads = {
    Code: { language?: string } & NodeConstructorBase;
    Decorator: NodeConstructorBase;
    Element: NodeConstructorBase;
    Hashtag: { text: string } & NodeConstructorBase;
    Heading: { tag: Headers } & NodeConstructorBase;
    LineBreak: NodeConstructorBase;
    Link: LinkAttributes & NodeConstructorBase;
    List: { listType: ListType, start?: number } & NodeConstructorBase;
    ListItem: { value?: number, checked?: boolean } & NodeConstructorBase;
    Paragraph: NodeConstructorBase;
    Quote: NodeConstructorBase;
    Root: NodeConstructorBase;
    Tab: NodeConstructorBase;
    Table: NodeConstructorBase;
    TableCell: { colSpan?: number, headerState?: TableCellHeaderState, width?: number } & NodeConstructorBase;
    TableRow: { height?: number } & NodeConstructorBase;
    Text: { text: string } & NodeConstructorBase;
};

export type NodeKey = string;

export type TextPointType = {
    _selection: BaseSelection;
    getNode: () => TextNode;
    is: (point: PointType) => boolean;
    isBefore: (point: PointType) => boolean;
    key: NodeKey;
    offset: number;
    set: (key: NodeKey, offset: number, type: "text" | "element") => void;
    type: "text";
};

export type ElementPointType = {
    _selection: BaseSelection;
    getNode: () => ElementNode;
    is: (point: PointType) => boolean;
    isBefore: (point: PointType) => boolean;
    key: NodeKey;
    offset: number;
    set: (key: NodeKey, offset: number, type: "text" | "element") => void;
    type: "element";
};

export type PointType = TextPointType | ElementPointType;

export interface BaseSelection {
    _cachedNodes: LexicalNode[] | null;
    dirty: boolean;

    clone(): BaseSelection;
    extract(): LexicalNode[];
    getNodes(): LexicalNode[];
    getTextContent(): string;
    insertText(text: string): void;
    insertRawText(text: string): void;
    is(selection: null | BaseSelection): boolean;
    insertNodes(nodes: LexicalNode[]): void;
    getStartEndPoints(): null | [PointType, PointType];
    isCollapsed(): boolean;
    isBackward(): boolean;
    getCachedNodes(): LexicalNode[] | null;
    setCachedNodes(nodes: LexicalNode[] | null): void;
}

export type Spread<T1, T2> = Omit<T2, keyof T1> & T1;

export type ElementFormatType =
    | "left"
    | "start"
    | "center"
    | "right"
    | "end"
    | "justify"
    | "";

export type DOMConversion<T extends HTMLElement = HTMLElement> = {
    conversion: DOMConversionFn<T>;
    priority?: 0 | 1 | 2 | 3 | 4;
};

export type DOMConversionFn<T extends HTMLElement = HTMLElement> = (
    element: T,
) => DOMConversionOutput | null;

export type DOMChildConversion = (
    lexicalNode: LexicalNode,
    parentLexicalNode: LexicalNode | null | undefined,
) => LexicalNode | null | undefined;

export type DOMConversionMap<T extends HTMLElement = HTMLElement> = Record<
    string,
    (element: T) => DOMConversion<T> | null
>;

export type DOMConversionCache = Record<string, ((element: HTMLElement) => DOMConversion | null)[]>;

export type DOMConversionOutput = {
    after?: (childLexicalNodes: LexicalNode[]) => LexicalNode[];
    forChild?: DOMChildConversion;
    node: null | LexicalNode | LexicalNode[];
};

export type DOMExportOutput = {
    after?: (
        generatedElement: HTMLElement | Text | null | undefined,
    ) => HTMLElement | Text | null | undefined;
    element: HTMLElement | Text | null;
};

export type PasteCommandType = ClipboardEvent | InputEvent | KeyboardEvent;

export type RootElementRemoveHandles = (() => void)[];
export type RootElementEvents = Array<
    [
        string,
        Record<string, unknown> | ((event: Event, editor: LexicalEditor) => void),
    ]
>;

export type EditorUpdateOptions = {
    onUpdate?: () => void;
    skipTransforms?: true;
    tag?: string;
    discrete?: true;
};

export type EditorSetOptions = {
    tag?: string;
};

export type EditorFocusOptions = {
    defaultSelection?: "rootStart" | "rootEnd";
};

export type EditorConfig = {
    namespace: string;
};

export type LexicalNodeReplacement = {
    replace: LexicalNodeClass;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    with: <T extends { new(...args: any): any }>(
        node: InstanceType<T>,
    ) => LexicalNode;
    withKlass?: LexicalNodeClass;
};

export type HTMLConfig = {
    export?: Map<
        LexicalNodeClass,
        (editor: LexicalEditor, target: LexicalNode) => DOMExportOutput
    >;
    import?: DOMConversionMap;
};

export type CreateEditorArgs = {
    namespace?: string;
};

export type Transform<T extends LexicalNode> = (node: T) => void;

export type ErrorHandler = (error: Error) => void;

export type MutationListeners = Map<MutationListener, LexicalNodeClass>;

export type MutatedNodes = { [K in NodeType]?: Record<NodeKey, NodeMutation> };

export type NodeMap = Map<NodeKey, LexicalNode>;

export type NodeMutation = "created" | "updated" | "destroyed";

export type UpdateListenerArgs = {
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>;
    dirtyLeaves: Set<NodeKey>;
    editorState: EditorState;
    normalizedNodes: Set<NodeKey>;
    prevEditorState: EditorState;
    tags: Set<string>;
};
export type UpdateListener = (updateArgs: UpdateListenerArgs) => void;

export type DecoratorListenerArgs<T = never> = Record<NodeKey, T>;
export type DecoratorListener<T = never> = (decorator: DecoratorListenerArgs) => void;

export type RootListenerArgs = {
    prevRootElement: null | HTMLElement,
    rootElement: null | HTMLElement,
};
export type RootListener = (rootArgs: RootListenerArgs) => void;

export type TextContentListenerArgs = string;
export type TextContentListener = (text: TextContentListenerArgs) => void;

export type MutationListener = (
    nodes: Record<NodeKey, NodeMutation>,
    payload: {
        dirtyLeaves: Set<string>;
        prevEditorState: EditorState;
        updateTags: Set<string>;
    },
) => void;

export type EditableListenerArgs = boolean;
export type EditableListener = (editable: EditableListenerArgs) => void;

export type EditorListener = {
    /** When the editor's decorator object is updated. The decorator object contains the mapping of node keys to decorators */
    decorator?: DecoratorListener;
    /** When the editor becomes editable or not */
    editable?: EditableListener;
    /** When the root element is registered or updated */
    root?: RootListener;
    /** When the text content of the editor changes */
    textcontent?: TextContentListener;
    /** When the editor state is updated */
    update?: UpdateListener;
} & {
        /** When a specific node type is updated */
        [K in NodeType]?: MutationListener;
    };
export type EditorListeners = { [K in keyof EditorListener]?: Set<NonNullable<EditorListener[K]>> };

export type EditorListenerPayload = {
    decorator: DecoratorListenerArgs;
    editable: EditableListenerArgs;
    root: RootListenerArgs;
    textcontent: TextContentListenerArgs;
    update: UpdateListenerArgs;
}

export type CommandListener<P> = (payload: P, editor: LexicalEditor) => boolean;

export type CommandListenerPriority = 0 | 1 | 2 | 3 | 4;

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export type LexicalCommand<TPayload> = {
    type?: string;
};

/**
 * Type helper for extracting the payload type from a command.
 *
 * @example
 * ```ts
 * const MY_COMMAND = createCommand<SomeType>();
 *
 * // ...
 *
 * editor.registerCommand(MY_COMMAND, payload => {
 *   // Type of `payload` is inferred here. But lets say we want to extract a function to delegate to
 *   handleMyCommand(editor, payload);
 *   return true;
 * });
 *
 * const handleMyCommand = (editor: LexicalEditor, payload: CommandPayloadType<typeof MY_COMMAND>) => {
 *   // `payload` is of type `SomeType`, extracted from the command.
 * }
 * ```
 */
export type CommandPayloadType<TCommand extends LexicalCommand<unknown>> =
    TCommand extends LexicalCommand<infer TPayload> ? TPayload : never;

export type CommandsMap = Map<
    LexicalCommand<unknown>,
    Array<Set<CommandListener<unknown>>>
>;

export type TransformerType = "text" | "decorator" | "element" | "root";

export type IntentionallyMarkedAsDirtyElement = boolean;

export type ShadowRootNode = Spread<
    { isShadowRoot(): true },
    ElementNode
>;

export interface BaseSerializedNode {
    __type: NodeType;
    children?: Array<BaseSerializedNode>;
    version: number;
}

export type SerializedElementNode<
    T extends SerializedLexicalNode = SerializedLexicalNode,
> = Spread<
    {
        children: Array<T>;
        direction: "ltr" | "rtl" | null;
        format: ElementFormatType;
        indent: number;
    },
    SerializedLexicalNode
>;

export type SerializedRootNode<
    T extends SerializedLexicalNode = SerializedLexicalNode,
> = SerializedElementNode<T>;

/**
 * A serialized (i.e. JSON) representation of a LexicalNode, 
 * so that we can reconstruct it later.
 */
export type SerializedLexicalNode = {
    __type: NodeType;
    /**
     * The latest version of the node. If we change the structure of the node, we increment this number
     * to indicate that it's incompatible
     */
    version: number;
};

export interface SerializedEditorState<
    T extends SerializedLexicalNode = SerializedLexicalNode,
> {
    root: SerializedRootNode<T>;
}

export type SerializedEditor = {
    editorState: SerializedEditorState;
};

export type SerializedTextNode = Spread<
    {
        detail: number;
        format: number;
        mode: TextModeType;
        style: string;
        text: string;
    },
    SerializedLexicalNode
>;

export type SerializedLineBreakNode = SerializedLexicalNode;

export type SerializedParagraphNode = Spread<
    {
        textFormat: number;
    },
    SerializedElementNode
>;

export type SerializedTabNode = SerializedTextNode;

export type SerializedHeadingNode = Spread<
    {
        tag: "h1" | "h2" | "h3" | "h4" | "h5" | "h6";
    },
    SerializedElementNode
>;

export const TableCellHeaderStates = {
    BOTH: 3,
    COLUMN: 2,
    NO_STATUS: 0,
    ROW: 1,
};

export type TableCellHeaderState =
    typeof TableCellHeaderStates[keyof typeof TableCellHeaderStates];

export type SerializedTableNode = SerializedElementNode;

export type SerializedTableCellNode = Spread<
    {
        colSpan?: number;
        rowSpan?: number;
        headerState: TableCellHeaderState;
        width?: number;
        backgroundColor?: null | string;
    },
    SerializedElementNode
>;

export type SerializedTableRowNode = Spread<
    {
        height?: number;
    },
    SerializedElementNode
>;

export type SerializedLinkNode = Spread<LinkAttributes, SerializedElementNode>;

export type SerializedAutoLinkNode = SerializedLinkNode;

export type SerializedListNode = Spread<
    {
        listType: ListType;
        start: number;
        tag: ListNodeTagType;
    },
    SerializedElementNode
>;

export type SerializedListItemNode = Spread<
    {
        checked: boolean | undefined;
        value: number;
    },
    SerializedElementNode
>;

export type ListType = "number" | "bullet" | "check";

export type ListNodeTagType = "ul" | "ol";

export type SerializedQuoteNode = SerializedElementNode;

export type SerializedCodeNode = Spread<
    {
        language: string | null | undefined;
    },
    SerializedElementNode
>;

export type SerializedCodeBlockNode = SerializedElementNode & {
    language: string;
};

export type SerializedCodeHighlightNode = Spread<
    {
        highlightType: string | null | undefined;
    },
    SerializedTextNode
>;

export type TextDetailType = "directionless" | "unmergable";

export type TextModeType = "normal" | "token" | "segmented";

export type TextMark = { end: null | number; id: string; start: null | number };

export type TextMarks = Array<TextMark>;

export type TextFormatType =
    | "BOLD"
    | "CODE_BLOCK"
    | "CODE_INLINE"
    | "HEADING"
    | "HIGHLIGHT"
    | "ITALIC"
    | "LIST_ORDERED"
    | "LIST_UNORDERED"
    | "QUOTE"
    | "SPOILER_LINES"
    | "SPOILER_TAGS"
    | "STRIKETHROUGH"
    | "SUBSCRIPT"
    | "SUPERSCRIPT"
    | "UNDERLINE_LINES"
    | "UNDERLINE_TAGS";

export type NodeType =
    // | "AutoLink" TODO
    | "LineBreak"
    | typeof ELEMENT_NODES[number]
    | typeof DECORATOR_NODES[number]
    | typeof TEXT_NODES[number]

export type TextFormatTransformer = Readonly<{
    /** The classes to apply to the DOM element */
    classes: string[];
    /** The DOM tag that represents the format */
    domTag: string;
    /** The type of format */
    format: TextFormatType;
    /**  Start and end tags (respectively) for the text format. */
    tags: string | [string, string];
    type: "text-format";
}>;

export type LinkAttributes = {
    rel?: null | string;
    target?: null | string;
    title?: null | string;
    url: string
};

export type TableSelectionShape = {
    fromX: number;
    fromY: number;
    toX: number;
    toY: number;
};

export type TableMapValueType = {
    cell: TableCellNode;
    startRow: number;
    startColumn: number;
};
export type TableMapType = Array<Array<TableMapValueType>>;

export type TableDOMCell = {
    elem: HTMLElement;
    highlighted: boolean;
    hasBackgroundColor: boolean;
    x: number;
    y: number;
};

export type TableDOMRows = Array<Array<TableDOMCell | undefined> | undefined>;

export type TableDOMTable = {
    domRows: TableDOMRows;
    columns: number;
    rows: number;
};

export type InsertTableCommandPayloadHeaders =
    | Readonly<{
        rows: boolean;
        columns: boolean;
    }>
    | boolean;

export type InsertTableCommandPayload = Readonly<{
    columns: number;
    rows: number;
    includeHeaders?: InsertTableCommandPayloadHeaders;
}>;

export type CustomDomElement<T = HTMLElement> = T & {
    __lexicalEditor?: LexicalEditor | null;
    __lexicalEventHandles?: Array<() => void>;
    __lexicalLineBreak?: HTMLBRElement | null;
    __lexicalListStart?: string;
    __lexicalListType?: ListType;
    __lexicalTableSelection?: TableObserver;
    [Key: `__lexicalKey_${NodeKey}`]: NodeKey;
}

export type CustomLexicalEvent = Event & {
    __lexicalHandled?: boolean;
};
