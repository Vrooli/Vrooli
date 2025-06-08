import Box from "@mui/material/Box";
import Dialog from "@mui/material/Dialog";
import DialogContent from "@mui/material/DialogContent";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { styled } from "@mui/material";
import { useTheme } from "@mui/material";
import type { Palette } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useHotkeys } from "../../hooks/useHotkeys.js";
import { useMenu } from "../../hooks/useMenu.js";
import { IconCommon, IconText } from "../../icons/Icons.js";
import { ELEMENT_IDS } from "../../utils/consts.js";
import { keyComboToString } from "../../utils/display/device.js";
import { SEARCH_HIGHLIGHT_CURRENT, highlightText } from "../../utils/display/documentTools.js";
import { type MenuPayloads } from "../../utils/pubsub.js";
import { TextInput } from "../inputs/TextInput/TextInput.js";

function commonButtonSx(palette: Palette, isEnabled: boolean) {
    return {
        background: isEnabled ? palette.secondary.dark : palette.background.paper,
        borderRadius: "0",
        padding: "4px",
        color: "inherit",
        width: "30px",
        height: "100%",
    };
}

const findInPageInputProps = {
    disableUnderline: true,
} as const;

const StyledDialog = styled(Dialog)(({ theme }) => ({
    zIndex: 999999999,
    "& > .MuiDialog-container": {
        "& > .MuiPaper-root": {
            zIndex: 999999999,
        },
    },
    "& .MuiDialog-paper": {
        background: theme.palette.background.paper,
        minWidth: "min(100%, 350px)",
        position: "absolute",
        top: "0%",
        right: "0%",
        overflowY: "visible",
        margin: { xs: "8px", sm: "16px" },
        boxShadow: 12,
    },
    "& .MuiDialogContent-root": {
        padding: "12px 8px",
        borderRadius: "4px",
    },
}));

export function FindInPage() {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const ref = useRef<HTMLDivElement | null>(null);

    const [isCaseSensitive, setIsCaseSensitive] = useState(false);
    const [isWholeWord, setIsWholeWord] = useState(false);
    const [isRegex, setIsRegex] = useState(false);

    const [results, setResults] = useState<HTMLSpanElement[]>([]);
    const [resultIndex, setResultIndex] = useState(0);
    const [searchString, setSearchString] = useState<string>("");

    const onSearchChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => { setSearchString(e.target.value); }, []);
    const onCaseSensitiveChange = useCallback(() => setIsCaseSensitive(o => !o), []);
    const onWholeWordChange = useCallback(() => setIsWholeWord(o => !o), []);
    const onRegexChange = useCallback(() => setIsRegex(o => !o), []);

    useEffect(() => {
        const highlights = highlightText(searchString, isCaseSensitive, isWholeWord, isRegex);
        setResults(highlights);
        setResultIndex(0);
    }, [searchString, isCaseSensitive, isWholeWord, isRegex]);

    useEffect(() => {
        // Remove highlights from every span
        results.forEach(span => span.classList.remove(SEARCH_HIGHLIGHT_CURRENT));
        // If there are results, highlight the current result and scroll to it
        if (results.length > resultIndex && resultIndex >= 0) {
            results[resultIndex].classList.add(SEARCH_HIGHLIGHT_CURRENT);
            results[resultIndex].scrollIntoView({ behavior: "smooth", block: "center" });
        }
    }, [resultIndex, results]);

    const onPrevious = useCallback(() => setResultIndex(o => {
        if (o > 0) return o - 1;
        else return results.length - 1;
    }), [results.length]);
    const onNext = useCallback(() => setResultIndex(o => {
        if (o < results.length - 1) return o + 1;
        else return 0;
    }), [results.length]);

    const onEvent = useCallback(function onEventCallback({ isOpen }: MenuPayloads[typeof ELEMENT_IDS.FindInPage]) {
        // If the command palette is closed
        if (!isOpen) {
            // Reset search values (but keep case sensitive and other buttons the same)
            setSearchString("");
            setResults([]);
            setResultIndex(0);
        }
    }, []);
    const { isOpen, close } = useMenu({
        id: ELEMENT_IDS.FindInPage,
        onEvent,
    });

    useHotkeys([
        { keys: ["c"], altKey: true, callback: onCaseSensitiveChange },
        { keys: ["w"], altKey: true, callback: onWholeWordChange },
        { keys: ["r"], altKey: true, callback: onRegexChange },
        { keys: ["Enter"], shiftKey: true, callback: onPrevious },
        { keys: ["Enter"], callback: onNext },
    ], isOpen, ref);

    /**
     * Handles dialog close. Ignores backdrop click
     */
    const onClose = useCallback((e: React.MouseEvent<HTMLDivElement, MouseEvent>, reason: "backdropClick" | "escapeKeyDown") => {
        if (reason === "backdropClick") return;
        close();
    }, [close]);

    return (
        <StyledDialog
            open={isOpen}
            onClose={onClose}
            ref={ref}
            disableScrollLock={true}
            hideBackdrop={true}
        >
            <DialogContent sx={{
                background: palette.background.default,
                position: "relative",
                overflowY: "visible",
            }}>
                <Stack direction="row">
                    <Stack direction="row" sx={{
                        background: palette.background.paper,
                        borderRadius: "4px",
                        border: `1px solid ${searchString.length > 0 && results.length === 0 ? "red" : palette.background.textPrimary}`,
                    }}>
                        {/* Search bar */}
                        <TextInput
                            id="command-palette-search"
                            autoComplete='off'
                            autoFocus={true}
                            placeholder={t("FindInPage")}
                            value={searchString}
                            onChange={onSearchChange}
                            size="small"
                            sx={{
                                paddingLeft: "4px",
                                paddingTop: "4px",
                                paddingBottom: "4px",
                                width: "100%",
                                border: "none",
                                borderRight: `1px solid ${palette.background.textPrimary}`,
                                "& .MuiInputBase-root": {
                                    height: "100%",
                                },
                            }}
                            variant="standard"
                            InputProps={findInPageInputProps}
                        />
                        {/* Display resultIndex and total results */}
                        {results.length > 0 &&
                            <Typography variant="body2" sx={{
                                padding: "4px",
                                borderRight: `1px solid ${palette.background.textPrimary}`,
                                width: 100,
                                display: "flex",
                                justifyContent: "center",
                                alignItems: "center",
                            }}>
                                {results.length > 0 ? `${resultIndex + 1}/${results.length}` : ""}
                            </Typography>
                        }
                        {/* Buttons for case-sensitive, match whole word, and regex */}
                        <Box display="flex" alignItems="center">
                            <Tooltip title={`${t("MatchCase")} (${keyComboToString("Alt", "C")})`}>
                                <IconButton
                                    aria-label={t("MatchCase")}
                                    component="button"
                                    sx={commonButtonSx(palette, isCaseSensitive)}
                                    onClick={onCaseSensitiveChange}
                                >
                                    <IconText
                                        decorative
                                        fill={isCaseSensitive ? palette.background.textPrimary : null}
                                        name="CaseSensitive"
                                        size={20}
                                    />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title={`${t("MatchWholeWord")} (${keyComboToString("Alt", "W")})`}>
                                <IconButton
                                    aria-label={t("MatchWholeWord")}
                                    component="button"
                                    onClick={onWholeWordChange}
                                    sx={commonButtonSx(palette, isWholeWord)}
                                >
                                    <IconText
                                        decorative
                                        fill={isWholeWord ? palette.background.textPrimary : null}
                                        name="WholeWord"
                                        size={20}
                                    />
                                </IconButton>
                            </Tooltip>
                            <Tooltip title={`${t("UseRegex")} (${keyComboToString("Alt", "R")})`}>
                                <IconButton
                                    aria-label={t("UseRegex")}
                                    component="button"
                                    onClick={onRegexChange}
                                    sx={commonButtonSx(palette, isRegex)}
                                >
                                    <IconText
                                        decorative
                                        fill={isRegex ? palette.background.textPrimary : null}
                                        name="Regex"
                                        size={20}
                                    />
                                </IconButton>
                            </Tooltip>
                        </Box>
                    </Stack>
                    {/* Up and down arrows, and close icon */}
                    <Box display="flex" alignItems="center" justifyContent="flex-end">
                        <Tooltip title={`${t("ResultPrevious")} (${keyComboToString("Shift", "Enter")})`}>
                            <IconButton
                                aria-label={t("ResultPrevious")}
                                onClick={onPrevious}
                                sx={commonButtonSx(palette, false)}
                            >
                                <IconCommon
                                    decorative
                                    fill={null}
                                    name="ArrowUp"
                                    size={20}
                                />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={`${t("ResultNext")} (${keyComboToString("Enter")})`}>
                            <IconButton
                                aria-label={t("ResultNext")}
                                onClick={onNext}
                                sx={commonButtonSx(palette, false)}
                            >
                                <IconCommon
                                    decorative
                                    fill={null}
                                    name="ArrowDown"
                                    size={20}
                                />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={`${t("Close")} (${keyComboToString("Escape")})`}>
                            <IconButton
                                aria-label={t("Close")}
                                onClick={close}
                                sx={commonButtonSx(palette, false)}
                            >
                                <IconCommon
                                    decorative
                                    fill={null}
                                    name="Close"
                                    size={20}
                                />
                            </IconButton>
                        </Tooltip>
                    </Box>
                </Stack>
            </DialogContent>
        </StyledDialog>
    );
}
