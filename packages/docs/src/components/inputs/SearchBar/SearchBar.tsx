import { IconButton, Input, Paper } from '@mui/material';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { SearchBarProps } from '../types';
import { Search as SearchIcon } from '@mui/icons-material';

export const SearchBar = ({
    id = 'search-bar',
    placeholder = 'Search...',
    value,
    onChange,
    debounce = 200,
    ...props
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
        < Paper
            component="form"
            sx={{ p: '2px 4px', display: 'flex', alignItems: 'center', borderRadius: '10px' }}
        >
            <Input
                id={id}
                sx={{ ml: 1, flex: 1 }}
                disableUnderline={true}
                value={internalValue}
                onChange={handleChange}
                placeholder={placeholder}
                autoFocus
                {...props}
                inputProps={props.inputProps}
            />
            <IconButton sx={{ p: '10px' }} aria-label="main-search-icon">
                <SearchIcon />
            </IconButton>
        </Paper>
    );
}