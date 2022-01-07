import { IconButton, InputAdornment, TextField } from '@mui/material';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { useCallback } from 'react';
import { SearchBarProps } from '../types';
import { Search as SearchIcon } from '@mui/icons-material';

export const SearchBar = ({
    label = 'Search...',
    value,
    onChange,
    debounce = 50,
    fullWidth = false,
    className,
    ...props
}: SearchBarProps) => {

    const onSearch = useCallback(() => AwesomeDebouncePromise(onChange, debounce), [debounce, onChange]);

    return (
        <TextField
            className={className}
            label={label}
            value={value}
            onChange={onSearch}
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