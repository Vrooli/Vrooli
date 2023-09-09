import { CommonKey, EmojisKey } from "@local/shared";
import { Box, Button, IconButton, Input, Palette, Paper, Popover, Typography, useTheme } from "@mui/material";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { useZIndex } from "hooks/useZIndex";
import i18next from "i18next";
import { AirplaneIcon, AwardIcon, CompleteIcon, FoodIcon, HistoryIcon, ProjectIcon, ReportIcon, RoutineValidIcon, SearchIcon, VrooliIcon } from "icons";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import emojis from "./data/emojis";

const DEFAULT_SELECTED_EMOJI = "1f60a"; //😊
const DEFAULT_SELECTION_CAPTION = "Select an emoji";
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
const emojiName = (emoji?: WithName | null | undefined): string => emoji ? emojiNames(emoji)[0] : "";

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

    if (allEmojisByUnified[unified]) {
        return allEmojisByUnified[unified];
    }

    const withoutSkinTone = unifiedWithoutSkinTone(unified);
    return allEmojisByUnified[withoutSkinTone];
}

const allEmojis = Object.values(emojis).map(category => category.map(emoji => ({
    ...emoji,
    name: (i18next.t(`emojis:${emoji.u.toLowerCase() as unknown as EmojisKey}`, { ns: "emojis" }) ?? "").split(", "),
}))) as unknown as Record<CategoryTabOption, DataEmoji[]>;
console.log("got all emojis", allEmojis, i18next.t("common:Submit"), i18next.t("emojis:1f606", { ns: "emojis" }));

const allEmojisByUnified: {
    [unified: string]: DataEmoji;
} = {};

const alphaNumericEmojiIndex: BaseIndex = {};

type BaseIndex = Record<string, Record<string, DataEmoji>>;

function indexEmoji(emoji: DataEmoji): void {
    const joinedNameString = emojiNames(emoji)
        .flat()
        .join("")
        .toLowerCase()
        .replace(/[^a-zA-Z\d]/g, "")
        .split("");

    joinedNameString.forEach(char => {
        alphaNumericEmojiIndex[char] = alphaNumericEmojiIndex[char] ?? {};

        alphaNumericEmojiIndex[char][emojiUnified(emoji)] = emoji;
    });
}

setTimeout(() => {
    Object.values(allEmojis).forEach(emojis => emojis.forEach(emoji => {
        indexEmoji(emoji);
        allEmojisByUnified[emoji[EmojiProperties.unified]] = emoji;
    }));
});

type BaseEmojiProps = {
    emoji?: DataEmoji;
    unified: string;
};


function ClickableEmoji({
    emoji,
    unified,
    hiddenOnSearch,
}: Readonly<BaseEmojiProps & {
    hiddenOnSearch?: boolean;
    emoji: DataEmoji;
}>) {
    return (
        <Button
            data-unified={unified}
            aria-label={emojiNames[0]}
            data-full-name={emojiNames}
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
}: {
    category: CategoryTabOption;
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
}: {
    activeSkinTone: SkinTone;
    index: number;
    category: CategoryTabOption;
}) {
    const [isPastInitialLoad, setIsPastInitialLoad] = useState(false);
    useEffect(() => {
        setIsPastInitialLoad(true);
    }, [setIsPastInitialLoad]);

    // Small trick to defer the rendering of all emoji categories until the first category is visible
    // This way the user gets to actually see something and not wait for the whole picker to render.
    const emojisToPush = (!isPastInitialLoad && index > 1) ? [] : (allEmojis[index] ?? []);

    const emojis = emojisToPush.filter(emoji => (!emoji.a || !MINIMUM_VERSION) || (emoji.a >= MINIMUM_VERSION)).map(emoji => {
        const unified = emojiUnified(emoji, activeSkinTone);
        return (
            <ClickableEmoji
                key={unified}
                emoji={emoji}
                unified={unified}
                hiddenOnSearch={false} //TODO
            />
        );
    });

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
}: {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
}) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const zIndex = useZIndex(Boolean(anchorEl));

    const visibleEmojis = Object.values(allEmojis).flat().filter(emoji => !emoji.a || (MINIMUM_VERSION && emoji.a >= MINIMUM_VERSION));

    const [selectedEmoji, setSelectedEmoji] = useState<DataEmoji | null>(visibleEmojis.find(emoji => emoji.u === DEFAULT_SELECTED_EMOJI) ?? null);
    const selectedEmojiName = emojiName(selectedEmoji) ?? DEFAULT_SELECTION_CAPTION;

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

    return (
        <Popover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
            sx={{
                zIndex: `${zIndex} !important`,
                "& .MuiPopover-paper": {
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
                <Box id="emoji-picker-body" p={1} sx={{ overflow: "scroll" }}>
                    {/* Emoji list */}
                    <ul style={{ padding: 0, margin: 0 }}>
                        {categoryTabParams.map((category, index) => {
                            if (category.tabType === CategoryTabOption.Suggested) {
                                return <Suggested key={category.tabType} category={category.tabType} />;
                            }
                            return (
                                <RenderCategory
                                    key={category.tabType}
                                    index={index}
                                    activeSkinTone={activeSkinTone}
                                    category={category.tabType}
                                />
                            );
                        })}
                    </ul>
                </Box>
                {/* Preview selected emoji */}
                {selectedEmoji && <Box display="flex" flexDirection="row" alignItems="center">
                    <Box sx={{ fontSize: "2rem" }}>{parseNativeEmoji(emojiUnified(selectedEmoji, activeSkinTone))}</Box>
                    <Typography variant="body1" sx={{ marginLeft: 1 }}>{selectedEmojiName}</Typography>
                </Box>}
            </Box>
        </Popover>
    );
};
