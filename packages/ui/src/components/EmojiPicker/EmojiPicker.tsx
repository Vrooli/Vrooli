import { Box, BoxProps, Button, IconButton, Input, Palette, Paper, Popover, PopoverProps, styled, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { PageTab, useTabs } from "hooks/useTabs";
import { useZIndex } from "hooks/useZIndex";
import { AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "icons";
import { memo, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { TabParam } from "utils/search/objectToSearch";
// import emojis from "./data/emojis";

// const KNOWN_FAILING_EMOJIS = ["2640-fe0f", "2642-fe0f", "2695-fe0f"];
const Z_INDEX_OFFSET = 1000;
const PERCENTS = 100;
const SUGGESTED_EMOJIS_LIMIT = 20;
const SKIN_OPTION_WIDTH_WITH_SPACING = 28;

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
    IsSearchable: false;
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

const categoryTabParams: TabParam<EmojiTabsInfo>[] = [{
    color: iconColor,
    Icon: HistoryIcon,
    key: "Suggested",
    titleKey: "EmojisSuggested",
}, {
    color: iconColor,
    Icon: RoutineValidIcon,
    key: "Smileys",
    titleKey: "EmojisSmileys",
}, {
    color: iconColor,
    Icon: VrooliIcon,
    key: "Nature",
    titleKey: "EmojisNature",
}, {
    color: iconColor,
    Icon: FoodIcon,
    key: "Food",
    titleKey: "EmojisFood",
}, {
    color: iconColor,
    Icon: AirplaneIcon,
    key: "Places",
    titleKey: "EmojisPlaces",
}, {
    color: iconColor,
    Icon: AwardIcon,
    key: "Activities",
    titleKey: "EmojisActivities",
}, {
    color: iconColor,
    Icon: ProjectIcon,
    key: "Objects",
    titleKey: "EmojisObjects",
}, {
    color: iconColor,
    Icon: CompleteIcon,
    key: "Symbols",
    titleKey: "EmojisSymbols",
}, {
    color: iconColor,
    Icon: ReportIcon,
    key: "Flags",
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
/**
 * Loads emoji names
 */
async function fetchEmojiNames() {
    const response = await fetch("/emojis/locales/en.json");
    if (!response.ok) {
        console.error("Failed to fetch emoji names: response was not ok");
        return;
    }
    const data = await response.json();
    if (typeof data !== "object") {
        console.error("Failed to fetch emoji names: response was not an object");
        return;
    }
    namesByCode = data;
    // Iterate over the namesByCode to create the codeByName mapping
    for (const code in namesByCode) {
        const names = namesByCode[code];
        if (Array.isArray(names)) {
            names.forEach(name => {
                codeByName[name] = code;
            });
        }
    }
}
if (process.env.NODE_ENV !== "test") {
    fetchEmojiNames();
}

let emojisByCategory: { [key in Exclude<CategoryTabOption, "Suggested">]?: DataEmoji[] } = {};
const emojiByCode: Record<string, DataEmoji | undefined> = {};
/**
 * Loads emoji data
 */
async function fetchEmojiData() {
    const response = await fetch("/emojis/data.json");
    if (!response.ok) {
        console.error("Failed to fetch emoji data: response was not ok");
        return;
    }
    const data = await response.json();
    if (typeof data !== "object") {
        console.error("Failed to fetch emoji data: response was not an object");
        return;
    }
    emojisByCategory = data;
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
}
if (process.env.NODE_ENV !== "test") {
    fetchEmojiData();
}

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
        width: "min(100%, 410px)",
        height: "min(100%, 410px)",
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

const FullEmojiList = styled("ul")(({ theme }) => ({
    padding: theme.spacing(1),
    margin: 0,
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

export function EmojiPicker({
    anchorEl,
    onClose,
    onSelect,
}: {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
    onSelect: (emoji: string) => unknown;
}) {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const zIndex = useZIndex(Boolean(anchorEl), false, Z_INDEX_OFFSET);

    const [searchString, setSearchString] = useState("");
    function handleChange(value: string) {
        setSearchString(value);
    }

    const [hoveredEmoji, setHoveredEmoji] = useState<string | null>(null);
    const [activeSkinTone, setActiveSkinTone] = useState(SkinTone.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);
    const [suggestedEmojis, setSuggestedEmojis] = useState<SuggestedItem[]>([]);

    // Handle categories
    const {
        currTab: activeCategory,
        handleTabChange,
        tabs,
    } = useTabs({ id: "emoji-picker-tabs", tabParams: categoryTabParams, display: "dialog" });

    function scrollCategoryIntoView(key: string) {
        const scrollContainer = scrollRef.current;
        if (!scrollContainer) return;

        const categoryDiv = scrollContainer.querySelector(`[data-name="${key}"]`);
        if (categoryDiv) {
            // Scroll to container, minus some spacing to account for the header
            const HEADER_HEIGHT_APPROX = 120;
            const categoryTop = (categoryDiv as HTMLElement).offsetTop - HEADER_HEIGHT_APPROX;
            scrollContainer.scrollTo({ top: categoryTop, behavior: "smooth" });
        } else {
            console.error(`Failed to scroll category into view: category "${key}" not found`);
        }
    }

    const setActiveCategory = useCallback(function setActiveCategoryCallback(_: unknown, newCategory: PageTab<EmojiTabsInfo>, disableScroll = false) {
        if (disableScroll) {
            handleTabChange(_, newCategory);
        } else {
            scrollCategoryIntoView(newCategory.key);
        }
    }, [handleTabChange]);

    useEffect(function getSuggestedEmojisCallback() {
        setSuggestedEmojis(getSuggested());
    }, []);
    const filteredTabs: PageTab<EmojiTabsInfo>[] = useMemo(() => {
        return tabs.filter(tab => {
            if (tab.key === CategoryTabOption.Suggested) {
                return suggestedEmojis.length > 0;
            }
            return true;
        }) as PageTab<EmojiTabsInfo>[];
    }, [suggestedEmojis, tabs]);

    // Change active category based on scroll position
    const scrollRef = useRef<HTMLDivElement>(null);
    function onScroll() {
        const scrollContainer = scrollRef.current;
        if (!scrollContainer) return;

        const scrollContainerRect = scrollContainer.getBoundingClientRect();
        const scrollContainerTop = scrollContainerRect.top;
        const scrollContainerBottom = scrollContainerRect.bottom;

        const categoryDivs = Array.from(scrollContainer.querySelectorAll("li"));

        let maxVisiblePercentage = 0;
        let mostVisibleCategoryKey: string = activeCategory.key;

        categoryDivs.forEach((div) => {
            const divRect = div.getBoundingClientRect();
            const visibleTop = Math.max(divRect.top, scrollContainerTop);
            const visibleBottom = Math.min(divRect.bottom, scrollContainerBottom);
            const visibleHeight = Math.max(visibleBottom - visibleTop, 0);
            const visiblePercentage = (visibleHeight / divRect.height) * PERCENTS;
            if (visiblePercentage > maxVisiblePercentage) {
                maxVisiblePercentage = visiblePercentage;
                const categoryKey = div.getAttribute("data-name");
                if (categoryKey) {
                    mostVisibleCategoryKey = categoryKey;
                }
            }
        });

        if (mostVisibleCategoryKey !== activeCategory.key) {
            const newActiveCategory = filteredTabs.find(tab => tab.key === mostVisibleCategoryKey);
            if (newActiveCategory) {
                setActiveCategory(undefined, newActiveCategory, true);
            }
        }
    }

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

    return (
        <EmojiPopover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            zIndex={zIndex}
        >
            <MainBox
                id="emoji-picker-main"
                component="aside"
            >
                <HeaderBox>
                    <Box display="flex" flexDirection="row" alignItems="center" padding={1} gap={1}>
                        <SearchBarPaper component="form">
                            <Input
                                id="emoji-search-input"
                                disableUnderline={true}
                                fullWidth={true}
                                value={searchString}
                                onChange={(e) => { handleChange(e.target.value); }}
                                placeholder={t("Search") + "..."}
                                sx={searchBarInputStyle}
                            />
                            <MicrophoneButton onTranscriptChange={handleChange} />
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
                <EmojiPickerBody
                    id="emoji-picker-body"
                    ref={scrollRef}
                    onScroll={onScroll}
                >
                    <FullEmojiList>
                        {filteredTabs.map((category) => {
                            if (category.key === CategoryTabOption.Suggested) {
                                return <Suggested
                                    key={category.key}
                                    category={category.key}
                                    onSelect={handleSelect}
                                    searchString={searchString}
                                    setHoveredEmoji={setHoveredEmoji}
                                />;
                            }
                            return (
                                <RenderCategory
                                    key={category.key}
                                    activeSkinTone={activeSkinTone}
                                    category={category.key}
                                    onSelect={handleSelect}
                                    searchString={searchString}
                                    setHoveredEmoji={setHoveredEmoji}
                                />
                            );
                        })}
                    </FullEmojiList>
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
