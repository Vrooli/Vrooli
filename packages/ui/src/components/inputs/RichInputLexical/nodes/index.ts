import { LexicalNodeClass, NodeType } from "../types";

/**
 * Maps the node types to the LexicalNode classes. 
 * Classes which are extended, such as ElementNode and DecoratorNode, are not included here. 
 * This is because we don't want to instantiate them directly, but rather use their subclasses.
 */
export class LexicalNodes {
    private static instance: LexicalNodes;
    private nodes: { [key in NodeType]?: LexicalNodeClass } | null = null;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    public static async init() {
        if (!LexicalNodes.instance.nodes) {
            LexicalNodes.instance.nodes = {
                "Code": (await import("../plugins/CodePlugin")).CodeBlockNode,
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
}

// Initialize the singleton instance
LexicalNodes.init();
