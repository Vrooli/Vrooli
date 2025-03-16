import { Box, BoxProps, Button, IconButton, Input, Palette, Paper, Popover, PopoverProps, styled, useTheme } from "@mui/material";
import { ChangeEvent, memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { VariableSizeList } from "react-window";
import { useDebounce } from "../../hooks/useDebounce.js";
import { PageTab, useTabs } from "../../hooks/useTabs.js";
import { AddIcon, AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "../../icons/common.js";
import { Z_INDEX } from "../../utils/consts.js";
import { TabParamBase } from "../../utils/search/objectToSearch.js";
import { PageTabs } from "../PageTabs/PageTabs.js";
import { MicrophoneButton } from "../buttons/MicrophoneButton/MicrophoneButton.js";
// import emojis from "./data/emojis";

// const KNOWN_FAILING_EMOJIS = ["2640-fe0f", "2642-fe0f", "2695-fe0f"];
const SUGGESTED_EMOJIS_LIMIT = 20;
const SKIN_OPTION_WIDTH_WITH_SPACING = 28;
const SEARCH_STRING_DEBOUNCE_MS = 200;
const NATIVE_PICKER_FAIL_TIMEOUT_MS = 500;

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

function iconColor(palette: Palette) {
    return palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary;
}

const categoryTabParams: TabParamBase<EmojiTabsInfo>[] = [{
    color: iconColor,
    Icon: HistoryIcon,
    key: CategoryTabOption.Suggested,
    titleKey: "EmojisSuggested",
}, {
    color: iconColor,
    Icon: RoutineValidIcon,
    key: CategoryTabOption.Smileys,
    titleKey: "EmojisSmileys",
}, {
    color: iconColor,
    Icon: VrooliIcon,
    key: CategoryTabOption.Nature,
    titleKey: "EmojisNature",
}, {
    color: iconColor,
    Icon: FoodIcon,
    key: CategoryTabOption.Food,
    titleKey: "EmojisFood",
}, {
    color: iconColor,
    Icon: AirplaneIcon,
    key: CategoryTabOption.Places,
    titleKey: "EmojisPlaces",
}, {
    color: iconColor,
    Icon: AwardIcon,
    key: CategoryTabOption.Activities,
    titleKey: "EmojisActivities",
}, {
    color: iconColor,
    Icon: ProjectIcon,
    key: CategoryTabOption.Objects,
    titleKey: "EmojisObjects",
}, {
    color: iconColor,
    Icon: CompleteIcon,
    key: CategoryTabOption.Symbols,
    titleKey: "EmojisSymbols",
}, {
    color: iconColor,
    Icon: ReportIcon,
    key: CategoryTabOption.Flags,
    titleKey: "EmojisFlags",
}];

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
const skinTones = Object.values(SkinTone);

const toneToHex: Record<SkinTone, string> = {
    [SkinTone.Neutral]: "#ffd225",
    [SkinTone.Light]: "#ffdfbd",
    [SkinTone.MediumLight]: "#e9c197",
    [SkinTone.Medium]: "#c88e62",
    [SkinTone.MediumDark]: "#a86637",
    [SkinTone.Dark]: "#60463a",
};

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
export function parseNativeEmoji(unified: string): string {
    if (typeof unified !== "string" || unified.length === 0) {
        return "";
    }
    return unified
        .split("-")
        .map(hex => String.fromCodePoint(parseInt(hex, 16)))
        .join("");
}

let namesByCode: Record<string, string[] | undefined> = {};
const codeByName: Record<string, string | undefined> = {};
let emojisByCategory: { [key in Exclude<CategoryTabOption, "Suggested">]?: DataEmoji[] } = {};
const emojiByCode: Record<string, DataEmoji | undefined> = {};

/**
 * Removes any skin tone data from a unified emoji code to normalize it.
 * @param unified - The unified string of an emoji code, potentially including skin tones.
 * @returns The unified code without any skin tone information.
 */
export function unifiedWithoutSkinTone(unified: string): string {
    return unified.split("-").filter((section) => !skinTones.includes(section as SkinTone)).join("-");
}

/**
 * Computes the unified string for an emoji, optionally applying a specified skin tone.
 * @param emoji The emoji data object.
 * @param Optional skin tone modifier.
 * @returns The unified string with or without a skin tone modification.
 */
export function emojiUnified(emoji: DataEmoji, skinTone?: string): string {
    const unified = emoji[EmojiProperties.unified];

    if (!skinTone || !Array.isArray(emoji[EmojiProperties.variations]) || emoji[EmojiProperties.variations].length === 0) {
        return unified;
    }

    return emojiVariationUnified(emoji, skinTone) ?? unified;
}

/**
 * Selects the appropriate emoji variation based on the given skin tone.
 * @param emoji The base emoji data object.
 * @param Optional skin tone to apply.
 * @returns The unified string of the emoji with the applied skin tone, if available.
 */
export function emojiVariationUnified(
    emoji: DataEmoji,
    skinTone?: string,
): string | undefined {
    return skinTone
        ? (emoji[EmojiProperties.variations] ?? []).find(variation => variation.includes(skinTone))
        : emojiUnified(emoji);
}

/**
 * Retrieves an emoji object by its unified code.
 * @param unified The unified string of an emoji.
 * @returns The emoji object corresponding to the unified code.
 */
export function emojiByUnified(unified?: string): DataEmoji | undefined {
    if (!unified) {
        return;
    }

    if (emojiByCode[unified]) {
        return emojiByCode[unified];
    }

    const withoutSkinTone = unifiedWithoutSkinTone(unified);
    return emojiByCode[withoutSkinTone];
}

const clickableEmojiButtonStyle = { fontSize: "1.5rem" } as const;
/**
 * Component representing a clickable emoji button in the UI.
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
    const names = namesByCode[unified];
    const name = Array.isArray(names) && names.length > 0 ? names[0] : "";

    function handleClick() {
        onSelect({ emoji: parseNativeEmoji(unified), unified });
    }
    function handleMouseEnter() {
        setHoveredEmoji(unified);
    }
    function handleMouseLeave() {
        setHoveredEmoji(null);
    }

    return (
        <Button
            data-unified={unified}
            aria-label={name}
            data-full-name={names}
            onClick={handleClick}
            onMouseEnter={handleMouseEnter}
            onMouseLeave={handleMouseLeave}
            sx={clickableEmojiButtonStyle}
        >
            {parseNativeEmoji(unified)}
        </Button>
    );
});
ClickableEmoji.displayName = "ClickableEmoji";

const SUGGESTED_LS_KEY = "epr_suggested";

type SuggestedItem = {
    unified: string;
    original: string;
    count: number;
};

function getSuggested(): SuggestedItem[] {
    try {
        if (!window?.localStorage) {
            return [];
        }
        const recent = JSON.parse(window?.localStorage.getItem(SUGGESTED_LS_KEY) ?? "[]") as SuggestedItem[];
        return recent.sort((a, b) => b.count - a.count);
    } catch {
        return [];
    }
}

function filterEmoji(searchString: string, emoji: DataEmoji): boolean {
    if (!searchString) return true;
    const searchStrLower = searchString.toLowerCase();
    const names = namesByCode[emoji[EmojiProperties.unified]];
    if (!Array.isArray(names)) return false;
    return names.some(name => name.toLowerCase().includes(searchStrLower));
}

const Suggested = memo(({
    category,
    onSelect,
    searchString,
    setHoveredEmoji,
}: {
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (selection: EmojiSelectProps) => unknown;
    searchString: string;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    const suggested = useMemo(() => getSuggested(), []);
    const filteredSuggested = useMemo(() => {
        if (!searchString) {
            return suggested;
        }
        return suggested.filter(suggestedItem => filterEmoji(searchString, { [EmojiProperties.unified]: suggestedItem.unified }));
    }, [suggested, searchString]);

    if (filteredSuggested.length === 0) {
        return null;
    }
    return (
        <EmojiCategory
            category={category}
        >
            {filteredSuggested.map((suggestedItem) => {
                // Don't render emoji if it fails to parse
                const emoji = emojiByUnified(suggestedItem.original);
                if (!emoji) {
                    return null;
                }

                return (
                    <ClickableEmoji
                        key={suggestedItem.unified}
                        unified={suggestedItem.unified}
                        onSelect={onSelect}
                        setHoveredEmoji={setHoveredEmoji}
                    />
                );
            })}
        </EmojiCategory>
    );
});
Suggested.displayName = "Suggested";

const emojiCategoryListStyle = { listStyleType: "none" } as const;
const emojiCategoryTitleStyle = { paddingLeft: "8px" } as const;

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
    searchString,
    setHoveredEmoji,
}: {
    activeSkinTone: SkinTone;
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (selection: EmojiSelectProps) => unknown;
    searchString: string;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    const emojis = useMemo(() => {
        const emojisToPush = emojisByCategory[category] ?? [];

        return emojisToPush
            // .filter(emoji => (!emoji.a || !MINIMUM_VERSION) || (emoji.a >= MINIMUM_VERSION))
            .filter(emoji => filterEmoji(searchString, emoji))
            .map(emoji => {
                const unified = emojiUnified(emoji, activeSkinTone);
                return (
                    <ClickableEmoji
                        key={unified}
                        unified={unified}
                        onSelect={onSelect}
                        setHoveredEmoji={setHoveredEmoji}
                    />
                );
            });
    }, [category, searchString, activeSkinTone, onSelect, setHoveredEmoji]);

    if (emojis.length === 0) {
        return null;
    }
    return (
        <EmojiCategory category={category}>
            {emojis}
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
        // width: "min(100%, 410px)",
        // height: "min(100%, 410px)",
    },
}));

const MainBox = styled(Box)(() => ({
    position: "relative",
    display: "flex",
    flexDirection: "column",
    width: "100%",
    height: "100%",
}));

const HeaderBox = styled(Box)(() => ({
    position: "relative",
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
}));

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
    overflow: "scroll",
    "&::-webkit-scrollbar": {
        display: "none",
    },
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

export function FallbackEmojiPicker({
    anchorEl,
    onClose,
    onSelect,
}: FallbackEmojiPickerProps) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const listRef = useRef<VariableSizeList | null>(null);

    const [searchString, setSearchString] = useState("");
    const [internalSearchString, setInternalSearchString] = useState("");
    const [debouncedSetSearchString] = useDebounce(setSearchString, SEARCH_STRING_DEBOUNCE_MS);
    function handleSearchChange(value: string) {
        setInternalSearchString(value);
        debouncedSetSearchString(value);
    }
    function handleSearchInputChange(event: React.ChangeEvent<HTMLInputElement>) {
        const value = event.target.value;
        setInternalSearchString(value);
        debouncedSetSearchString(value);
    }
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

    const [hoveredEmoji, setHoveredEmoji] = useState<string | null>(null);
    const [activeSkinTone, setActiveSkinTone] = useState(SkinTone.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);
    const [suggestedEmojis, setSuggestedEmojis] = useState<SuggestedItem[]>([]);

    // State for loading
    const [isLoading, setIsLoading] = useState(true);

    useEffect(function loadData() {
        async function fetchData() {
            try {
                // Fetch emoji names and data
                const [namesResponse, dataResponse] = await Promise.all([
                    fetch("/emojis/locales/en.json"),
                    fetch("/emojis/data.json"),
                ]);
                const [namesData, emojiData] = await Promise.all([
                    namesResponse.json(),
                    dataResponse.json(),
                ]);
                if (typeof namesData !== "object" || typeof emojiData !== "object") {
                    console.error("Failed to fetch emoji data: response was not an object");
                    return;
                }
                emojisByCategory = emojiData;
                namesByCode = namesData;

                // Iterate over the namesByCode to create the codeByName mapping
                for (const code in namesByCode) {
                    const names = namesByCode[code];
                    if (Array.isArray(names)) {
                        names.forEach(name => {
                            codeByName[name] = code;
                        });
                    }
                }
                // Iterate over categories to create the emojiByCode mapping
                Object.entries(emojisByCategory).forEach(([category, emojis]) => {
                    if (!(category in CategoryTabOption) || !Array.isArray(emojis)) {
                        console.warn(`Invalid category "${category}" or emojis data for category`);
                        return;
                    }
                    emojis.map((emojiData: DataEmoji) => {
                        const unified = emojiData[EmojiProperties.unified];
                        emojiByCode[unified] = emojiData;
                    });
                });

                setIsLoading(false);
            } catch (error) {
                console.error("Failed to fetch emoji data", error);
            }
        }

        fetchData();
    }, []);

    // Handle categories
    const {
        currTab: activeCategory,
        handleTabChange,
        tabs,
    } = useTabs({ id: "emoji-picker-tabs", tabParams: categoryTabParams, display: "dialog" });

    useEffect(function getSuggestedEmojisCallback() {
        setSuggestedEmojis(getSuggested());
    }, []);
    const filteredTabs: PageTab<TabParamBase<EmojiTabsInfo>>[] = useMemo(() => {
        return tabs.filter(tab => {
            if (tab.key === CategoryTabOption.Suggested) {
                return suggestedEmojis.length > 0;
            }
            return true;
        });
    }, [suggestedEmojis, tabs]);

    const handleSelect = useCallback(function handleSelectCallback({ emoji, unified }: EmojiSelectProps) {
        setSuggestedEmojis(prevSuggested => {
            const existingIndex = prevSuggested.findIndex(item => item.unified === unified);
            let newSuggested: SuggestedItem[];

            if (existingIndex >= 0) {
                // Update count
                newSuggested = [...prevSuggested];
                newSuggested[existingIndex].count += 1;
            } else {
                // Add new item
                newSuggested = [
                    ...prevSuggested,
                    {
                        unified,
                        original: unifiedWithoutSkinTone(unified),
                        count: 1,
                    },
                ];
            }

            // Sort by count descending
            newSuggested.sort((a, b) => b.count - a.count);

            // Limit the number of suggested emojis
            newSuggested = newSuggested.slice(0, SUGGESTED_EMOJIS_LIMIT);

            // Update local storage
            window.localStorage.setItem(SUGGESTED_LS_KEY, JSON.stringify(newSuggested));

            return newSuggested;
        });

        onSelect(emoji);
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

    const getItemSize = useCallback((index: number) => {
        const tab = filteredTabs[index];

        // Height for the category title
        const categoryTitleHeight = 24;

        // Get emojis for this category after filtering
        let numEmojisInCategory = 0;
        if (tab.key === CategoryTabOption.Suggested) {
            numEmojisInCategory = getSuggested().filter(suggestedItem => filterEmoji(searchString, { [EmojiProperties.unified]: suggestedItem.unified })).length;
        } else {
            numEmojisInCategory = (emojisByCategory[tab.key]?.filter(emoji => filterEmoji(searchString, emoji)) ?? []).length;
        }

        // Calculate the number of rows needed for the emojis
        const emojisPerRow = 6;
        const emojiRowHeight = 54;

        const numberOfRows = Math.ceil(numEmojisInCategory / emojisPerRow);
        const emojisHeight = numberOfRows * emojiRowHeight;

        if (emojisHeight === 0) {
            return 0;
        }
        return categoryTitleHeight + emojisHeight;
    }, [filteredTabs, searchString]);

    const onItemsRendered = useCallback(({ visibleStartIndex }) => {
        const mostVisibleIndex = visibleStartIndex;
        const mostVisibleTab = filteredTabs[mostVisibleIndex];

        if (mostVisibleTab && mostVisibleTab.key !== activeCategory.key) {
            // Update active category without scrolling
            setActiveCategory(undefined, mostVisibleTab, true);
        }
    }, [filteredTabs, activeCategory, setActiveCategory]);

    return (
        <EmojiPopover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            zIndex={Z_INDEX.Popup}
        >
            <MainBox
                id="emoji-picker-main"
                component="aside"
            >
                <HeaderBox>
                    <Box display="flex" flexDirection="row" alignItems="center" padding={1} gap={1}>
                        <SearchBarPaper
                            // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                            // @ts-ignore
                            action="#" // Needed for iOS to accept enterKeyHint: https://stackoverflow.com/a/39485162/406725
                            component="form"
                        >
                            <Input
                                id="emoji-search-input"
                                disableUnderline={true}
                                fullWidth={true}
                                inputProps={getInputProps()}
                                value={internalSearchString}
                                onChange={handleSearchInputChange}
                                placeholder={t("Search") + "..."}
                                sx={searchBarInputStyle}
                            />
                            <MicrophoneButton onTranscriptChange={handleSearchChange} />
                            <IconButton sx={searchIconButtonStyle} aria-label="main-search-icon">
                                <SearchIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </SearchBarPaper>
                        {/* Skin tone picker */}
                        <SkinColorPicker isSkinTonePickerOpen={isSkinTonePickerOpen}>
                            {Object.entries(SkinTone).map(([skinToneKey, skinToneValue], index) => {
                                const isActive = skinToneValue === activeSkinTone;
                                function handleClick() {
                                    if (isSkinTonePickerOpen) {
                                        if (!isActive) {
                                            setActiveSkinTone(skinToneValue);
                                        }
                                        setIsSkinTonePickerOpen(false);
                                    } else {
                                        setIsSkinTonePickerOpen(true);
                                    }
                                }

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
                                        onClick={handleClick}
                                    />
                                );
                            })}
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
                    {isLoading && <div>Loading...</div>}
                    {!isLoading && (
                        // TODO Remove this once @types/react-window is updated
                        // @ts-expect-error Incompatible JSX type definitions
                        <VariableSizeList
                            ref={listRef}
                            height={320}
                            width={400}
                            itemCount={filteredTabs.length}
                            itemSize={getItemSize}
                            onItemsRendered={onItemsRendered}
                            overscanCount={2}
                        >
                            {({ index, style }) => {
                                const tab = filteredTabs[index];

                                return (
                                    <div style={style}>
                                        {tab.key === CategoryTabOption.Suggested ? (
                                            <Suggested
                                                key={tab.key}
                                                category={tab.key}
                                                onSelect={handleSelect}
                                                searchString={searchString}
                                                setHoveredEmoji={setHoveredEmoji}
                                            />
                                        ) : (
                                            <RenderCategory
                                                key={tab.key}
                                                activeSkinTone={activeSkinTone}
                                                category={tab.key}
                                                onSelect={handleSelect}
                                                searchString={searchString}
                                                setHoveredEmoji={setHoveredEmoji}
                                            />
                                        )}
                                    </div>
                                );
                            }}
                        </VariableSizeList>
                    )}
                    <HoveredEmojiBox>
                        <HoveredEmojiDisplay>
                            {hoveredEmoji ? parseNativeEmoji(hoveredEmoji) : "‎"}
                        </HoveredEmojiDisplay>
                        <HoveredEmojiLabel>
                            {/* eslint-disable-next-line @typescript-eslint/no-non-null-assertion */}
                            {hoveredEmoji && Array.isArray(namesByCode[hoveredEmoji]) && namesByCode[hoveredEmoji]!.length > 0 ? namesByCode[hoveredEmoji]![0] : ""}
                        </HoveredEmojiLabel>
                    </HoveredEmojiBox>
                </EmojiPickerBody>
            </MainBox>
        </EmojiPopover>
    );
}

const addEmojiIconButtonStyle = { borderRadius: 0, background: "transparent" } as const;
const hiddenInputStyle = { position: "absolute", opacity: 0, pointerEvents: "none", width: 0, height: 0 } as const;

export function EmojiPicker({
    onSelect,
}: {
    onSelect: (emoji: string) => unknown;
}) {
    // Hidden input for native emoji picker
    const inputRef = useRef<HTMLInputElement>(null);
    const [anchorEl, setAnchorEl] = useState<Element | null>(null);

    function supportsNativeEmojiPicker() {
        // See https://developer.mozilla.org/en-US/docs/Web/API/HTMLInputElement/showPicker
        // return "showPicker" in HTMLInputElement.prototype; // Currently doesn't work on any browser. For now, showPicker only supports, dates, colors, etc. Basically everything except emojis.
        return false;
    }

    function handleButtonClick(event: React.MouseEvent<Element>) {
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
    }

    return (
        <>
            <IconButton
                size="small"
                style={addEmojiIconButtonStyle}
                onClick={handleButtonClick}
            >
                <AddIcon />
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
            />
        </>
    );
}
