import * as React from "react";
import { useEffect, useRef } from "react";
import { useActiveSkinToneState, useDisallowClickRef, useEmojiVariationPickerState, useUpdateSuggested } from "../components/context/PickerContext";
import { useOnEmojiClickConfig } from "../config/useConfig";
import { DataEmoji } from "../dataUtils/DataTypes";
import { activeVariationFromUnified, emojiHasVariations, emojiNames, emojiUnified } from "../dataUtils/emojiSelectors";
import { parseNativeEmoji } from "../dataUtils/parseNativeEmoji";
import { setSuggested } from "../dataUtils/suggested";
import { emojiFromElement, isEmojiElement, NullableElement } from "../DomUtils/selectors";
import { EmojiClickData, SkinTones } from "../types";
import { useCloseAllOpenToggles } from "./useCloseAllOpenToggles";
import useSetVariationPicker from "./useSetVariationPicker";

export function useMouseDownHandlers(
    BodyRef: React.MutableRefObject<NullableElement>,
) {
    const mouseDownTimerRef = useRef<undefined | number>();
    const setVariationPicker = useSetVariationPicker();
    const disallowClickRef = useDisallowClickRef();
    const [, setEmojiVariationPicker] = useEmojiVariationPickerState();
    const closeAllOpenToggles = useCloseAllOpenToggles();
    const [activeSkinTone] = useActiveSkinToneState();
    const onEmojiClick = useOnEmojiClickConfig();
    const [, updateSuggested] = useUpdateSuggested();

    const onClick = React.useCallback(
        function onClick(event: MouseEvent) {
            if (disallowClickRef.current) {
                return;
            }

            closeAllOpenToggles();

            const [emoji, unified] = emojiFromEvent(event);

            if (!emoji || !unified) {
                return;
            }

            const skinToneToUse =
                activeVariationFromUnified(unified) || activeSkinTone;

            updateSuggested();
            setSuggested(emoji, skinToneToUse);
            onEmojiClick(emojiClickOutput(emoji, skinToneToUse), event);
        },
        [
            activeSkinTone,
            closeAllOpenToggles,
            disallowClickRef,
            onEmojiClick,
            updateSuggested,
        ],
    );

    const onMouseDown = React.useCallback(
        function onMouseDown(event: MouseEvent) {
            if (mouseDownTimerRef.current) {
                clearTimeout(mouseDownTimerRef.current);
            }

            const [emoji] = emojiFromEvent(event);

            if (!emoji || !emojiHasVariations(emoji)) {
                return;
            }

            mouseDownTimerRef.current = window?.setTimeout(() => {
                disallowClickRef.current = true;
                mouseDownTimerRef.current = undefined;
                closeAllOpenToggles();
                setVariationPicker(event.target as HTMLElement);
                setEmojiVariationPicker(emoji);
            }, 500);
        },
        [
            disallowClickRef,
            closeAllOpenToggles,
            setVariationPicker,
            setEmojiVariationPicker,
        ],
    );
    const onMouseUp = React.useCallback(
        function onMouseUp() {
            if (mouseDownTimerRef.current) {
                clearTimeout(mouseDownTimerRef.current);
                mouseDownTimerRef.current = undefined;
            } else if (disallowClickRef.current) {
                // The problem we're trying to overcome here
                // is that the emoji has both mouseup and click events
                // and when releasing a mouseup event
                // the click gets triggered too
                // So we're disallowing the click event for a short time

                requestAnimationFrame(() => {
                    disallowClickRef.current = false;
                });
            }
        },
        [disallowClickRef],
    );

    useEffect(() => {
        if (!BodyRef.current) {
            return;
        }
        const bodyRef = BodyRef.current;
        bodyRef.addEventListener("click", onClick, {
            passive: true,
        });

        bodyRef.addEventListener("mousedown", onMouseDown, {
            passive: true,
        });
        bodyRef.addEventListener("mouseup", onMouseUp, {
            passive: true,
        });

        return () => {
            bodyRef?.removeEventListener("click", onClick);
            bodyRef?.removeEventListener("mousedown", onMouseDown);
            bodyRef?.removeEventListener("mouseup", onMouseUp);
        };
    }, [BodyRef, onClick, onMouseDown, onMouseUp]);
}

function emojiFromEvent(event: MouseEvent): [DataEmoji, string] | [] {
    const target = event?.target as HTMLElement;
    if (!isEmojiElement(target)) {
        return [];
    }

    return emojiFromElement(target);
}

function emojiClickOutput(
    emoji: DataEmoji,
    activeSkinTone: SkinTones,
): EmojiClickData {
    const unified = emojiUnified(emoji, activeSkinTone);
    return {
        activeSkinTone,
        emoji: parseNativeEmoji(unified),
        names: emojiNames(emoji),
        unified,
        unifiedWithoutSkinTone: emojiUnified(emoji),
    };
}
