/* eslint-disable no-magic-numbers */
import { ElementFormatType, TextDetailType, TextFormatType, TextModeType } from "./types.js";

export const NO_DIRTY_NODES = 0;
export const HAS_DIRTY_NODES = 1;
export const FULL_RECONCILE = 2;

export const COMMAND_PRIORITY_EDITOR = 0;
export const COMMAND_PRIORITY_LOW = 1;
export const COMMAND_PRIORITY_NORMAL = 2;
export const COMMAND_PRIORITY_HIGH = 3;
export const COMMAND_PRIORITY_CRITICAL = 4;

// Element node formatting
export const IS_ALIGN_LEFT = 1;
export const IS_ALIGN_CENTER = 2;
export const IS_ALIGN_RIGHT = 3;
export const IS_ALIGN_JUSTIFY = 4;
export const IS_ALIGN_START = 5;
export const IS_ALIGN_END = 6;

// Text node details
export const IS_DIRECTIONLESS = 1;
export const IS_UNMERGEABLE = 1 << 1;

// Text node modes
export const IS_NORMAL = 0;
export const IS_TOKEN = 1;
export const IS_SEGMENTED = 2;
export const TEXT_MODE_TO_TYPE: Record<TextModeType, 0 | 1 | 2> = {
    normal: IS_NORMAL,
    segmented: IS_SEGMENTED,
    token: IS_TOKEN,
};
export const TEXT_TYPE_TO_MODE: Record<number, TextModeType> = {
    [IS_NORMAL]: "normal",
    [IS_SEGMENTED]: "segmented",
    [IS_TOKEN]: "token",
};

export const DETAIL_TYPE_TO_DETAIL: Record<TextDetailType | string, number> = {
    directionless: IS_DIRECTIONLESS,
    unmergeable: IS_UNMERGEABLE,
};

export const ELEMENT_TYPE_TO_FORMAT: Record<
    Exclude<ElementFormatType, "">,
    number
> = {
    center: IS_ALIGN_CENTER,
    end: IS_ALIGN_END,
    justify: IS_ALIGN_JUSTIFY,
    left: IS_ALIGN_LEFT,
    right: IS_ALIGN_RIGHT,
    start: IS_ALIGN_START,
};

export const ELEMENT_FORMAT_TO_TYPE: Record<number, ElementFormatType> = {
    [IS_ALIGN_CENTER]: "center",
    [IS_ALIGN_END]: "end",
    [IS_ALIGN_JUSTIFY]: "justify",
    [IS_ALIGN_LEFT]: "left",
    [IS_ALIGN_RIGHT]: "right",
    [IS_ALIGN_START]: "start",
};

export const ELEMENT_NODES = [
    "Code",
    "Element",
    "Heading",
    "Link",
    "List",
    "ListItem",
    "Paragraph",
    "Quote",
    "Root",
    "Table",
    "TableCell",
    "TableRow",
] as const;
export const DECORATOR_NODES = [
    "Decorator",
] as const;
export const TEXT_NODES = [
    "Hashtag",
    "Tab",
    "Text",
] as const;

/**
 * Bit flags representing the type of text formatting. This is useful because
 * text nodes can have multiple types of formatting.
 */
export const TEXT_FLAGS: Record<TextFormatType, number> = {
    BOLD: 1 << 0,
    CODE_BLOCK: 1 << 1,
    CODE_INLINE: 1 << 2,
    HEADING: 1 << 3,
    HIGHLIGHT: 1 << 4,
    ITALIC: 1 << 5,
    LIST_ORDERED: 1 << 6,
    LIST_UNORDERED: 1 << 7,
    QUOTE: 1 << 8,
    SPOILER_LINES: 1 << 9,
    SPOILER_TAGS: 1 << 10,
    STRIKETHROUGH: 1 << 11,
    SUBSCRIPT: 1 << 12,
    SUPERSCRIPT: 1 << 13,
    UNDERLINE_LINES: 1 << 14,
    UNDERLINE_TAGS: 1 << 15,
} as const;

export const DOUBLE_LINE_BREAK = "\n\n";

export const LIST_INDENT_SIZE = 4;

export const DOM_ELEMENT_TYPE = 1;
export const DOM_TEXT_TYPE = 3;

export const CAN_USE_DOM: boolean =
    typeof window !== "undefined" &&
    typeof window.document !== "undefined" &&
    typeof window.document.createElement !== "undefined";

const documentMode =
    CAN_USE_DOM && "documentMode" in document ? document.documentMode : null;

export const IS_APPLE: boolean =
    CAN_USE_DOM && /Mac|iPod|iPhone|iPad/.test(navigator.userAgent);

export const IS_FIREFOX: boolean =
    CAN_USE_DOM && /^(?!.*Seamonkey)(?=.*Firefox).*/i.test(navigator.userAgent);

export const CAN_USE_BEFORE_INPUT: boolean =
    CAN_USE_DOM && "InputEvent" in window && !documentMode
        ? "getTargetRanges" in new window.InputEvent("input")
        : false;

export const IS_SAFARI: boolean =
    CAN_USE_DOM && /Version\/[\d.]+.*Safari/.test(navigator.userAgent);

export const IS_IOS: boolean =
    CAN_USE_DOM &&
    /iPad|iPhone|iPod/.test(navigator.userAgent) &&
    !(window as { MSStream?: unknown }).MSStream;

export const IS_ANDROID: boolean =
    CAN_USE_DOM && /Android/.test(navigator.userAgent);

// Keep these in case we need to use them in the future.
// export const IS_WINDOWS: boolean = CAN_USE_DOM && /Win/.test(navigator.platform);
export const IS_CHROME: boolean =
    CAN_USE_DOM && /^(?=.*Chrome).*/i.test(navigator.userAgent);
// export const canUseTextInputEvent: boolean = CAN_USE_DOM && 'TextEvent' in window && !documentMode;

export const IS_ANDROID_CHROME: boolean =
    CAN_USE_DOM && IS_ANDROID && IS_CHROME;

export const IS_APPLE_WEBKIT =
    CAN_USE_DOM && /AppleWebKit\/[\d.]+/.test(navigator.userAgent) && !IS_CHROME;

export const NON_BREAKING_SPACE = "\u00A0";
const ZERO_WIDTH_SPACE = "\u200b";

// For iOS/Safari we use a non breaking space, otherwise the cursor appears
// overlapping the composed text.
export const COMPOSITION_SUFFIX: string =
    IS_SAFARI || IS_IOS || IS_APPLE_WEBKIT
        ? NON_BREAKING_SPACE
        : ZERO_WIDTH_SPACE;

// For FF, we need to use a non-breaking space, or it gets composition
// in a stuck state.
export const COMPOSITION_START_CHAR: string = IS_FIREFOX
    ? NON_BREAKING_SPACE
    : COMPOSITION_SUFFIX;

export const ANDROID_COMPOSITION_LATENCY = 30;

/** The time between a text entry event and the mutation observer firing. */
export const TEXT_MUTATION_VARIANCE = 100;

const RTL = "\u0591-\u07FF\uFB1D-\uFDFD\uFE70-\uFEFC";
const LTR =
    "A-Za-z\u00C0-\u00D6\u00D8-\u00F6" +
    "\u00F8-\u02B8\u0300-\u0590\u0800-\u1FFF\u200E\u2C00-\uFB1C" +
    "\uFE00-\uFE6F\uFEFD-\uFFFF";
// eslint-disable-next-line no-misleading-character-class
export const RTL_REGEX = new RegExp("^[^" + LTR + "]*[" + RTL + "]");
// eslint-disable-next-line no-misleading-character-class
export const LTR_REGEX = new RegExp("^[^" + RTL + "]*[" + LTR + "]");

export const PIXEL_VALUE_REG_EXP = /^(\d+(?:\.\d+)?)px$/;

export const CSS_TO_STYLES: Map<string, Record<string, string>> = new Map();

export const IGNORE_TAGS = new Set(["STYLE", "SCRIPT"]);

export const DEFAULT_CODE_LANGUAGE = "javascript";
