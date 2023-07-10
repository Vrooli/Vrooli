import { ArrowDownIcon, ArrowUpIcon, CaseSensitiveIcon, CloseIcon, RegexIcon, WholeWordIcon } from "@local/shared";
import { Box, Dialog, DialogContent, IconButton, Palette, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { keyComboToString } from "utils/display/device";
import { getTextNodes, normalizeText, removeHighlights, wrapMatches } from "utils/display/documentTools";
import { PubSub } from "utils/pubsub";

const commonButtonSx = (palette: Palette) => ({
    borderRadius: "0",
    padding: "4px",
    color: "inherit",
    width: "30px",
    height: "100%",
});

const commonIconProps = (palette: Palette) => ({
    fill: palette.background.textPrimary,
    width: "20px",
    height: "20px",
});

/**
 * Highlights all instances of the search term. Accomplishes this by doing the following: 
 * 1. Remove all previous highlights
 * 2. Finds all text nodes in the document
 * 3. Maps diacritics from the search term and text nodes to their base characters
 * 4. Checks if the search term is a substring of the text node, while adhering to the search options (case sensitive, whole word, regex)
 * 5. Highlights the text node if it matches the search term, by wrapping it in a span with a custom class (custom class is necessary to remove the highlight later)
 * @param searchString The search term
 * @param isCaseSensitive Whether or not the search should be case sensitive
 * @param isWholeWord Whether or not the search should be whole word
 * @param isRegex Whether or not the search should be regex
 * @returns Highlight spans
 */
const highlightText = (
    searchString: string,
    isCaseSensitive: boolean,
    isWholeWord: boolean,
    isRegex: boolean,
): HTMLSpanElement[] => {
    // Remove all previous highlights
    removeHighlights("search-highlight"); // General highlight class
    removeHighlights("search-highlight-current"); // Highlight class for the current match
    removeHighlights("search-highlight-wrap"); // Wrapper class for a highlight's text node
    // If text is empty, return
    if (searchString.trim().length === 0) return [];
    // Finds all text nodes in the document
    const textNodes: Text[] = getTextNodes();
    // Normalize the search term
    const normalizedSearchString = normalizeText(searchString);
    // Build the regex
    let regexString = normalizedSearchString;
    // If not regex, escape regex characters
    if (!isRegex) { regexString = regexString.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&"); }
    // If whole word, wrap the search term in word boundaries
    if (isWholeWord) { regexString = `\\b${regexString}\\b`; }
    // Create global regex expression 
    const regex = new RegExp(regexString, isCaseSensitive ? "g" : "gi");
    // Loop through all text nodes, and store highlights. 
    // These will be used for previous/next buttons
    const highlightSpans: HTMLSpanElement[] = [];
    textNodes.forEach((textNode) => {
        const spans = wrapMatches(textNode, regex, "search-highlight-wrap", "search-highlight");
        highlightSpans.push(...spans);
    });
    // If there is at least one highlight, change the first highlight's class to 'search-highlight-current'
    if (highlightSpans.length > 0) {
        highlightSpans[0].classList.add("search-highlight-current");
    }
    return highlightSpans;
};

const FindInPage = () => {
    console.log("rendering findinpage...");
    const { palette } = useTheme();
    const { t } = useTranslation();

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
        results.forEach(span => span.classList.remove("search-highlight-current"));
        // If there are results, highlight the current result and scroll to it
        if (results.length > resultIndex && resultIndex >= 0) {
            results[resultIndex].classList.add("search-highlight-current");
            results[resultIndex].scrollIntoView({ behavior: "smooth", block: "center" });
        }
    }, [resultIndex, results]);

    const [open, setOpen] = useState(false);
    const close = useCallback(() => {
        setOpen(false);
        setSearchString("");
        setResults([]);
        setResultIndex(0);
    }, []);

    const onPrevious = useCallback(() => setResultIndex(o => {
        if (o > 0) return o - 1;
        else return results.length - 1;
    }), [results.length]);
    const onNext = useCallback(() => setResultIndex(o => {
        if (o < results.length - 1) return o + 1;
        else return 0;
    }), [results.length]);

    useEffect(() => {
        const dialogSub = PubSub.get().subscribeFindInPage(() => {
            setOpen(o => {
                // If turning off, reset search values (but keep case sensitive and other buttons the same)
                if (o) { close(); }
                return !o;
            });
        });
        return () => { PubSub.get().unsubscribe(dialogSub); };
    }, [close]);

    // Handle keyboard shortcuts
    useEffect(() => {
        const handleKeyDown = (e: KeyboardEvent) => {
            // ALT + C - Match case
            if (e.altKey && e.key === "c") { onCaseSensitiveChange(); }
            // ALT + W - Match whole word
            else if (e.altKey && e.key === "w") { onWholeWordChange(); }
            // ALT + R - Use regex
            else if (e.altKey && e.key === "r") { onRegexChange(); }
            // SHIFT + ENTER - Previous result
            else if (e.shiftKey && e.key === "Enter") { onPrevious(); }
            // ENTER - Next result
            else if (e.key === "Enter") { onNext(); }
        };
        // attach the event listener
        document.addEventListener("keydown", handleKeyDown);
        // remove the event listener
        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [close, onCaseSensitiveChange, onNext, onPrevious, onRegexChange, onWholeWordChange]);

    /**
     * Handles dialog close. Ignores backdrop click
     */
    const handleClose = useCallback((e: React.MouseEvent<HTMLDivElement, MouseEvent>, reason: "backdropClick" | "escapeKeyDown") => {
        if (reason === "backdropClick") return;
        close();
    }, [close]);

    return (
        <Dialog
            open={open}
            onClose={handleClose}
            disableScrollLock={true}
            BackdropProps={{ invisible: true }}
            sx={{
                "& .MuiDialog-paper": {
                    background: palette.background.paper,
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
            }}
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
                        <TextField
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
                            InputProps={{
                                disableUnderline: true,
                            }}
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
                                <ColorIconButton
                                    aria-label="case-sensitive"
                                    background={isCaseSensitive ? palette.secondary.dark : palette.background.paper}
                                    sx={commonButtonSx(palette)}
                                    onClick={onCaseSensitiveChange}
                                >
                                    <CaseSensitiveIcon {...commonIconProps(palette)} />
                                </ColorIconButton>
                            </Tooltip>
                            <Tooltip title={`${t("MatchWholeWord")} (${keyComboToString("Alt", "W")})`}>
                                <ColorIconButton
                                    aria-label="match whole word"
                                    background={isWholeWord ? palette.secondary.dark : palette.background.paper}
                                    sx={commonButtonSx(palette)}
                                    onClick={onWholeWordChange}
                                >
                                    <WholeWordIcon {...commonIconProps(palette)} />
                                </ColorIconButton>
                            </Tooltip>
                            <Tooltip title={`${t("UseRegex")} (${keyComboToString("Alt", "R")})`}>
                                <ColorIconButton
                                    aria-label="match regex"
                                    background={isRegex ? palette.secondary.dark : palette.background.paper}
                                    sx={commonButtonSx(palette)}
                                    onClick={onRegexChange}
                                >
                                    <RegexIcon {...commonIconProps(palette)} />
                                </ColorIconButton>
                            </Tooltip>
                        </Box>
                    </Stack>
                    {/* Up and down arrows, and close icon */}
                    <Box display="flex" alignItems="center" justifyContent="flex-end">
                        <Tooltip title={`${t("ResultPrevious")} (${keyComboToString("Shift", "Enter")})`}>
                            <IconButton
                                aria-label="previous result"
                                sx={commonButtonSx(palette)}
                                onClick={onPrevious}
                            >
                                <ArrowUpIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={`${t("ResultNext")} (${keyComboToString("Enter")})`}>
                            <IconButton
                                aria-label="next result"
                                sx={commonButtonSx(palette)}
                                onClick={onNext}
                            >
                                <ArrowDownIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                        <Tooltip title={`${t("Close")} (${keyComboToString("Escape")})`}>
                            <IconButton
                                aria-label="close"
                                sx={commonButtonSx(palette)}
                                onClick={close}
                            >
                                <CloseIcon {...commonIconProps(palette)} />
                            </IconButton>
                        </Tooltip>
                    </Box>
                </Stack>
            </DialogContent>
        </Dialog >
    );
};

export { FindInPage };
