import { Autocomplete, IconButton, Input, Paper } from '@mui/material';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { SearchBarProps } from '../types';
import { Search as SearchIcon } from '@mui/icons-material';

export const SearchBar = ({
    id = 'search-bar',
    placeholder = 'Search...',
    options = [],
    getOptionLabel = (option) => option,
    value,
    onChange,
    onInputChange,
    debounce = 200,
    sx,
}: SearchBarProps) => {
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
            options={options}
            getOptionLabel={getOptionLabel}
            sx={sx}
            onInputChange={onInputChange}
            renderInput={(params) => (
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