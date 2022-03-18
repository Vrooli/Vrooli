import { Autocomplete, IconButton, Input, ListItemText, MenuItem, Paper, Typography } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { Search as SearchIcon } from '@mui/icons-material';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';

export function AutocompleteSearchBar<T>({
    id = 'search-bar',
    placeholder = 'Search...',
    options = [],
    getOptionKey,
    getOptionLabel,
    getOptionLabelSecondary,
    value,
    onChange,
    onInputChange,
    debounce = 200,
    loading = false,
    sx,
    ...props
}: AutocompleteSearchBarProps<T>) {
    const [popupOpen, setPopupOpen] = useState(false);
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const handleClick = useCallback((event) => {
        setAnchorEl(event.currentTarget);
        setPopupOpen(true);
    }, []);
    const handleClose = useCallback(() => setPopupOpen(false), []);

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

    return (
        <Autocomplete
            disablePortal
            id={id}
            sx={sx}
            options={options}
            inputValue={internalValue}
            getOptionLabel={getOptionLabel}
            renderOption={(_, option) => { return (
                <MenuItem
                    key={getOptionKey(option)}
                    onClick={() => {
                        setInternalValue(getOptionLabel(option));
                        onChangeDebounced(getOptionLabel(option));
                        handleClose();
                        onInputChange(option);
                    }}
                >
                    <ListItemText>{getOptionLabel(option)}</ListItemText>
                    {getOptionLabelSecondary ? <Typography color="text.secondary">{getOptionLabelSecondary(option)}</Typography> : null}
                </MenuItem>
            )}}  
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
                        autoFocus
                        {...params.InputProps}
                        inputProps={params.inputProps}
                    />
                    <IconButton sx={{ p: '10px' }} aria-label="main-search-icon">
                        <SearchIcon />
                    </IconButton>
                </Paper>
            )
            }
        />
    );
}