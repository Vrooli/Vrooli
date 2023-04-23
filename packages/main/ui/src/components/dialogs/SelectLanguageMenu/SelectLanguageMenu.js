import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { ArrowDropDownIcon, ArrowDropUpIcon, CompleteIcon, DeleteIcon, LanguageIcon } from "@local/icons";
import { IconButton, ListItem, Popover, Stack, TextField, Tooltip, Typography, useTheme } from "@mui/material";
import { useCallback, useContext, useMemo, useState } from "react";
import { FixedSizeList } from "react-window";
import { translateTranslate } from "../../../api/generated/endpoints/translate_translate";
import { useCustomLazyQuery } from "../../../api/hooks";
import { queryWrapper } from "../../../api/utils";
import { AllLanguages, getLanguageSubtag, getUserLanguages } from "../../../utils/display/translationTools";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { ListMenu } from "../ListMenu/ListMenu";
import { MenuTitle } from "../MenuTitle/MenuTitle";
const autoTranslateLanguages = [
    "zh",
    "es",
    "en",
    "hi",
    "pt",
    "ru",
    "ja",
    "ko",
    "tr",
    "fr",
    "de",
    "it",
    "ar",
    "id",
    "fa",
    "pl",
    "uk",
    "nl",
    "da",
    "fi",
    "az",
    "el",
    "hu",
    "cs",
    "sk",
    "ga",
    "sv",
    "he",
    "eo",
];
const titleId = "select-language-dialog-title";
export const SelectLanguageMenu = ({ currentLanguage, handleDelete, handleCurrent, isEditing = false, languages, sxs, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event) => {
        setSearchString(event.target.value);
    }, []);
    const [getAutoTranslation] = useCustomLazyQuery(translateTranslate);
    const autoTranslate = useCallback((source, target) => {
        const sourceTranslation = languages.find(l => l === source);
        if (!sourceTranslation) {
            PubSub.get().publishSnack({ messageKey: "CouldNotFindTranslation", severity: "Error" });
            return;
        }
        queryWrapper({
            query: getAutoTranslation,
            input: { fields: JSON.stringify(sourceTranslation), languageSource: source, languageTarget: target },
            onSuccess: (data) => {
                if (data) {
                    console.log("TODO");
                }
                else {
                    PubSub.get().publishSnack({ messageKey: "FailedToTranslate", severity: "Error" });
                }
            },
            errorMessage: () => ({ key: "FailedToTranslate" }),
        });
    }, [getAutoTranslation, languages]);
    const translateSourceOptions = useMemo(() => {
        const autoTranslateLanguagesFiltered = autoTranslateLanguages.filter(l => languages?.indexOf(l) !== -1);
        return autoTranslateLanguagesFiltered.map(l => ({ label: AllLanguages[l], value: l }));
    }, [languages]);
    const [translateSourceAnchor, setTranslateSourceAnchor] = useState(null);
    const openTranslateSource = useCallback((ev, targetLanguage) => {
        ev.stopPropagation();
        if (translateSourceOptions.length === 1) {
            autoTranslate(translateSourceOptions[0].value, targetLanguage);
        }
        else {
            console.log("openTranslateSource", targetLanguage);
            setTranslateSourceAnchor(ev.currentTarget);
        }
    }, [autoTranslate, translateSourceOptions]);
    const closeTranslateSource = useCallback(() => setTranslateSourceAnchor(null), []);
    const handleTranslateSourceSelect = useCallback((path) => {
        console.log("TODO");
    }, []);
    const languageOptions = useMemo(() => {
        const userLanguages = getUserLanguages(session);
        const selected = languages.map((l) => getLanguageSubtag(l));
        const sortedSelectedLanguages = selected.sort((a, b) => {
            const aIndex = userLanguages.indexOf(a);
            const bIndex = userLanguages.indexOf(b);
            if (aIndex === -1 && bIndex === -1) {
                return 0;
            }
            else if (aIndex === -1) {
                return 1;
            }
            else if (bIndex === -1) {
                return -1;
            }
            else {
                return aIndex - bIndex;
            }
        }) ?? [];
        const userLanguagesFiltered = userLanguages.filter(l => selected.indexOf(l) === -1);
        const allLanguagesFiltered = isEditing ? (Object.keys(AllLanguages)).filter(l => selected.indexOf(l) === -1 && userLanguages.indexOf(l) === -1 && autoTranslateLanguages.indexOf(l) === -1) : [];
        const displayed = [...sortedSelectedLanguages, ...userLanguagesFiltered, ...allLanguagesFiltered];
        let options = displayed.map(l => [l, AllLanguages[l]]);
        if (searchString.length > 0) {
            options = options.filter((o) => o[1].toLowerCase().includes(searchString.toLowerCase()));
        }
        return options;
    }, [isEditing, searchString, session, languages]);
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event) => {
        setAnchorEl(event.currentTarget);
        if (currentLanguage)
            handleCurrent(currentLanguage);
    }, [currentLanguage, handleCurrent]);
    const onClose = useCallback(() => {
        setSearchString("");
        setAnchorEl(null);
    }, []);
    const onDelete = useCallback((e, language) => {
        e.preventDefault();
        e.stopPropagation();
        if (handleDelete)
            handleDelete(language);
    }, [handleDelete]);
    return (_jsxs(_Fragment, { children: [_jsx(ListMenu, { id: "auto-translate-from-menu", anchorEl: translateSourceAnchor, title: 'Translate from...', data: translateSourceOptions, onSelect: handleTranslateSourceSelect, onClose: closeTranslateSource, zIndex: zIndex + 2 }), _jsxs(Popover, { open: open, anchorEl: anchorEl, onClose: onClose, "aria-labelledby": titleId, sx: {
                    zIndex: zIndex + 1,
                    "& .MuiPopover-paper": {
                        background: "transparent",
                        border: "none",
                        paddingBottom: 1,
                    },
                }, anchorOrigin: {
                    vertical: "bottom",
                    horizontal: "center",
                }, transformOrigin: {
                    vertical: "top",
                    horizontal: "center",
                }, children: [_jsx(MenuTitle, { ariaLabel: titleId, title: "Select Language", onClose: onClose }), _jsxs(Stack, { direction: "column", spacing: 2, sx: {
                            width: "min(100vw, 400px)",
                            maxHeight: "min(100vh, 600px)",
                            maxWidth: "100%",
                            overflowX: "auto",
                            overflowY: "hidden",
                            background: palette.background.default,
                            borderRadius: "8px",
                            padding: "8px",
                            "&::-webkit-scrollbar": {
                                width: 10,
                            },
                            "&::-webkit-scrollbar-track": {
                                backgroundColor: "#dae5f0",
                            },
                            "&::-webkit-scrollbar-thumb": {
                                borderRadius: "100px",
                                backgroundColor: "#409590",
                            },
                        }, children: [_jsx(TextField, { placeholder: "Enter language...", autoFocus: true, value: searchString, onChange: updateSearchString, sx: {
                                    paddingLeft: 1,
                                    paddingRight: 1,
                                } }), _jsx(FixedSizeList, { height: 600, width: 384, itemSize: 46, itemCount: languageOptions.length, overscanCount: 5, style: {
                                    scrollbarColor: "#409590 #dae5f0",
                                    scrollbarWidth: "thin",
                                    maxWidth: "100%",
                                }, children: (props) => {
                                    const { index, style } = props;
                                    const option = languageOptions[index];
                                    const isSelected = option[0] === currentLanguage || languages.some((l) => getLanguageSubtag(l) === getLanguageSubtag(option[0]));
                                    const isCurrent = option[0] === currentLanguage;
                                    const canDelete = isSelected && isEditing && languages.length > 1;
                                    const canAutoTranslate = !isSelected && translateSourceOptions.length > 0 && autoTranslateLanguages.includes(option[0]);
                                    return (_jsxs(ListItem, { style: style, disablePadding: true, button: true, onClick: () => { handleCurrent(option[0]); onClose(); }, sx: {
                                            background: isCurrent ? palette.secondary.light : palette.background.default,
                                            color: isCurrent ? palette.secondary.contrastText : palette.background.textPrimary,
                                            "&:hover": {
                                                background: isCurrent ? palette.secondary.light : palette.background.default,
                                                filter: "brightness(105%)",
                                            },
                                        }, children: [isSelected && (_jsx(CompleteIcon, { fill: (isCurrent) ? palette.secondary.contrastText : palette.background.textPrimary })), _jsx(Typography, { variant: "body2", style: {
                                                    display: "block",
                                                    marginRight: "auto",
                                                    marginLeft: isSelected ? "8px" : "0",
                                                }, children: option[1] }), canDelete && (_jsx(Tooltip, { title: "Delete translation for this language", children: _jsx(IconButton, { size: "small", onClick: (e) => onDelete(e, option[0]), children: _jsx(DeleteIcon, { fill: isCurrent ? palette.secondary.contrastText : palette.background.textPrimary }) }) }))] }, index));
                                } })] })] }), _jsx(Tooltip, { title: AllLanguages[currentLanguage] ?? "", placement: "top", children: _jsxs(Stack, { direction: "row", spacing: 0, onClick: onOpen, sx: {
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                        borderRadius: "50px",
                        cursor: "pointer",
                        background: "#4e7d31",
                        boxShadow: 4,
                        "&:hover": {
                            filter: "brightness(120%)",
                        },
                        transition: "all 0.2s ease-in-out",
                        width: "fit-content",
                        ...(sxs?.root ?? {}),
                    }, children: [_jsx(IconButton, { size: "large", sx: { padding: "4px" }, children: _jsx(LanguageIcon, { fill: "white" }) }), isEditing && _jsx(Typography, { variant: "body2", sx: { color: "white", marginRight: "8px" }, children: currentLanguage?.toLocaleUpperCase() }), _jsx(IconButton, { size: "large", "aria-label": "language-select", sx: { padding: "4px", marginLeft: "-8px" }, children: open ? _jsx(ArrowDropUpIcon, { fill: "white" }) : _jsx(ArrowDropDownIcon, { fill: "white" }) })] }) })] }));
};
//# sourceMappingURL=SelectLanguageMenu.js.map