import { IconButton, InputAdornment, TextField } from '@mui/material';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { ChangeEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { SearchBarProps } from '../types';
import { Search as SearchIcon } from '@mui/icons-material';

export const SearchBar = ({
    label = 'Search...',
    value,
    onChange,
    debounce = 2000,
    fullWidth = false,
    className,
    ...props
}: SearchBarProps) => {
    const [internalValue, setInternalValue] = useState<string>(value);
    const onChangeDebounced = useMemo(() => AwesomeDebouncePromise(
        onChange,
        debounce,
    ), [onChange, debounce]);
    useEffect(() => setInternalValue(value), [value]);

    const handleChange = useCallback((event: ChangeEvent<HTMLInputElement>) => {
        const { value } = event.target;
        setInternalValue(value);
        onChangeDebounced(value);
    }, [onChangeDebounced]);

    return (
        <TextField
            className={className}
            label={label}
            value={internalValue}
            onChange={handleChange}
            fullWidth={fullWidth}
            InputProps={{
                endAdornment: (
                    <InputAdornment position="end">
                        <IconButton>
                            <SearchIcon />
                        </IconButton>
                    </InputAdornment>
                )
            }}
            {...props}
        />
    );
}