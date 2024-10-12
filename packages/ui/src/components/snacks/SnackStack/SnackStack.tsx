import { uuid } from "@local/shared";
import { Box, BoxProps, styled } from "@mui/material";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { translateSnackMessage } from "utils/display/translationTools";
import { PubSub, TranslatedSnackMessage, UntranslatedSnackMessage } from "utils/pubsub";
import { BasicSnack, SnackSeverity } from "../BasicSnack/BasicSnack";
import { CookiesSnack } from "../CookiesSnack/CookiesSnack";
import { BasicSnackProps } from "../types";

const MAX_SNACKS = 3;

interface StyledBoxProps extends BoxProps {
    isVisible: boolean;
}

const StyledBox = styled(Box, {
    shouldForwardProp: (prop) => prop !== "isVisible",
})<StyledBoxProps>(({ isVisible, theme }) => ({
    display: isVisible ? "flex" : "none",
    flexDirection: "column",
    gap: theme.spacing(1),
    position: "fixed",
    left: "calc(8px + env(safe-area-inset-left))",
    zIndex: 20000,
    pointerEvents: "none",
    // Displays above the bottom nav bar, accounting for PWA inset-bottom
    bottom: "calc(8px + env(safe-area-inset-bottom))",
    [theme.breakpoints.down("md")]: {
        bottom: "calc(64px + env(safe-area-inset-bottom))",
    },
}));

/**
 * Displays a stack of snack messages. 
 * Most messages disappear after a few seconds, but some are persistent (e.g. No Internet Connection).
 * No more than 3 ephemeral messages are displayed at once.
 */
export function SnackStack() {
    const { t } = useTranslation();

    // FIFO queue of basic snackbars
    const [snacks, setSnacks] = useState<BasicSnackProps[]>([]);
    // Special cookie snack
    const [isCookieSnackOpen, setIsCookieSnackOpen] = useState<boolean>(false);

    // Remove an item from the queue by id
    function handleClose(id: string) {
        setSnacks((prev) => [...prev.filter((snack) => snack.id !== id)]);
    }

    function closeCookieSnack() {
        setIsCookieSnackOpen(false);
    }

    // Subscribe to snack events
    useEffect(function subscribeEffect() {
        // Subscribe to basic snacks
        const snackSub = PubSub.get().subscribe("snack", (o) => {
            // Add the snack to the queue
            setSnacks((snacks) => {
                // event can define an id, or we generate one
                const id = o.id ?? uuid();
                const newSnack: BasicSnackProps = {
                    autoHideDuration: o.autoHideDuration,
                    buttonClicked: (props) => { o.buttonClicked?.(props); handleClose(id); },
                    buttonText: o.buttonKey ? t(o.buttonKey, { ...o.buttonVariables, defaultValue: o.buttonKey }) : undefined,
                    data: o.data,
                    handleClose: () => handleClose(id),
                    id,
                    message: (o as UntranslatedSnackMessage).message ?
                        (o as UntranslatedSnackMessage).message :
                        translateSnackMessage((o as TranslatedSnackMessage).messageKey, (o as TranslatedSnackMessage).messageVariables, o.severity === "Error" ? "error" : undefined).message,
                    severity: o.severity as SnackSeverity,
                };
                // If a snack with the same id is already in the queue, replace it
                const alreadyHasId = snacks.some((snack) => snack.id === id);
                let updatedSnacks = [...snacks.map((snack) => snack.id === id ? newSnack : snack)];
                // If snack did not replace any existing snack, add it to the end
                if (!alreadyHasId) {
                    updatedSnacks.push(newSnack);
                }
                if (updatedSnacks.length > MAX_SNACKS) {
                    updatedSnacks = updatedSnacks.slice(1);
                }
                return updatedSnacks;
            });
        });
        // Subscribe to special snack events
        const cookiesSub = PubSub.get().subscribe("cookies", () => {
            setIsCookieSnackOpen(true);
        });
        return () => {
            snackSub();
            cookiesSub();
        };
    }, [t]);

    const isVisible = useMemo(() => snacks.length > 0 || isCookieSnackOpen, [snacks, isCookieSnackOpen]);

    return (
        // Snacks displayed in bottom left corner
        <StyledBox isVisible={isVisible}>
            {/* Basic snacks on top */}
            {snacks.map((snack) => (
                <BasicSnack
                    key={snack.id}
                    {...snack}
                />
            ))}
            {/* Special snacks below */}
            {/* Cookie snack */}
            {isCookieSnackOpen && <CookiesSnack handleClose={closeCookieSnack} />}
        </StyledBox>
    );
}
