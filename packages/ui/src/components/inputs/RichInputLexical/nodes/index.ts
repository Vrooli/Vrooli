import { DOMConversionCache, LexicalNodeClass, NodeType } from "../types";

/**
 * Maps the node types to the LexicalNode classes and HTML -> LexicalNode conversion functions.
 * Classes which are extended, such as ElementNode and DecoratorNode, are not included here. 
 * This is because we don't want to instantiate them directly, but rather use their subclasses.
 */
export class LexicalNodes {
    private static instance: LexicalNodes = new LexicalNodes();
    private nodes: { [key in NodeType]?: LexicalNodeClass } = {};
    private domConversions: DOMConversionCache = {};
    private isInitialized = false;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static async init() {
        if (LexicalNodes.instance.isInitialized) {
            return;
        }
        await LexicalNodes.loadNodes();
        LexicalNodes.createDOMConversions();
        LexicalNodes.instance.isInitialized = true;
    }

    private static async loadNodes() {
        LexicalNodes.instance.nodes = {
            "Code": (await import("../plugins/CodePlugin")).CodeBlockNode,
            "Decorator": (await import("./DecoratorNode")).DecoratorNode,
            "Element": (await import("./ElementNode")).ElementNode,
            "Hashtag": (await import("./HashtagNode")).HashtagNode,
            "Heading": (await import("./HeadingNode")).HeadingNode,
            "HorizontalRule": (await import("./HorizontalRuleNode")).HorizontalRuleNode,
            "LineBreak": (await import("./LineBreakNode")).LineBreakNode,
            "Link": (await import("./LinkNode")).LinkNode,
            "List": (await import("./ListNode")).ListNode,
            "ListItem": (await import("./ListItemNode")).ListItemNode,
            "Paragraph": (await import("./ParagraphNode")).ParagraphNode,
            "Quote": (await import("./QuoteNode")).QuoteNode,
            "Root": (await import("./RootNode")).RootNode,
            "Tab": (await import("./TabNode")).TabNode,
            "Table": (await import("./TableNode")).TableNode,
            "TableCell": (await import("./TableCellNode")).TableCellNode,
            "TableRow": (await import("./TableRowNode")).TableRowNode,
            "Text": (await import("./TextNode")).TextNode,
        };
    }

    private static createDOMConversions() {
        // Nodes must be loaded before this function is called
        if (Object.keys(LexicalNodes.instance.nodes).length === 0) {
            console.error("Call LexicalNodes.loadNodes() before LexicalNodes.createDOMConversions()");
            return;
        }
        // Iterate over each node
        for (const nodeType in LexicalNodes.instance.nodes) {
            const NodeClass = LexicalNodes.instance.nodes[nodeType as NodeType];
            if (NodeClass) {
                const importDOM = NodeClass.importDOM?.() || {};
                // Iterate over keys returned by the importDOM function of the class
                for (const tag in importDOM) {
                    const convertFunction = importDOM[tag];
                    // Check if this tag already has a conversion function array
                    if (!LexicalNodes.instance.domConversions[tag]) {
                        LexicalNodes.instance.domConversions[tag] = [];
                    }
                    // Push the new conversion function to the array for this tag
                    LexicalNodes.instance.domConversions[tag].push(convertFunction);
                }
            }
        }
    }

    public static isInitialized() {
        return LexicalNodes.instance.isInitialized;
    }

    public static get(nodeType: NodeType): LexicalNodeClass | undefined {
        // Check if nodes is initialized
        if (!LexicalNodes.instance.nodes) {
            console.error("LexicalNodes not initialized");
            return undefined;
        }
        return LexicalNodes.instance.nodes[nodeType];
    }

    public static getAll() {
        return LexicalNodes.instance.nodes;
    }

    public static getConversionsForTag(tag: string): DOMConversionCache[string] {
        return LexicalNodes.instance.domConversions[tag] || [];
    }
}

// Initialize the singleton instance
LexicalNodes.init();
