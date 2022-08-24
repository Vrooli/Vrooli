import { Autocomplete, AutocompleteChangeDetails, AutocompleteChangeReason, AutocompleteHighlightChangeReason, CircularProgress, IconButton, Input, ListItemText, MenuItem, Paper, TextField, Typography } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import {
    Search as SearchIcon
} from '@mui/icons-material';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';
import { AutocompleteOption } from 'types';

export function AutocompleteSearchBar({
    id = 'search-bar',
    placeholder = 'Search...',
    options = [],
    value,
    onChange,
    onInputChange,
    debounce = 200,
    loading = false,
    session,
    showSecondaryLabel = false,
    sxs,
    ...props
}: AutocompleteSearchBarProps) {
    // Input internal value (since value passed back is debounced)
    const [internalValue, setInternalValue] = useState<string>(value);
    // Highlighted option (if navigated with keyboard)
    const [highlightedOption, setHighlightedOption] = useState<AutocompleteOption | null>(null);

    const onChangeDebounced = useMemo(() => AwesomeDebouncePromise(
        onChange,
        debounce,
    ), [onChange, debounce]);
    useEffect(() => setInternalValue(value), [value]);
    const handleChange = useCallback((event: ChangeEvent<any>) => {
        // Get the new input string
        const { value } = event.target;
        // Update state
        setInternalValue(value);
        // Remove the highlight
        setHighlightedOption(null);
        // Debounce onChange
        onChangeDebounced(value);
    }, [onChangeDebounced]);

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
                        sx={{
                            marginRight: 1,
                        }}
                    />
                    Loading...
                </>
            )
        }
        return 'No options';
    }, [loading, value, internalValue]);

    const onHighlightChange = useCallback((event: React.SyntheticEvent<Element, Event>, option: AutocompleteOption | null, reason: AutocompleteHighlightChangeReason) => {
        if (option && option.label && reason === 'keyboard') {
            setHighlightedOption(option);
        }
    }, [])

    const onSubmit = useCallback((event: React.SyntheticEvent<Element, Event>, value: AutocompleteOption | null, reason: AutocompleteChangeReason, details?: AutocompleteChangeDetails<any> | undefined) => {
        // If there is a highlighted option, use that
        if (highlightedOption) {
            onInputChange(highlightedOption);
        }
        // Otherwise, don't submit
    }, [highlightedOption, onInputChange])

    return (
        <Autocomplete
            disablePortal
            id={id}
            options={options}
            onHighlightChange={onHighlightChange}
            inputValue={internalValue}
            getOptionLabel={(option: AutocompleteOption) => option.label ?? ''}
            // Stop default onSubmit, since this reloads the page for some reason
            onSubmit={(event: any) => {
                event.preventDefault();
                event.stopPropagation();
            }}
            // The real onSubmit, since onSubmit is only triggered after 2 presses of the enter key (don't know why)
            onChange={onSubmit}
            renderOption={(props, option) => {
                // If loading, display spinner
                if (option.__typename === 'Loading') {
                    return (
                        <MenuItem
                            key="loading"
                        >
                            {/* Object title */}
                            <ListItemText sx={{
                                '& .MuiTypography-root': {
                                    textOverflow: 'ellipsis',
                                    whiteSpace: 'nowrap',
                                    overflow: 'hidden',
                                },
                            }}>
                                <CircularProgress
                                    color="secondary"
                                    size={20}
                                    sx={{
                                        marginRight: 1,
                                    }}
                                />
                                {option.label ?? ''}
                            </ListItemText>
                        </MenuItem>
                    )
                }
                return (
                    <MenuItem
                        {...props}
                        key={option.id}
                        onClick={() => {
                            const label = option.label ?? '';
                            setInternalValue(label);
                            onChangeDebounced(label);
                            onInputChange(option);
                        }}
                    >
                        {/* Object title */}
                        <ListItemText sx={{
                            '& .MuiTypography-root': {
                                textOverflow: 'ellipsis',
                                whiteSpace: 'nowrap',
                                overflow: 'hidden',
                            },
                        }}>
                            {option.label}
                        </ListItemText>
                        {/* Type of object */}
                        {
                            showSecondaryLabel ?
                                <Typography color="text.secondary">
                                    {option.__typename === 'Shortcut' ? "↪ Shortcut" : option.__typename}
                                </Typography> :
                                null
                        }
                    </MenuItem>
                )
            }}
            renderInput={(params) => (
                <Paper
                    component="form"
                    sx={{ ...(sxs?.paper ?? {}), p: '2px 4px', display: 'flex', alignItems: 'center', borderRadius: '10px' }}
                >
                    <Input
                        id={params.id}
                        disabled={params.disabled}
                        disableUnderline={true}
                        fullWidth={params.fullWidth}
                        value={internalValue}
                        onChange={handleChange}
                        placeholder={placeholder}
                        autoFocus={props.autoFocus ?? false}
                        // {...params.InputLabelProps}
                        inputProps={params.inputProps}
                        ref={params.InputProps.ref}
                        size={params.size}
                        sx={{
                            ml: 1,
                            flex: 1,
                            // Drop down/up icon
                            '& .MuiAutocomplete-endAdornment': {
                                width: '48px',
                                height: '48px',
                                top: '0',
                                position: 'relative',
                                '& .MuiButtonBase-root': {
                                    width: '48px',
                                    height: '48px',
                                }
                            }
                        }}
                    />
                    <IconButton sx={{
                        width: '48px',
                        height: '48px',
                    }} aria-label="main-search-icon">
                        <SearchIcon />
                    </IconButton>
                </Paper>
            )}
            noOptionsText={noOptionsText}
            sx={{
                ...(sxs?.root ?? {}),
                '& .MuiAutocomplete-inputRoot': {
                    paddingRight: '0 !important',
                },
            }}
        />
    );
}