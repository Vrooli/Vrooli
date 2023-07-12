import { SearchIcon } from "@local/shared";
import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, IconButton, Input, ListItemText, MenuItem, Paper, Popper, useTheme } from "@mui/material";
import { ChangeEvent, useCallback, useState } from "react";
import { useTranslation } from "react-i18next";
import { findSearchResults, SearchItem, shapeSearchText } from "utils/search/siteToSearch";
import { SettingsSearchBarProps } from "../types";

const FullWidthPopper = function (props) {
    return <Popper {...props} style={{
        width: "fit-content",
        maxWidth: "100%",
    }} placement="bottom-start" />;
};

export const SettingsSearchBar = ({
    debounce = 200,
    id = "search-bar",
    options = [],
    onChange,
    onInputChange,
    placeholder,
    value,
    ...props
}: SettingsSearchBarProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

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
                        placeholder={placeholder ?? t("SearchEllipsis")}
                        autoFocus={props.autoFocus ?? false}
                        // {...params.InputLabelProps}
                        inputProps={params.inputProps}
                        ref={params.InputProps.ref}
                        size={params.size}
                        sx={{
                            ml: 1,
                            flex: 1,
                            // Drop down/up icon
                            "& .MuiAutocomplete-endAdornment": {
                                width: "48px",
                                height: "48px",
                                top: "0",
                                position: "relative",
                                "& .MuiButtonBase-root": {
                                    width: "48px",
                                    height: "48px",
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
            }}
        />
    );
};
