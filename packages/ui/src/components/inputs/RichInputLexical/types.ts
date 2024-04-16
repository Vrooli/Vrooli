import { LEXICAL_ELEMENT_KEY } from "./consts";
import { type EditorState, type LexicalEditor } from "./editor";
import { type ElementNode } from "./nodes/ElementNode";
import { type LexicalNode } from "./nodes/LexicalNode";
import { type TableCellNode } from "./nodes/TableCellNode";
import { type TextNode } from "./nodes/TextNode";
import { type TableObserver } from "./plugins/TablePlugin";

type GenericConstructor<T> = new (...args: any[]) => T;

// https://github.com/microsoft/TypeScript/issues/3841
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export type KlassConstructor<Cls extends GenericConstructor<any>> =
    GenericConstructor<InstanceType<Cls>> & { [k in keyof Cls]: Cls[k] };

export type Klass<T extends LexicalNode> = InstanceType<T["constructor"]> extends T ? T["constructor"] : GenericConstructor<T> & T["constructor"];

export type ElementTransformer = {
    dependencies: Array<Klass<LexicalNode>>;
    export: (node: LexicalNode, traverseChildren: (node: ElementNode) => string) => string | null;
    regExp: RegExp;
    replace: (parentNode: ElementNode, children: Array<LexicalNode>, match: Array<string>, isImport: boolean) => void;
    type: "element";
};
export type TextMatchTransformer = Readonly<{
    dependencies: Array<Klass<LexicalNode>>;
    export: (node: LexicalNode, exportChildren: (node: ElementNode) => string, exportFormat: (node: TextNode, textContent: string) => string) => string | null;
    importRegExp: RegExp;
    regExp: RegExp;
    replace: (node: TextNode, match: RegExpMatchArray) => void;
    trigger: string;
    type: "text-match";
}>;
export type LexicalTransformer = ElementTransformer | TextFormatTransformer | TextMatchTransformer;

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
    _cachedNodes: Array<LexicalNode> | null;
    dirty: boolean;

    clone(): BaseSelection;
    extract(): Array<LexicalNode>;
    getNodes(): Array<LexicalNode>;
    getTextContent(): string;
    insertText(text: string): void;
    insertRawText(text: string): void;
    is(selection: null | BaseSelection): boolean;
    insertNodes(nodes: Array<LexicalNode>): void;
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
    NodeName,
    (node: T) => DOMConversion<T> | null
>;
type NodeName = string;

export type DOMConversionOutput = {
    after?: (childLexicalNodes: Array<LexicalNode>) => Array<LexicalNode>;
    forChild?: DOMChildConversion;
    node: null | LexicalNode | Array<LexicalNode>;
};

export type DOMExportOutput = {
    after?: (
        generatedElement: HTMLElement | Text | null | undefined,
    ) => HTMLElement | Text | null | undefined;
    element: HTMLElement | Text | null;
};

export type PasteCommandType = ClipboardEvent | InputEvent | KeyboardEvent;

export type RootElementRemoveHandles = Array<() => void>;
export type RootElementEvents = Array<
    [
        string,
        Record<string, unknown> | ((event: Event, editor: LexicalEditor) => void),
    ]
>;

export type EditorThemeClassName = string;

export type TextNodeThemeClasses = {
    base?: EditorThemeClassName;
    bold?: EditorThemeClassName;
    code?: EditorThemeClassName;
    highlight?: EditorThemeClassName;
    italic?: EditorThemeClassName;
    strikethrough?: EditorThemeClassName;
    subscript?: EditorThemeClassName;
    superscript?: EditorThemeClassName;
    underline?: EditorThemeClassName;
    underlineStrikethrough?: EditorThemeClassName;
    [key: string]: EditorThemeClassName | undefined;
};

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

export type EditorThemeClasses = {
    blockCursor?: EditorThemeClassName;
    characterLimit?: EditorThemeClassName;
    code?: EditorThemeClassName;
    codeHighlight?: Record<string, EditorThemeClassName>;
    hashtag?: EditorThemeClassName;
    heading?: {
        h1?: EditorThemeClassName;
        h2?: EditorThemeClassName;
        h3?: EditorThemeClassName;
        h4?: EditorThemeClassName;
        h5?: EditorThemeClassName;
        h6?: EditorThemeClassName;
    };
    image?: EditorThemeClassName;
    link?: EditorThemeClassName;
    list?: {
        ul?: EditorThemeClassName;
        ulDepth?: Array<EditorThemeClassName>;
        ol?: EditorThemeClassName;
        olDepth?: Array<EditorThemeClassName>;
        checklist?: EditorThemeClassName;
        listitem?: EditorThemeClassName;
        listitemChecked?: EditorThemeClassName;
        listitemUnchecked?: EditorThemeClassName;
        nested?: {
            list?: EditorThemeClassName;
            listitem?: EditorThemeClassName;
        };
    };
    ltr?: EditorThemeClassName;
    mark?: EditorThemeClassName;
    markOverlap?: EditorThemeClassName;
    paragraph?: EditorThemeClassName;
    quote?: EditorThemeClassName;
    root?: EditorThemeClassName;
    rtl?: EditorThemeClassName;
    table?: EditorThemeClassName;
    tableAddColumns?: EditorThemeClassName;
    tableAddRows?: EditorThemeClassName;
    tableCellActionButton?: EditorThemeClassName;
    tableCellActionButtonContainer?: EditorThemeClassName;
    tableCellPrimarySelected?: EditorThemeClassName;
    tableCellSelected?: EditorThemeClassName;
    tableCell?: EditorThemeClassName;
    tableCellEditing?: EditorThemeClassName;
    tableCellHeader?: EditorThemeClassName;
    tableCellResizer?: EditorThemeClassName;
    tableCellSortedIndicator?: EditorThemeClassName;
    tableResizeRuler?: EditorThemeClassName;
    tableRow?: EditorThemeClassName;
    tableSelected?: EditorThemeClassName;
    text?: TextNodeThemeClasses;
    embedBlock?: {
        base?: EditorThemeClassName;
        focus?: EditorThemeClassName;
    };
    indent?: EditorThemeClassName;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    [key: string]: any;
};

export type EditorConfig = {
    disableEvents?: boolean;
    namespace: string;
    theme: EditorThemeClasses;
};

export type InitialEditorStateType =
    | null
    | string
    | EditorState
    | ((editor: LexicalEditor) => void);

export type LexicalNodeReplacement = {
    replace: Klass<LexicalNode>;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    with: <T extends { new(...args: any): any }>(
        node: InstanceType<T>,
    ) => LexicalNode;
    withKlass?: Klass<LexicalNode>;
};

export type HTMLConfig = {
    export?: Map<
        Klass<LexicalNode>,
        (editor: LexicalEditor, target: LexicalNode) => DOMExportOutput
    >;
    import?: DOMConversionMap;
};

export type CreateEditorArgs = {
    disableEvents?: boolean;
    editorState?: EditorState;
    namespace?: string;
    nodes?: ReadonlyArray<Klass<LexicalNode> | LexicalNodeReplacement>;
    onError?: ErrorHandler;
    parentEditor?: LexicalEditor;
    editable?: boolean;
    theme?: EditorThemeClasses;
    html?: HTMLConfig;
};

export type RegisteredNodes = Map<string, RegisteredNode>;

export type RegisteredNode = {
    klass: Klass<LexicalNode>;
    transforms: Set<Transform<LexicalNode>>;
    replace: null | ((node: LexicalNode) => LexicalNode);
    replaceWithKlass: null | Klass<LexicalNode>;
    exportDOM?: (
        editor: LexicalEditor,
        targetNode: LexicalNode,
    ) => DOMExportOutput;
};

export type Transform<T extends LexicalNode> = (node: T) => void;

export type ErrorHandler = (error: Error) => void;

export type MutationListeners = Map<MutationListener, Klass<LexicalNode>>;

export type MutatedNodes = Map<Klass<LexicalNode>, Map<NodeKey, NodeMutation>>;

export type NodeMap = Map<NodeKey, LexicalNode>;

export type NodeMutation = "created" | "updated" | "destroyed";

export type UpdateListener = (arg0: {
    dirtyElements: Map<NodeKey, IntentionallyMarkedAsDirtyElement>;
    dirtyLeaves: Set<NodeKey>;
    editorState: EditorState;
    normalizedNodes: Set<NodeKey>;
    prevEditorState: EditorState;
    tags: Set<string>;
}) => void;

export type DecoratorListener<T = never> = (
    decorator: Record<NodeKey, T>,
) => void;

export type RootListener = (
    rootElement: null | HTMLElement,
    prevRootElement: null | HTMLElement,
) => void;

export type TextContentListener = (text: string) => void;

export type MutationListener = (
    nodes: Map<NodeKey, NodeMutation>,
    payload: {
        updateTags: Set<string>;
        dirtyLeaves: Set<string>;
        prevEditorState: EditorState;
    },
) => void;

export type CommandListener<P> = (payload: P, editor: LexicalEditor) => boolean;

export type EditableListener = (editable: boolean) => void;

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
export type Listeners = {
    decorator: Set<DecoratorListener>;
    mutation: MutationListeners;
    editable: Set<EditableListener>;
    root: Set<RootListener>;
    textcontent: Set<TextContentListener>;
    update: Set<UpdateListener>;
};

export type Listener =
    | DecoratorListener
    | EditableListener
    | MutationListener
    | RootListener
    | TextContentListener
    | UpdateListener;

export type ListenerType =
    | "update"
    | "root"
    | "decorator"
    | "textcontent"
    | "mutation"
    | "editable";

export type TransformerType = "text" | "decorator" | "element" | "root";

export type HeadingTagType = "h1" | "h2" | "h3" | "h4" | "h5" | "h6";

export type IntentionallyMarkedAsDirtyElement = boolean;

export type ShadowRootNode = Spread<
    { isShadowRoot(): true },
    ElementNode
>;

export type DOMConversionCache = Map<
    string,
    Array<(node: Node) => DOMConversion | null>
>;

export interface BaseSerializedNode {
    children?: Array<BaseSerializedNode>;
    type: string;
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

export type SerializedLexicalNode = {
    type: string;
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

export type SerializedLinkNode = Spread<
    {
        url: string;
    },
    Spread<LinkAttributes, SerializedElementNode>
>;
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

export type SerializedHorizontalRuleNode = SerializedLexicalNode;

export type ListType = "number" | "bullet" | "check";

export type ListNodeTagType = "ul" | "ol";

export type SerializedQuoteNode = SerializedElementNode;

export type SerializedCodeNode = Spread<
    {
        language: string | null | undefined;
    },
    SerializedElementNode
>;

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

export type TextFormatTransformer = Readonly<{
    /**
     * The type of format
     */
    format: TextFormatType;
    /** 
     * Start and end tags (respectively) for the text format.
     */
    tags: string | [string, string];
    type: "text-format";
}>;

export type LinkAttributes = {
    rel?: null | string;
    target?: null | string;
    title?: null | string;
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

export type ObjectKlass<T> = new (...args: any[]) => T;

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

export type HTMLTableElementWithWithTableSelectionState = HTMLTableElement &
    Record<typeof LEXICAL_ELEMENT_KEY, TableObserver>;

