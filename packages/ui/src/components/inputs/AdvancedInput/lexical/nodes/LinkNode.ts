import { RangeSelection } from "../selection.js";
import { BaseSelection, DOMConversionMap, DOMConversionOutput, LinkAttributes, NodeConstructorPayloads, NodeType, SerializedAutoLinkNode, SerializedLinkNode } from "../types.js";
import { $applyNodeReplacement, $createNode, $isNode, $isRangeSelection, getParent, isHTMLAnchorElement } from "../utils.js";
import { ElementNode } from "./ElementNode.js";
import { type LexicalNode } from "./LexicalNode.js";

const SUPPORTED_URL_PROTOCOLS = new Set([
    "http:",
    "https:",
    "mailto:",
    "sms:",
    "tel:",
]);

export class LinkNode extends ElementNode {
    static __type: NodeType = "Link";
    __url: string;
    __target: null | string;
    __rel: null | string;
    __title: null | string;

    constructor({ rel, target, title, url, ...rest }: NodeConstructorPayloads["Link"]) {
        super(rest);
        this.__url = url;
        this.__target = target || null;
        this.__rel = rel || null;
        this.__title = title || null;
    }

    static clone(node: LinkNode): LinkNode {
        const { __rel, __target, __title, __url, __key } = node;
        return $createNode("Link", { rel: __rel, target: __target, title: __title, url: __url, key: __key });
    }

    createDOM(): HTMLAnchorElement {
        const element = document.createElement("a");
        element.href = this.sanitizeUrl(this.__url);
        if (this.__target !== null) {
            element.target = this.__target;
        }
        if (this.__rel !== null) {
            element.rel = this.__rel;
        }
        if (this.__title !== null) {
            element.title = this.__title;
        }
        return element;
    }

    updateDOM(
        prevNode: LinkNode,
        anchor: HTMLAnchorElement,
    ): boolean {
        const url = this.__url;
        const target = this.__target;
        const rel = this.__rel;
        const title = this.__title;
        if (url !== prevNode.__url) {
            anchor.href = url;
        }

        if (target !== prevNode.__target) {
            if (target) {
                anchor.target = target;
            } else {
                anchor.removeAttribute("target");
            }
        }

        if (rel !== prevNode.__rel) {
            if (rel) {
                anchor.rel = rel;
            } else {
                anchor.removeAttribute("rel");
            }
        }

        if (title !== prevNode.__title) {
            if (title) {
                anchor.title = title;
            } else {
                anchor.removeAttribute("title");
            }
        }
        return false;
    }

    static importDOM(): DOMConversionMap {
        return {
            a: (node: Node) => ({
                conversion: convertAnchorElement,
                priority: 1,
            }),
        };
    }

    static importJSON({
        direction,
        format,
        indent,
        rel,
        target,
        title,
        url,
    }: SerializedLinkNode): LinkNode {
        const node = $createNode("Link", { rel, target, title, url });
        node.setFormat(format);
        node.setIndent(indent);
        node.setDirection(direction);
        return node;
    }

    sanitizeUrl(url: string): string {
        try {
            const parsedUrl = new URL(url);
            // eslint-disable-next-line no-script-url
            if (!SUPPORTED_URL_PROTOCOLS.has(parsedUrl.protocol)) {
                return "about:blank";
            }
        } catch {
            return url;
        }
        return url;
    }

    exportJSON(): SerializedLinkNode | SerializedAutoLinkNode {
        return {
            ...super.exportJSON(),
            __type: "Link",
            rel: this.getRel(),
            target: this.getTarget(),
            title: this.getTitle(),
            url: this.getURL(),
            version: 1,
        };
    }

    getURL(): string {
        return this.getLatest().__url;
    }

    setURL(url: string): void {
        const writable = this.getWritable();
        writable.__url = url;
    }

    getTarget(): null | string {
        return this.getLatest().__target;
    }

    setTarget(target: null | string): void {
        const writable = this.getWritable();
        writable.__target = target;
    }

    getRel(): null | string {
        return this.getLatest().__rel;
    }

    setRel(rel: null | string): void {
        const writable = this.getWritable();
        writable.__rel = rel;
    }

    getTitle(): null | string {
        return this.getLatest().__title;
    }

    setTitle(title: null | string): void {
        const writable = this.getWritable();
        writable.__title = title;
    }

    insertNewAfter(
        _: RangeSelection,
        restoreSelection = true,
    ): null | ElementNode {
        const linkNode = $createNode("Link", {
            rel: this.__rel,
            target: this.__target,
            title: this.__title,
            url: this.__url,
        });
        this.insertAfter(linkNode, restoreSelection);
        return linkNode;
    }

    canInsertTextBefore(): false {
        return false;
    }

    canInsertTextAfter(): false {
        return false;
    }

    canBeEmpty(): false {
        return false;
    }

    isInline(): true {
        return true;
    }

    extractWithChild(
        child: LexicalNode,
        selection: BaseSelection,
        destination: "clone" | "html",
    ): boolean {
        if (!$isRangeSelection(selection)) {
            return false;
        }

        const anchorNode = selection.anchor.getNode();
        const focusNode = selection.focus.getNode();

        return (
            this.isParentOf(anchorNode) &&
            this.isParentOf(focusNode) &&
            selection.getTextContent().length > 0
        );
    }
}

function convertAnchorElement(domNode: Node): DOMConversionOutput {
    let node: LinkNode | null = null;
    if (isHTMLAnchorElement(domNode)) {
        const content = domNode.textContent;
        if ((content !== null && content !== "") || domNode.children.length > 0) {
            node = $createNode("Link", {
                rel: domNode.getAttribute("rel"),
                target: domNode.getAttribute("target"),
                title: domNode.getAttribute("title"),
                url: domNode.getAttribute("href") || "",
            });
        }
    }
    return { node };
}

// Custom node type to override `canInsertTextAfter` that will
// allow typing within the link
export class AutoLinkNode extends LinkNode { //TODO might need to set this.__type

    static clone(node: AutoLinkNode): AutoLinkNode {
        const { __rel, __target, __title, __url, __key } = node;
        return new AutoLinkNode({ rel: __rel, target: __target, title: __title, url: __url, key: __key });
    }

    static importJSON(serializedNode: SerializedAutoLinkNode): AutoLinkNode {
        const node = $createAutoLinkNode({
            rel: serializedNode.rel,
            target: serializedNode.target,
            title: serializedNode.title,
            url: serializedNode.url,
        });
        node.setFormat(serializedNode.format);
        node.setIndent(serializedNode.indent);
        node.setDirection(serializedNode.direction);
        return node;
    }

    static importDOM(): DOMConversionMap {
        return {};
    }

    exportJSON(): SerializedAutoLinkNode {
        return {
            ...super.exportJSON(),
            __type: "Link",
            version: 1,
        };
    }

    insertNewAfter(
        selection: RangeSelection,
        restoreSelection = true,
    ): null | ElementNode {
        const element = getParent(this, { throwIfNull: true }).insertNewAfter(
            selection,
            restoreSelection,
        );
        if ($isNode("Element", element)) {
            const linkNode = $createAutoLinkNode({
                rel: this.__rel,
                target: this.__target,
                title: this.__title,
                url: this.__url,
            });
            element.append(linkNode);
            return linkNode;
        }
        return null;
    }
}

/**
 * Takes a URL and creates an AutoLinkNode. AutoLinkNodes are generally automatically generated
 * during typing, which is especially useful when a button to generate a LinkNode is not practical.
 * @param url - The URL the LinkNode should direct to.
 * @param attributes - Optional HTML a tag attributes. { target, rel, title }
 * @returns The LinkNode.
 */
export function $createAutoLinkNode(attributes: LinkAttributes): AutoLinkNode {
    return $applyNodeReplacement(new AutoLinkNode(attributes));
}
