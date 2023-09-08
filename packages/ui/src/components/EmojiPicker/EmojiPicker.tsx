import { CommonKey } from "@local/shared";
import { Box, Button, IconButton, Input, Paper, Popover, Typography, useTheme } from "@mui/material";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton";
import { PageTabs } from "components/PageTabs/PageTabs";
import { useTabs } from "hooks/useTabs";
import { SearchIcon } from "icons";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { EmojiVariationPicker } from "./components/body/EmojiVariationPicker";
import { ClickableEmoji } from "./components/emoji/Emoji";
import { ViewOnlyEmoji } from "./components/emoji/ViewOnlyEmoji";
import { allEmojis, emojiName, emojiUnified } from "./dataUtils/emojiSelectors";

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
    titleKey: "EmojisSuggested" as CommonKey,
    tabType: CategoryTabOption.Suggested,
}, {
    titleKey: "EmojisSmileysPeople" as CommonKey,
    tabType: CategoryTabOption.SmileysPeople,
}, {
    titleKey: "EmojisAnimalsNature" as CommonKey,
    tabType: CategoryTabOption.AnimalsNature,
}, {
    titleKey: "EmojisFoodDrink" as CommonKey,
    tabType: CategoryTabOption.FoodDrink,
}, {
    titleKey: "EmojisTravelPlaces" as CommonKey,
    tabType: CategoryTabOption.TravelPlaces,
}, {
    titleKey: "EmojisActivities" as CommonKey,
    tabType: CategoryTabOption.Activities,
}, {
    titleKey: "EmojisObjects" as CommonKey,
    tabType: CategoryTabOption.Objects,
}, {
    titleKey: "EmojisSymbols" as CommonKey,
    tabType: CategoryTabOption.Symbols,
}, {
    titleKey: "EmojisFlags" as CommonKey,
    tabType: CategoryTabOption.Flags,
}];

type EmojiClickData = {
    activeSkinTone: SkinTones;
    unified: string;
    unifiedWithoutSkinTone: string;
    emoji: string;
    names: string[];
};

export enum SkinTones {
    Neutral = "neutral",
    Light = "1f3fb",
    MediumLight = "1f3fc",
    Medium = "1f3fd",
    MediumDark = "1f3fe",
    Dark = "1f3ff"
}

const SUGGESTED_LS_KEY = "epr_suggested";

type SuggestedItem = {
    unified: string;
    original: string;
    count: number;
};

export function getSuggested(mode?: SuggestionMode): Suggested {
    try {
        if (!window?.localStorage) {
            return [];
        }
        const recent = JSON.parse(
            window?.localStorage.getItem(SUGGESTED_LS_KEY) ?? "[]",
        ) as Suggested;

        if (mode === SuggestionMode.FREQUENT) {
            return recent.sort((a, b) => b.count - a.count);
        }

        return recent;
    } catch {
        return [];
    }
}

export function setSuggested(emoji: DataEmoji, skinTone: SkinTones) {
    const recent = getSuggested();

    const unified = emojiUnified(emoji, skinTone);
    const originalUnified = emojiUnified(emoji);

    let existing = recent.find(({ unified: u }) => u === unified);

    let nextList: SuggestedItem[];

    if (existing) {
        nextList = [existing].concat(recent.filter(i => i !== existing));
    } else {
        existing = {
            unified,
            original: originalUnified,
            count: 0,
        };
        nextList = [existing, ...recent];
    }

    existing.count++;

    nextList.length = Math.min(nextList.length, 14);

    try {
        window?.localStorage.setItem(SUGGESTED_LS_KEY, JSON.stringify(nextList));
        // Prevents the change from being seen immediately.
    } catch {
        // ignore
    }
}

function Suggested({
    category,
}: {
    category: Categories;
}) {
    const [suggestedUpdated] = useUpdateSuggested();
    const suggestedEmojisModeConfig = useSuggestedEmojisModeConfig();
    const suggested = React.useMemo(
        () => getSuggested(suggestedEmojisModeConfig) ?? [],
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [suggestedUpdated, suggestedEmojisModeConfig],
    );

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
                        showVariations={false}
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
            aria-label={t(`Category${category}`, { ns: "emojis", defaultValue: category })}
        >
            <div>{t(`Category${category}`, { ns: "emojis", defaultValue: category })}</div>
            <div>{children}</div>
        </li>
    );
}

function RenderCategory({
    activeSkinTone,
    index,
    category,
}: {
    activeSkinTone: SkinTones;
    index: number;
    category: CategoryTabOption;
}) {
    const [isPastInitialLoad, setIsPastInitialLoad] = useState(false);
    useEffect(() => {
        setIsPastInitialLoad(true);
    }, [setIsPastInitialLoad]);

    // Small trick to defer the rendering of all emoji categories until the first category is visible
    // This way the user gets to actually see something and not wait for the whole picker to render.
    const emojisToPush =
        !isPastInitialLoad && index > 1 ? [] : allEmojis[category];

    const emojis = emojisToPush.filter(emoji => !emoji.a || (MINIMUM_VERSION && emoji.a >= MINIMUM_VERSION)).map(emoji => {
        const unified = emojiUnified(emoji, activeSkinTone);
        return (
            <ClickableEmoji
                key={unified}
                emoji={emoji}
                unified={unified}
                hiddenOnSearch={filteredOut}
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

export default function EmojiPicker({
    anchorEl,
    onClose,
}: {
    anchorEl: HTMLElement | null;
    onClose: () => unknown;
}) {
    const { t } = useTranslation();
    const { palette } = useTheme();

    const visibleEmojis = allEmojis.map(category => category.filter(emoji => !emoji.a || (MINIMUM_VERSION && emoji.a >= MINIMUM_VERSION)));

    const [selectedEmoji, setSelectedEmoji] = useState<DataEmoji | null>(DEFAULT_SELECTED_EMOJI);
    const selectedEmojiName = emojiName(selectedEmoji) ?? DEFAULT_SELECTION_CAPTION;

    const [searchString, setSearchString] = useState("");
    const handleChange = (value: string) => {
        setSearchString(value);
    };
    const isSearching = useMemo(() => searchString.length > 0, [searchString]);

    const {
        currTab: activeCategory,
        handleTabChange: setActiveCategory,
        tabs: categories,
    } = useTabs<CategoryTabOption>({ id: "emoji-picker-tabs", tabParams: categoryTabParams as any[], display: "dialog" });

    const [activeSkinTone, setActiveSkinTone] = useState(SkinTones.Neutral);
    const [isSkinTonePickerOpen, setIsSkinTonePickerOpen] = useState(false);

    return (
        <Popover
            anchorEl={anchorEl}
            open={Boolean(anchorEl)}
            onClose={onClose}
        >
            <Box
                id="emoji-picker-main"
                component="aside"
                sx={{
                    position: "relative",
                    display: "flex",
                    flexDirection: "column",
                    borderWidth: "1px",
                    borderStyle: "solid",
                    borderRadius: 2,
                    width: "min(100%, 400px)",
                    height: "min(100%, 400px)",
                }}>
                {/* Header */}
                <Box sx={{ position: "relative" }}>
                    <Box display="flex" flexDirection="row" alignItems="center" sx={{ padding: 1 }}>
                        {/* Search bar */}
                        <Paper
                            component="form"
                            sx={{
                                p: "2px 4px",
                                display: "flex",
                                alignItems: "center",
                                borderRadius: "10px",
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
                        <Box display="relative">
                            {skinToneVariations.map((skinToneVariation, i) => {
                                const active = skinToneVariation === activeSkinTone;
                                return (
                                    <Button
                                        onClick={() => {
                                            if (isSkinTonePickerOpen) {
                                                setActiveSkinTone(skinToneVariation);
                                                // focusSearchInput();
                                            } else {
                                                setIsSkinTonePickerOpen(true);
                                            }
                                            // closeAllOpenToggles();
                                        }}
                                        key={skinToneVariation}
                                        tabIndex={isSkinTonePickerOpen ? 0 : -1}
                                        aria-pressed={active}
                                        aria-label={`Skin tone ${skinTonesNamed[skinToneVariation as SkinTones]
                                            }`}
                                    ></Button>
                                );
                            })}
                        </Box>
                    </Box>
                    {/* Category tabs */}
                    <PageTabs
                        ariaLabel="emoji-picker-tabs"
                        fullWidth
                        id="emoji-picker-tabs"
                        ignoreIcons
                        currTab={activeCategory}
                        onChange={setActiveCategory}
                        tabs={categories}
                    />
                </Box>
                {/* Body */}
                <div id="emoji-picker-body">
                    <EmojiVariationPicker selectedEmoji={selectedEmoji} />
                    {/* Emoji list */}
                    <ul>
                        {categoryTabParams.map((category, index) => {
                            if (category.tabType === CategoryTabOption.Suggested) {
                                return <Suggested key={category.tabType} category={category.tabType} />;
                            }
                            return (
                                <RenderCategory
                                    key={category.tabType}
                                    index={index}
                                    category={category.tabType}
                                />
                            );
                        })}
                    </ul>
                </div>
                {/* Preview selected emoji */}
                {selectedEmoji && <Box display="flex" flexDirection="row" alignItems="center">
                    <ViewOnlyEmoji
                        unified={selectedEmoji?.u as string}
                        emoji={selectedEmoji}
                        size={45}
                    />
                    <Typography variant="body1" sx={{ marginLeft: 1 }}>{selectedEmojiName}</Typography>
                </Box>}
            </Box>
        </Popover>
    );
}
