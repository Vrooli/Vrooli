import IconButton from "@mui/material/IconButton";
import ListItem from "@mui/material/ListItem";
import Popover from "@mui/material/Popover";
import Stack from "@mui/material/Stack";
import Tooltip from "@mui/material/Tooltip";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material";
import { type MouseEvent, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { FixedSizeList } from "react-window";
import { SessionContext } from "../../../contexts/session.js";
import { IconCommon } from "../../../icons/Icons.js";
import { Z_INDEX } from "../../../utils/consts.js";
import { AllLanguages, getLanguageSubtag, getUserLanguages } from "../../../utils/display/translationTools.js";
import { TextInput } from "../../inputs/TextInput/TextInput.js";
import { MenuTitle } from "../MenuTitle/MenuTitle.js";
import { type SelectLanguageMenuProps } from "../types.js";

const titleId = "select-language-dialog-title";
const anchorOrigin = {
    vertical: "bottom",
    horizontal: "center",
} as const;
const transformOrigin = {
    vertical: "top",
    horizontal: "center",
} as const;

// Pre-defined styles
const popoverSx = {
    zIndex: Z_INDEX.Popup,
    "& .MuiPopover-paper": {
        background: "transparent",
        border: "none",
        paddingBottom: 1,
        zIndex: Z_INDEX.Popup,
    },
};

const fixedSizeListStyle = {
    maxWidth: "100%",
};

// ListItem component to avoid JSX object creation in render
function LanguageListItem({
    index,
    style,
    option,
    isSelected,
    isCurrent,
    canDelete,
    palette,
    onDelete,
    handleCurrent,
    onClose,
}: {
    index: number;
    style: React.CSSProperties;
    option: [string, string];
    isSelected: boolean;
    isCurrent: boolean;
    canDelete: boolean;
    palette: any;
    onDelete: (e: MouseEvent<HTMLButtonElement>, language: string) => void;
    handleCurrent: (language: string) => void;
    onClose: () => void;
}) {
    const handleItemClick = useCallback(() => {
        handleCurrent(option[0]);
        onClose();
    }, [handleCurrent, onClose, option]);

    const handleDeleteClick = useCallback((e: MouseEvent<HTMLButtonElement>) => {
        onDelete(e, option[0]);
    }, [onDelete, option]);

    const listItemSx = useMemo(() => ({
        background: isCurrent ? palette.secondary.light : palette.background.default,
        color: isCurrent ? palette.secondary.contrastText : palette.background.textPrimary,
        "&:hover": {
            background: isCurrent ? palette.secondary.light : palette.background.default,
            filter: "brightness(105%)",
        },
    }), [isCurrent, palette]);

    const typographyStyle = useMemo(() => ({
        display: "block",
        marginRight: "auto",
        marginLeft: isSelected ? "8px" : "0",
    }), [isSelected]);

    return (
        <ListItem
            key={index}
            style={style}
            disablePadding
            button
            onClick={handleItemClick}
            sx={listItemSx}
        >
            {/* Display check mark if selected */}
            {isSelected && (
                <IconCommon
                    decorative
                    fill={(isCurrent) ? palette.secondary.contrastText : palette.background.textPrimary}
                    name="Complete"
                />
            )}
            <Typography variant="body2" style={typographyStyle}>{option[1]}</Typography>
            {/* Delete icon */}
            {canDelete && (
                <Tooltip title="Delete translation for this language">
                    <IconButton
                        size="small"
                        onClick={handleDeleteClick}
                    >
                        <IconCommon
                            decorative
                            fill={isCurrent ? palette.secondary.contrastText : palette.background.textPrimary}
                            name="Delete"
                        />
                    </IconButton>
                </Tooltip>
            )}
        </ListItem>
    );
}

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

    // Memoized styles that depend on theme or props
    const stackSx = useMemo(() => ({
        maxHeight: "min(100vh, 600px)",
        maxWidth: "100%",
        overflowX: "auto",
        overflowY: "hidden",
        background: palette.background.default,
        borderRadius: "4px",
        padding: "8px",
    }), [palette.background.default]);

    const textInputSx = useMemo(() => ({
        paddingLeft: 1,
        paddingRight: 1,
    }), []);

    const languageIconButtonSx = useMemo(() => ({
        padding: "4px",
        color: palette.background.textSecondary,
    }), [palette.background.textSecondary]);

    const dropIconButtonSx = useMemo(() => ({
        padding: "4px",
        marginLeft: "-8px",
        color: palette.background.textSecondary,
    }), [palette.background.textSecondary]);

    const languageTypographySx = useMemo(() => ({
        color: palette.background.textSecondary,
        marginRight: "8px",
        fontSize: "0.85rem",
    }), [palette.background.textSecondary]);

    const rootStackSx = useMemo(() => ({
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        borderRadius: "50px",
        cursor: "pointer",
        background: "transparent",
        border: `1px solid ${palette.background.textSecondary}`,
        transition: "all 0.2s ease-in-out",
        width: "fit-content",
        height: "32px",
        ...(sxs?.root ?? {}),
    }), [palette.background.textSecondary, sxs?.root]);

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
        const allLanguagesFiltered = isEditing ? (Object.keys(AllLanguages)).filter(l => selected.indexOf(l as any) === -1 && userLanguages.indexOf(l) === -1) : [];
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
    const [anchorEl, setAnchorEl] = useState<Element | null>(null);
    const open = Boolean(anchorEl);
    const onOpen = useCallback((event: MouseEvent<Element>) => {
        setAnchorEl(event.currentTarget);
        // Force parent to save current translation
        if (currentLanguage) handleCurrent(currentLanguage);
    }, [currentLanguage, handleCurrent]);
    const onClose = useCallback(() => {
        // Clear text field
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
            {/* Language select popover with disablePortal={false} to ensure it's rendered in a portal */}
            <Popover
                open={open}
                anchorEl={anchorEl}
                onClose={onClose}
                aria-labelledby={titleId}
                sx={popoverSx}
                anchorOrigin={anchorOrigin}
                transformOrigin={transformOrigin}
                disablePortal={false}
                container={document.body}
            >
                {/* Title */}
                <MenuTitle
                    ariaLabel={titleId}
                    title={t("LanguageSelect")}
                    onClose={onClose}
                />
                {/* Search bar and list of languages */}
                <Stack direction="column" spacing={2} sx={stackSx}>
                    <TextInput
                        placeholder={t("LanguageEnter")}
                        // Removing autoFocus to fix accessibility issue
                        value={searchString}
                        onChange={updateSearchString}
                        sx={textInputSx}
                    />
                    {/* TODO Remove this once react-window is updated */}
                    {/* @ts-expect-error Incompatible JSX type definitions */}
                    <FixedSizeList
                        height={600}
                        width={400}
                        itemSize={46}
                        itemCount={languageOptions.length}
                        overscanCount={5}
                        style={fixedSizeListStyle}
                    >
                        {(props) => {
                            const { index, style } = props;
                            const option: [string, string] = languageOptions[index];
                            const isSelected = option[0] === currentLanguage || languages.some((l) => getLanguageSubtag(l) === getLanguageSubtag(option[0]));
                            const isCurrent = option[0] === currentLanguage;
                            // Can delete if selected, editing, and there are more than 1 selected languages
                            const canDelete = isSelected && isEditing && languages.length > 1;

                            return (
                                <LanguageListItem
                                    index={index}
                                    style={style}
                                    option={option}
                                    isSelected={isSelected}
                                    isCurrent={isCurrent}
                                    canDelete={canDelete}
                                    palette={palette}
                                    onDelete={onDelete}
                                    handleCurrent={handleCurrent}
                                    onClose={onClose}
                                />
                            );
                        }}
                    </FixedSizeList>
                </Stack>
            </Popover>
            {/* Selected language label */}
            <Tooltip title={AllLanguages[currentLanguage] ?? ""} placement="top">
                <Stack direction="row" spacing={0} onClick={onOpen} sx={rootStackSx}>
                    <IconButton size="small" sx={languageIconButtonSx}>
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name="Language"
                        />
                    </IconButton>
                    {/* Only show language code when editing to save space. You'll know what language you're reading just by reading */}
                    {isEditing && <Typography variant="body2" sx={languageTypographySx}>
                        {currentLanguage?.toLocaleUpperCase()}
                    </Typography>}
                    {/* Drop down or drop up icon */}
                    <IconButton
                        aria-label="language-select"
                        size="small"
                        sx={dropIconButtonSx}
                    >
                        <IconCommon
                            decorative
                            fill="background.textSecondary"
                            name={open ? "ArrowDropUp" : "ArrowDropDown"}
                        />
                    </IconButton>
                </Stack>
            </Tooltip>
        </>
    );
}
