import { EmojisKey } from "@local/shared";
import { Box, Button, IconButton, Input, Palette, Paper, Popover, useTheme } from "@mui/material";
import { PageTabs } from "components/PageTabs/PageTabs";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { useTabs } from "hooks/useTabs";
import { useZIndex } from "hooks/useZIndex";
import i18next from "i18next";
import { AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { TabParam } from "utils/search/objectToSearch";
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
    key: "SmileysPeople",
    titleKey: "EmojisSmileysPeople",
}, {
    color: iconColor,
    Icon: VrooliIcon,
    key: "AnimalsNature",
    titleKey: "EmojisAnimalsNature",
}, {
    color: iconColor,
    Icon: FoodIcon,
    key: "FoodDrink",
    titleKey: "EmojisFoodDrink",
}, {
    color: iconColor,
    Icon: AirplaneIcon,
    key: "TravelPlaces",
    titleKey: "EmojisTravelPlaces",
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

const allEmojis = Object.values(emojis).map(category => category.map(emoji => ({
    ...emoji,
    name: (i18next.t(`emojis:${emoji.u.toLowerCase() as unknown as EmojisKey}`, { ns: "emojis" }) ?? "").split(", "),
}))) as unknown as Record<CategoryTabOption, DataEmoji[]>;

const allEmojisByUnified = new Map<string, DataEmoji>();
const allEmojisByName = new Map<string, DataEmoji>();

function indexEmoji(emoji: DataEmoji): void {
    emojiNames(emoji).forEach(name => {
        allEmojisByName.set(name, emoji);
    });
}

setTimeout(() => {
    Object.values(allEmojis).forEach(emojis => emojis.forEach(emoji => {
        indexEmoji(emoji);
        allEmojisByUnified.set(emoji[EmojiProperties.unified], emoji);
    }));
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
    onSelect,
}: {
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (emoji: string) => unknown;
}) {
    const suggested = useMemo(() => getSuggested(), []);
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
    category: CategoryTabOption | `${CategoryTabOption}`;
    children: React.ReactNode;
    hidden: boolean;
}) {
    const { t } = useTranslation();

    // if (hidden) {
    //     return null;
    // }
    return (
        <li
            data-name={category}
            aria-label={t(`Emojis${category}`, { ns: "common", defaultValue: category })}
            style={{ listStyleType: "none" }}
        >
            <div>{t(`Emojis${category}`, { ns: "common", defaultValue: category })}</div>
            {!hidden && <div>{children}</div>}
        </li>
    );
}

const RenderCategory = ({
    activeSkinTone,
    index,
    isInView,
    category,
    onSelect,
}: {
    activeSkinTone: SkinTone;
    index: number;
    isInView: boolean;
    category: CategoryTabOption | `${CategoryTabOption}`;
    onSelect: (emoji: string) => unknown;
}) => {
    const emojis = useMemo(() => {
        const emojisToPush = isInView ? (allEmojis[index] ?? []) : [];
        return emojisToPush
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
    }, [index, isInView, activeSkinTone, onSelect]);

    return (
        <EmojiCategory
            category={category}
            hidden={emojis.length === 0}
        >
            {emojis}
        </EmojiCategory>
    );
};

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
    const handleChange = (value: string) => {
        setSearchString(value);
    };

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
        console.log("visible categories", newVisibleCategories);
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
                <Box
                    id="emoji-picker-body"
                    ref={scrollRef}
                    p={1}
                    onScroll={onScroll}
                    sx={{ overflow: "scroll" }}
                >
                    {/* Emoji list */}
                    <ul style={{ padding: 0, margin: 0 }}>
                        {categoryTabParams.map((category, index) => {
                            if (category.key === CategoryTabOption.Suggested) {
                                return <Suggested
                                    key={category.key}
                                    category={category.key}
                                    onSelect={onSelect}
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
                                />
                            );
                        })}
                    </ul>
                </Box>
            </Box>
        </Popover>
    );
};
