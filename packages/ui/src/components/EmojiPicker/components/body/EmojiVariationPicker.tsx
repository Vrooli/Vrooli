import { DataEmoji, EmojiProperties } from "components/EmojiPicker/dataUtils/DataTypes";
import { emojiUnified } from "../../dataUtils/emojiSelectors";
import { ClickableEmoji } from "../emoji/Emoji";

export function EmojiVariationPicker({
    selectedEmoji,
}: {
    selectedEmoji: DataEmoji | null;
}) {

    const visible = true;//TODO

    return (
        <div id="emoji-variation-picker">
            {visible && selectedEmoji
                ? [emojiUnified(selectedEmoji)]
                    .concat(selectedEmoji[EmojiProperties.variations] ?? [])
                    .slice(0, 6)
                    .map(unified => (
                        <ClickableEmoji
                            key={unified}
                            emoji={selectedEmoji}
                            unified={unified}
                        />
                    ))
                : null}
        </div>
    );
}
