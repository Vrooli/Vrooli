import { LINKS } from "@local/shared";
import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, IconButton, Input, ListItemText, MenuItem, Paper, Popper, PopperProps, useTheme } from "@mui/material";
import { SessionContext } from "contexts";
import { ApiIcon, HistoryIcon, LightModeIcon, LockIcon, NotificationsCustomizedIcon, ObjectIcon, ProfileIcon, SearchIcon, VisibleIcon, WalletIcon } from "icons";
import { ChangeEvent, FormEvent, useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PreSearchItem, SearchItem, findSearchResults, shapeSearchText, translateSearchItems } from "utils/search/siteToSearch";
import { SettingsSearchBarProps } from "../types";

const DEFAULT_DEBOUNCE_MS = 200;

const searchItems: PreSearchItem[] = [
    {
        Icon: ProfileIcon,
        label: "Profile",
        keywords: ["Bio", "Handle", "Name"],
        value: LINKS.SettingsProfile,
    },
    {
        Icon: VisibleIcon,
        label: "Privacy",
        keywords: ["History", "Private"],
        value: LINKS.SettingsPrivacy,
    },
    {
        Icon: ObjectIcon,
        label: "Data",
        keywords: ["Export"],
        value: LINKS.SettingsData,
    },
    {
        Icon: LockIcon,
        label: "Authentication",
        keywords: [{ key: "Wallet", count: 1 }, { key: "Wallet", count: 2 }, { key: "Email", count: 1 }, { key: "Email", count: 2 }, "LogOut", "Security"],
        value: LINKS.SettingsAuthentication,
    },
    {
        Icon: WalletIcon,
        label: "Payment",
        labelArgs: { count: 2 },
        keywords: [],
        value: LINKS.SettingsPayments,
    },
    {
        Icon: ApiIcon,
        label: "Api",
        keywords: [{ key: "Api", count: 2 }],
        value: LINKS.SettingsApi,
    },
    {
        Icon: LightModeIcon,
        label: "Display",
        keywords: ["Theme", "Light", "Dark", "Interests", "Hidden", { key: "Tag", count: 1 }, { key: "Tag", count: 2 }, "History"],
        value: LINKS.SettingsDisplay,
    },
    {
        Icon: NotificationsCustomizedIcon,
        label: "Notification",
        labelArgs: { count: 2 },
        keywords: [{ key: "Alert", count: 1 }, { key: "Alert", count: 2 }, { key: "PushNotification", count: 1 }, { key: "PushNotification", count: 2 }],
        value: LINKS.SettingsNotifications,
    },
    {
        Icon: HistoryIcon,
        label: "FocusMode",
        labelArgs: { count: 2 },
        keywords: [{ key: "Schedule", count: 1 }, { key: "Schedule", count: 2 }, { key: "FocusMode", count: 1 }],
        value: LINKS.SettingsFocusModes,
    },
];

function getSearchItemOptionLabel(option: SearchItem) { return option.label ?? ""; }

/**
 * Stop default onSubmit, since when the search bar is a form this causes the page to reload
 */
function stopDefaultSubmit(event: FormEvent<HTMLDivElement>) {
    event.preventDefault();
    event.stopPropagation();
}

function FullWidthPopper(props: PopperProps) {
    const parentWidth = props.anchorEl && (props.anchorEl as HTMLElement).parentElement ? (props.anchorEl as HTMLElement).parentElement?.clientWidth : null;

    return <Popper {...props} sx={{
        left: "-12px!important",
        minWidth: parentWidth ?? (props.anchorEl as HTMLElement)?.clientWidth ?? "fit-content",
        maxWidth: "100%",
    }} placement="bottom-start" /> as JSX.Element;
}

export function SettingsSearchBar({
    debounce = DEFAULT_DEBOUNCE_MS,
    id = "search-bar",
    onChange,
    onInputChange,
    placeholder,
    value,
    ...props
}: SettingsSearchBarProps) {
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
            getOptionLabel={getSearchItemOptionLabel}
            onSubmit={stopDefaultSubmit}
            // The real onSubmit, since onSubmit is only triggered after 2 presses of the enter key (don't know why)
            onChange={onSubmit}
            PopperComponent={FullWidthPopper}
            renderOption={(props, option) => {
                const { Icon, keywords, label, unshapedKeywords, value } = option;
                // Check if any of the keywords matches the search string
                const shapedSearchString = shapeSearchText(internalValue);
                let displayedKeyword = "";
                if (keywords && shapedSearchString.length > 0) {
                    for (let i = 0; i < keywords?.length; i++) {
                        const keyword = keywords[i];
                        const unshapedKeyword = unshapedKeywords![i];
                        // Skip label
                        if (unshapedKeyword === label) continue;
                        // If keyword includes search string, display it
                        if (keyword.includes(shapedSearchString)) {
                            displayedKeyword = unshapedKeyword;
                            break;
                        }
                    }
                }
                console.log("shapedSearchString", shapedSearchString);
                console.log("option", option);
                console.log("displayedKeyword", displayedKeyword);
                return (
                    <MenuItem
                        {...props}
                        key={value}
                        onClick={() => {
                            setInternalValue(label ?? "");
                            onChange(label ?? "");
                            handleSelect(option);
                        }}
                    >
                        {/* Icon */}
                        {Icon && (
                            <Icon fill={palette.background.textSecondary} style={{
                                width: "24px",
                                height: "24px",
                                marginRight: "8px",
                            }} />
                        )}
                        {/* Object title */}
                        <ListItemText sx={{
                            "& .MuiTypography-root": {
                                textOverflow: "ellipsis",
                                whiteSpace: "nowrap",
                                overflow: "hidden",
                            },
                        }}>
                            {label}
                        </ListItemText>
                        {/* If search string matches a keyword, display the first match */}
                        {displayedKeyword && (
                            <ListItemText sx={{
                                "& .MuiTypography-root": {
                                    textOverflow: "ellipsis",
                                    whiteSpace: "nowrap",
                                    overflow: "hidden",
                                    textAlign: "right",
                                    color: palette.background.textSecondary,
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
                        inputProps={{
                            ...params.inputProps,
                            enterKeyHint: "search",
                        }}
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
}
