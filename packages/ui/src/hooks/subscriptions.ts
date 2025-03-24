import { Breakpoints, useTheme } from "@mui/material";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Dimensions } from "../types.js";
import { getCookie, removeCookie, setCookie } from "../utils/localStorage.js";
import { PubSub, RichInputToolbarViewSize } from "../utils/pubsub.js";
import { useDimensions } from "./useDimensions.js";

/**
 * Tracks if the site should display in left-handed mode.
 */
export function useIsLeftHanded() {
    const [isLeftHanded, setIsLeftHanded] = useState<boolean>(getCookie("IsLeftHanded"));
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("isLeftHanded", (data) => {
            setIsLeftHanded(data);
        });
        return unsubscribe;
    }, []);

    return isLeftHanded;
}

const SHOW_MINIMAL_VIEW_AT_PX = 375;

function determineViewSize(
    dimensions: Dimensions,
    breakpoints: Breakpoints,
    storedViewSize: RichInputToolbarViewSize,
) {
    // Determine detault view based on dimensions
    let viewSize: RichInputToolbarViewSize = "full";
    if (dimensions.width <= SHOW_MINIMAL_VIEW_AT_PX) {
        viewSize = "minimal";
    }
    else if (dimensions.width <= breakpoints.values.sm) {
        viewSize = "partial";
    }
    // If the stored view size is smaller than the current view size, use the stored view size
    const viewOrder = ["minimal", "partial", "full"];
    const storedViewIndex = viewOrder.indexOf(storedViewSize);
    const currentViewIndex = viewOrder.indexOf(viewSize);
    if (storedViewIndex < currentViewIndex) {
        viewSize = storedViewSize;
    }
    return viewSize;
}

export function useRichInputToolbarViewSize() {
    const { breakpoints } = useTheme();
    const { dimensions, ref: dimRef } = useDimensions();

    const [richInputToolbarViewSize, setRichInputToolbarViewSize] = useState<RichInputToolbarViewSize>(getCookie("RichInputToolbarViewSize"));
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("richInputToolbarViewSize", (data) => {
            setRichInputToolbarViewSize(data);
        });
        return unsubscribe;
    }, []);

    const viewSize = useMemo(function viewSizeMemo() {
        return determineViewSize(dimensions, breakpoints, richInputToolbarViewSize);
    }, [dimensions, breakpoints, richInputToolbarViewSize]);

    const handleUpdateViewSize = useCallback(function handleUpdateViewSize(viewSize: RichInputToolbarViewSize) {
        setCookie("RichInputToolbarViewSize", viewSize);
        PubSub.get().publish("richInputToolbarViewSize", viewSize);
    }, []);

    return { dimRef, handleUpdateViewSize, viewSize };
}

export function useShowBotWarning() {
    const [showBotWarning, setShowBotWarning] = useState<boolean>(getCookie("ShowBotWarning"));
    useEffect(() => {
        const unsubscribe = PubSub.get().subscribe("showBotWarning", (data) => {
            setShowBotWarning(data);
        });
        return unsubscribe;
    }, []);

    const handleUpdateShowBotWarning = useCallback(function handleUpdateShowBotWarning(showWarning: boolean | null | undefined) {
        if (typeof showWarning !== "boolean") {
            removeCookie("ShowBotWarning");
            PubSub.get().publish("showBotWarning", true);
        } else {
            setCookie("ShowBotWarning", showWarning);
            PubSub.get().publish("showBotWarning", showWarning);
        }
    }, []);

    return { handleUpdateShowBotWarning, showBotWarning };
}
