import { CommonKey, EmojisKey } from "@local/shared";
import { Box, Button, IconButton, Input, Palette, Paper, Popover, useTheme } from "@mui/material";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { useZIndex } from "hooks/useZIndex";
import i18next from "i18next";
import { AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "icons";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { VariableSizeList } from "react-window";
import emojis from "./data/emojis";

const MINIMUM_VERSION: number | null = null;
const KNOWN_FAILING_EMOJIS = ["2640-fe0f", "2642-fe0f", "2695-fe0f"];

enum CategoryTabOption {
    Suggested = "Suggested",
    SmileysPeople = "SmileysPeople",
    AnimalsNature = "AnimalsNature",
    FoodDrink = "FoodDrink",
    TravelPlaces = "TravelPlaces",
    Activities = "Activities",
    Objects = "Objects",
    Symbols = "Symbols",
    Flags = "Flags"
}

const categoryTabParams = [{
    Icon: HistoryIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisSuggested" as CommonKey,
    tabType: CategoryTabOption.Suggested,
}, {
    Icon: RoutineValidIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisSmileysPeople" as CommonKey,
    tabType: CategoryTabOption.SmileysPeople,
}, {
    Icon: VrooliIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisAnimalsNature" as CommonKey,
    tabType: CategoryTabOption.AnimalsNature,
}, {
    Icon: FoodIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisFoodDrink" as CommonKey,
    tabType: CategoryTabOption.FoodDrink,
}, {
    Icon: AirplaneIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisTravelPlaces" as CommonKey,
    tabType: CategoryTabOption.TravelPlaces,
}, {
    Icon: AwardIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisActivities" as CommonKey,
    tabType: CategoryTabOption.Activities,
}, {
    Icon: ProjectIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisObjects" as CommonKey,
    tabType: CategoryTabOption.Objects,
}, {
    Icon: CompleteIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisSymbols" as CommonKey,
    tabType: CategoryTabOption.Symbols,
}, {
    Icon: ReportIcon,
    color: (palette: Palette) => palette.mode === "light" ? palette.secondary.light : palette.background.textPrimary,
    titleKey: "EmojisFlags" as CommonKey,
    tabType: CategoryTabOption.Flags,
}];

enum EmojiProperties {
    name = "name",
    unified = "u",
    variations = "v",
    added_in = "a",
    imgUrl = "imgUrl"
}

interface DataEmoji extends WithName {
    [EmojiProperties.unified]: string;
    [EmojiProperties.variations]?: string[];
    [EmojiProperties.added_in]: number;
    [EmojiProperties.imgUrl]?: string;
}

type WithName = {
    name: string[];
};

enum SkinTone {
    Neutral = "neutral",
    Light = "1f3fb",
    MediumLight = "1f3fc",
    Medium = "1f3fd",
    MediumDark = "1f3fe",
    Dark = "1f3ff"
}

const toneToHex: Record<SkinTone, string> = {
    [SkinTone.Neutral]: "#ffd225",
    [SkinTone.Light]: "#ffdfbd",
    [SkinTone.MediumLight]: "#e9c197",
    [SkinTone.Medium]: "#c88e62",
    [SkinTone.MediumDark]: "#a86637",
    [SkinTone.Dark]: "#60463a",
};

function parseNativeEmoji(unified: string): string {
    return unified
        .split("-")
        .map(hex => String.fromCodePoint(parseInt(hex, 16)))
        .join("");
}

const emojiNames = (emoji: WithName): string[] => emoji[EmojiProperties.name] ?? [];

console.time("all emojis");
const allEmojis = Object.values(emojis).map(category => category.map(emoji => ({
    ...emoji,
    name: (i18next.t(`emojis:${emoji.u.toLowerCase() as unknown as EmojisKey}`, { ns: "emojis" }) ?? "").split(", "),
}))) as unknown as Record<CategoryTabOption, DataEmoji[]>;
console.timeEnd("all emojis");
console.log("got all emojis", allEmojis, i18next.t("common:Submit"), i18next.t("emojis:1f606", { ns: "emojis" }));

const allEmojisByUnified = new Map<string, DataEmoji>();
const allEmojisByName = new Map<string, DataEmoji>();

function indexEmoji(emoji: DataEmoji): void {
    emojiNames(emoji).forEach(name => {
        allEmojisByName.set(name, emoji);
    });
}

setTimeout(() => {
    console.time("indexing");
    Object.values(allEmojis).forEach(emojis => emojis.forEach(emoji => {
        indexEmoji(emoji);
        allEmojisByUnified.set(emoji[EmojiProperties.unified], emoji);
    }));
    console.timeEnd("indexing");
});

function unifiedWithoutSkinTone(unified: string): string {
    const splat = unified.split("-");
    const [skinTone] = splat.splice(1, 1);

    if (SkinTone[skinTone]) {
        return splat.join("-");
    }

    return unified;
}

function emojiUnified(emoji: DataEmoji, skinTone?: string): string {
    const unified = emoji[EmojiProperties.unified];

    if (!skinTone || !Array.isArray(emoji[EmojiProperties.variations]) || emoji[EmojiProperties.variations].length === 0) {
        return unified;
    }

    return emojiVariationUnified(emoji, skinTone) ?? unified;
}

function emojiVariationUnified(
    emoji: DataEmoji,
    skinTone?: string,
): string | undefined {
    return skinTone
        ? (emoji[EmojiProperties.variations] ?? []).find(variation => variation.includes(skinTone))
        : emojiUnified(emoji);
}

function emojiByUnified(unified?: string): DataEmoji | undefined {
    if (!unified) {
        return;
    }

    if (allEmojisByUnified.has(unified)) {
        return allEmojisByUnified.get(unified);
    }

    const withoutSkinTone = unifiedWithoutSkinTone(unified);
    return allEmojisByUnified.get(withoutSkinTone);
}

type BaseEmojiProps = {
    emoji?: DataEmoji;
    unified: string;
};

function ClickableEmoji({
    emoji,
    unified,
    hiddenOnSearch,
    onSelect,
}: Readonly<BaseEmojiProps & {
    hiddenOnSearch?: boolean;
    emoji: DataEmoji;
    onSelect: (emoji: string) => unknown;
}>) {
    return (
        <Button
            data-unified={unified}
            aria-label={emojiNames[0]}
            data-full-name={emojiNames}
            onClick={() => { onSelect(parseNativeEmoji(unified)); }}
            sx={{
                fontSize: "1.5rem",
            }}
        >
            {parseNativeEmoji(unified)}
        </Button>
    );
}

const calculateCategoryHeight = (numEmoji: number, pickerWidth: number): number => {
    const emojiWidth = 64;
    const emojiHeight = 54;
    const totalPaddingIncludingScrollbar = 8 + 8 + 8;
    const numEmojisPerRow = Math.floor((pickerWidth - totalPaddingIncludingScrollbar) / emojiWidth);
    const numRows = Math.ceil(numEmoji / numEmojisPerRow);
    return numRows * emojiHeight;
};

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

function Suggested({
    category,
    onHeightChange,
    onSelect,
    pickerWidth,
}: {
    category: CategoryTabOption;
    onHeightChange: (index: number, height: number) => unknown;
    onSelect: (emoji: string) => unknown;
    pickerWidth: number;
}) {
    const suggested = useMemo(() => getSuggested(), []);
    const categoryHeight = useMemo(() => calculateCategoryHeight(suggested.length, pickerWidth), [suggested, pickerWidth]);
    useEffect(() => {
        console.log("calculating suggested height change", categoryHeight);
        onHeightChange(0, categoryHeight);
    }, [categoryHeight, onHeightChange]);

    return (
        <EmojiCategory
            category={category}
            hidden={suggested.length === 0}
        >
            {suggested.map((suggestedItem) => {
                const emoji = emojiByUnified(suggestedItem.original);

                if (!emoji) {
                    return null;
                }

                return (
                    <ClickableEmoji
                        unified={suggestedItem.unified}
                        emoji={emoji}
                        key={suggestedItem.unified}
                        onSelect={onSelect}
                    />
                );
            })}
        </EmojiCategory>
    );
}

function EmojiCategory({
    category,
    children,
    hidden,
}: {
    category: CategoryTabOption;
    children: React.ReactNode;
    hidden: boolean;
}) {
    const { t } = useTranslation();

    if (hidden) {
        return null;
    }
    return (
        <li
            data-name={category}
            aria-label={t(`Emojis${category}`, { ns: "common", defaultValue: category })}
            style={{ listStyleType: "none" }}
        >
            <div>{t(`Emojis${category}`, { ns: "common", defaultValue: category })}</div>
            <div>{children}</div>
        </li>
    );
}

function RenderCategory({
    activeSkinTone,
    index,
    category,
    onHeightChange,
    onSelect,
    pickerWidth,
}: {
    activeSkinTone: SkinTone;
    index: number;
    category: CategoryTabOption;
    onHeightChange: (index: number, height: number) => unknown;
    onSelect: (emoji: string) => unknown;
    pickerWidth: number;
}) {
    const emojis = useMemo(() => {
        console.log("calculating emojis in category");
        return allEmojis[index]
            .filter(emoji => (!emoji.a || !MINIMUM_VERSION) || (emoji.a >= MINIMUM_VERSION))
            .map(emoji => {
                const unified = emojiUnified(emoji, activeSkinTone);
                return (
                    <ClickableEmoji
                        key={unified}
                        emoji={emoji}
                        unified={unified}
                        hiddenOnSearch={false} //TODO
                        onSelect={onSelect}
                    />
                );
            });
    }, [index, activeSkinTone, onSelect]);
    const categoryHeight = useMemo(() => calculateCategoryHeight(emojis.length, pickerWidth), [emojis, pickerWidth]);
    useEffect(() => {
        console.log("calculating category height change", categoryHeight);
        onHeightChange(index, categoryHeight);
    }, [categoryHeight, index, onHeightChange]);

    return (
        <EmojiCategory
            category={category}
            hidden={emojis.length === 0}
        >
            {emojis}
        </EmojiCategory>
    );
}

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
    const zIndex = useZIndex(Boolean(anchorEl));

    const [searchString, setSearchString] = useState("");
    const handleChange = (value: string) => {
        setSearchString(value);
    };

    const {
        currTab: activeCategory,
        handleTabChange: setActiveCategory,
        tabs: categories,
    } = useTabs<CategoryTabOption>({ id: "emoji-picker-tabs", tabParams: categoryTabParams as any[], display: "dialog" });
    useEffect(() => {
        const categoryDiv = document.querySelector(`[data-name="${activeCategory.tabType}"]`);
        console.log("category divvvvv", categoryDiv, activeCategory);
        if (categoryDiv) {
            categoryDiv.scrollIntoView();
        }
    }, [activeCategory]);

    const [activeSkinTone, setActiveSkinTone] = useState(SkinTone.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);

    // Logic for calculating each category's height. Needed for virtualized list.
    const [pickerWidth, setPickerWidth] = useState(410);
    const pickerRef = useRef<HTMLDivElement>(null);
    useEffect(() => {
        if (pickerRef.current) {
            // setPickerWidth(pickerRef.current.clientWidth);
        }
    }, []);
    useEffect(() => {
        const handleResize = () => {
            if (pickerRef.current) {
                // setPickerWidth(pickerRef.current.clientWidth);
            }
        };
        window.addEventListener("resize", handleResize);
        return () => { window.removeEventListener("resize", handleResize); };
    }, []);
    const categoryHeights = useRef<number[]>(new Array(categoryTabParams.length).fill(0));
    const onCategoryHeightChange = useCallback((index: number, height: number) => {
        console.log("height change", index, height);
        categoryHeights.current[index] = height;
    }, []);

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
                    width: "min(100%, 410px)",
                    height: "min(100%, 410px)",
                },
            }}
        >
            <Box
                id="emoji-picker-main"
                ref={pickerRef}
                component="aside"
                sx={{
                    position: "relative",
                    display: "flex",
                    flexDirection: "column",
                    borderRadius: 2,
                    width: "100%",
                    height: "100%",
                }}>
                {/* Header */}
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
                                border: palette.mode === "light" ? `1px solid ${palette.divider}` : "none",
                            }}
                        >
                            <Input
                                id="emoji-search-input"
                                disableUnderline={true}
                                fullWidth={true}
                                value={searchString}
                                onChange={(e) => handleChange(e.target.value)}
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
                            gap: "2px",
                        }}>
                            {Object.entries(SkinTone).map(([skinToneKey, skinToneValue], i) => {
                                const active = skinToneValue === activeSkinTone;
                                return (
                                    <Box
                                        onClick={() => {
                                            if (isSkinTonePickerOpen) {
                                                setActiveSkinTone(skinToneValue);
                                                // focusSearchInput();
                                                setIsSkinTonePickerOpen(false);
                                            } else {
                                                setIsSkinTonePickerOpen(true);
                                            }
                                            // closeAllOpenToggles();
                                        }}
                                        key={skinToneKey}
                                        tabIndex={isSkinTonePickerOpen ? 0 : -1}
                                        aria-pressed={active}
                                        aria-label={`Skin tone ${t(`SkinTone${skinToneKey}`, { ns: "emojis", defaultValue: skinToneKey })}`}
                                        sx={{
                                            background: toneToHex[skinToneValue],
                                            borderRadius: 2,
                                            cursor: "pointer",
                                            width: "24px",
                                            height: "24px",
                                            position: "absolute",
                                            left: isSkinTonePickerOpen ? `${i * 28}px` : "0px",
                                            transition: "left 0.2s ease-in-out",
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
                <Box id="emoji-picker-body" p={1} sx={{ overflow: "hidden" }}>
                    {/* Emoji list */}
                    <VariableSizeList
                        height={410}
                        width={410}
                        itemCount={categoryTabParams.length}
                        itemSize={(index) => categoryHeights[index] ?? 0}
                        overscanCount={2}
                        style={{
                            maxWidth: "100%",
                            padding: 0,
                            margin: 0,
                        }}
                    >
                        {({ index }) => {
                            const category = categoryTabParams[index];
                            if (category.tabType === CategoryTabOption.Suggested) {
                                return (
                                    <Suggested
                                        key={category.tabType}
                                        category={category.tabType}
                                        onHeightChange={onCategoryHeightChange}
                                        onSelect={onSelect}
                                        pickerWidth={pickerWidth}
                                    />
                                );
                            }
                            return (
                                <RenderCategory
                                    key={category.tabType}
                                    index={index}
                                    activeSkinTone={activeSkinTone}
                                    category={category.tabType}
                                    onHeightChange={onCategoryHeightChange}
                                    onSelect={onSelect}
                                    pickerWidth={pickerWidth}
                                />
                            );
                        }}
                    </VariableSizeList>
                </Box>
            </Box>
        </Popover>
    );
};
