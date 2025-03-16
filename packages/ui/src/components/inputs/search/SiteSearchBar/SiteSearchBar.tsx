import { AutocompleteOption, BookmarkFor } from "@local/shared";
import { Autocomplete, AutocompleteHighlightChangeReason, AutocompleteRenderInputParams, CircularProgress, IconButton, Input, ListItemIcon, ListItemText, MenuItem, Paper, Popper, PopperProps, Tooltip, styled, useTheme } from "@mui/material";
import { BookmarkButton } from "components/buttons/BookmarkButton/BookmarkButton.js";
import { MicrophoneButton } from "components/buttons/MicrophoneButton/MicrophoneButton.js";
import { useDebounce } from "hooks/useDebounce.js";
import { ActionIcon, ApiIcon, BookmarkFilledIcon, DeleteIcon, HelpIcon, HistoryIcon, NoteIcon, PlayIcon, ProjectIcon, RoutineIcon, SearchIcon, ShortcutIcon, StandardIcon, TeamIcon, TerminalIcon, UserIcon, VisibleIcon } from "icons/common.js";
import { ChangeEvent, FormEvent, HTMLAttributes, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SvgComponent } from "types";
import { getCurrentUser } from "utils/authentication/session.js";
import { randomString } from "utils/codes.js";
import { DUMMY_LIST_LENGTH } from "utils/consts.js";
import { getLocalStorageKeys } from "utils/localStorage.js";
import { performAction } from "utils/navigation/quickActions.js";
import { SessionContext } from "../../../../contexts.js";
import { SiteSearchBarProps } from "../types.js";

type OptionHistory = { timestamp: number, option: AutocompleteOption };

const DEFAULT_DEBOUNCE_MS = 200;
const MAX_HISTORY_LENGTH = 500;

function getAutocompleteOptionLabel(option: AutocompleteOption) { return option.label ?? ""; }

/**
 * Gets search history from local storage, unique by search bar and account
 * @param searchBarId The search bar ID
 * @param userId The user ID
 */
function getSearchHistory(searchBarId: string, userId: string): { [label: string]: OptionHistory } {
    const existingHistoryString: string = localStorage.getItem(`search-history-${searchBarId}-${userId}`) ?? "{}";
    let existingHistory: { [label: string]: OptionHistory } = {};
    // Try to parse existing history
    try {
        const parsedHistory: any = JSON.parse(existingHistoryString);
        // If it's not an object, set it to an empty object
        if (typeof existingHistory !== "object") existingHistory = {};
        else existingHistory = parsedHistory;
    } catch (e) {
        existingHistory = {};
    }
    return existingHistory;
}

/**
 * For a list of options, checks if they are stored in local storage and need their bookmarks/isBookmarked updated. If so, updates them.
 * @param searchBarId The search bar ID
 * @param userId The user ID
 * @param options The options to check
 */
function updateHistoryItems(searchBarId: string, userId: string, options: AutocompleteOption[] | undefined) {
    if (!options) return;
    // Find all search history objects in localStorage
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: "search-history-",
        suffix: userId,
    });
    // For each search history object, perform update
    searchHistoryKeys.forEach(key => {
        // Get history data
        let existingHistory: { [label: string]: OptionHistory } = {};
        // Try to parse existing history
        try {
            const parsedHistory: any = JSON.parse(localStorage.getItem(key) ?? "{}");
            // If it's not an object, set it to an empty object
            if (typeof existingHistory !== "object") existingHistory = {};
            else existingHistory = parsedHistory;
        } catch (e) {
            existingHistory = {};
        }
        // Find options that are in history, and update if bookmarks or isBookmarked are different
        const updatedHistory = options.map(option => {
            // bookmarks and isBookmarked are not in shortcuts or actions
            if (option.__typename === "Shortcut" || option.__typename === "Action") return null;
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
        }).filter(Boolean) as OptionHistory[];
        // Update changed options
        if (updatedHistory.length > 0) {
            for (const historyItem of updatedHistory) {
                existingHistory[historyItem.option.label] = historyItem;
            }
            localStorage.setItem(key, JSON.stringify(existingHistory));
        }
    });
}

/**
 * Stop default onSubmit, since when the search bar is a form this causes the page to reload
 */
function stopDefaultSubmit(event: FormEvent<HTMLDivElement>) {
    event.preventDefault();
    event.stopPropagation();
}

const IconMap = {
    Action: ActionIcon,
    Api: ApiIcon,
    Bookmark: BookmarkFilledIcon,
    Code: TerminalIcon,
    Note: NoteIcon,
    Project: ProjectIcon,
    Question: HelpIcon,
    Routine: RoutineIcon,
    Run: PlayIcon,
    Shortcut: ShortcutIcon,
    Standard: StandardIcon,
    Team: TeamIcon,
    User: UserIcon,
    View: VisibleIcon,
};

/**
 * Maps object types to icons
 */
function typeToIcon(type: string, fill: string): JSX.Element | null {
    const Icon: SvgComponent | undefined = IconMap[type];
    return Icon ? <Icon fill={fill} /> : null;
}

interface PopperComponentProps extends PopperProps {
    clientWidth: number | null | undefined
    parentWidth: number | null | undefined
}

const PopperComponent = styled(Popper, {
    shouldForwardProp: (prop) => prop !== "clientWidth" && prop !== "parentWidth",
})<PopperComponentProps>(({ clientWidth, parentWidth }) => ({
    left: "-12px!important",
    minWidth: parentWidth ?? clientWidth ?? "fit-content",
    maxWidth: "100%",
}));

function FullWidthPopper(props: PopperProps) {
    const parentWidth = props.anchorEl && (props.anchorEl as HTMLElement).parentElement ? (props.anchorEl as HTMLElement).parentElement?.clientWidth : null;

    return <PopperComponent
        {...props}
        clientWidth={(props.anchorEl as HTMLElement)?.clientWidth}
        parentWidth={parentWidth}
        placement="bottom-start"
    />;
}

function NoPopper() {
    return null;
}

const loadingIndicatorStyle = { marginRight: 1 } as const;
const searchIconStyle = { width: "48px", height: "48px" } as const;
const optionIconStyle = { display: { xs: "none", sm: "block" } } as const;
const bookmarkStyle = { root: { marginRight: 1 } } as const;
const listItemTextStyle = {
    "& .MuiTypography-root": {
        textOverflow: "ellipsis",
        whiteSpace: "nowrap",
        overflow: "hidden",
    },
} as const;
const inputStyle = {
    ml: 1,
    flex: 1,
    // Drop down should be as large as the full width of the screen
    "& .MuiAutocomplete-popper": {
        width: "100vw!important",
        left: "0",
        right: "0",
        // The drop down should be below the search bar
        "& .MuiPaper-root": {
            marginTop: "0",
        },
    },
} as const;

/**
 * A customized search bar for searching user-generated content, quick actions, and shortcuts. 
 * Supports search history and starring items.
 */
export function SiteSearchBar({
    enterKeyHint,
    id = "search-bar",
    isNested,
    placeholder,
    options,
    value,
    onChange,
    onInputChange,
    debounce = DEFAULT_DEBOUNCE_MS,
    loading = false,
    sxs,
    ...props
}: SiteSearchBarProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);
    const { t } = useTranslation();

    // Input internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Highlighted option (if navigated with keyboard)
    const [highlightedOption, setHighlightedOption] = useState<AutocompleteOption | null>(null);

    const [onChangeDebounced] = useDebounce(onChange, debounce);
    useEffect(() => setInternalValue(value), [value]);
    const handleChange = useCallback((value: string) => {
        // Update state
        setInternalValue(value);
        // Remove the highlight
        setHighlightedOption(null);
        // Debounce onChange
        onChangeDebounced(value);
    }, [onChangeDebounced]);
    const handleChangeEvent = useCallback((event: ChangeEvent<any>) => {
        // Get the new input string
        const { value } = event.target;
        // Call handleChange
        handleChange(value);
    }, [handleChange]);

    const [optionsWithHistory, setOptionsWithHistory] = useState<readonly AutocompleteOption[]>(options ?? []);
    useEffect(function getOptionsWithHistoryEffect() {
        if (!options) {
            return;
        }
        // Grab history from local storage
        const searchHistory = getSearchHistory(id, userId ?? "");
        // Filter out history keys that don't contain internal value
        let filteredHistory = Object.entries(searchHistory).filter(([key]) => key.toLowerCase().includes(internalValue.toLowerCase()));
        // Order remaining history keys by most recent. Value is stored as { timestamp: string, value: AutocompleteOption }
        filteredHistory = filteredHistory.sort((a, b) => { return b[1].timestamp - a[1].timestamp; });
        // Convert history keys to options
        let historyOptions: AutocompleteOption[] = filteredHistory.map(([, value]) => ({ ...value.option, isFromHistory: true }));
        // Limit size
        historyOptions = historyOptions.slice(0, DUMMY_LIST_LENGTH);
        // Filter out options that are in history (use id to check)
        const filteredOptions = options.filter(option => !historyOptions.some(historyOption => historyOption.id === option.id));
        // If any options have a bookmarks/isBookmarked values which differs from history, update history
        updateHistoryItems(id, userId ?? "", filteredOptions);
        // Combine history and options
        let combinedOptions = [...historyOptions, ...filteredOptions];
        // In case there are bad options, filter out anything with: an empty or whitespace-only label, or no id
        combinedOptions = combinedOptions.filter(option => option.label && option.label.trim() !== "" && option.id);
        // Create unique keys for each option
        const keySet = new Set<string>();
        combinedOptions = combinedOptions.map(option => {
            let key = `${option.id}-${option.label}-${option.__typename}-${option.isFromHistory ? "history" : "regular"}`;
            while (keySet.has(key)) {
                console.warn("Duplicate key found in search bar", key, option);
                key += `-${randomString()}`;
            }
            keySet.add(key);
            return { ...option, key };
        });
        // Update state
        setOptionsWithHistory(combinedOptions);
    }, [options, internalValue, id, userId]);

    const removeFromHistory = useCallback((option: AutocompleteOption) => {
        // Get existing history
        const existingHistory = getSearchHistory(id, userId ?? "");
        // Remove the option from history
        delete existingHistory[option.label];
        // Save the new history
        localStorage.setItem(`search-history-${id}-${userId ?? ""}`, JSON.stringify(existingHistory));
        // Update options with history
        setOptionsWithHistory(optionsWithHistory.filter(o => o.key !== option.key));
    }, [id, optionsWithHistory, userId]);

    /**
     * If no options but loading, display a loading indicator
     */
    const noOptionsText = useMemo(() => {
        if (loading || (internalValue !== value)) {
            return (
                <>
                    <CircularProgress
                        color="secondary"
                        size={20}
                        sx={loadingIndicatorStyle}
                    />
                    {t("Loading")}
                </>
            );
        }
        return t("NoResults", { ns: "error" });
    }, [loading, internalValue, value, t]);

    const onHighlightChange = useCallback((event: React.SyntheticEvent<Element, Event>, option: AutocompleteOption | null, reason: AutocompleteHighlightChangeReason) => {
        if (option && option.label && reason === "keyboard") {
            setHighlightedOption(option);
        }
    }, []);

    const handleSelect = useCallback((option: AutocompleteOption) => {
        // Add to search history
        const existingHistory = getSearchHistory(id, userId ?? "");
        // If history has more than 500 entries, remove the oldest
        if (Object.keys(existingHistory).length > MAX_HISTORY_LENGTH) {
            const oldestKey = Object.keys(existingHistory).sort((a, b) => existingHistory[a].timestamp - existingHistory[b].timestamp)[0];
            delete existingHistory[oldestKey];
        }
        // Add new entry
        existingHistory[option.label] = {
            timestamp: Date.now(),
            option: ({ ...option, isFromHistory: true }),
        };
        // Save to local storage
        localStorage.setItem(`search-history-${id}-${userId ?? ""}`, JSON.stringify(existingHistory));
        // If action, perform action and clear input
        if (option.__typename === "Action") {
            performAction(option, session);
            // Update state
            setInternalValue("");
            // Remove the highlight
            setHighlightedOption(null);
            // Debounce onChange
            onChangeDebounced("");
        }
        // Trigger onInputChange
        onInputChange(option);
    }, [id, onChangeDebounced, onInputChange, session, userId]);

    const onSubmit = useCallback(function onSubmitCallback() {
        // If there is a highlighted option, use that
        if (highlightedOption) {
            handleSelect(highlightedOption);
        }
        // Otherwise, don't submit
    }, [highlightedOption, handleSelect]);

    const optionColor = useCallback((isFromHistory: boolean | undefined, isSecondary: boolean): string => {
        if (isFromHistory) return palette.mode === "dark" ? "hotpink" : "purple";
        return isSecondary ? palette.background.textSecondary : palette.background.textPrimary;
    }, [palette]);

    const handleBookmark = useCallback((option: AutocompleteOption, isBookmarked: boolean, event: any) => {
        if (option.__typename === "Shortcut" || option.__typename === "Action") return;
        // Stop propagation so that the option isn't selected
        event.stopPropagation();
        // Update the option's isBookmarked and bookmarks value
        const updatedOption = { ...option, isBookmarked, bookmarks: isBookmarked ? (option.bookmarks ?? 0) + 1 : (option.bookmarks ?? 1) - 1 };
        // Update history and state
        updateHistoryItems(id, userId ?? "", [updatedOption]);
        setOptionsWithHistory(optionsWithHistory.map(o => o.key && option.key && o.key === option.key ? updatedOption : o));
    }, [id, optionsWithHistory, userId]);

    // On key down, fill search input with highlighted option if right arrow is pressed
    const onKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (event.key === "ArrowRight" && highlightedOption) {
            // Update state
            setInternalValue(highlightedOption.label);
            // Debounce onChange
            onChangeDebounced("");
        }
    }, [highlightedOption, onChangeDebounced]);

    const autocompleteSx = useMemo(function autocompleteSxMemo() {
        return {
            ...(sxs?.root ?? {}),
            "& .MuiAutocomplete-inputRoot": {
                paddingRight: "0 !important",
            },
            // Make sure menu is at least as wide as the search bar
            "& .MuiAutocomplete-popper": {
                minWidth: "100vw",
            },
        };
    }, [sxs?.root]);

    const paperSx = useMemo(function paperSxMemo() {
        return {
            ...(sxs?.paper ?? {}),
            p: "2px 4px",
            display: "flex",
            alignItems: "center",
            borderRadius: "10px",
            height: "48px",
        };
    }, [sxs?.paper]);

    const renderOption = useCallback(function renderOptionCallback(
        props: HTMLAttributes<HTMLLIElement>,
        option: AutocompleteOption,
    ) {
        function onOptionClick() {
            const label = option.label ?? "";
            setInternalValue(label);
            onChangeDebounced(label);
            handleSelect(option);
        }
        function onBookmarkChange(isBookmarked: boolean, event: any) {
            handleBookmark(option, isBookmarked, event);
        }
        function onDeleteClick(event: React.MouseEvent<HTMLButtonElement, MouseEvent>) {
            event.stopPropagation();
            removeFromHistory(option);
        }
        const optionStyle = { color: optionColor(option.isFromHistory, false) } as const;

        return (
            <MenuItem
                {...props}
                key={option.key}
                onClick={onOptionClick}
                sx={optionStyle}
            >
                {/* Show history icon if from history */}
                {option.isFromHistory && (
                    <ListItemIcon sx={optionIconStyle}>
                        <HistoryIcon fill={optionColor(true, false)} />
                    </ListItemIcon>
                )}
                {/* Object icon */}
                <ListItemIcon>
                    {typeToIcon(option.__typename, optionColor(option.isFromHistory, true))}
                </ListItemIcon>
                {/* Object title */}
                <ListItemText sx={listItemTextStyle}>
                    {option.label}
                </ListItemText>
                {/* Bookmark button */}
                {option.__typename !== "Shortcut" && option.__typename !== "Action" && option.bookmarks !== undefined && <BookmarkButton
                    isBookmarked={option.isBookmarked}
                    objectId={option.id}
                    onChange={onBookmarkChange}
                    showBookmarks={true}
                    bookmarkFor={option.__typename as unknown as BookmarkFor}
                    bookmarks={option.bookmarks}
                    sxs={bookmarkStyle}
                />}
                {/* If history, show delete icon */}
                {option.isFromHistory && <Tooltip placement='right' title='Remove'>
                    <IconButton size="small" onClick={onDeleteClick}>
                        <DeleteIcon fill={optionColor(true, true)} />
                    </IconButton>
                </Tooltip>}
            </MenuItem>
        );
    }, [handleBookmark, handleSelect, onChangeDebounced, optionColor, removeFromHistory]);

    const renderInput = useCallback(function renderInputCallback(params: AutocompleteRenderInputParams) {
        function getInputProps() {
            return {
                ...params.inputProps,
                enterKeyHint: enterKeyHint || "search",
            };
        }
        return (
            <Paper
                action="#" // Needed for iOS to accept enterKeyHint: https://stackoverflow.com/a/39485162/406725
                component={isNested ? "div" : "form"}
                sx={paperSx}
            >
                <Input
                    id={params.id}
                    disabled={params.disabled}
                    disableUnderline={true}
                    fullWidth={true}
                    value={internalValue}
                    onChange={handleChangeEvent}
                    placeholder={t(`${placeholder || "Search"}`) + "..."}
                    autoFocus={props.autoFocus ?? false}
                    inputProps={getInputProps()}
                    // inputProps={params.inputProps}
                    ref={params.InputProps.ref}
                    size={params.size}
                    sx={inputStyle}
                />
                <MicrophoneButton onTranscriptChange={handleChange} />
                <IconButton sx={searchIconStyle} aria-label="main-search-icon">
                    <SearchIcon fill={palette.background.textSecondary} />
                </IconButton>
            </Paper>
        );
    }, [enterKeyHint, handleChange, handleChangeEvent, internalValue, isNested, palette.background.textSecondary, paperSx, placeholder, props.autoFocus, t]);

    // If no options were passed (not even an empty array), then we probably don't want to see it
    const PopperComponent = useMemo(() => options !== undefined ? FullWidthPopper : NoPopper, [options]);

    return (
        <Autocomplete
            disablePortal
            id={id}
            options={optionsWithHistory}
            onHighlightChange={onHighlightChange}
            onKeyDown={onKeyDown}
            inputValue={internalValue}
            getOptionLabel={getAutocompleteOptionLabel}
            onSubmit={stopDefaultSubmit}
            // The real onSubmit, since onSubmit is only triggered after 2 presses of the enter key (don't know why)
            onChange={onSubmit}
            PopperComponent={PopperComponent}
            renderOption={renderOption}
            renderInput={renderInput}
            noOptionsText={noOptionsText}
            sx={autocompleteSx}
        />
    );
}
