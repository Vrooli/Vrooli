import { useState } from "react";
import { usePreviewConfig } from "../../config/useConfig";
import { emojiByUnified, emojiName, emojiUnified } from "../../dataUtils/emojiSelectors";
import { useEmojiPreviewEvents } from "../../hooks/useEmojiPreviewEvents";
import { useIsSkinToneInPreview } from "../../hooks/useShouldShowSkinTonePicker";
import { useEmojiVariationPickerState } from "../context/PickerContext";
import { ViewOnlyEmoji } from "../emoji/ViewOnlyEmoji";
import { SkinTonePickerMenu } from "../header/SkinTonePicker";
import Flex from "../Layout/Flex";
import Space from "../Layout/Space";
import "./Preview.css";

export function Preview() {
    const previewConfig = usePreviewConfig();
    const isSkinToneInPreview = useIsSkinToneInPreview();

    if (!previewConfig.showPreview) {
        return null;
    }

    return (
        <Flex className="epr-preview">
            <PreviewBody />
            <Space />
            {isSkinToneInPreview ? <SkinTonePickerMenu /> : null}
        </Flex>
    );
}

export function PreviewBody() {
    const previewConfig = usePreviewConfig();
    const [previewEmoji, setPreviewEmoji] = useState<PreviewEmoji>(null);
    const [variationPickerEmoji] = useEmojiVariationPickerState();

    useEmojiPreviewEvents(previewConfig.showPreview, setPreviewEmoji);

    const emoji = emojiByUnified(previewEmoji?.originalUnified);

    const show = emoji != null && previewEmoji != null;

    return <PreviewContent />;

    function PreviewContent() {
        const defaultEmoji = variationPickerEmoji ?? emojiByUnified(previewConfig.defaultEmoji);
        if (!defaultEmoji) {
            return null;
        }
        const defaultText = variationPickerEmoji
            ? emojiName(variationPickerEmoji)
            : previewConfig.defaultCaption;

        return (
            <>
                <div>
                    {show ? (
                        <ViewOnlyEmoji
                            unified={previewEmoji?.unified as string}
                            emoji={emoji}
                            size={45}
                        />
                    ) : defaultEmoji ? (
                        <ViewOnlyEmoji
                            unified={emojiUnified(defaultEmoji)}
                            emoji={defaultEmoji}
                            size={45}
                        />
                    ) : null}
                </div>
                {show ? (
                    <div className="epr-preview-emoji-label">
                        {emojiName(emoji)}
                    </div>
                ) : (
                    <div className="epr-preview-emoji-label">{defaultText}</div>
                )}
            </>
        );
    }
}

export type PreviewEmoji = null | {
    unified: string;
    originalUnified: string;
};
