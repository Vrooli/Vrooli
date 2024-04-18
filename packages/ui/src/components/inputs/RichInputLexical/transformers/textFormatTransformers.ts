import { TEXT_FLAGS } from "../consts";
import { LexicalNode } from "../nodes/LexicalNode";
import { TextFormatTransformer, TextFormatType } from "../types";
import { $isNode } from "../utils";

export const BOLD: TextFormatTransformer = {
    format: "BOLD",
    tags: ["**", "**"],
    type: "text-format",
};

export const CODE_INLINE: TextFormatTransformer = {
    format: "CODE_INLINE",
    tags: ["`", "`"],
    type: "text-format",
};

export const HIGHLIGHT: TextFormatTransformer = {
    format: "HIGHLIGHT",
    tags: ["==", "=="],
    type: "text-format",
};

export const ITALIC: TextFormatTransformer = {
    format: "ITALIC",
    tags: ["*", "*"],
    type: "text-format",
};

export const UNDERLINE_LINES: TextFormatTransformer = {
    format: "UNDERLINE_LINES",
    tags: ["__", "__"],
    type: "text-format",
};

export const UNDERLINE_TAGS: TextFormatTransformer = {
    format: "UNDERLINE_TAGS",
    tags: ["<u>", "</u>"],
    type: "text-format",
};

export const SPOILER_LINES: TextFormatTransformer = {
    format: "SPOILER_LINES",
    tags: ["||", "||"],
    type: "text-format",
};

export const SPOILER_TAGS: TextFormatTransformer = {
    format: "SPOILER_TAGS",
    tags: ["<spoiler>", "</spoiler>"],
    type: "text-format",
};


export const STRIKETHROUGH: TextFormatTransformer = {
    format: "STRIKETHROUGH",
    tags: ["~~", "~~"],
    type: "text-format",
};

// NOTE: Order of text format transformers matters:
//
// - code should go first as it prevents any transformations inside
// - then longer tags match (e.g. ** should go before *)
export const TEXT_TRANSFORMERS: TextFormatTransformer[] = [
    BOLD,
    CODE_INLINE,
    HIGHLIGHT,
    ITALIC,
    SPOILER_LINES,
    SPOILER_TAGS,
    STRIKETHROUGH,
    UNDERLINE_LINES,
    UNDERLINE_TAGS,
];

/**
 * Finds every text transformer that matches the given format number. 
 * 
 * Format numbers store a bitwise representation of all the formats applied to a node.
 * @param format The format number to match.
 * @returns An array of text transformers that match the given format number.
 */
export const findAppliedTextTransformers = (format: number): TextFormatTransformer[] => {
    return TEXT_TRANSFORMERS.filter(transformer => (TEXT_FLAGS[transformer.format] & format) !== 0);
};

/**
 * Applies text transformers to the given text.
 * 
 * We assume that the text does not 
 * currently contain any tags, meaning THIS FUNCTION IS NOT IDEMPOTENT.
 * @param text The text to apply the transformers to. 
 * @param format The format number to apply to the text.
 */
export const applyTextTransformers = (text: string, format: number): string => {
    const transformers = findAppliedTextTransformers(format);
    let transformedText = text;

    transformers.forEach(transformer => {
        transformedText = `${transformer.tags[0]}${transformedText}${transformer.tags[1]}`;
    });

    return transformedText;
};

/**
 * @returns True if the given format has one of the specified flags, false otherwise.
 */
export const hasTextFormat = (
    format: number,
    ...flags: TextFormatType[]
): boolean => {
    return flags.some(flag => (format & TEXT_FLAGS[flag]) !== 0);
};

/**
 * Like `hasTextFormat`, but also checks if the given node is a text node.
 * @param node The node to check.
 * @param format The format to check for.
 */
export const hasFormat = (node: LexicalNode | null, format: TextFormatType) => {
    return $isNode("Text", node) && hasTextFormat(node.getFormat(), format);
};

/** 
 * Toggle a text format type on or off in a format number.
 * @param format The current format number. 
 * @param type The type of format to toggle.
 * @param alignWithFormat Locks certain formats so they can't be toggled off
 * @returns 
 */
export const toggleTextFormatType = (
    format: number,
    type: TextFormatType,
    alignWithFormat: null | number = null,
): number => {
    const flag = TEXT_FLAGS[type];
    const isOn = (format & flag) !== 0;
    const isLocked = alignWithFormat !== null && (alignWithFormat & flag) !== 0;

    if (isLocked) {
        return format;
    }

    if (isOn) {
        return format & ~flag;
    } else {
        return format | flag;
    }
};
