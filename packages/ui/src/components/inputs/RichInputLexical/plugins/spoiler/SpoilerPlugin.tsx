import { TextMatchTransformer } from "@lexical/markdown";
import { useLexicalComposerContext } from "@lexical/react/LexicalComposerContext";
import { $getSelection, $isRangeSelection, $isTextNode, COMMAND_PRIORITY_LOW, DOMConversionMap, DOMConversionOutput, ElementNode, SerializedLexicalNode, Spread, TextNode, createCommand, type EditorConfig, type LexicalNode, type NodeKey } from "lexical";

export type SerializedSpoilerNode = Spread<
    {
        children: any[];
    },
    SerializedLexicalNode
>;

export class SpoilerNode extends ElementNode {
    constructor(key?: NodeKey) {
        super(key);
    }

    static getType(): string {
        return "spoiler";
    }

    static clone(node: SpoilerNode): SpoilerNode {
        return new SpoilerNode(node.__key);
    }

    /** Switch between showing text and hiding text */
    toggleSpoilerVisibility = (event: Event) => {
        const target = event.currentTarget as HTMLElement;
        if (target.classList.contains("revealed")) {
            target.classList.remove("revealed");
            target.classList.add("hidden");
        } else {
            target.classList.remove("hidden");
            target.classList.add("revealed");
        }
    };

    createDOM(_config: EditorConfig): HTMLElement {
        const element = document.createElement("span");
        element.classList.add("spoiler", "hidden"); // Start in hidden state
        element.setAttribute("spellcheck", "false"); // Disable spellcheck so that it doesn't block the toggle
        element.addEventListener("click", this.toggleSpoilerVisibility);
        return element;
    }

    updateDOM(_prevNode: SpoilerNode, span: HTMLElement, _config: EditorConfig): boolean {
        // Assuming you might want to update some styles or attributes in the future, but for now, it's a no-op
        // Simply checks that the element is a spoiler and updates if needed
        if (!span.classList.contains("spoiler")) {
            span.classList.add("spoiler");
            span.setAttribute("spellcheck", "false");
            span.addEventListener("click", this.toggleSpoilerVisibility);
            return true; // Indicates that the DOM was updated
        }
        return false; // Indicates that no updates were necessary
    }

    static importDOM(): DOMConversionMap | null {
        return {
            span: (node: Node) => {
                // Check if the span has the correct class or attribute to identify it as a spoiler
                if ((node as HTMLElement).classList.contains("spoiler")) {
                    return {
                        conversion: SpoilerNode.fromDOMElement,
                        priority: 1,
                    };
                }
                return null;
            },
        };
    }

    static fromDOMElement(element: HTMLElement): DOMConversionOutput {
        // Creates a new SpoilerNode from the given DOM element
        return { node: new SpoilerNode() };
    }

    isInline(): true {
        return true;
    }

    static importJSON(serializedNode: SerializedSpoilerNode): SpoilerNode {
        if (serializedNode.type === "spoiler" && serializedNode.version === 1) {
            const node = new SpoilerNode();
            serializedNode.children.forEach(childJSON => {
                node.append(childJSON);
            });
            return node;
        }
        throw new Error("Invalid serialized spoiler node.");
    }

    exportJSON(): any {//SerializedSpoilerNode { TODO
        const childrenJSON = this.getChildren().map(child => child.exportJSON());
        return {
            type: "spoiler",
            version: 1,
            children: childrenJSON,
        } as any; //const;
    }

    // insertNewAfter(
    //     _selection: RangeSelection,
    //     restoreSelection?: boolean | undefined
    // ): LexicalNode | null {
    //     const newBlock = $createParagraphNode();
    //     const direction = this.getDirection();
    //     newBlock.setDirection(direction);
    //     this.insertAfter(newBlock, restoreSelection);
    //     return newBlock;
    // }

    // collapseAtStart(): boolean {
    //     const paragraph = $createParagraphNode();
    //     const children = this.getChildren();
    //     children.forEach((child) => paragraph.append(child));
    //     this.replace(paragraph);
    //     return true;
    // }
}

export function $createSpoilerNode(): SpoilerNode {
    return new SpoilerNode();
}

export function $isSpoilerNode(node: LexicalNode): node is SpoilerNode {
    return node instanceof SpoilerNode;
}

export const TOGGLE_SPOILER_COMMAND = createCommand("TOGGLE_SPOILER");

const spoilerCommandListener = () => {
    const selection = $getSelection();
    if (!$isRangeSelection(selection)) {
        return false;
    }
    const nodes = selection.getNodes();
    console.log("got nodes", nodes, nodes.map(node => node.getParent()));
    // Check if there is a spoiler node in the selection, or if the selection is a single node with a parent spoiler node
    const isAlreadySpoiled = (nodes.length > 0 && nodes.some((node) => node instanceof SpoilerNode)) ||
        (nodes.length === 1 && nodes[0].getParent() instanceof SpoilerNode);
    console.log("is already spoiled", isAlreadySpoiled);
    if (isAlreadySpoiled) {
        // Remove each spoiler effect by replacing it with its children
        nodes.forEach((node) => {
            let spoilerNode = node;
            // If node isn't a spoiler, try the parent
            if (!(spoilerNode instanceof SpoilerNode)) {
                spoilerNode = node.getParent() as LexicalNode;
                if (!(spoilerNode instanceof SpoilerNode)) {
                    return;
                }
            }
            const children = spoilerNode.getChildren<LexicalNode>();
            for (const child of children) {
                spoilerNode.insertBefore(child);
            }
            spoilerNode.remove();
        });
    } else if (nodes.length > 0) {
        const spoilerNode = $createSpoilerNode();

        const startNode = nodes[0] as TextNode;
        if (!$isTextNode(startNode)) {
            console.error("Failed to find a text node for the start of the selection!");
            return false;
        }
        console.log("got start node", startNode.getTextContent());
        const endNode = nodes[nodes.length - 1] as TextNode;
        if (!$isTextNode(endNode)) {
            console.error("Failed to find a text node for the end of the selection!");
            return false;
        }
        console.log("got end node", endNode.getTextContent());
        const startParent = startNode.getParent();
        console.log("start parent at beginning", startParent?.getTextContent());
        const nodeAfterStartNode = startNode.getNextSibling();

        // Get offsets from smallest to largest
        const [startOffset, endOffset] = [selection.anchor.offset, selection.focus.offset].sort((a, b) => a - b);
        console.log("got offsets", startOffset, endOffset);

        // Split the startNode if needed
        if (startNode.getTextContentSize() > startOffset) {
            const beforeText = startNode.getTextContent().substring(0, startOffset);
            const newNode: TextNode = new TextNode(beforeText); // Assuming TextNode constructor accepts a string
            console.log("new node 1", newNode.getTextContent());
            startNode.insertBefore(newNode);
            startNode.setTextContent(startNode.getTextContent().substring(startOffset));
        }
        console.log("got start node", startNode.getTextContent());

        // If the spoiler spans multiple nodes:
        if (startNode !== endNode) {
            let currentNode = startNode.getNextSibling();
            while (currentNode && currentNode !== endNode) {
                spoilerNode.append(currentNode);
                currentNode = currentNode.getNextSibling();
            }
        }

        // Split the endNode if needed
        if (endOffset < endNode.getTextContentSize()) {
            const afterText = endNode.getTextContent().substring(endOffset - startOffset);
            const newNode: TextNode = new TextNode(afterText);
            console.log("new node 2", newNode.getTextContent());
            endNode.insertAfter(newNode);
            endNode.setTextContent(endNode.getTextContent().substring(0, endOffset - startOffset));
        }
        console.log("got end node", endNode.getTextContent());

        // Append the (possibly split) endNode to spoilerNode
        spoilerNode.append(endNode);

        // Insert the spoilerNode in the correct location
        console.log("insert the spoiler node", startNode.getTextContent(), spoilerNode.getTextContent());
        if (startParent) {
            startParent.insertAfter(spoilerNode);
        } else {
            // Handle cases where the startParent is not available
            console.error("Failed to find a parent for the startNode!");
        }

        // Adjust the selection to the entire spoilerNode
        spoilerNode.select();
    }
    return true;
};

export function SpoilerPlugin(): null {
    const [editor] = useLexicalComposerContext();
    if (!editor.hasNodes([SpoilerNode])) {
        throw new Error("SpoilerPlugin: SpoilerNode not registered on editor");
    }
    editor.registerCommand(
        TOGGLE_SPOILER_COMMAND,
        spoilerCommandListener,
        COMMAND_PRIORITY_LOW,
    );
    return null;
}

// Spoiler with ||spoiler|| syntax
export const SPOILER_LINES_TRANSFORMER: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        const isSpoiler = (node as TextNode).__type === "spoiler";
        const shouldExportChildren = node instanceof ElementNode;
        if (isSpoiler) {
            return `||${shouldExportChildren ? exportChildren(node as ElementNode) : (node as TextNode).__text}||`;
        }
        return null;
    },
    importRegExp: /\|\|(.*?)\|\|/,
    regExp: /\|\|(.*?)\|\|$/,
    replace: (textNode, match) => {
        // Extract the matched spoiler text
        const spoilerText = match[1];

        // Create a new styled text node with the matched spoiler text
        const spoilerTextNode = new TextNode(spoilerText);

        // Create a SpoilerNode
        const spoilerNode = $createSpoilerNode();
        spoilerNode.append(spoilerTextNode);

        // Replace the original text node with the spoiler node
        textNode.replace(spoilerNode);
    },
    trigger: "||",
    type: "text-match",
};

// Spoiler with <spoiler>spoiler</spoiler> syntax
export const SPOILER_TAGS_TRANSFORMER: TextMatchTransformer = {
    dependencies: [],
    export: (node, exportChildren, exportFormat) => {
        const isSpoiler = (node as TextNode).__type === "spoiler";
        const shouldExportChildren = node instanceof ElementNode;
        if (isSpoiler) {
            return `<spoiler>${shouldExportChildren ? exportChildren(node as ElementNode) : (node as TextNode).__text}</spoiler>`;
        }
        return null;
    },
    importRegExp: /<spoiler>(.*?)<\/spoiler>/,
    regExp: /<spoiler>(.*?)<\/spoiler>$/,
    replace: (textNode, match) => {
        // Extract the matched spoiler text
        const spoilerText = match[1];
        console.log("SPOILER TEXT", match, textNode);

        // Create a new styled text node with the matched spoiler text
        const spoilerTextNode = new TextNode(spoilerText);
        console.log("SPOILER TEXT NODE", spoilerTextNode);

        // Create a SpoilerNode
        const spoilerNode = $createSpoilerNode();
        spoilerNode.append(spoilerTextNode);

        // Replace the original text node with the spoiler node
        textNode.replace(spoilerNode);
    },
    trigger: "<spoiler>",
    type: "text-match",
};
