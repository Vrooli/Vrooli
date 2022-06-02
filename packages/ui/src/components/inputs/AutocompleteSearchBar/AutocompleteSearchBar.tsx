import { Autocomplete, CircularProgress, IconButton, Input, ListItemText, MenuItem, Paper, Typography } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { Search as SearchIcon } from '@mui/icons-material';
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
    sx,
    ...props
}: AutocompleteSearchBarProps) {
    const [internalValue, setInternalValue] = useState<string>(value);
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

    return (
        <Autocomplete
            disablePortal
            id={id}
            sx={sx}
            options={options}
            inputValue={internalValue}
            getOptionLabel={(option: AutocompleteOption) => option.label ?? ''}
            onSubmit={(event: any) => {
                // This is triggered when the user presses enter
                // We want to prevent the form from submitting, since this reloads the page 
                // for some reason.
                event.preventDefault();
                event.stopPropagation();
            }}
            renderOption={(_, option) => {
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
                    sx={{ p: '2px 4px', display: 'flex', alignItems: 'center', borderRadius: '10px' }}
                >
                    <Input
                        sx={{ ml: 1, flex: 1 }}
                        disableUnderline={true}
                        value={internalValue}
                        onChange={handleChange}
                        placeholder={placeholder}
                        autoFocus={props.autoFocus ?? false}
                        {...params.InputProps}
                        inputProps={params.inputProps}
                    />
                    <IconButton sx={{ p: '10px' }} aria-label="main-search-icon">
                        <SearchIcon />
                    </IconButton>
                </Paper>
            )
            }
            noOptionsText={noOptionsText}
        />
    );
}