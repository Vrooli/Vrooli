import { TextField } from '@material-ui/core';
import AwesomeDebouncePromise from 'awesome-debounce-promise';
import { useCallback } from 'react';

interface Props {
    label?: string;
    value: string;
    onChange: (updatedText: string) => any;
    debounce?: number;
    fullWidth?: boolean;
    className?: string;
}

export const SearchBar = ({
    label = 'Search...',
    value,
    onChange,
    debounce = 50,
    fullWidth = false,
    className,
    ...props
}: Props) => {

    const onSearch = useCallback(() => AwesomeDebouncePromise(onChange, debounce), [debounce, onChange]);

    return (
        <TextField
            className={className}
            label={label}
            value={value}
            onChange={onSearch}
            fullWidth={fullWidth}
            InputProps={{
                // endAdornment: (
                //     <InputAdornment>
                //         <IconButton>
                //             <SearchIcon />
                //         </IconButton>
                //     </InputAdornment>
                // )
            }}
            {...props}
        />
    );
}