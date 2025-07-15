import Box from "@mui/material/Box";
import CircularProgress from "@mui/material/CircularProgress";
import { IconButton } from "../buttons/IconButton.js";
import Input from "@mui/material/Input";
import Paper from "@mui/material/Paper";
import Popover from "@mui/material/Popover";
import type { BoxProps, Palette, PopoverProps } from "@mui/material";
import { styled, useTheme } from "@mui/material";
import { type ChangeEvent, memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { VariableSizeList } from "react-window";
import { useDebounce } from "../../hooks/useDebounce.js";
import { type PageTab, useTabs } from "../../hooks/useTabs.js";
import { IconCommon } from "../../icons/Icons.js";
import { Z_INDEX } from "../../utils/consts.js";
import { type TabParamBase } from "../../utils/search/objectToSearch.js";
import { PageTabs } from "../PageTabs/PageTabs.js";
import { MicrophoneButton } from "../buttons/MicrophoneButton.js";
// import emojis from "./data/emojis";

type FallbackEmojiPickerProps = {
    anchorEl: Element | null;
    onClose: () => unknown;
    onSelect: (emoji: string) => unknown;
};

enum CategoryTabOption {
    Suggested = "Suggested",
    Smileys = "Smileys",
    Nature = "Nature",
    Food = "Food",
    Places = "Places",
    Activities = "Activities",
    Objects = "Objects",
    Symbols = "Symbols",
    Flags = "Flags"
}

type EmojiTabsInfo = {
    Key: CategoryTabOption;
    Payload: undefined;
    WhereParams: undefined;
}

type EmojiSelectProps = {
    emoji: string;
    unified: string;
};

export enum EmojiProperties {
    unified = "u",
    variations = "v",
}

export interface DataEmoji {
    [EmojiProperties.unified]: string;
    [EmojiProperties.variations]?: string[];
}

export enum SkinTone {
    Neutral = "neutral",
    Light = "1f3fb",
    MediumLight = "1f3fc",
    Medium = "1f3fd",
    MediumDark = "1f3fe",
    Dark = "1f3ff"
}

// const KNOWN_FAILING_EMOJIS = ["2640-fe0f", "2642-fe0f", "2695-fe0f"];
const SUGGESTED_EMOJIS_LIMIT = 20;
const SKIN_OPTION_WIDTH_WITH_SPACING = 28;
const SEARCH_STRING_DEBOUNCE_MS = 200;
const NATIVE_PICKER_FAIL_TIMEOUT_MS = 500;

const emptyArray = [];

const EMOJI_CACHE_KEY = "emoji_data_cache";
const EMOJI_CACHE_VERSION = "1"; // Increment this when the emoji data structure changes

// Add this near the top with other constants
const EMOJIS_PER_ROW = 7;
const EMOJI_ROW_HEIGHT = 48;
const CATEGORY_TITLE_HEIGHT = 36;
const EMOJI_GRID_HEIGHT = 400;
const EMOJI_GRID_WIDTH = EMOJIS_PER_ROW * EMOJI_ROW_HEIGHT;


// Removed as we're using search index instead

// Removed as we're using search index instead

// Add this near other constants
const SUGGESTED_BATCH_UPDATE_DELAY = 100; // ms

function iconColor(palette: Palette) {
    return palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary;
}

const categoryTabParams: TabParamBase<EmojiTabsInfo>[] = [{
    color: iconColor,
    iconInfo: { name: "History", type: "Common" } as const,
    key: CategoryTabOption.Suggested,
    titleKey: "EmojisSuggested",
}, {
    color: iconColor,
    iconInfo: { name: "RoutineValid", type: "Routine" } as const,
    key: CategoryTabOption.Smileys,
    titleKey: "EmojisSmileys",
}, {
    color: iconColor,
    iconInfo: { name: "Vrooli", type: "Common" } as const,
    key: CategoryTabOption.Nature,
    titleKey: "EmojisNature",
}, {
    color: iconColor,
    iconInfo: { name: "Food", type: "Common" } as const,
    key: CategoryTabOption.Food,
    titleKey: "EmojisFood",
}, {
    color: iconColor,
    iconInfo: { name: "Airplane", type: "Common" } as const,
    key: CategoryTabOption.Places,
    titleKey: "EmojisPlaces",
}, {
    color: iconColor,
    iconInfo: { name: "Award", type: "Common" } as const,
    key: CategoryTabOption.Activities,
    titleKey: "EmojisActivities",
}, {
    color: iconColor,
    iconInfo: { name: "Project", type: "Common" } as const,
    key: CategoryTabOption.Objects,
    titleKey: "EmojisObjects",
}, {
    color: iconColor,
    iconInfo: { name: "Complete", type: "Common" } as const,
    key: CategoryTabOption.Symbols,
    titleKey: "EmojisSymbols",
}, {
    color: iconColor,
    iconInfo: { name: "Report", type: "Common" } as const,
    key: CategoryTabOption.Flags,
    titleKey: "EmojisFlags",
}];

const skinTones = Object.values(SkinTone);

const toneToHex: Record<SkinTone, string> = {
    [SkinTone.Neutral]: "#ffd225",
    [SkinTone.Light]: "#ffdfbd",
    [SkinTone.MediumLight]: "#e9c197",
    [SkinTone.Medium]: "#c88e62",
    [SkinTone.MediumDark]: "#a86637",
    [SkinTone.Dark]: "#60463a",
};

let namesByCode: Record<string, string[] | undefined> = {};
let codeByName: Record<string, string | undefined> = {};
let emojisByCategory: { [key in Exclude<CategoryTabOption, "Suggested">]?: DataEmoji[] } = {};
let emojiByCode: Record<string, DataEmoji | undefined> = {};

class EmojiTools {
    /**
     * Parses a unified emoji string and converts it to a native emoji character. The unified string is a
     * sequence of Unicode code points (hexadecimal numbers) separated by hyphens. Each code point is converted
     * to its corresponding character, and the resulting characters are combined into a single emoji.
     * 
     * @param unified A string representing the unified code of an emoji.
     * @returns The native emoji character.
     * 
     * Example:
     * - Input: "1f1fa-1f1f8"
     * - Output: "🇺🇸"
     * This example takes the unified code for the U.S. flag emoji, splits it into "1f1fa" and "1f1f8",
     * converts these hex codes into their corresponding characters, and then combines them to form the flag emoji.
     */
    static parseNativeEmoji(unified: string): string {
        if (typeof unified !== "string" || unified.length === 0) {
            return "";
        }
        return unified
            .split("-")
            .map(hex => String.fromCodePoint(parseInt(hex, 16)))
            .join("");
    }

    /**
     * Removes any skin tone data from a unified emoji code to normalize it.
     * 
     * @param unified The unified string of an emoji code, potentially including skin tones.
     * @returns The unified code without any skin tone information.
     */
    static unifiedWithoutSkinTone(unified: string): string {
        if (typeof unified !== "string" || unified.length === 0) {
            return "";
        }
        return unified.split("-").filter((section) => !skinTones.includes(section as SkinTone)).join("-");
    }

    /**
     * Computes the unified string for an emoji, optionally applying a specified skin tone.
     * 
     * @param emoji The emoji data object.
     * @param Optional skin tone modifier.
     * @returns The unified string with or without a skin tone modification.
     */
    static emojiUnified(emoji: DataEmoji | SuggestedItem, skinTone?: string): string {
        const unified = emoji[EmojiProperties.unified];

        if (!skinTone || !Array.isArray(emoji[EmojiProperties.variations]) || emoji[EmojiProperties.variations].length === 0) {
            return unified;
        }

        return EmojiTools.emojiVariationUnified(emoji, skinTone) ?? unified;
    }

    /**
     * Selects the appropriate emoji variation based on the given skin tone.
     * @param emoji The base emoji data object.
     * @param Optional skin tone to apply.
     * @returns The unified string of the emoji with the applied skin tone, if available.
     */
    static emojiVariationUnified(
        emoji: DataEmoji | SuggestedItem,
        skinTone?: string,
    ): string | undefined {
        return skinTone
            ? (emoji[EmojiProperties.variations] ?? []).find(variation => variation.includes(skinTone))
            : EmojiTools.emojiUnified(emoji);
    }

    /**
     * Retrieves an emoji object by its unified code.
     * @param unified The unified string of an emoji.
     * @returns The emoji object corresponding to the unified code.
     */
    static emojiByUnified(unified?: string): DataEmoji | undefined {
        if (!unified) {
            return;
        }

        if (emojiByCode[unified]) {
            return emojiByCode[unified];
        }

        const withoutSkinTone = EmojiTools.unifiedWithoutSkinTone(unified);
        return emojiByCode[withoutSkinTone];
    }

    static getEmojiName(unified: string): string {
        const names = namesByCode[unified];
        return Array.isArray(names) && names.length > 0 ? names[0] : "";
    }
}

// Move styled component outside of render scope for better performance
const ClickableEmojiButton = styled("div", {
    name: "ClickableEmojiButton",
})(({ theme }) => ({
    fontSize: "1.5rem",
    cursor: "pointer",
    padding: "8px",
    transition: "background-color 0.2s",
    "&:hover": {
        backgroundColor: theme.palette.action.hover,
    },
    display: "inline-flex",
    alignItems: "center",
    justifyContent: "center",
    width: EMOJI_ROW_HEIGHT,
    height: EMOJI_ROW_HEIGHT,
    userSelect: "none",
    WebkitTapHighlightColor: "transparent", // Remove tap highlight on mobile
    "&:active": {
        backgroundColor: theme.palette.action.selected,
    },
}));

// Memoize handlers at module level to share across instances
function createHandlers(onSelect: (selection: EmojiSelectProps) => unknown, setHoveredEmoji: (emoji: string | null) => unknown) {
    const handlers = new Map<string, {
        handleClick: () => void;
        handleMouseEnter: () => void;
        handleMouseLeave: () => void;
    }>();

    return function getHandlers(unified: string, emoji: string) {
        const key = `${unified}:${emoji}`;
        let cachedHandlers = handlers.get(key);

        if (!cachedHandlers) {
            cachedHandlers = {
                handleClick: () => onSelect({ emoji, unified }),
                handleMouseEnter: () => setHoveredEmoji(unified),
                handleMouseLeave: () => setHoveredEmoji(null),
            };
            handlers.set(key, cachedHandlers);
        }

        return cachedHandlers;
    };
}

/**
 * Optimized clickable emoji component that renders a single emoji with click and hover handlers.
 * Uses module-level memoization and minimal re-renders for maximum performance.
 */
const ClickableEmoji = memo(({
    onSelect,
    unified,
    setHoveredEmoji,
}: {
    onSelect: (selection: EmojiSelectProps) => unknown;
    unified: string;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    // Memoize name and emoji at module level
    const name = useMemo(() => EmojiTools.getEmojiName(unified), [unified]);
    const emoji = useMemo(() => EmojiTools.parseNativeEmoji(unified), [unified]);

    // Get cached handlers
    const { handleClick, handleMouseEnter, handleMouseLeave } = useMemo(
        () => createHandlers(onSelect, setHoveredEmoji)(unified, emoji),
        [onSelect, setHoveredEmoji, unified, emoji],
    );

    return (
        <ClickableEmojiButton
            role="button"
            data-unified={unified}
            aria-label={name}
            data-full-name={namesByCode[unified]}
            onClick={handleClick}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
        >
            {emoji}
        </ClickableEmojiButton>
    );
}, (prevProps, nextProps) => {
    // Custom equality check for maximum performance
    return prevProps.unified === nextProps.unified &&
        prevProps.onSelect === nextProps.onSelect &&
        prevProps.setHoveredEmoji === nextProps.setHoveredEmoji;
});
ClickableEmoji.displayName = "ClickableEmoji";

const SUGGESTED_LS_KEY = "epr_suggested";

type SuggestedItem = {
    unified: string;
    original: string;
    count: number;
};

// Async localStorage operations with in-memory cache
let suggestedCache: SuggestedItem[] | null = null;
let suggestedCachePromise: Promise<SuggestedItem[]> | null = null;

async function getSuggestedAsync(): Promise<SuggestedItem[]> {
    // Return cached value if available
    if (suggestedCache !== null) {
        return suggestedCache;
    }
    
    // Reuse existing promise if already fetching
    if (suggestedCachePromise) {
        return suggestedCachePromise;
    }
    
    suggestedCachePromise = new Promise((resolve) => {
        // Use requestIdleCallback for better performance
        const callback = () => {
            try {
                if (!window?.localStorage) {
                    suggestedCache = [];
                    resolve([]);
                    return;
                }
                const recent = JSON.parse(window.localStorage.getItem(SUGGESTED_LS_KEY) ?? "[]") as SuggestedItem[];
                const sorted = recent.sort((a, b) => b.count - a.count);
                suggestedCache = sorted;
                resolve(sorted);
            } catch {
                suggestedCache = [];
                resolve([]);
            } finally {
                suggestedCachePromise = null;
            }
        };
        
        if ("requestIdleCallback" in window) {
            window.requestIdleCallback(callback);
        } else {
            setTimeout(callback, 0);
        }
    });
    
    return suggestedCachePromise;
}

async function saveSuggestedAsync(items: SuggestedItem[]): Promise<void> {
    suggestedCache = items; // Update cache immediately
    
    return new Promise((resolve) => {
        const callback = () => {
            try {
                if (window?.localStorage) {
                    window.localStorage.setItem(SUGGESTED_LS_KEY, JSON.stringify(items));
                }
            } catch (error) {
                console.error("Failed to save suggested emojis:", error);
            }
            resolve();
        };
        
        if ("requestIdleCallback" in window) {
            window.requestIdleCallback(callback);
        } else {
            setTimeout(callback, 0);
        }
    });
}

// Optimized filter using the search index
function filterEmojiWithIndex(searchString: string, emoji: DataEmoji, searchResults: Set<string> | null): boolean {
    if (!searchString) return true;
    if (!searchResults) return false;
    return searchResults.has(emoji[EmojiProperties.unified]);
}

const variableSizeListStyle = { willChange: "transform" } as const;
const emojiCategoryListStyle = { listStyleType: "none" } as const;
const emojiCategoryTitleStyle = {
    paddingLeft: "8px",
    height: CATEGORY_TITLE_HEIGHT,
    lineHeight: `${CATEGORY_TITLE_HEIGHT}px`,
} as const;

const EmojiCategory = memo(({
    category,
    children,
}: {
    category: CategoryTabOption | `${CategoryTabOption}`;
    children: React.ReactNode;
}) => {
    const { t } = useTranslation();

    return (
        <li
            data-name={category}
            aria-label={t(`Emojis${category}`, { ns: "common", defaultValue: category })}
            style={emojiCategoryListStyle}
        >
            <div style={emojiCategoryTitleStyle}>{t(`Emojis${category}`, { ns: "common", defaultValue: category })}</div>
            <div>{children}</div>
        </li>
    );
});
EmojiCategory.displayName = "EmojiCategory";

const RenderCategory = memo(({
    activeSkinTone,
    category,
    onSelect,
    setHoveredEmoji,
    filteredEmojis,
}: {
    activeSkinTone: SkinTone;
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (selection: EmojiSelectProps) => unknown;
    setHoveredEmoji: (emoji: string | null) => unknown;
    filteredEmojis: (DataEmoji | SuggestedItem)[];
}) => {
    // Pre-calculate grid positions for better performance
    const rows = useMemo(() => {
        const rows: React.ReactNode[][] = [];
        let currentRow: React.ReactNode[] = [];

        for (const emoji of filteredEmojis) {
            if (currentRow.length === EMOJIS_PER_ROW) {
                rows.push([...currentRow]);
                currentRow = [];
            }

            const unified = category === CategoryTabOption.Suggested
                ? (emoji as SuggestedItem).original // Use the original unified code for suggested emojis
                : EmojiTools.emojiUnified(emoji as DataEmoji, activeSkinTone);

            currentRow.push(
                <ClickableEmoji
                    key={unified}
                    unified={unified}
                    onSelect={onSelect}
                    setHoveredEmoji={setHoveredEmoji}
                />,
            );
        }

        if (currentRow.length > 0) {
            rows.push([...currentRow]);
        }

        return rows;
    }, [filteredEmojis, category, activeSkinTone, onSelect, setHoveredEmoji]);

    if (rows.length === 0) return null;

    return (
        <EmojiCategory category={category}>
            {rows.map((row, i) => (
                <EmojiRow key={i}>
                    {row}
                </EmojiRow>
            ))}
        </EmojiCategory>
    );
});
RenderCategory.displayName = "RenderCategory";

interface EmojiPopoverProps extends PopoverProps {
    zIndex: number;
}
const EmojiPopover = styled(Popover, {
    shouldForwardProp: (prop) => prop !== "zIndex",
})<EmojiPopoverProps>(({ theme, zIndex }) => ({
    zIndex: `${zIndex} !important`,
    "& .MuiPopover-paper": {
        background: theme.palette.background.default,
        border: theme.palette.mode === "light" ? "none" : `1px solid ${theme.palette.divider}`,
        borderRadius: theme.spacing(2),
    },
}));

const MainBox = styled("aside")(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    width: EMOJI_GRID_WIDTH,
    height: "100%",
}));

const HeaderBox = styled("div")(() => ({
    position: "relative",
}));

const EmojiRow = styled("div")(() => ({
    display: "flex",
    flexDirection: "row",
    justifyContent: "flex-start",
    gap: 0,
    height: EMOJI_ROW_HEIGHT,
}));

const SearchBarPaper = styled(Paper)(({ theme }) => ({
    padding: "2px 4px",
    display: "flex",
    alignItems: "center",
    borderRadius: "10px",
    boxShadow: "none",
    width: "-webkit-fill-available",
    flex: theme.spacing(1),
    border: theme.palette.mode === "light" ? `1px solid ${theme.palette.divider}` : "none",
})) as typeof Paper;

interface SkinColorPickerProps extends BoxProps {
    isSkinTonePickerOpen: boolean;
}
const SkinColorPicker = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isSkinTonePickerOpen",
})<SkinColorPickerProps>(({ isSkinTonePickerOpen }) => ({
    position: "relative",
    width: isSkinTonePickerOpen ? `${SKIN_OPTION_WIDTH_WITH_SPACING * Object.values(SkinTone).length}px` : `${SKIN_OPTION_WIDTH_WITH_SPACING}px`,
    height: `${SKIN_OPTION_WIDTH_WITH_SPACING}px`,
    transition: "width 0.2s ease-in-out",
    display: "flex",
    flexDirection: "row",
    flexShrink: 0, // Prevent shrinking
    gap: "2px",
}));

interface SkinColorOptionProps extends BoxProps {
    index: number;
    isActive: boolean;
    isSkinTonePickerOpen: boolean;
    skinToneValue: SkinTone;
}
const SkinColorOption = styled(Box, {
    shouldForwardProp: (prop) => prop !== "index" && prop !== "isActive" && prop !== "skinToneValue" && prop !== "isSkinTonePickerOpen",
})<SkinColorOptionProps>(({ index, isActive, isSkinTonePickerOpen, skinToneValue, theme }) => ({
    background: toneToHex[skinToneValue],
    borderRadius: isActive ? theme.spacing(1) : "100%",
    cursor: "pointer",
    width: isActive ? "24px" : "20px",
    height: isActive ? "24px" : "20px",
    position: "absolute",
    left: isSkinTonePickerOpen ?
        `${index * SKIN_OPTION_WIDTH_WITH_SPACING}px` :
        isActive ?
            "0px" :
            "2px",
    top: isActive ? "0px" : "2px",
    transition: "left 0.2s ease-in-out",
    zIndex: isActive ? 1 : 0, // Ensure selected is on top
}));

const EmojiPickerBody = styled(Box)(() => ({
    overflow: "auto",
    height: EMOJI_GRID_HEIGHT,
    width: EMOJI_GRID_WIDTH,
    "&::-webkit-scrollbar": {
        display: "none",
    },
    position: "relative",
}));

const HoveredEmojiBox = styled(Box)(({ theme }) => ({
    position: "absolute",
    display: "flex",
    bottom: 0,
    width: "100%",
    textAlign: "start",
    background: theme.palette.background.paper,
    paddingLeft: theme.spacing(2),
    paddingRight: theme.spacing(2),
}));

const HoveredEmojiDisplay = styled("div")(() => ({
    display: "contents",
    fontSize: "2.5rem",
}));

const HoveredEmojiLabel = styled("div")(({ theme }) => ({
    display: "inline",
    alignSelf: "center",
    marginLeft: theme.spacing(1),
}));

const searchBarInputStyle = { marginLeft: 1 } as const;
const searchIconButtonStyle = {
    width: "48px",
    height: "48px",
} as const;

function getCachedEmojiData() {
    try {
        const cached = localStorage.getItem(EMOJI_CACHE_KEY);
        if (!cached) return null;

        const { version, data } = JSON.parse(cached);
        if (version !== EMOJI_CACHE_VERSION) return null;

        return data;
    } catch (error) {
        console.error("Failed to get cached emoji data:", error);
        return null;
    }
}

function setCachedEmojiData(data: any) {
    try {
        localStorage.setItem(EMOJI_CACHE_KEY, JSON.stringify({
            version: EMOJI_CACHE_VERSION,
            data,
        }));
    } catch (error) {
        console.error("Failed to cache emoji data:", error);
    }
}

// Create a search index for O(1) lookups
type SearchIndex = Map<string, Set<string>>; // searchTerm -> Set of unified codes
let searchIndex: SearchIndex | null = null;
let emojiDataPromise: Promise<any> | null = null;

function buildSearchIndex(namesByCode: Record<string, string[] | undefined>): SearchIndex {
    const index = new Map<string, Set<string>>();
    
    for (const [unified, names] of Object.entries(namesByCode)) {
        if (!Array.isArray(names)) continue;
        
        for (const name of names) {
            const words = name.toLowerCase().split(/\s+/);
            for (const word of words) {
                // Index all prefixes of each word for faster searching
                for (let i = 1; i <= word.length; i++) {
                    const prefix = word.substring(0, i);
                    if (!index.has(prefix)) {
                        index.set(prefix, new Set());
                    }
                    index.get(prefix)!.add(unified);
                }
            }
        }
    }
    
    return index;
}

function searchEmojisByIndex(searchString: string, index: SearchIndex): Set<string> {
    if (!searchString) return new Set();
    
    const searchLower = searchString.toLowerCase();
    const words = searchLower.split(/\s+/).filter(w => w.length > 0);
    
    if (words.length === 0) return new Set();
    
    // Find emojis that match all search words
    let results: Set<string> | null = null;
    
    for (const word of words) {
        const wordMatches = index.get(word) || new Set();
        
        if (results === null) {
            results = new Set(wordMatches);
        } else {
            // Intersection of results with current word matches
            results = new Set([...results].filter(x => wordMatches.has(x)));
        }
        
        if (results.size === 0) break;
    }
    
    return results || new Set();
}

function useEmojiData(shouldLoad: boolean) {
    const [emojiData, setEmojiData] = useState<{
        namesByCode: typeof namesByCode;
        emojisByCategory: typeof emojisByCategory;
        emojiByCode: typeof emojiByCode;
        codeByName: typeof codeByName;
        searchIndex: SearchIndex;
    } | null>(null);

    useEffect(function loadData() {
        if (!shouldLoad) return;
        
        let isMounted = true;

        async function fetchData() {
            // Try to get cached data first
            const cachedData = getCachedEmojiData();
            if (cachedData) {
                if (!isMounted) return;

                // Build search index if not cached
                if (!cachedData.searchIndex) {
                    cachedData.searchIndex = buildSearchIndex(cachedData.namesByCode);
                }
                
                setEmojiData(cachedData);
                namesByCode = cachedData.namesByCode;
                emojisByCategory = cachedData.emojisByCategory;
                emojiByCode = cachedData.emojiByCode;
                codeByName = cachedData.codeByName;
                searchIndex = cachedData.searchIndex;
                return;
            }

            // Reuse existing promise if data is already being fetched
            if (emojiDataPromise) {
                try {
                    const data = await emojiDataPromise;
                    if (isMounted && data) {
                        setEmojiData(data);
                    }
                } catch (error) {
                    console.error("Failed to load emoji data from promise:", error);
                }
                return;
            }

            // Fetch fresh data if no cache
            emojiDataPromise = (async () => {
                try {
                    const [namesResponse, dataResponse] = await Promise.all([
                        fetch("/emojis/locales/en.json"),
                        fetch("/emojis/data.json"),
                    ]);

                    const [namesData, emojiData] = await Promise.all([
                        namesResponse.json(),
                        dataResponse.json(),
                    ]);

                    if (typeof namesData !== "object" || typeof emojiData !== "object") {
                        throw new Error("Invalid emoji data format");
                    }

                    const newNamesByCode: typeof namesByCode = namesData;
                    const newEmojisByCategory: typeof emojisByCategory = emojiData;
                    const newEmojiByCode: typeof emojiByCode = {};
                    const newCodeByName: typeof codeByName = {};

                    // Process data in a single pass
                    for (const code in newNamesByCode) {
                        const names = newNamesByCode[code];
                        if (Array.isArray(names)) {
                            names.forEach(name => {
                                newCodeByName[name] = code;
                            });
                        }
                    }

                    Object.entries(newEmojisByCategory).forEach(([category, emojis]) => {
                        if (!(category in CategoryTabOption) || !Array.isArray(emojis)) {
                            if (process.env.NODE_ENV !== "test") {
                                console.warn(`Invalid category "${category}" or emojis data for category`);
                            }
                            return;
                        }
                        emojis.forEach((emojiData: DataEmoji) => {
                            const unified = emojiData[EmojiProperties.unified];
                            newEmojiByCode[unified] = emojiData;
                        });
                    });

                    // Build search index
                    const newSearchIndex = buildSearchIndex(newNamesByCode);

                    const newData = {
                        namesByCode: newNamesByCode,
                        emojisByCategory: newEmojisByCategory,
                        emojiByCode: newEmojiByCode,
                        codeByName: newCodeByName,
                        searchIndex: newSearchIndex,
                    };

                    // Cache the processed data
                    setCachedEmojiData(newData);

                    namesByCode = newNamesByCode;
                    emojisByCategory = newEmojisByCategory;
                    emojiByCode = newEmojiByCode;
                    codeByName = newCodeByName;
                    searchIndex = newSearchIndex;
                    
                    return newData;
                } catch (error) {
                    console.error("Failed to fetch emoji data:", error);
                    emojiDataPromise = null; // Reset on error
                    throw error;
                }
            })();
            
            try {
                const data = await emojiDataPromise;
                if (isMounted) {
                    setEmojiData(data);
                }
            } catch (error) {
                // Error already logged
            }
        }

        fetchData();

        return () => {
            isMounted = false;
        };
    }, [shouldLoad]);

    return { emojiData };
}

// Memoized skin tone options to avoid recreating handlers in render
const SkinToneOptions = memo(({ 
    activeSkinTone, 
    isSkinTonePickerOpen, 
    setActiveSkinTone, 
    setIsSkinTonePickerOpen,
    t,
}: {
    activeSkinTone: SkinTone;
    isSkinTonePickerOpen: boolean;
    setActiveSkinTone: (tone: SkinTone) => void;
    setIsSkinTonePickerOpen: (open: boolean) => void;
    t: (key: string, options?: Record<string, unknown>) => string;
}) => {
    const handleSkinToneClick = useCallback((skinToneValue: SkinTone, isActive: boolean) => {
        if (isSkinTonePickerOpen) {
            if (!isActive) {
                setActiveSkinTone(skinToneValue);
            }
            setIsSkinTonePickerOpen(false);
        } else {
            setIsSkinTonePickerOpen(true);
        }
    }, [isSkinTonePickerOpen, setActiveSkinTone, setIsSkinTonePickerOpen]);
    
    return (
        <>
            {Object.entries(SkinTone).map(([skinToneKey, skinToneValue], index) => {
                const isActive = skinToneValue === activeSkinTone;
                return (
                    <SkinColorOption
                        aria-pressed={isActive}
                        aria-label={`Skin tone ${t(`EmojiSkinTone${skinToneKey}`, { defaultValue: skinToneKey })}`}
                        index={index}
                        isActive={isActive}
                        isSkinTonePickerOpen={isSkinTonePickerOpen}
                        key={skinToneKey}
                        skinToneValue={skinToneValue}
                        tabIndex={isSkinTonePickerOpen ? 0 : -1}
                        onClick={() => handleSkinToneClick(skinToneValue, isActive)}
                    />
                );
            })}
        </>
    );
});
SkinToneOptions.displayName = "SkinToneOptions";

// Add this new component
const MemoizedHoveredEmojiContent = memo(({ hoveredEmoji }: { hoveredEmoji: string | null }) => {
    const names = hoveredEmoji ? namesByCode[hoveredEmoji] : undefined;
    const displayName = names && Array.isArray(names) && names.length > 0 ? names[0] : "";
    const emojiDisplay = hoveredEmoji ? EmojiTools.parseNativeEmoji(hoveredEmoji) : "‎"; // Empty character to prevent layout shift

    return (
        <>
            <HoveredEmojiDisplay>
                {emojiDisplay}
            </HoveredEmojiDisplay>
            <HoveredEmojiLabel>
                {displayName}
            </HoveredEmojiLabel>
        </>
    );
});
MemoizedHoveredEmojiContent.displayName = "MemoizedHoveredEmojiContent";

export function FallbackEmojiPicker({
    anchorEl,
    onClose,
    onSelect,
    emojiData,
}: FallbackEmojiPickerProps & {
    emojiData: ReturnType<typeof useEmojiData>["emojiData"];
}) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const listRef = useRef<VariableSizeList | null>(null);

    const [searchString, setSearchString] = useState("");
    const searchInputRef = useRef<HTMLInputElement>(null);
    const [debouncedSetSearchString] = useDebounce(setSearchString, SEARCH_STRING_DEBOUNCE_MS);
    
    // Reset search when picker closes
    useEffect(() => {
        if (!anchorEl) {
            setSearchString("");
            if (searchInputRef.current) {
                searchInputRef.current.value = "";
            }
        }
    }, [anchorEl]);

    // Use uncontrolled input with debounced search to avoid re-renders on every keystroke
    const handleSearchInputChange = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        const value = event.target.value;
        debouncedSetSearchString(value);
    }, [debouncedSetSearchString]);
    
    // For voice input
    const handleSearchChange = useCallback((value: string) => {
        if (searchInputRef.current) {
            searchInputRef.current.value = value;
        }
        debouncedSetSearchString(value);
    }, [debouncedSetSearchString]);

    function getInputProps() {
        return {
            enterKeyHint: "search",
        } as const;
    }
    useEffect(function updateListSizeOnSearchStringChange() {
        if (listRef.current) {
            listRef.current.resetAfterIndex(0, true);
        }
    }, [searchString]);

    // Optimize hover state with ref to avoid re-renders
    const [hoveredEmoji, setHoveredEmoji] = useState<string | null>(null);
    const hoveredEmojiRef = useRef<string | null>(null);
    
    const setHoveredEmojiOptimized = useCallback((emoji: string | null) => {
        // Only update state if the value actually changed
        if (hoveredEmojiRef.current !== emoji) {
            hoveredEmojiRef.current = emoji;
            // Batch the state update
            requestAnimationFrame(() => {
                setHoveredEmoji(emoji);
            });
        }
    }, []);
    const [activeSkinTone, setActiveSkinTone] = useState(SkinTone.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);
    const [suggestedEmojis, setSuggestedEmojis] = useState<SuggestedItem[]>([]);

    // Memoize search results using the search index
    const searchResults = useMemo(() => {
        if (!emojiData || !searchString) return null;
        return searchEmojisByIndex(searchString, emojiData.searchIndex);
    }, [emojiData, searchString]);
    
    // Memoize filtered emojis by category with better performance
    const filteredEmojisByCategory = useMemo(() => {
        if (!emojiData) return {};
        if (!searchString) return emojiData.emojisByCategory;

        const result: Record<CategoryTabOption, DataEmoji[]> = {} as Record<CategoryTabOption, DataEmoji[]>;

        Object.entries(emojiData.emojisByCategory).forEach(([category, emojis]) => {
            if (!(category in CategoryTabOption) || !Array.isArray(emojis)) return;
            
            // Filter using the pre-built search index for O(1) lookups
            const filtered = emojis.filter(emoji => 
                filterEmojiWithIndex(searchString, emoji, searchResults),
            );
            
            result[category as CategoryTabOption] = filtered;
        });

        return result;
    }, [emojiData, searchString, searchResults]);

    // Handle categories
    const {
        currTab: activeCategory,
        handleTabChange,
        tabs,
    } = useTabs({ id: "emoji-picker-tabs", tabParams: categoryTabParams, display: "Dialog" });

    useEffect(function getSuggestedEmojisCallback() {
        getSuggestedAsync().then(setSuggestedEmojis);
    }, []);
    const filteredTabs: PageTab<TabParamBase<EmojiTabsInfo>>[] = useMemo(() => {
        return tabs.filter(tab => {
            if (tab.key === CategoryTabOption.Suggested) {
                return suggestedEmojis.length > 0;
            }
            return true;
        });
    }, [suggestedEmojis, tabs]);

    const updateSuggestedEmojisRef = useRef<NodeJS.Timeout>();

    const handleSelect = useCallback(function handleSelectCallback({ emoji, unified }: EmojiSelectProps) {
        // Batch updates using React 18's automatic batching
        // Call onSelect immediately for best responsiveness
        onSelect(emoji);

        // Debounce the suggested emojis update
        if (updateSuggestedEmojisRef.current) {
            clearTimeout(updateSuggestedEmojisRef.current);
        }

        updateSuggestedEmojisRef.current = setTimeout(() => {
            setSuggestedEmojis(prevSuggested => {
                const existingIndex = prevSuggested.findIndex(item => item.unified === unified);
                let newSuggested: SuggestedItem[];

                if (existingIndex >= 0) {
                    // Update count using immutable update
                    newSuggested = [
                        ...prevSuggested.slice(0, existingIndex),
                        { ...prevSuggested[existingIndex], count: prevSuggested[existingIndex].count + 1 },
                        ...prevSuggested.slice(existingIndex + 1),
                    ];
                } else {
                    // Add new item
                    newSuggested = [
                        ...prevSuggested,
                        {
                            unified,
                            original: EmojiTools.unifiedWithoutSkinTone(unified),
                            count: 1,
                        },
                    ];
                }

                // Sort by count descending and limit size
                newSuggested.sort((a, b) => b.count - a.count);
                newSuggested = newSuggested.slice(0, SUGGESTED_EMOJIS_LIMIT);

                // Update storage asynchronously
                saveSuggestedAsync(newSuggested);

                return newSuggested;
            });
        }, SUGGESTED_BATCH_UPDATE_DELAY);
    }, [onSelect]);

    const scrollToCategory = useCallback((categoryKey) => {
        const index = filteredTabs.findIndex(tab => tab.key === categoryKey);
        if (index >= 0 && listRef.current) {
            listRef.current.scrollToItem(index, "start");
        }
    }, [filteredTabs]);

    const setActiveCategory = useCallback(function setActiveCategoryCallback(
        _: ChangeEvent<unknown> | undefined,
        newCategory: PageTab<TabParamBase<EmojiTabsInfo>>,
        disableScroll = false,
    ) {
        if (disableScroll) {
            handleTabChange(_, newCategory);
        } else {
            scrollToCategory(newCategory.key);
        }
    }, [handleTabChange, scrollToCategory]);

    // Pre-calculate and cache category sizes for better performance
    const categorySizes = useMemo(() => {
        const sizes = new Map<string, number>();
        
        filteredTabs.forEach((tab, index) => {
            if (!tab) {
                sizes.set(`${index}`, 0);
                return;
            }

            // Get emojis for this category after filtering
            let numEmojisInCategory = 0;
            if (tab.key === CategoryTabOption.Suggested) {
                if (!searchString) {
                    numEmojisInCategory = suggestedEmojis.length;
                } else {
                    numEmojisInCategory = suggestedEmojis.filter(suggestedItem =>
                        searchResults?.has(suggestedItem.unified) ?? false,
                    ).length;
                }
            } else {
                numEmojisInCategory = filteredEmojisByCategory[tab.key]?.length ?? 0;
            }

            const numberOfRows = Math.ceil(numEmojisInCategory / EMOJIS_PER_ROW);
            const emojisHeight = numberOfRows * EMOJI_ROW_HEIGHT;

            if (emojisHeight === 0) {
                sizes.set(`${index}`, 0);
            } else {
                sizes.set(`${index}`, CATEGORY_TITLE_HEIGHT + emojisHeight);
            }
        });
        
        return sizes;
    }, [filteredTabs, searchString, suggestedEmojis, filteredEmojisByCategory, searchResults]);
    
    const getItemSize = useCallback((index: number) => {
        return categorySizes.get(`${index}`) ?? 0;
    }, [categorySizes]);

    const onItemsRendered = useCallback(({ visibleStartIndex }) => {
        const mostVisibleIndex = visibleStartIndex;
        const mostVisibleTab = filteredTabs[mostVisibleIndex];

        if (mostVisibleTab && mostVisibleTab.key !== activeCategory.key) {
            // Update active category without scrolling
            setActiveCategory(undefined, mostVisibleTab, true);
        }
    }, [filteredTabs, activeCategory, setActiveCategory]);
    
    // Early return if no data and picker is not open
    if (!anchorEl && !emojiData) {
        return null;
    }

    return (
        <EmojiPopover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            zIndex={Z_INDEX.Popup}
        >
            <MainBox
                id="emoji-picker-main"
            >
                <HeaderBox>
                    <Box display="flex" flexDirection="row" alignItems="center" p={1} gap={1}>
                        <SearchBarPaper
                            component="form"
                        >
                            <Input
                                id="emoji-search-input"
                                inputRef={searchInputRef}
                                disableUnderline={true}
                                fullWidth={true}
                                inputProps={getInputProps()}
                                onChange={handleSearchInputChange}
                                placeholder={t("Search") + "..."}
                                disabled={emojiData === null}
                                sx={searchBarInputStyle}
                            />
                            <MicrophoneButton
                                onTranscriptChange={handleSearchChange}
                                disabled={emojiData === null}
                            />
                            <IconButton
                                aria-label={t("Search")}
                                disabled={emojiData === null}
                                variant="transparent"
                                sx={searchIconButtonStyle}
                            >
                                <IconCommon
                                    decorative
                                    fill={palette.background.textSecondary}
                                    name="Search"
                                />
                            </IconButton>
                        </SearchBarPaper>
                        <SkinColorPicker
                            isSkinTonePickerOpen={isSkinTonePickerOpen}
                        >
                            <SkinToneOptions
                                activeSkinTone={activeSkinTone}
                                isSkinTonePickerOpen={isSkinTonePickerOpen}
                                setActiveSkinTone={setActiveSkinTone}
                                setIsSkinTonePickerOpen={setIsSkinTonePickerOpen}
                                t={t}
                            />
                        </SkinColorPicker>
                    </Box>
                    <PageTabs
                        ariaLabel="emoji-picker-tabs"
                        fullWidth
                        id="emoji-picker-tabs"
                        currTab={activeCategory}
                        onChange={setActiveCategory}
                        tabs={filteredTabs}
                    />
                </HeaderBox>
                <EmojiPickerBody id="emoji-picker-body">
                    {emojiData === null ? (
                        <LoadingContainer>
                            <CircularProgress size={40} />
                        </LoadingContainer>
                    ) : (
                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                        // @ts-ignore - VariableSizeList can be used as a JSX component despite the TypeScript error
                        <VariableSizeList
                            ref={listRef}
                            height={EMOJI_GRID_HEIGHT}
                            width={EMOJI_GRID_WIDTH}
                            itemCount={filteredTabs.length}
                            itemSize={getItemSize}
                            onItemsRendered={onItemsRendered}
                            overscanCount={2}
                            style={variableSizeListStyle}
                            useIsScrolling
                        >
                            {({ index, style }) => {
                                const tab = filteredTabs[index];
                                if (!tab) return null;

                                return (
                                    <div style={style}>
                                        {tab.key === CategoryTabOption.Suggested ? (
                                            <RenderCategory
                                                key={tab.key}
                                                activeSkinTone={activeSkinTone}
                                                category={tab.key}
                                                onSelect={handleSelect}
                                                setHoveredEmoji={setHoveredEmojiOptimized}
                                                filteredEmojis={searchString ? suggestedEmojis.filter(item => searchResults?.has(item.unified) ?? false) : suggestedEmojis}
                                            />
                                        ) : (
                                            <RenderCategory
                                                key={tab.key}
                                                activeSkinTone={activeSkinTone}
                                                category={tab.key}
                                                onSelect={handleSelect}
                                                setHoveredEmoji={setHoveredEmojiOptimized}
                                                filteredEmojis={filteredEmojisByCategory[tab.key] ?? emptyArray}
                                            />
                                        )}
                                    </div>
                                );
                            }}
                        </VariableSizeList>
                    )}
                    {emojiData && (
                        <HoveredEmojiBox>
                            <MemoizedHoveredEmojiContent hoveredEmoji={hoveredEmoji} />
                        </HoveredEmojiBox>
                    )}
                </EmojiPickerBody>
            </MainBox>
        </EmojiPopover>
    );
}

const addEmojiIconButtonStyle = { borderRadius: 0, background: "transparent" } as const;
const hiddenInputStyle = { position: "absolute", opacity: 0, pointerEvents: "none", width: 0, height: 0 } as const;

export function EmojiPicker({
    disableNative,
    onSelect,
    onOpen,
    onClose: onCloseProp,
}: {
    disableNative?: boolean;
    onSelect: (emoji: string) => unknown;
    onOpen?: () => void;
    onClose?: () => void;
}) {
    const { t } = useTranslation();

    // Only load emoji data when picker is opened
    const [hasOpened, setHasOpened] = useState(false);
    const { emojiData } = useEmojiData(hasOpened);

    // Hidden input for native emoji picker
    const inputRef = useRef<HTMLInputElement>(null);
    const [anchorEl, setAnchorEl] = useState<Element | null>(null);

    function supportsNativeEmojiPicker() {
        // See https://developer.mozilla.org/en-US/docs/Web/API/HTMLInputElement/showPicker
        // return "showPicker" in HTMLInputElement.prototype; // Currently doesn't work on any browser. For now, showPicker only supports, dates, colors, etc. Basically everything except emojis.
        return disableNative === true;
    }

    function handleButtonClick(event: React.MouseEvent<Element>) {
        onOpen?.();
        setHasOpened(true); // Trigger data loading
        // Get target now, or it will be null after the timeout
        const anchorEl = event.currentTarget;

        // Fallback to custom emoji picker if native picker is not supported
        if (!inputRef.current || !supportsNativeEmojiPicker()) {
            setAnchorEl(anchorEl);
            return;
        }

        let pickerOpened = false;

        function onInputChange() {
            pickerOpened = true;
            inputRef.current?.removeEventListener("input", onInputChange);
        }

        inputRef.current.addEventListener("input", onInputChange);
        inputRef.current.focus();

        try {
            inputRef.current.showPicker();
        } catch (error) {
            console.error("showPicker failed:", error);
            inputRef.current.removeEventListener("input", onInputChange);
            // Fallback to custom emoji picker
            setAnchorEl(anchorEl);
            return;
        }

        setTimeout(function fallbackAfterTimeout() {
            if (pickerOpened) {
                return;
            }
            // Picker didn't open or user didn't select an emoji
            inputRef.current?.removeEventListener("input", onInputChange);
            // Fallback to custom emoji picker
            setAnchorEl(anchorEl);
        }, NATIVE_PICKER_FAIL_TIMEOUT_MS);
    }

    function handleInputChange(event: React.ChangeEvent<HTMLInputElement>) {
        const emoji = event.target.value;
        onSelect(emoji);
        event.target.value = ""; // Reset the input
    }

    function handleCustomPickerSelect(emoji: string) {
        onSelect(emoji);
        setAnchorEl(null);
    }

    function handleCustomPickerClose() {
        setAnchorEl(null);
        onCloseProp?.();
    }

    return (
        <>
            <IconButton
                aria-label={t("Add")}
                size="sm"
                variant="transparent"
                style={addEmojiIconButtonStyle}
                onClick={handleButtonClick}
            >
                <IconCommon
                    decorative
                    name="Add"
                />
            </IconButton>
            <input
                ref={inputRef}
                type="text"
                style={hiddenInputStyle}
                onChange={handleInputChange}
            />
            <FallbackEmojiPicker
                anchorEl={anchorEl}
                onClose={handleCustomPickerClose}
                onSelect={handleCustomPickerSelect}
                emojiData={emojiData}
            />
        </>
    );
}

const LoadingContainer = styled(Box)(() => ({
    position: "absolute",
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    display: "flex",
    flexDirection: "column",
    justifyContent: "center",
    alignItems: "center",
    gap: 2,
}));

// Export helper functions for testing
export const parseNativeEmoji = EmojiTools.parseNativeEmoji;
export const unifiedWithoutSkinTone = EmojiTools.unifiedWithoutSkinTone;
export const emojiUnified = EmojiTools.emojiUnified;
export const emojiVariationUnified = EmojiTools.emojiVariationUnified;
