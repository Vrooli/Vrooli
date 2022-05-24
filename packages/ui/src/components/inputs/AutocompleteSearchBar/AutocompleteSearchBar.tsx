import { Autocomplete, IconButton, Input, ListItemText, MenuItem, Paper, Typography } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { Search as SearchIcon } from '@mui/icons-material';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';
import { getUserLanguages } from 'utils';

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
    session,
    sx,
    ...props
}: AutocompleteSearchBarProps<T>) {
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

    const languages = useMemo(() => getUserLanguages(session), [session]);

    return (
        <Autocomplete
            disablePortal
            id={id}
            sx={sx}
            options={options}
            inputValue={internalValue}
            getOptionLabel={(option: any) => getOptionLabel(option, languages)}
            renderOption={(_, option) => { return (
                <MenuItem
                    key={getOptionKey(option, languages)}
                    onClick={() => {
                        setInternalValue(getOptionLabel(option, languages));
                        onChangeDebounced(getOptionLabel(option, languages));
                        onInputChange(option);
                    }}
                >
                    <ListItemText>{getOptionLabel(option, languages)}</ListItemText>
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
        />
    );
}