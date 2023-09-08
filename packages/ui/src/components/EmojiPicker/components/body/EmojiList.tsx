import { Categories, CategoryConfig, categoryFromCategoryConfig } from "../../config/categoryConfig";
import { useCategoriesConfig, useLazyLoadEmojisConfig, useSkinTonesDisabledConfig } from "../../config/useConfig";
import { emojisByCategory, emojiUnified } from "../../dataUtils/emojiSelectors";
import { ClassNames } from "../../DomUtils/classNames";
import { useIsEmojiDisallowed } from "../../hooks/useDisallowedEmojis";
import { useIsEmojiHidden } from "../../hooks/useIsEmojiHidden";
import { useActiveSkinToneState, useIsPastInitialLoad } from "../context/PickerContext";
import { ClickableEmoji } from "../emoji/Emoji";
import { EmojiCategory } from "./EmojiCategory";
import "./EmojiList.css";
import { Suggested } from "./Suggested";

export function EmojiList() {
    const categories = useCategoriesConfig();

    return (
        <ul className={ClassNames.emojiList}>
            {categories.map((categoryConfig, index) => {
                const category = categoryFromCategoryConfig(categoryConfig);

                if (category === Categories.SUGGESTED) {
                    return <Suggested key={category} categoryConfig={categoryConfig} />;
                }

                return (
                    <RenderCategory
                        key={category}
                        index={index}
                        category={category}
                        categoryConfig={categoryConfig}
                    />
                );
            })}
        </ul>
    );
}

function RenderCategory({
    index,
    category,
    categoryConfig,
}: {
    index: number;
    category: Categories;
    categoryConfig: CategoryConfig;
}) {
    const isEmojiHidden = useIsEmojiHidden();
    const lazyLoadEmojis = useLazyLoadEmojisConfig();
    const isPastInitialLoad = useIsPastInitialLoad();
    const [activeSkinTone] = useActiveSkinToneState();
    const isEmojiDisallowed = useIsEmojiDisallowed();
    const showVariations = !useSkinTonesDisabledConfig();

    // Small trick to defer the rendering of all emoji categories until the first category is visible
    // This way the user gets to actually see something and not wait for the whole picker to render.
    const emojisToPush =
        !isPastInitialLoad && index > 1 ? [] : emojisByCategory(category);

    let hiddenCounter = 0;

    const emojis = emojisToPush.map(emoji => {
        const unified = emojiUnified(emoji, activeSkinTone);
        const { failedToLoad, filteredOut, hidden } = isEmojiHidden(emoji);

        const isDisallowed = isEmojiDisallowed(emoji);

        if (hidden || isDisallowed) {
            hiddenCounter++;
        }

        if (isDisallowed) {
            return null;
        }

        return (
            <ClickableEmoji
                showVariations={showVariations}
                key={unified}
                emoji={emoji}
                unified={unified}
                hidden={failedToLoad}
                hiddenOnSearch={filteredOut}
                lazyLoad={lazyLoadEmojis}
            />
        );
    });

    return (
        <EmojiCategory
            categoryConfig={categoryConfig}
            // Indicates that there are no visible emojis
            // Hence, the category should be hidden
            hidden={hiddenCounter === emojis.length}
        >
            {emojis}
        </EmojiCategory>
    );
}
