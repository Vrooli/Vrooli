import { LINKS } from "@local/shared";
import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, IconButton, Input, ListItemText, MenuItem, Paper, Popper, PopperProps, useTheme } from "@mui/material";
import { SessionContext } from "contexts/SessionContext";
import { SearchIcon } from "icons";
import { ChangeEvent, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { findSearchResults, PreSearchItem, SearchItem, shapeSearchText, translateSearchItems } from "utils/search/siteToSearch";
import { SettingsSearchBarProps } from "../types";

const searchItems: PreSearchItem[] = [
    {
        label: "Profile",
        keywords: ["Bio", "Handle", "Name"],
        value: LINKS.SettingsProfile,
    },
    {
        label: "Privacy",
        keywords: ["History", "Private"],
        value: LINKS.SettingsPrivacy,
    },
    {
        label: "Data",
        keywords: [],
        value: LINKS.SettingsData,
    },
    {
        label: "Authentication",
        keywords: [{ key: "Wallet", count: 1 }, { key: "Wallet", count: 2 }, { key: "Email", count: 1 }, { key: "Email", count: 2 }, "LogOut", "Security"],
        value: LINKS.SettingsAuthentication,
    },
    {
        label: "Payment",
        labelArgs: { count: 2 },
        keywords: [],
        value: LINKS.SettingsPayments,
    },
    {
        label: "Api",
        keywords: [{ key: "Api", count: 2 }],
        value: LINKS.SettingsApi,
    },
    {
        label: "Display",
        keywords: ["Theme", "Light", "Dark", "Interests", "Hidden", { key: "Tag", count: 1 }, { key: "Tag", count: 2 }, "History"],
        value: LINKS.SettingsDisplay,
    },
    {
        label: "Notification",
        labelArgs: { count: 2 },
        keywords: [{ key: "Alert", count: 1 }, { key: "Alert", count: 2 }, { key: "PushNotification", count: 1 }, { key: "PushNotification", count: 2 }],
        value: LINKS.SettingsNotifications,
    },
    {
        label: "FocusMode",
        labelArgs: { count: 2 },
        keywords: [{ key: "Schedule", count: 1 }, { key: "Schedule", count: 2 }, { key: "FocusMode", count: 1 }],
        value: LINKS.SettingsFocusModes,
    },
];

const FullWidthPopper = function (props: PopperProps) {
    const parentWidth = props.anchorEl && (props.anchorEl as HTMLElement).parentElement ? (props.anchorEl as HTMLElement).parentElement?.clientWidth : null;
    return <Popper {...props} sx={{
        left: "-12px!important",
        minWidth: parentWidth ?? (props.anchorEl as HTMLElement)?.clientWidth ?? "fit-content",
        maxWidth: "100%",
    }} placement="bottom-start" /> as JSX.Element;
};

export const SettingsSearchBar = ({
    debounce = 200,
    id = "search-bar",
    onChange,
    onInputChange,
    placeholder,
    value,
    ...props
}: SettingsSearchBarProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const session = useContext(SessionContext);

    const options = useMemo(() => translateSearchItems(searchItems, session), [session]);

    // Input internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Highlighted option (if navigated with keyboard)
    const [highlightedOption, setHighlightedOption] = useState<SearchItem | null>(null);

    const handleChange = useCallback((event: ChangeEvent<any>) => {
        // Get the new input string
        const { value } = event.target;
        // Update state
        setInternalValue(value);
        // Remove the highlight
        setHighlightedOption(null);
        // Call onChange
        onChange(value);
    }, [onChange]);

    const onHighlightChange = useCallback((event: React.SyntheticEvent<Element, Event>, option: SearchItem | null, reason: AutocompleteHighlightChangeReason) => {
        if (option && option.label && reason === "keyboard") {
            setHighlightedOption(option);
        }
    }, []);

    const handleSelect = useCallback((option: SearchItem) => {
        // Trigger onInputChange
        onInputChange(option);
    }, [onInputChange]);

    const onSubmit = useCallback((event: React.SyntheticEvent<Element, Event>, value: SearchItem | null, reason: AutocompleteChangeReason, details?: AutocompleteChangeDetails<any> | undefined) => {
        // If there is a highlighted option, use that
        if (highlightedOption) {
            handleSelect(highlightedOption);
        }
        // Otherwise, don't submit
    }, [highlightedOption, handleSelect]);

    // On key down, fill search input with highlighted option if right arrow is pressed
    const onKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (event.key === "ArrowRight" && highlightedOption) {
            // Update state
            setInternalValue(highlightedOption.label);
            // Debounce onChange
            onChange("");
        }
    }, [highlightedOption, onChange]);

    return (
        <Autocomplete
            disablePortal
            id={id}
            filterOptions={findSearchResults}
            options={options}
            onHighlightChange={onHighlightChange}
            onKeyDown={onKeyDown}
            inputValue={internalValue}
            getOptionLabel={(option: SearchItem) => option.label ?? ""}
            // Stop default onSubmit, since this reloads the page for some reason
            onSubmit={(event: any) => {
                event.preventDefault();
                event.stopPropagation();
            }}
            // The real onSubmit, since onSubmit is only triggered after 2 presses of the enter key (don't know why)
            onChange={onSubmit}
            PopperComponent={FullWidthPopper}
            renderOption={(props, option) => {
                // Check if any of the keywords matches the search string
                const shapedSearchString = shapeSearchText(internalValue);
                let displayedKeyword = "";
                if (option.keywords && shapedSearchString.length > 0) {
                    for (let i = 0; i < option.keywords?.length; i++) {
                        const keyword = option.keywords[i];
                        const unshapedKeyword = option.unshapedKeywords![i];
                        // Skip label
                        if (unshapedKeyword === option.label || keyword.includes(shapedSearchString)) continue;
                        // If keyword includes search string, display it
                        if (keyword.includes(shapedSearchString)) {
                            displayedKeyword = unshapedKeyword;
                            break;
                        }
                    }
                }
                return (
                    <MenuItem
                        {...props}
                        key={option.value}
                        onClick={() => {
                            const label = option.label ?? "";
                            setInternalValue(label);
                            onChange(label);
                            handleSelect(option);
                        }}
                    >
                        {/* Object title */}
                        <ListItemText sx={{
                            "& .MuiTypography-root": {
                                textOverflow: "ellipsis",
                                whiteSpace: "nowrap",
                                overflow: "hidden",
                            },
                        }}>
                            {option.label}
                        </ListItemText>
                        {/* If search string matches a keyword, display the first match */}
                        {displayedKeyword && (
                            <ListItemText sx={{
                                "& .MuiTypography-root": {
                                    textOverflow: "ellipsis",
                                    whiteSpace: "nowrap",
                                    overflow: "hidden",
                                },
                            }}>
                                {displayedKeyword}
                            </ListItemText>
                        )}
                    </MenuItem>
                );
            }}
            renderInput={(params) => (
                <Paper
                    component="form"
                    sx={{
                        p: "2px 4px",
                        display: "flex",
                        alignItems: "center",
                        borderRadius: "10px",
                    }}
                >
                    <Input
                        id={params.id}
                        disabled={params.disabled}
                        disableUnderline={true}
                        fullWidth={params.fullWidth}
                        value={internalValue}
                        onChange={handleChange}
                        placeholder={t(`${placeholder ?? "Search"}`) + "..."}
                        autoFocus={props.autoFocus ?? false}
                        // {...params.InputLabelProps}
                        inputProps={params.inputProps}
                        ref={params.InputProps.ref}
                        size={params.size}
                        sx={{
                            width: "100vw",
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
                        }}
                    />
                    <IconButton sx={{
                        width: "48px",
                        height: "48px",
                    }} aria-label="main-search-icon">
                        <SearchIcon fill={palette.background.textSecondary} />
                    </IconButton>
                </Paper>
            )}
            noOptionsText={t("NoResults", { ns: "error" })}
            sx={{
                "& .MuiAutocomplete-inputRoot": {
                    paddingRight: "0 !important",
                },
                // Make sure menu is at least as wide as the search bar
                "& .MuiAutocomplete-popper": {
                    minWidth: "100vw",
                },
            }}
        />
    );
};
