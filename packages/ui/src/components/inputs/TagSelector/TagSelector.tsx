import { useCallback, useEffect, useMemo, useState } from 'react';
import { tags, tagsVariables } from 'graphql/generated/tags';
import { tagsQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { StarFor, TagSortBy } from '@local/shared';
import { TagSelectorProps } from '../types';
import { Autocomplete, Chip, ListItemText, MenuItem, TextField, useTheme } from '@mui/material';
import { StarButton } from 'components';
import { PubSub, TagShape } from 'utils';
import { Tag } from 'types';

export const TagSelector = ({
    disabled,
    session,
    tags,
    placeholder = 'Enter tags, followed by commas...',
    onTagAdd,
    onTagRemove,
    onTagsClear,
}: TagSelectorProps) => {
    const { palette } = useTheme();
    const [inputValue, setInputValue] = useState<string>('');
    const clearText = useCallback(() => { setInputValue(''); }, []);
    const onChange = useCallback((change: any) => {
        // Remove invalid characters (i.e. ',' or ';')
        setInputValue(change.target.value.replace(/[,;]/g, ''))
    }, []);
    const onKeyDown = useCallback((event: any) => {
        let tagLabel;
        // Check if the user pressed ',' or ';'
        if (event.code === 'Comma' || event.code === 'Semicolon') {
            tagLabel = inputValue;
        }
        // Check if the user pressed enter
        else if (event.code === 'enter' && event.target.value) {
            tagLabel = inputValue + event.key
        }
        else return;
        // Remove invalid characters (i.e. ',' or ';')
        tagLabel = tagLabel.replace(/[,;]/g, '');
        // Check if tag is valid length
        if (tagLabel.length < 2) {
            PubSub.get().publishSnack({ message: 'Tag too short.', severity: 'error' });
            return;
        }
        if (tagLabel.length > 30) {
            PubSub.get().publishSnack({ message: 'Tag too long.', severity: 'error' });
            return;
        }
        // Determine if tag is already selected
        const isSelected = tags.some(t => t.tag === tagLabel);
        if (isSelected) {
            PubSub.get().publishSnack({ message: 'Tag already selected.', severity: 'error' });
            return;
        }
        // Add tag
        onTagAdd({ tag: tagLabel });
        // Clear input
        clearText();
    }, [clearText, inputValue, onTagAdd, tags]);

    const onInputSelect = useCallback((tag: Tag) => {
        setInputValue('');
        // Determine if tag is already selected
        const isSelected = tags.some(t => t.tag === tag.tag);
        if (isSelected) onTagRemove(tag);
        else onTagAdd(tag);
    }, [tags, onTagAdd, onTagRemove]);
    const onChipDelete = useCallback((tag: TagShape) => {
        onTagRemove(tag);
    }, [onTagRemove]);
    const { data: autocompleteData, refetch: refetchAutocomplete } = useQuery<tags, tagsVariables>(tagsQuery, {
        variables: {
            input: {
                searchString: inputValue,
                sortBy: TagSortBy.StarsDesc,
                take: 25,
            }
        }
    });
    useEffect(() => { refetchAutocomplete() }, [inputValue, refetchAutocomplete]);
    const autocompleteOptions: TagShape[] = useMemo(() => {
        if (!autocompleteData) return [];
        return autocompleteData.tags.edges.map(({ node }) => node);
    }, [autocompleteData]);

    // //TODO store all queried tag data, and query unknown tag data (e.g. tags set in URL, but you don't know isStarred yet)
    // const [fullTagData, setFullTagData] = useState<{ [tag: string]: Tag }>({});

    return (
        <Autocomplete
            id="tags-input"
            disabled={disabled}
            fullWidth
            multiple
            freeSolo={true}
            options={autocompleteOptions}
            getOptionLabel={(o: string | TagShape) => (typeof o === 'string' ? o : o.tag)}
            inputValue={inputValue}
            noOptionsText={'No suggestions'}
            limitTags={3}
            onClose={clearText}
            value={tags}
            // Filter out what has already been selected
            filterOptions={(options, params) => options.filter(o => !tags.some(t => t.tag === o.tag))}
            renderTags={(value, getTagProps) =>
                value.map((option: TagShape, index) => (
                    <Chip
                        variant="filled"
                        label={option.tag}
                        {...getTagProps({ index })}
                        onDelete={() => onChipDelete(option)}
                        sx={{
                            backgroundColor: palette.mode === 'light' ? '#8148b0' : '#8148b0', //'#a068ce',
                            color: 'white',
                        }}
                    />
                )
                )}
            renderOption={(props, option: TagShape) => (
                <MenuItem
                    {...props}
                    onClick={() => onInputSelect(option as Tag)} //TODO
                >
                    <ListItemText>{option.tag}</ListItemText>
                    <StarButton
                        session={session}
                        objectId={option.tag ?? ''}
                        starFor={StarFor.Tag}
                        // isStar={option.isStarred} // TODO
                        // stars={option.stars} //TODO
                        isStar={false}
                        stars={0}
                        onChange={(isStar: boolean) => {}} //TODO
                    />
                </MenuItem>
            )}
            renderInput={(params) => (
                <TextField
                    value={inputValue}
                    onChange={onChange}
                    placeholder={placeholder}
                    InputProps={params.InputProps}
                    inputProps={params.inputProps}
                    onKeyDown={onKeyDown}
                    fullWidth
                    sx={{ paddingRight: 0, minWidth: '250px' }}
                />
            )}
        />
    )
}