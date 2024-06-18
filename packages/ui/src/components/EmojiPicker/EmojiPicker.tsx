import { Box, Button, IconButton, Input, Palette, Paper, Popover, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { useTabs } from "hooks/useTabs";
import { useZIndex } from "hooks/useZIndex";
import { AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "icons";
import { memo, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { TabParam } from "utils/search/objectToSearch";
// import emojis from "./data/emojis";

const KNOWN_FAILING_EMOJIS = ["2640-fe0f", "2642-fe0f", "2695-fe0f"];

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

const iconColor = (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary;

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

export interface DataEmoji extends WithName {
    [EmojiProperties.unified]: string;
    [EmojiProperties.variations]?: string[];
}

type WithName = {
    name: string[];
};

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
export const parseNativeEmoji = (unified: string): string => {
    if (typeof unified !== "string" || unified.length === 0) {
        return "";
    }
    return unified
        .split("-")
        .map(hex => String.fromCodePoint(parseInt(hex, 16)))
        .join("");
};

let namesByCode: Record<string, string[] | undefined> = {};
const codeByName: Record<string, string | undefined> = {};
/**
 * Loads emoji names
 */
const fetchEmojiNames = async () => {
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
};
if (process.env.NODE_ENV !== "test") {
    fetchEmojiNames();
}

let emojisByCategory: Record<Exclude<CategoryTabOption, "Suggested">, DataEmoji[] | undefined> = {} as any;
const emojiByCode: Record<string, DataEmoji | undefined> = {};
/**
 * Loads emoji data
 */
const fetchEmojiData = async () => {
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
};
if (process.env.NODE_ENV !== "test") {
    fetchEmojiData();
}

/**
 * Removes any skin tone data from a unified emoji code to normalize it.
 * @param unified - The unified string of an emoji code, potentially including skin tones.
 * @returns The unified code without any skin tone information.
 */
export const unifiedWithoutSkinTone = (unified: string): string => {
    return unified.split("-").filter((section) => !skinTones.includes(section as SkinTone)).join("-");
};

/**
 * Computes the unified string for an emoji, optionally applying a specified skin tone.
 * @param emoji The emoji data object.
 * @param Optional skin tone modifier.
 * @returns The unified string with or without a skin tone modification.
 */
export const emojiUnified = (emoji: DataEmoji, skinTone?: string): string => {
    const unified = emoji[EmojiProperties.unified];

    if (!skinTone || !Array.isArray(emoji[EmojiProperties.variations]) || emoji[EmojiProperties.variations].length === 0) {
        return unified;
    }

    return emojiVariationUnified(emoji, skinTone) ?? unified;
};

/**
 * Selects the appropriate emoji variation based on the given skin tone.
 * @param emoji The base emoji data object.
 * @param Optional skin tone to apply.
 * @returns The unified string of the emoji with the applied skin tone, if available.
 */
export const emojiVariationUnified = (
    emoji: DataEmoji,
    skinTone?: string,
): string | undefined => {
    return skinTone
        ? (emoji[EmojiProperties.variations] ?? []).find(variation => variation.includes(skinTone))
        : emojiUnified(emoji);
};

/**
 * Retrieves an emoji object by its unified code.
 * @param unified The unified string of an emoji.
 * @returns The emoji object corresponding to the unified code.
 */
export const emojiByUnified = (unified?: string): DataEmoji | undefined => {
    if (!unified) {
        return;
    }

    if (emojiByCode[unified]) {
        return emojiByCode[unified];
    }

    const withoutSkinTone = unifiedWithoutSkinTone(unified);
    return emojiByCode[withoutSkinTone];
};

/**
 * Component representing a clickable emoji button in the UI.
 */
const ClickableEmoji = memo(({
    onSelect,
    unified,
    setHoveredEmoji,
}: {
    onSelect: (emoji: string) => unknown;
    unified: string;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    const names = namesByCode[unified];
    const name = Array.isArray(names) && names.length > 0 ? names[0] : "";

    return (
        <Button
            data-unified={unified}
            aria-label={name}
            data-full-name={names}
            onClick={() => { onSelect(parseNativeEmoji(unified)); }}
            onMouseEnter={() => { setHoveredEmoji(unified); }}
            onMouseLeave={() => { setHoveredEmoji(null); }}
            sx={{ fontSize: "1.5rem" }}
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
        // return recent.sort((a, b) => b.count - a.count);
        return recent;
    } catch {
        return [];
    }
}

const Suggested = memo(({
    category,
    onSelect,
    setHoveredEmoji,
}: {
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (emoji: string) => unknown;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    const suggested = useMemo(() => getSuggested(), []);
    if (suggested.length === 0) {
        return null;
    }
    return (
        <EmojiCategory
            category={category}
        >
            {suggested.map((suggestedItem) => {
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
            style={{ listStyleType: "none" }}
        >
            <div style={{ paddingLeft: "8px" }}>{t(`Emojis${category}`, { ns: "common", defaultValue: category })}</div>
            <div>{children}</div>
        </li>
    );
});
EmojiCategory.displayName = "EmojiCategory";

const RenderCategory = memo(({
    activeSkinTone,
    index,
    isInView,
    category,
    onSelect,
    setHoveredEmoji,
}: {
    activeSkinTone: SkinTone;
    index: number;
    isInView: boolean;
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (emoji: string) => unknown;
    setHoveredEmoji: (emoji: string | null) => unknown;
}) => {
    const emojis = useMemo(() => {
        const emojisToPush = isInView ? (emojisByCategory[category] ?? []) : [];
        return emojisToPush
            // .filter(emoji => (!emoji.a || !MINIMUM_VERSION) || (emoji.a >= MINIMUM_VERSION)) TODO search filter
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
    }, [index, isInView, activeSkinTone, onSelect]);

    if (emojis.length === 0) {
        return null;
    }
    return (
        <EmojiCategory
            category={category}
        >
            {emojis}
        </EmojiCategory>
    );
});
RenderCategory.displayName = "RenderCategory";

export const EmojiPicker = ({
    anchorEl,
    onClose,
    onSelect,
}: {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
    onSelect: (emoji: string) => unknown;
}) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const zIndex = useZIndex(Boolean(anchorEl), false, 1000);

    const [searchString, setSearchString] = useState("");
    const handleChange = (value: string) => { setSearchString(value); };

    const [hoveredEmoji, setHoveredEmoji] = useState<string | null>(null);

    const {
        currTab: activeCategory,
        handleTabChange: setActiveCategory,
        tabs: categories,
    } = useTabs({ id: "emoji-picker-tabs", tabParams: categoryTabParams, display: "dialog" });
    useEffect(() => {
        const categoryDiv = document.querySelector(`[data-name="${activeCategory.key}"]`);
        if (categoryDiv) {
            categoryDiv.scrollIntoView();
        }
    }, [activeCategory]);

    const [activeSkinTone, setActiveSkinTone] = useState(SkinTone.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);

    const scrollRef = useRef<HTMLDivElement>(null);
    const [visibleCategories, setVisibleCategories] = useState(new Set([0, 1, 2]));
    // Find which category is the most in view, and set that and surrounding categories to visible
    const onScroll = () => {
        const scrollContainer = scrollRef.current;
        if (!scrollContainer) return;

        const scrollContainerRect = scrollContainer.getBoundingClientRect();
        const scrollContainerTop = scrollContainerRect.top;
        const scrollContainerBottom = scrollContainerRect.bottom;

        const categoryDivs = Array.from(scrollContainer.querySelectorAll("li"));

        let maxVisiblePercentage = 0;
        let mostVisibleIndex = 0;

        categoryDivs.forEach((div, index) => {
            const divRect = div.getBoundingClientRect();
            const visibleTop = Math.max(divRect.top, scrollContainerTop);
            const visibleBottom = Math.min(divRect.bottom, scrollContainerBottom);
            // Calculate visible height
            const visibleHeight = Math.max(visibleBottom - visibleTop, 0);
            // Calculate percentage visible
            const visiblePercentage = (visibleHeight / divRect.height) * 100;
            if (visiblePercentage > maxVisiblePercentage) {
                maxVisiblePercentage = visiblePercentage;
                mostVisibleIndex = index;
            }
        });
        // Determine surrounding categories. Move them into range if needed, making sure there are always 3 categories visible
        let categoriesToShow = [
            mostVisibleIndex - 1,
            mostVisibleIndex,
            mostVisibleIndex + 1,
        ];
        if (categoriesToShow[0] < 0) {
            categoriesToShow = categoriesToShow.map((index) => index + Math.abs(categoriesToShow[0]));
        }
        if (categoriesToShow[2] >= categoryDivs.length) {
            categoriesToShow = categoriesToShow.map((index) => index - (categoriesToShow[2] - categoryDivs.length + 1));
        }
        const newVisibleCategories = new Set(visibleCategories);
        categoriesToShow.forEach((index) => {
            newVisibleCategories.add(index);
        });
        setVisibleCategories(newVisibleCategories);
    };

    return (
        <Popover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            sx={{
                zIndex: `${zIndex} !important`,
                "& .MuiPopover-paper": {
                    background: palette.background.default,
                    border: palette.mode === "light" ? "none" : `1px solid ${palette.divider}`,
                    borderRadius: 2,
                    width: "min(100%, 410px)",
                    height: "min(100%, 410px)",
                },
            }}
        >
            <Box
                id="emoji-picker-main"
                component="aside"
                sx={{
                    position: "relative",
                    display: "flex",
                    flexDirection: "column",
                    width: "100%",
                    height: "100%",
                }}>
                <Box sx={{ position: "relative" }}>
                    <Box display="flex" flexDirection="row" alignItems="center" sx={{ padding: 1, gap: 1 }}>
                        {/* Search bar */}
                        <Paper
                            component="form"
                            sx={{
                                p: "2px 4px",
                                display: "flex",
                                alignItems: "center",
                                borderRadius: "10px",
                                boxShadow: 0,
                                width: "-webkit-fill-available",
                                flex: 1,
                                border: palette.mode === "light" ? `1px solid ${palette.divider}` : "none",
                            }}
                        >
                            <Input
                                id="emoji-search-input"
                                disableUnderline={true}
                                fullWidth={true}
                                value={searchString}
                                onChange={(e) => { handleChange(e.target.value); }}
                                placeholder={t("Search") + "..."}
                                autoFocus={true}
                                sx={{ marginLeft: 1 }}
                            />
                            <MicrophoneButton onTranscriptChange={handleChange} />
                            <IconButton sx={{
                                width: "48px",
                                height: "48px",
                            }} aria-label="main-search-icon">
                                <SearchIcon fill={palette.background.textSecondary} />
                            </IconButton>
                        </Paper>
                        {/* Skin tone picker */}
                        <Box sx={{
                            position: "relative",
                            width: isSkinTonePickerOpen ? `${28 * Object.values(SkinTone).length}px` : "28px",
                            height: "28px",
                            transition: "width 0.2s ease-in-out",
                            display: "flex",
                            flexDirection: "row",
                            flexShrink: 0, // Prevent shrinking
                            gap: "2px",
                        }}>
                            {Object.entries(SkinTone).map(([skinToneKey, skinToneValue], i) => {
                                const active = skinToneValue === activeSkinTone;
                                return (
                                    <Box
                                        onClick={() => {
                                            if (isSkinTonePickerOpen) {
                                                if (!active) {
                                                    setActiveSkinTone(skinToneValue);
                                                }
                                                setIsSkinTonePickerOpen(false);
                                            } else {
                                                setIsSkinTonePickerOpen(true);
                                            }
                                        }}
                                        key={skinToneKey}
                                        tabIndex={isSkinTonePickerOpen ? 0 : -1}
                                        aria-pressed={active}
                                        aria-label={`Skin tone ${t(`EmojiSkinTone${skinToneKey}`, { defaultValue: skinToneKey })}`}
                                        sx={{
                                            background: toneToHex[skinToneValue],
                                            borderRadius: active ? 2 : "100%",
                                            cursor: "pointer",
                                            width: active ? "24px" : "20px",
                                            height: active ? "24px" : "20px",
                                            position: "absolute",
                                            left: isSkinTonePickerOpen ?
                                                `${i * 28}px` :
                                                active ?
                                                    "0px" :
                                                    "2px",
                                            top: active ? "0px" : "2px",
                                            transition: "left 0.2s ease-in-out",
                                            zIndex: active ? 1 : 0, // Ensure selected is on top
                                        }}
                                    ></Box>
                                );
                            })}
                        </Box>
                    </Box>
                    {/* Category tabs */}
                    <PageTabs
                        ariaLabel="emoji-picker-tabs"
                        fullWidth
                        id="emoji-picker-tabs"
                        currTab={activeCategory}
                        onChange={setActiveCategory}
                        tabs={categories}
                    />
                </Box>
                {/* Body */}
                <Box
                    id="emoji-picker-body"
                    ref={scrollRef}
                    onScroll={onScroll}
                    sx={{
                        overflow: "scroll",
                        "&::-webkit-scrollbar": {
                            display: "none",
                        },
                    }}
                >
                    {/* Emoji list */}
                    <ul style={{ padding: 1, margin: 0 }}>
                        {categoryTabParams.map((category, index) => {
                            if (category.key === CategoryTabOption.Suggested) {
                                return <Suggested
                                    key={category.key}
                                    category={category.key}
                                    onSelect={onSelect}
                                    setHoveredEmoji={(emoji) => { setHoveredEmoji(emoji); }}
                                />;
                            }
                            return (
                                <RenderCategory
                                    key={category.key}
                                    index={index}
                                    activeSkinTone={activeSkinTone}
                                    category={category.key}
                                    isInView={visibleCategories.has(index)}
                                    onSelect={onSelect}
                                    setHoveredEmoji={(emoji) => { setHoveredEmoji(emoji); }}
                                />
                            );
                        })}
                    </ul>
                    {/* Display hovered emoji name */}
                    <Box sx={{
                        position: "absolute",
                        display: "flex",
                        bottom: 0,
                        width: "100%",
                        textAlign: "start",
                        background: palette.background.paper,
                        paddingLeft: 2,
                        paddingRight: 2,
                    }}>
                        <div style={{ display: "contents", fontSize: "2.5rem" }}>
                            {hoveredEmoji ? parseNativeEmoji(hoveredEmoji) : "‎"}
                        </div>
                        <div style={{ display: "inline", alignSelf: "center", marginLeft: 1 }}>
                            {/* eslint-disable-next-line @typescript-eslint/no-non-null-assertion */}
                            {hoveredEmoji && Array.isArray(namesByCode[hoveredEmoji]) && namesByCode[hoveredEmoji]!.length > 0 ? namesByCode[hoveredEmoji]![0] : ""}
                        </div>
                    </Box>
                </Box>
            </Box>
        </Popover>
    );
};
