import { SPOILER_COMMAND } from "../commands";
import { COMMAND_PRIORITY_LOW, TEXT_FLAGS } from "../consts";
import { useLexicalComposerContext } from "../context";
import { type TextNode } from "../nodes/TextNode";
import { hasFormat, toggleTextFormatType } from "../transformers/textFormatTransformers";
import { $getSelection, $isNode, $isRangeSelection } from "../utils";

const spoilerCommandListener = () => {
    const selection = $getSelection();
    if (!$isRangeSelection(selection)) {
        return false;
    }

    const nodes = selection.getNodes();
    const isAlreadySpoiled = nodes.some(node => hasFormat(node, "SPOILER_LINES") || hasFormat(node, "SPOILER_TAGS"));

    if (isAlreadySpoiled) {
        nodes.forEach(node => {
            if (hasFormat(node, "SPOILER_LINES") || hasFormat(node, "SPOILER_TAGS")) {
                (node as TextNode).__format = toggleTextFormatType((node as TextNode).__format, "SPOILER_LINES");
            }
        });
    } else {
        const { anchor, focus } = selection;
        const [startOffset, endOffset] = [anchor.offset, focus.offset].sort((a, b) => a - b);

        if (nodes.length === 1 && $isNode("Text", nodes[0])) {
            // Simple case: single text node in selection
            const node = nodes[0];
            node.splitText(startOffset);
            const endNode = node.splitText(endOffset - startOffset);
            node.setFormat(node.getFormat() | TEXT_FLAGS.SPOILER_LINES);  // Applying spoiler format
        } else {
            // Complex case: multiple nodes
            nodes.forEach((node, index) => {
                if ($isNode("Text", node)) {
                    if (index === 0) {
                        node.splitText(startOffset);
                        node.setFormat(node.getFormat() | TEXT_FLAGS.SPOILER_LINES);
                    } else if (index === nodes.length - 1) {
                        node.splitText(endOffset - startOffset);
                        node.setFormat(node.getFormat() | TEXT_FLAGS.SPOILER_LINES);
                    } else {
                        node.setFormat(node.getFormat() | TEXT_FLAGS.SPOILER_LINES);
                    }
                }
            });
        }
    }
    return true;
};

export function SpoilerPlugin(): null {
    const editor = useLexicalComposerContext();
    editor?.registerCommand(
        SPOILER_COMMAND,
        spoilerCommandListener,
        COMMAND_PRIORITY_LOW,
    );
    return null;
}
