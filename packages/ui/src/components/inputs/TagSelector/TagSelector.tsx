import { useCallback, useEffect, useMemo, useState } from 'react';
import { tags, tagsVariables } from 'graphql/generated/tags';
import { tagsQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { TagSortBy } from '@local/shared';
import { TagSelectorProps } from '../types';
import { Autocomplete, Chip, Paper, TextField } from '@mui/material';

export const TagSelector = ({
    tags,
    onTagAdd,
    onTagRemove,
    onTagsClear
}: TagSelectorProps) => {
    const [inputValue, setInputValue] = useState<string>('');
    const onChange = useCallback((change: any) => {
        console.log('onChange', change);
        // setSearchString(newValue) 
    }, []);
    const onInputChange = useCallback((newValue: any) => {
        console.log('onInputChange', newValue);
    }, []);
    const { data: autocompleteData, refetch: refetchAutocomplete } = useQuery<tags, tagsVariables>(tagsQuery, {
        variables: {
            input: {
                searchString: inputValue,
                sortBy: TagSortBy.AlphabeticalDesc,
                take: 25,
            }
        }
    });
    useEffect(() => { refetchAutocomplete() }, [inputValue]);
    const autocompleteOptions: string[] = useMemo(() => {
        if (!autocompleteData) return [];
        return autocompleteData.tags.edges.map(({ node }) => node.tag);
    }, [autocompleteData]);

    console.log('tags', tags);
    console.log('autocomplete options', autocompleteOptions);

    return (
        // <Selector
        //     id="favorite-tags-input"
        //     placeholder='Enter topics you are interested in, followed by spaces or arrows...'
        //     multiple
        //     fullWidth
        //     selected={tags}
        //     options={autocompleteOptions}
        //     getOptionLabel={(o: any) => `${o.tag} | ${o.description}`}
        //     // selected={searchString}
        //     handleChange={updateSearchString}
        //     // onInputChange={() => {}}
        //     sx={{ width: 'min(100%, 600px)' }}
        // />

        <Autocomplete
            id="tags-input"
            disablePortal
            fullWidth
            multiple
            options={autocompleteOptions}
            getOptionLabel={(o: any) => o}
            inputValue={inputValue}
            onInputChange={onInputChange}
            value={tags}
            renderTags={(value, getTagProps) =>
                value.map((option, index) => (
                    <Chip variant="outlined" label={option} {...getTagProps({ index })} />
                ))
            }
            renderInput={(params) => (
                <TextField
                    sx={{ paddingRight: 0 }}
                    value={inputValue}
                    onChange={onChange}
                    placeholder="Enter tags, followed by commas..."
                    autoFocus
                    fullWidth
                    InputProps={params.InputProps}
                    inputProps={params.inputProps}
                />
            )}
        />
    )
}