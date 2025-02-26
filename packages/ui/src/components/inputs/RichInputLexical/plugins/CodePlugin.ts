import { copyIconPath } from "icons";
import { PubSub } from "utils/pubsub.js";
import { CODE_BLOCK_COMMAND } from "../commands.js";
import { COMMAND_PRIORITY_HIGH } from "../consts.js";
import { useLexicalComposerContext } from "../context.js";
import { ElementNode } from "../nodes/ElementNode.js";
import { DOMConversionMap, DOMConversionOutput, NodeConstructorPayloads, NodeType, SerializedCodeBlockNode } from "../types.js";
import { $createNode, $getSelection, $isNode, $isRangeSelection, getParent } from "../utils.js";

const LANGUAGE_DATA_ATTRIBUTE = "data-highlight-language";

export class CodeBlockNode extends ElementNode {
    static __type: NodeType = "Code";
    __language: string;

    constructor({ language, ...rest }: NodeConstructorPayloads["Code"]) {
        super(rest);
        this.__language = language || "";
    }

    static clone(node: CodeBlockNode): CodeBlockNode {
        const { __language, __key } = node;
        return new CodeBlockNode({ language: __language, key: __key });
    }

    createDOM(): HTMLElement {
        // Create the outer div to position the copy button relative to it
        const topElement = document.createElement("div");
        topElement.style.backgroundColor = "#23241f";
        topElement.style.color = "#f8f8f2";
        topElement.style.border = "1px solid #ddd";
        topElement.style.fontFamily = "Menlo, Consolas, Monaco, monospace";
        topElement.style.display = "block";
        topElement.style.lineHeight = "1.53";
        topElement.style.fontSize = "13px";
        topElement.style.margin = "0";
        topElement.style.marginTop = "8px";
        topElement.style.marginBottom = "8px";
        topElement.style.overflowX = "auto";
        topElement.style.position = "relative";
        topElement.style.tabSize = "2";
        topElement.style.borderRadius = "4px";
        topElement.setAttribute("spellcheck", "false"); // Disable spellcheck
        topElement.setAttribute("autocomplete", "off"); // Disable autocomplete
        topElement.setAttribute("autocorrect", "off"); // Disable autocorrect
        topElement.setAttribute("autocapitalize", "off"); // Disable autocapitalize

        // Create the top bar
        const topBar = document.createElement("div");
        topBar.style.display = "flex";
        topBar.style.justifyContent = "space-between";
        topBar.style.alignItems = "center";
        topBar.style.background = "#464c51";
        topBar.style.padding = "4px 8px";
        topBar.style.borderBottom = "1px solid #ddd"; // Example styling, adjust as needed

        // Add language label to the top bar
        const languageLabel = document.createElement("span");
        // languageLabel.textContent = this.__language.toUpperCase(); // Display language in uppercase
        if (this.__language && this.__language.length > 0) {
            languageLabel.textContent = this.__language.toUpperCase();
        } else {
            languageLabel.textContent = "Select Language"; // Placeholder text
            languageLabel.style.cursor = "pointer";
            languageLabel.onclick = () => {
                //TODO
                // Trigger your language selection modal or dropdown here
                // Update the node with the selected language afterwards
            };
        }
        languageLabel.style.fontFamily = "\"Roboto\",\"Helvetica\",\"Arial\",sans-serif";
        languageLabel.style.fontSize = "0.75rem"; // Smaller font size for the language label
        topBar.appendChild(languageLabel);

        // Create the copy button with an icon
        const copyButton = document.createElement("button");
        copyButton.setAttribute("type", "button"); // Set the button type to 'button' to prevent form submission
        copyButton.setAttribute("aria-label", "Copy code to clipboard");
        copyButton.style.background = "transparent";
        copyButton.style.border = "none";
        copyButton.style.cursor = "pointer";

        // Create the SVG element for the copy icon
        const svgNS = "http://www.w3.org/2000/svg";
        const copyIcon = document.createElementNS(svgNS, "svg");
        copyIcon.setAttributeNS(null, "viewBox", "0 0 24 24");
        copyIcon.setAttributeNS(null, "width", "24");
        copyIcon.setAttributeNS(null, "height", "24");
        const path = document.createElementNS(svgNS, "path");
        path.setAttributeNS(null, "d", copyIconPath);
        path.setAttributeNS(null, "fill", "white");
        copyIcon.appendChild(path);
        copyButton.appendChild(copyIcon);

        copyButton.onclick = () => {
            navigator.clipboard.writeText(codeElement.textContent || "");
            PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
        };
        topBar.appendChild(copyButton);

        // Append the top bar to the outer element
        topElement.appendChild(topBar);

        // Pre and code elements
        const preElement = document.createElement("pre");
        preElement.style.margin = "0"; // Reset margin for consistency
        const codeElement = document.createElement("code");
        codeElement.setAttribute("class", `language-${this.__language}`);
        codeElement.style.padding = "16px"; // Add padding to the code text
        codeElement.style.display = "block"; // Ensure the code element is block-level for proper padding
        preElement.appendChild(codeElement);

        topElement.appendChild(preElement);

        return topElement;
    }

    updateDOM(_prevNode: CodeBlockNode, dom: HTMLElement): boolean {
        const codeElement = dom.firstChild as HTMLElement;
        codeElement.className = `language-${this.__language}`;
        return false; // No need to update the DOM structure itself
    }
    // TODO might be able to use updateDOM to move the text (which is automatically placed in a span) to the code element for hljs to work
    // updateDOM(prevNode, dom) {
    //     const codeElement = dom.querySelector('code');
    //     const spans = dom.querySelectorAll('span');
    //     spans.forEach(span => codeElement.appendChild(span));
    //     return true; // Return true to indicate that the DOM structure has been manually updated
    // }

    exportDOM() {
        const element = document.createElement("pre");
        element.setAttribute("spellcheck", "false");
        const language = this.__language;
        if (language && language.length > 0) {
            element.setAttribute(LANGUAGE_DATA_ATTRIBUTE, language);
        }
        return {
            element,
        };
    }

    static importJSON(serializedNode: SerializedCodeBlockNode): CodeBlockNode {
        const { language } = serializedNode;
        return new CodeBlockNode({ language });
    }

    exportJSON(): SerializedCodeBlockNode {
        return {
            ...super.exportJSON(),
            __type: "Code",
            language: this.__language,
        };
    }

    static importDOM(): DOMConversionMap {
        return {
            pre: (node: Node) => {
                const codeElement = node.firstChild;
                if (codeElement && codeElement.nodeName.toLowerCase() === "code") {
                    return {
                        conversion: CodeBlockNode.fromDOMElement,
                        priority: 1,
                    };
                }
                return null;
            },
        };
    }

    static fromDOMElement(element: HTMLElement): DOMConversionOutput {
        const codeElement = element.querySelector("code");
        let language = "";

        // Extract language class if present (assuming class format is language-xxx)
        if (codeElement) {
            const classList = Array.from(codeElement.classList);
            const languageClass = classList.find(className => className.startsWith("language-"));
            if (languageClass) {
                language = languageClass.replace("language-", "");
            }
        }

        // Create a new CodeBlockNode with the extracted language and the text content
        const node = $createNode("Code", { language });
        // Assuming you have a way to set the text content for the code block, e.g., by appending a TextNode
        if (codeElement) {
            const textNode = $createNode("Text", { text: codeElement.textContent || "" });
            node.append(textNode);
        }

        return { node };
    }

    canIndent() {
        return false;
    }

    collapseAtStart() {
        const paragraph = $createNode("Paragraph", {});
        const children = this.getChildren();
        children.forEach(child => paragraph.append(child));
        this.replace(paragraph);
        return true;
    }

    getMarkdownContent() {
        const markdown = this.getChildren().map(child => {
            const text = child.getMarkdownContent();
            if ($isNode("Element", child) && !child.isInline()) {
                return text + "\n";
            }
            return text;
        }).join("");
        // Return with backticks and language identifier
        return `\`\`\`${this.__language}\n${markdown}\n\`\`\``;
    }

    // insertNewAfter(
    //     _selection: RangeSelection,
    //     restoreSelection?: boolean | undefined
    // ): LexicalNode | null {
    //     const newBlock = $createNode("Paragraph", {});
    //     const direction = this.getDirection();
    //     newBlock.setDirection(direction);
    //     this.insertAfter(newBlock, restoreSelection);
    //     return newBlock;
    // }
}

const codeBlockCommandListener = () => {
    const selection = $getSelection();
    if (!$isRangeSelection(selection)) {
        return false;
    }
    const nodes = selection.getNodes();

    // Check if there is a code block node in the selection
    const isInCodeBlock = nodes.some(node => $isNode("Code", node) || getParent(node) !== null);

    if (isInCodeBlock) {
        // Logic to unwrap the selected text from the CodeBlockNode
        nodes.forEach(node => {
            const codeBlockNode = $isNode("Code", node) ? node : getParent(node);
            if (codeBlockNode) {
                const textContent = codeBlockNode.getTextContent();
                const newTextNode = $createNode("Text", { text: textContent });
                codeBlockNode.replace(newTextNode);
            }
        });
    } else {
        // Logic to wrap the selected text in a new CodeBlockNode
        const textContent = selection.getTextContent();
        const codeBlockNode = $createNode("Code", {}); // Optionally pass a language identifier
        const textNode = $createNode("Text", { text: textContent });
        codeBlockNode.append(textNode);

        // $wrapNodes([codeBlockNode], selection);
    }

    return true;
};

export function CodeBlockPlugin(): null {
    const editor = useLexicalComposerContext();
    editor?.registerCommand(
        CODE_BLOCK_COMMAND,
        codeBlockCommandListener,
        COMMAND_PRIORITY_HIGH, // Higher priority than built-in code block command
    );
    return null;
}
