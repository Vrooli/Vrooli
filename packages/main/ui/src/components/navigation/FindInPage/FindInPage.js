import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { ArrowDownIcon, ArrowUpIcon, CaseSensitiveIcon, CloseIcon, RegexIcon, WholeWordIcon } from "@local/icons";
import { Box, Dialog, DialogContent, IconButton, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { Stack } from "@mui/system";
import { useCallback, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { keyComboToString } from "../../../utils/display/device";
import { getTextNodes, normalizeText, removeHighlights, wrapMatches } from "../../../utils/display/documentTools";
import { PubSub } from "../../../utils/pubsub";
import { ColorIconButton } from "../../buttons/ColorIconButton/ColorIconButton";
const commonButtonSx = (palette) => ({
    borderRadius: "0",
    padding: "4px",
    color: "inherit",
    width: "30px",
    height: "100%",
});
const commonIconProps = (palette) => ({
    fill: palette.background.textPrimary,
    width: "20px",
    height: "20px",
});
const highlightText = (searchString, isCaseSensitive, isWholeWord, isRegex) => {
    removeHighlights("search-highlight");
    removeHighlights("search-highlight-current");
    removeHighlights("search-highlight-wrap");
    if (searchString.trim().length === 0)
        return [];
    const textNodes = getTextNodes();
    const normalizedSearchString = normalizeText(searchString);
    let regexString = normalizedSearchString;
    if (!isRegex) {
        regexString = regexString.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&");
    }
    if (isWholeWord) {
        regexString = `\\b${regexString}\\b`;
    }
    const regex = new RegExp(regexString, isCaseSensitive ? "g" : "gi");
    const highlightSpans = [];
    textNodes.forEach((textNode) => {
        const spans = wrapMatches(textNode, regex, "search-highlight-wrap", "search-highlight");
        highlightSpans.push(...spans);
    });
    if (highlightSpans.length > 0) {
        highlightSpans[0].classList.add("search-highlight-current");
    }
    return highlightSpans;
};
const FindInPage = () => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [isCaseSensitive, setIsCaseSensitive] = useState(false);
    const [isWholeWord, setIsWholeWord] = useState(false);
    const [isRegex, setIsRegex] = useState(false);
    const [results, setResults] = useState([]);
    const [resultIndex, setResultIndex] = useState(0);
    const [searchString, setSearchString] = useState("");
    const onSearchChange = useCallback((e) => { setSearchString(e.target.value); }, []);
    const onCaseSensitiveChange = useCallback(() => setIsCaseSensitive(o => !o), []);
    const onWholeWordChange = useCallback(() => setIsWholeWord(o => !o), []);
    const onRegexChange = useCallback(() => setIsRegex(o => !o), []);
    useEffect(() => {
        const highlights = highlightText(searchString, isCaseSensitive, isWholeWord, isRegex);
        setResults(highlights);
        setResultIndex(0);
    }, [searchString, isCaseSensitive, isWholeWord, isRegex]);
    useEffect(() => {
        results.forEach(span => span.classList.remove("search-highlight-current"));
        if (results.length > resultIndex) {
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
        if (o > 0)
            return o - 1;
        else
            return results.length - 1;
    }), [results.length]);
    const onNext = useCallback(() => setResultIndex(o => {
        if (o < results.length - 1)
            return o + 1;
        else
            return 0;
    }), [results.length]);
    useEffect(() => {
        const dialogSub = PubSub.get().subscribeFindInPage(() => {
            setOpen(o => {
                if (o) {
                    close();
                }
                return !o;
            });
        });
        return () => { PubSub.get().unsubscribe(dialogSub); };
    }, [close]);
    useEffect(() => {
        const handleKeyDown = (e) => {
            if (e.altKey && e.key === "c") {
                onCaseSensitiveChange();
            }
            else if (e.altKey && e.key === "w") {
                onWholeWordChange();
            }
            else if (e.altKey && e.key === "r") {
                onRegexChange();
            }
            else if (e.shiftKey && e.key === "Enter") {
                onPrevious();
            }
            else if (e.key === "Enter") {
                onNext();
            }
        };
        document.addEventListener("keydown", handleKeyDown);
        return () => {
            document.removeEventListener("keydown", handleKeyDown);
        };
    }, [close, onCaseSensitiveChange, onNext, onPrevious, onRegexChange, onWholeWordChange]);
    const handleClose = useCallback((e, reason) => {
        if (reason === "backdropClick")
            return;
        close();
    }, [close]);
    return (_jsx(Dialog, { open: open, onClose: handleClose, disableScrollLock: true, BackdropProps: { invisible: true }, sx: {
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
        }, children: _jsx(DialogContent, { sx: {
                background: palette.background.default,
                position: "relative",
                overflowY: "visible",
            }, children: _jsxs(Stack, { direction: "row", children: [_jsxs(Stack, { direction: "row", sx: {
                            background: palette.background.paper,
                            borderRadius: "4px",
                            border: `1px solid ${searchString.length > 0 && results.length === 0 ? "red" : palette.background.textPrimary}`,
                        }, children: [_jsx(TextField, { id: "command-palette-search", autoComplete: 'off', autoFocus: true, placeholder: t("FindInPage"), value: searchString, onChange: onSearchChange, size: "small", sx: {
                                    paddingLeft: "4px",
                                    paddingTop: "4px",
                                    paddingBottom: "4px",
                                    width: "100%",
                                    border: "none",
                                    borderRight: `1px solid ${palette.background.textPrimary}`,
                                    "& .MuiInputBase-root": {
                                        height: "100%",
                                    },
                                }, variant: "standard", InputProps: {
                                    disableUnderline: true,
                                } }), results.length > 0 &&
                                _jsx(Typography, { variant: "body2", sx: {
                                        padding: "4px",
                                        borderRight: `1px solid ${palette.background.textPrimary}`,
                                        width: 100,
                                        display: "flex",
                                        justifyContent: "center",
                                        alignItems: "center",
                                    }, children: results.length > 0 ? `${resultIndex + 1}/${results.length}` : "" }), _jsxs(Box, { display: "flex", alignItems: "center", children: [_jsx(Tooltip, { title: `${t("MatchCase")} ${keyComboToString("Alt", "C")}`, children: _jsx(ColorIconButton, { "aria-label": "case-sensitive", background: isCaseSensitive ? palette.secondary.dark : palette.background.paper, sx: commonButtonSx(palette), onClick: onCaseSensitiveChange, children: _jsx(CaseSensitiveIcon, { ...commonIconProps(palette) }) }) }), _jsx(Tooltip, { title: `${t("MatchWholeWord")} ${keyComboToString("Alt", "W")}`, children: _jsx(ColorIconButton, { "aria-label": "match whole word", background: isWholeWord ? palette.secondary.dark : palette.background.paper, sx: commonButtonSx(palette), onClick: onWholeWordChange, children: _jsx(WholeWordIcon, { ...commonIconProps(palette) }) }) }), _jsx(Tooltip, { title: `${t("UseRegex")} ${keyComboToString("Alt", "R")}`, children: _jsx(ColorIconButton, { "aria-label": "match regex", background: isRegex ? palette.secondary.dark : palette.background.paper, sx: commonButtonSx(palette), onClick: onRegexChange, children: _jsx(RegexIcon, { ...commonIconProps(palette) }) }) })] })] }), _jsxs(Box, { display: "flex", alignItems: "center", justifyContent: "flex-end", children: [_jsx(Tooltip, { title: `${t("ResultPrevious")} ${keyComboToString("Shift", "Enter")}`, children: _jsx(IconButton, { "aria-label": "previous result", sx: commonButtonSx(palette), onClick: onPrevious, children: _jsx(ArrowUpIcon, { ...commonIconProps(palette) }) }) }), _jsx(Tooltip, { title: `${t("ResultNext")} ${keyComboToString("Enter")}`, children: _jsx(IconButton, { "aria-label": "next result", sx: commonButtonSx(palette), onClick: onNext, children: _jsx(ArrowDownIcon, { ...commonIconProps(palette) }) }) }), _jsx(Tooltip, { title: `${t("Close")} ${keyComboToString("Escape")}`, children: _jsx(IconButton, { "aria-label": "close", sx: commonButtonSx(palette), onClick: close, children: _jsx(CloseIcon, { ...commonIconProps(palette) }) }) })] })] }) }) }));
};
export { FindInPage };
//# sourceMappingURL=FindInPage.js.map