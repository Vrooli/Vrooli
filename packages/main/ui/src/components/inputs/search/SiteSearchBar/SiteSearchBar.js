import { createElement as _createElement } from "react";
import { jsx as _jsx, Fragment as _Fragment, jsxs as _jsxs } from "react/jsx-runtime";
import { ActionIcon, ApiIcon, BookmarkFilledIcon, DeleteIcon, HelpIcon, HistoryIcon, NoteIcon, OrganizationIcon, PlayIcon, ProjectIcon, RoutineIcon, SearchIcon, ShortcutIcon, SmartContractIcon, StandardIcon, UserIcon, VisibleIcon } from "@local/icons";
import { Autocomplete, CircularProgress, IconButton, Input, ListItemIcon, ListItemText, MenuItem, Paper, Popper, Tooltip, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { getCurrentUser } from "../../../../utils/authentication/session";
import { useDebounce } from "../../../../utils/hooks/useDebounce";
import { getLocalStorageKeys } from "../../../../utils/localStorage";
import { performAction } from "../../../../utils/navigation/quickActions";
import { SessionContext } from "../../../../utils/SessionContext";
import { BookmarkButton } from "../../../buttons/BookmarkButton/BookmarkButton";
import { MicrophoneButton } from "../../../buttons/MicrophoneButton/MicrophoneButton";
const getSearchHistory = (searchBarId, userId) => {
    const existingHistoryString = localStorage.getItem(`search-history-${searchBarId}-${userId}`) ?? "{}";
    let existingHistory = {};
    try {
        const parsedHistory = JSON.parse(existingHistoryString);
        if (typeof existingHistory !== "object")
            existingHistory = {};
        else
            existingHistory = parsedHistory;
    }
    catch (e) {
        existingHistory = {};
    }
    return existingHistory;
};
const updateHistoryItems = (searchBarId, userId, options) => {
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: "search-history-",
        suffix: userId,
    });
    searchHistoryKeys.forEach(key => {
        let existingHistory = {};
        try {
            const parsedHistory = JSON.parse(localStorage.getItem(key) ?? "{}");
            if (typeof existingHistory !== "object")
                existingHistory = {};
            else
                existingHistory = parsedHistory;
        }
        catch (e) {
            existingHistory = {};
        }
        const updatedHistory = options.map(option => {
            if (option.__typename === "Shortcut" || option.__typename === "Action")
                return null;
            const historyItem = existingHistory[option.label];
            if (historyItem && historyItem.option.__typename !== "Shortcut" && historyItem.option.__typename !== "Action") {
                const { bookmarks, isBookmarked } = option;
                if (bookmarks !== historyItem.option.bookmarks || isBookmarked !== historyItem.option.isBookmarked) {
                    return {
                        timestamp: historyItem.timestamp,
                        option: {
                            ...option,
                            bookmarks,
                            isBookmarked,
                        },
                    };
                }
            }
            return null;
        }).filter(Boolean);
        if (updatedHistory.length > 0) {
            for (const historyItem of updatedHistory) {
                existingHistory[historyItem.option.label] = historyItem;
            }
            localStorage.setItem(key, JSON.stringify(existingHistory));
        }
    });
};
const IconMap = {
    Action: ActionIcon,
    Api: ApiIcon,
    Bookmark: BookmarkFilledIcon,
    Note: NoteIcon,
    Organization: OrganizationIcon,
    Project: ProjectIcon,
    Question: HelpIcon,
    Routine: RoutineIcon,
    Run: PlayIcon,
    Shortcut: ShortcutIcon,
    SmartContract: SmartContractIcon,
    Standard: StandardIcon,
    User: UserIcon,
    View: VisibleIcon,
};
const typeToIcon = (type, fill) => {
    const Icon = IconMap[type];
    return Icon ? _jsx(Icon, { fill: fill }) : null;
};
const FullWidthPopper = function (props) {
    return _jsx(Popper, { ...props, style: {
            width: "fit-content",
            maxWidth: "100%",
        }, placement: "bottom-start" });
};
export function SiteSearchBar({ id = "search-bar", placeholder, options = [], value, onChange, onInputChange, debounce = 200, loading = false, showSecondaryLabel = false, sxs, ...props }) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();
    const [internalValue, setInternalValue] = useState(value);
    const [highlightedOption, setHighlightedOption] = useState(null);
    const onChangeDebounced = useDebounce(onChange, debounce);
    useEffect(() => setInternalValue(value), [value]);
    const handleChange = useCallback((value) => {
        setInternalValue(value);
        setHighlightedOption(null);
        onChangeDebounced(value);
    }, [onChangeDebounced]);
    const handleChangeEvent = useCallback((event) => {
        const { value } = event.target;
        handleChange(value);
    }, [handleChange]);
    const [optionsWithHistory, setOptionsWithHistory] = useState(options);
    useEffect(() => {
        const searchHistory = getSearchHistory(id, userId ?? "");
        let filteredHistory = Object.entries(searchHistory).filter(([key]) => key.toLowerCase().includes(internalValue.toLowerCase()));
        filteredHistory = filteredHistory.sort((a, b) => { return b[1].timestamp - a[1].timestamp; });
        let historyOptions = filteredHistory.map(([, value]) => ({ ...value.option, isFromHistory: true }));
        historyOptions = historyOptions.slice(0, 5);
        const filteredOptions = options.filter(option => !historyOptions.some(historyOption => historyOption.id === option.id));
        updateHistoryItems(id, userId ?? "", filteredOptions);
        let combinedOptions = [...historyOptions, ...filteredOptions];
        combinedOptions = combinedOptions.filter(option => option.label && option.label.trim() !== "" && option.id);
        setOptionsWithHistory(combinedOptions);
    }, [options, internalValue, id, userId]);
    const removeFromHistory = useCallback((option) => {
        const existingHistory = getSearchHistory(id, userId ?? "");
        delete existingHistory[option.label];
        localStorage.setItem(`search-history-${id}-${userId ?? ""}`, JSON.stringify(existingHistory));
        setOptionsWithHistory(optionsWithHistory.filter(o => o.id !== option.id));
    }, [id, optionsWithHistory, userId]);
    const noOptionsText = useMemo(() => {
        if (loading || (internalValue !== value)) {
            return (_jsxs(_Fragment, { children: [_jsx(CircularProgress, { color: "secondary", size: 20, sx: {
                            marginRight: 1,
                        } }), t("Loading")] }));
        }
        return t("NoResults", { ns: "error" });
    }, [loading, internalValue, value, t]);
    const onHighlightChange = useCallback((event, option, reason) => {
        if (option && option.label && reason === "keyboard") {
            setHighlightedOption(option);
        }
    }, []);
    const handleSelect = useCallback((option) => {
        const existingHistory = getSearchHistory(id, userId ?? "");
        if (Object.keys(existingHistory).length > 500) {
            const oldestKey = Object.keys(existingHistory).sort((a, b) => existingHistory[a].timestamp - existingHistory[b].timestamp)[0];
            delete existingHistory[oldestKey];
        }
        existingHistory[option.label] = {
            timestamp: Date.now(),
            option: ({ ...option, isFromHistory: true }),
        };
        localStorage.setItem(`search-history-${id}-${userId ?? ""}`, JSON.stringify(existingHistory));
        if (option.__typename === "Action") {
            performAction(option, session);
            setInternalValue("");
            setHighlightedOption(null);
            onChangeDebounced("");
        }
        onInputChange(option);
    }, [id, onChangeDebounced, onInputChange, session, userId]);
    const onSubmit = useCallback((event, value, reason, details) => {
        if (highlightedOption) {
            handleSelect(highlightedOption);
        }
    }, [highlightedOption, handleSelect]);
    const optionColor = useCallback((isFromHistory, isSecondary) => {
        if (isFromHistory)
            return palette.mode === "dark" ? "hotpink" : "purple";
        return isSecondary ? palette.background.textSecondary : palette.background.textPrimary;
    }, [palette]);
    const handleBookmark = useCallback((option, isBookmarked, event) => {
        if (option.__typename === "Shortcut" || option.__typename === "Action")
            return;
        event.stopPropagation();
        const updatedOption = { ...option, isBookmarked, bookmarks: isBookmarked ? (option.bookmarks ?? 0) + 1 : (option.bookmarks ?? 1) - 1 };
        updateHistoryItems(id, userId ?? "", [updatedOption]);
        setOptionsWithHistory(optionsWithHistory.map(o => o.id === option.id ? updatedOption : o));
    }, [id, optionsWithHistory, userId]);
    const onKeyDown = useCallback((event) => {
        if (event.key === "ArrowRight" && highlightedOption) {
            setInternalValue(highlightedOption.label);
            onChangeDebounced("");
        }
    }, [highlightedOption, onChangeDebounced]);
    return (_jsx(Autocomplete, { disablePortal: true, id: id, options: optionsWithHistory, onHighlightChange: onHighlightChange, onKeyDown: onKeyDown, inputValue: internalValue, getOptionLabel: (option) => option.label ?? "", onSubmit: (event) => {
            event.preventDefault();
            event.stopPropagation();
        }, onChange: onSubmit, PopperComponent: FullWidthPopper, renderOption: (props, option) => {
            return (_createElement(MenuItem, { ...props, key: option.id, onClick: () => {
                    const label = option.label ?? "";
                    setInternalValue(label);
                    onChangeDebounced(label);
                    handleSelect(option);
                }, sx: {
                    color: optionColor(option.isFromHistory, false),
                } },
                option.isFromHistory && (_jsx(ListItemIcon, { sx: { display: { xs: "none", sm: "block" } }, children: _jsx(HistoryIcon, { fill: optionColor(true, false) }) })),
                _jsx(ListItemIcon, { children: typeToIcon(option.__typename, optionColor(option.isFromHistory, true)) }),
                _jsx(ListItemText, { sx: {
                        "& .MuiTypography-root": {
                            textOverflow: "ellipsis",
                            whiteSpace: "nowrap",
                            overflow: "hidden",
                        },
                    }, children: option.label }),
                option.__typename !== "Shortcut" && option.__typename !== "Action" && option.bookmarks !== undefined && _jsx(BookmarkButton, { isBookmarked: option.isBookmarked, objectId: option.id, onChange: (isBookmarked, event) => handleBookmark(option, isBookmarked, event), showBookmarks: true, bookmarkFor: option.__typename, bookmarks: option.bookmarks, sxs: { root: { marginRight: 1 } } }),
                option.isFromHistory && _jsx(Tooltip, { placement: 'right', title: 'Remove', children: _jsx(IconButton, { size: "small", onClick: (event) => {
                            event.stopPropagation();
                            removeFromHistory(option);
                        }, children: _jsx(DeleteIcon, { fill: optionColor(true, true) }) }) })));
        }, renderInput: (params) => (_jsxs(Paper, { component: "form", sx: {
                ...(sxs?.paper ?? {}),
                p: "2px 4px",
                display: "flex",
                alignItems: "center",
                borderRadius: "10px",
            }, children: [_jsx(Input, { id: params.id, disabled: params.disabled, disableUnderline: true, fullWidth: true, value: internalValue, onChange: handleChangeEvent, placeholder: t(`${placeholder ?? "Search"}`) + "...", autoFocus: props.autoFocus ?? false, inputProps: params.inputProps, ref: params.InputProps.ref, size: params.size, sx: {
                        width: "100vw",
                        ml: 1,
                        flex: 1,
                        "& .MuiAutocomplete-popper": {
                            width: "100vw!important",
                            left: "0",
                            right: "0",
                            "& .MuiPaper-root": {
                                marginTop: "0",
                            },
                        },
                    } }), _jsx(MicrophoneButton, { onTranscriptChange: handleChange }), _jsx(IconButton, { sx: {
                        width: "48px",
                        height: "48px",
                    }, "aria-label": "main-search-icon", children: _jsx(SearchIcon, { fill: palette.background.textSecondary }) })] })), noOptionsText: noOptionsText, sx: {
            ...(sxs?.root ?? {}),
            "& .MuiAutocomplete-inputRoot": {
                paddingRight: "0 !important",
            },
        } }));
}
//# sourceMappingURL=SiteSearchBar.js.map