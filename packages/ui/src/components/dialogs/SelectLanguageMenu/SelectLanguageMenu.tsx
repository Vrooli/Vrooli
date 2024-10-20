import { endpointGetTranslate, Translate, TranslateInput } from "@local/shared";
import { IconButton, ListItem, Popover, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { SessionContext } from "contexts";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useZIndex } from "hooks/useZIndex";
import { ArrowDropDownIcon, ArrowDropUpIcon, CompleteIcon, DeleteIcon, LanguageIcon } from "icons";
import { MouseEvent, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FixedSizeList } from "react-window";
import { AllLanguages, getLanguageSubtag, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { ListMenu } from "../ListMenu/ListMenu";
import { MenuTitle } from "../MenuTitle/MenuTitle";
import { ListMenuItemData, SelectLanguageMenuProps } from "../types";

/**
 * Languages which support auto-translations through LibreTranslate. 
 * These are sorted by popularity (as best as I could).
 */
const autoTranslateLanguages = [
    "zh", // Chinese
    "es", // Spanish
    "en", // English
    "hi", // Hindi
    "pt", // Portuguese
    "ru", // Russian
    "ja", // Japanese
    "ko", // Korean
    "tr", // Turkish
    "fr", // French
    "de", // German
    "it", // Italian
    "ar", // Arabic
    "id", // Indonesian
    "fa", // Persian
    "pl", // Polish
    "uk", // Ukrainian
    "nl", // Dutch
    "da", // Danish
    "fi", // Finnish
    "az", // Azerbaijani
    "el", // Greek
    "hu", // Hungarian
    "cs", // Czech
    "sk", // Slovak
    "ga", // Irish
    "sv", // Swedish
    "he", // Hebrew
    "eo", // Esperanto
] as const;

const titleId = "select-language-dialog-title";

export function SelectLanguageMenu({
    currentLanguage,
    handleDelete,
    handleCurrent,
    isEditing = false,
    languages,
    sxs,
}: SelectLanguageMenuProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    // Auto-translates from source to target language
    const [getAutoTranslation] = useLazyFetch<TranslateInput, Translate>(endpointGetTranslate);
    const autoTranslate = useCallback((source: string, target: string) => {
        // Get source translation
        const sourceTranslation = languages.find(l => l === source);
        if (!sourceTranslation) {
            PubSub.get().publish("snack", { messageKey: "CouldNotFindTranslation", severity: "Error" });
            return;
        }
        fetchLazyWrapper<TranslateInput, Translate>({
            fetch: getAutoTranslation,
            inputs: { fields: JSON.stringify(sourceTranslation), languageSource: source, languageTarget: target },
            onSuccess: (data) => {
                // Try parse
                if (data) {
                    console.log("TODO");
                } else {
                    PubSub.get().publish("snack", { messageKey: "FailedToTranslate", severity: "Error" });
                }
            },
            errorMessage: () => ({ messageKey: "FailedToTranslate" }),
        });
    }, [getAutoTranslation, languages]);

    // Menu for selecting language to auto-translate from
    const translateSourceOptions = useMemo<ListMenuItemData<string>[]>(() => {
        // Find all languages which support auto-translations in selected languages.
        // Filter languages that already have a translation
        const autoTranslateLanguagesFiltered = autoTranslateLanguages.filter(l => languages?.indexOf(l) !== -1);
        // Convert to list menu item data
        return autoTranslateLanguagesFiltered.map(l => ({ label: AllLanguages[l], value: l }));
    }, [languages]);
    const [translateSourceAnchor, setTranslateSourceAnchor] = useState<HTMLElement | null>(null);
    const openTranslateSource = useCallback((ev: React.MouseEvent<any>, targetLanguage: string) => {
        // Stop propagation so that the list item is not selected
        ev.stopPropagation();
        // If there's only one source language, auto-translate
        if (translateSourceOptions.length === 1) {
            autoTranslate(translateSourceOptions[0].value, targetLanguage);
        }
        // Otherwise, open menu to select source language
        else {
            console.log("openTranslateSource", targetLanguage);
            setTranslateSourceAnchor(ev.currentTarget);
        }
    }, [autoTranslate, translateSourceOptions]);
    const closeTranslateSource = useCallback(() => setTranslateSourceAnchor(null), []);
    const handleTranslateSourceSelect = useCallback((path: string) => {
        console.log("TODO");
    }, []);

    const languageOptions = useMemo<Array<[string, string]>>(() => {
        // Find user languages
        const userLanguages = getUserLanguages(session);
        const selected = languages.map((l) => getLanguageSubtag(l));
        // Sort selected languages. Selected languages which are also user languages are first.
        const sortedSelectedLanguages = selected.sort((a, b) => {
            const aIndex = userLanguages.indexOf(a);
            const bIndex = userLanguages.indexOf(b);
            if (aIndex === -1 && bIndex === -1) {
                return 0;
            } else if (aIndex === -1) {
                return 1;
            } else if (bIndex === -1) {
                return -1;
            } else {
                return aIndex - bIndex;
            }
        }) ?? [];
        // Filter selected languages from user languages
        const userLanguagesFiltered = userLanguages.filter(l => selected.indexOf(l) === -1);
        // Filter selected and user languages from auto-translateLanguages TODO put back when this is implemented
        //const autoTranslateLanguagesFiltered = autoTranslateLanguages.filter(l => selected.indexOf(l) === -1 && userLanguages.indexOf(l) === -1);
        // Filter selected and user and auto-translate languages from all languages (only when editing)
        const allLanguagesFiltered = isEditing ? (Object.keys(AllLanguages)).filter(l => selected.indexOf(l as any) === -1 && userLanguages.indexOf(l) === -1 && autoTranslateLanguages.indexOf(l as any) === -1) : [];
        // Create array with all available languages.
        const displayed = [...sortedSelectedLanguages, ...userLanguagesFiltered, ...allLanguagesFiltered];//, ...autoTranslateLanguagesFiltered, ...allLanguagesFiltered]; TODO
        // Convert to array of [languageCode, languageDisplayName]
        let options: Array<[string, string]> = displayed.map(l => [l, AllLanguages[l]]);
        // Filter options with search string
        if (searchString.length > 0) {
            options = options.filter((o: [string, string]) => o[1].toLowerCase().includes(searchString.toLowerCase()));
        }
        return options;
    }, [isEditing, searchString, session, languages]);

    // Popup for selecting language
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = Boolean(anchorEl);
    const zIndex = useZIndex(open, false, 1000);
    const onOpen = useCallback((event: MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
        // Force parent to save current translation TODO this causes infinite render in multi-step routine. not sure why
        if (currentLanguage) handleCurrent(currentLanguage);
    }, [currentLanguage, handleCurrent]);
    const onClose = useCallback(() => {
        // Chear text field
        setSearchString("");
        setAnchorEl(null);
    }, []);

    const onDelete = useCallback((e: MouseEvent<HTMLButtonElement>, language: string) => {
        e.preventDefault();
        e.stopPropagation();
        if (handleDelete) handleDelete(language);
    }, [handleDelete]);

    return (
        <>
            {/* Select auto-translate source popover */}
            <ListMenu
                id={"auto-translate-from-menu"}
                anchorEl={translateSourceAnchor}
                title='Translate from...'
                data={translateSourceOptions}
                onSelect={handleTranslateSourceSelect}
                onClose={closeTranslateSource}
            />
            {/* Language select popover */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                aria-labelledby={titleId}
                sx={{
                    zIndex,
                    "& .MuiPopover-paper": {
                        background: "transparent",
                        border: "none",
                        paddingBottom: 1,
                    },
                }}
                anchorOrigin={{
                    vertical: "bottom",
                    horizontal: "center",
                }}
                transformOrigin={{
                    vertical: "top",
                    horizontal: "center",
                }}
            >
                {/* Title */}
                <MenuTitle
                    ariaLabel={titleId}
                    title={t("LanguageSelect")}
                    onClose={onClose}
                />
                {/* Search bar and list of languages */}
                <Stack direction="column" spacing={2} sx={{
                    // width: "min(100vw, 400px)",
                    maxHeight: "min(100vh, 600px)",
                    maxWidth: "100%",
                    overflowX: "auto",
                    overflowY: "hidden",
                    background: palette.background.default,
                    borderRadius: "0 4px 4px",
                    padding: "8px",
                }}>
                    <TextInput
                        placeholder={t("LanguageEnter")}
                        autoFocus={true}
                        value={searchString}
                        onChange={updateSearchString}
                        sx={{
                            paddingLeft: 1,
                            paddingRight: 1,
                        }}
                    />
                    <FixedSizeList
                        height={600}
                        width={400}
                        itemSize={46}
                        itemCount={languageOptions.length}
                        overscanCount={5}
                        style={{
                            maxWidth: "100%",
                        }}
                    >
                        {(props) => {
                            const { index, style } = props;
                            const option: [string, string] = languageOptions[index];
                            const isSelected = option[0] === currentLanguage || languages.some((l) => getLanguageSubtag(l) === getLanguageSubtag(option[0]));
                            const isCurrent = option[0] === currentLanguage;
                            // Can delete if selected, editing, and there are more than 1 selected languages
                            const canDelete = isSelected && isEditing && languages.length > 1;
                            // Can auto-translate if the language is not selected, is in auto-translate languages, and one of 
                            // the existing translations is in the auto-translate languages.
                            const canAutoTranslate = !isSelected && translateSourceOptions.length > 0 && autoTranslateLanguages.includes(option[0] as any);
                            return (
                                <ListItem
                                    key={index}
                                    style={style}
                                    disablePadding
                                    button
                                    onClick={() => { handleCurrent(option[0]); onClose(); }}
                                    // Darken/lighten selected language (depending on light/dark mode)
                                    sx={{
                                        background: isCurrent ? palette.secondary.light : palette.background.default,
                                        color: isCurrent ? palette.secondary.contrastText : palette.background.textPrimary,
                                        "&:hover": {
                                            background: isCurrent ? palette.secondary.light : palette.background.default,
                                            filter: "brightness(105%)",
                                        },
                                    }}
                                >
                                    {/* Display check mark if selected */}
                                    {isSelected && (
                                        <CompleteIcon fill={(isCurrent) ? palette.secondary.contrastText : palette.background.textPrimary} />
                                    )}
                                    <Typography variant="body2" style={{
                                        display: "block",
                                        marginRight: "auto",
                                        marginLeft: isSelected ? "8px" : "0",
                                    }}>{option[1]}</Typography>
                                    {/* Delete icon */}
                                    {canDelete && (
                                        <Tooltip title="Delete translation for this language">
                                            <IconButton
                                                size="small"
                                                onClick={(e) => onDelete(e, option[0])}
                                            >
                                                <DeleteIcon fill={isCurrent ? palette.secondary.contrastText : palette.background.textPrimary} />
                                            </IconButton>
                                        </Tooltip>
                                    )}
                                    {/* Auto-translate icon TODO add back later*/}
                                    {/* {canAutoTranslate && (
                                        <Tooltip title="Auto-translate from an existing translation">
                                            <IconButton
                                                size="small"
                                                onClick={(e) => { openTranslateSource(e, option[0]) }}
                                            >
                                                <TranslateIcon fill={isCurrent ? palette.secondary.contrastText : palette.background.textPrimary} />
                                            </IconButton>
                                        </Tooltip>
                                    )} */}
                                </ListItem>
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title={AllLanguages[currentLanguage] ?? ""} placement="top">
                <Stack direction="row" spacing={0} onClick={onOpen} sx={{
                    display: "flex",
                    justifyContent: "center",
                    alignItems: "center",
                    borderRadius: "50px",
                    cursor: "pointer",
                    background: "#4e7d31",
                    boxShadow: 2,
                    "&:hover": {
                        filter: "brightness(120%)",
                    },
                    transition: "all 0.2s ease-in-out",
                    width: "fit-content",
                    ...(sxs?.root ?? {}),
                }}>
                    <IconButton size="large" sx={{ padding: "4px" }}>
                        <LanguageIcon fill={"white"} />
                    </IconButton>
                    {/* Only show language code when editing to save space. You'll know what language you're reading just by reading */}
                    {isEditing && <Typography variant="body2" sx={{ color: "white", marginRight: "8px" }}>
                        {currentLanguage?.toLocaleUpperCase()}
                    </Typography>}
                    {/* Drop down or drop up icon */}
                    <IconButton size="large" aria-label="language-select" sx={{ padding: "4px", marginLeft: "-8px" }}>
                        {open ? <ArrowDropUpIcon fill={"white"} /> : <ArrowDropDownIcon fill={"white"} />}
                    </IconButton>
                </Stack>
            </Tooltip>
        </>
    );
}
