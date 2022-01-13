import { Autocomplete, IconButton, Input, Paper } from '@mui/material';
import { AutocompleteSearchBarProps } from '../types';
import { SearchBar } from '..';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { Search as SearchIcon } from '@mui/icons-material';
import { ChangeEvent, useCallback, useState, useEffect, useMemo } from 'react';

export const AutocompleteSearchBar = ({
    id = 'search-bar',
    placeholder='Search...',
    options = [],
    getOptionLabel = (option) => option,
    value,
    onChange,
    onInputChange,
    debounce = 200,
    sx,
    ...props
}: AutocompleteSearchBarProps) => {
    console.log('AutocompleteSearchBar', options, value);
    const [internalValue, setInternalValue] = useState<string>(value);
    const onChangeDebounced = useMemo(() => AwesomeDebouncePromise(
        onChange,
        debounce,
    ), [onChange, debounce]);
    useEffect(() => setInternalValue(value), [value]);

    const handleChange = useCallback((event: ChangeEvent<any>) => {
        const { value } = event.target;
        setInternalValue(value);
        onChangeDebounced(value);
    }, [onChangeDebounced]);
    return (
        <Autocomplete
            disablePortal
            id={id}
            sx={sx}
            options={options}
            getOptionLabel={getOptionLabel}
            onInputChange={onInputChange}
            renderInput={(params) => (
                // <SearchBar TODO doesn't work for some reason
                //     value={value}
                //     onChange={onChange}
                //     debounce={debounce}
                //     {...props}
                //     {...params.InputProps}
                //     inputProps={params.inputProps}
                // />
                < Paper
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