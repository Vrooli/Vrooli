import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { tags, tagsVariables } from 'graphql/generated/tags';
import { tagsQuery } from 'graphql/query';
import { useQuery } from '@apollo/client';
import { StarFor, TagSortBy } from '@shared/consts';
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
    const onChipDelete = useCallback((tag: TagShape | Tag) => {
        onTagRemove(tag);
    }, [onTagRemove]);

    // Map of tag strings to queried tag data, so we can exclude tags that have already been queried before
    type TagsRef = { [key: string]: TagShape | Tag }
    const tagsRef = useRef<TagsRef | null>(null);
    // Whenever selected tags change, add unknown tags to the tag map
    useEffect(() => {
        if (!tagsRef.current) return;
        tags.forEach(tag => {
            if (!(tagsRef.current as TagsRef)[tag.tag]) (tagsRef.current as TagsRef)[tag.tag] = tag;
        });
    }, [tags]);

    const { data: autocompleteData, refetch: refetchAutocomplete } = useQuery<tags, tagsVariables>(tagsQuery, {
        variables: {
            input: {
                // Exclude tags that have already been fully queried, and match the search string
                // (i.e. in tag map, and have an ID)
                excludeIds: tagsRef.current !== null ? 
                    Object.values(tagsRef.current)
                        .filter(t => (t as Tag).id && t.tag.toLowerCase().includes(inputValue.toLowerCase()))
                        .map(t => t.id) as string[] : 
                    [],
                searchString: inputValue,
                sortBy: TagSortBy.StarsDesc,
                take: 25,
            }
        }
    });
    useEffect(() => { refetchAutocomplete() }, [inputValue, refetchAutocomplete]);

    /**
     * Store queried tags in the tag ref
     */
    useEffect(() => {
        if (!autocompleteData) return;
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        queried.forEach(tag => {
            if (!tagsRef.current) tagsRef.current = {};
            (tagsRef.current as TagsRef)[tag.tag] = tag;
        });
    }, [autocompleteData, tagsRef]);

    const autocompleteOptions: (TagShape | Tag)[] = useMemo(() => {
        if (!autocompleteData) return [];
        // Find queried
        const queried = autocompleteData.tags.edges.map(({ node }) => node);
        // Find already known, that match the search string
        const known = tagsRef.current ? 
            Object.values(tagsRef.current)
                .filter(tag => tag.tag.toLowerCase().includes(inputValue.toLowerCase())) :
            [];
        // Return all queried and known
        return [...queried, ...known];
    }, [autocompleteData, inputValue, tagsRef]);

    const handleIsStarred = useCallback((tag: string, isStarred: boolean) => {
        // Update tag ref
        if (!tagsRef.current) tagsRef.current = {};
        ((tagsRef.current as TagsRef)[tag] as any) = { ...(tagsRef.current as TagsRef)[tag], isStarred };
    }, [tagsRef]);

    return (
        <Autocomplete
            id="tags-input"
            disabled={disabled}
            fullWidth
            multiple
            freeSolo={true}
            options={autocompleteOptions}
            getOptionLabel={(o: string | TagShape | Tag) => (typeof o === 'string' ? o : o.tag)}
            inputValue={inputValue}
            noOptionsText={'No suggestions'}
            limitTags={3}
            onClose={clearText}
            value={tags}
            // Filter out what has already been selected
            filterOptions={(options, params) => options.filter(o => !tags.some(t => t.tag === o.tag))}
            renderTags={(value, getTagProps) =>
                value.map((option: TagShape | Tag, index) => (
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
            renderOption={(props, option: TagShape | Tag) => (
                <MenuItem
                    {...props}
                    onClick={() => onInputSelect(option as Tag)} //TODO
                >
                    <ListItemText>{option.tag}</ListItemText>
                    <StarButton
                        session={session}
                        objectId={option.id ?? ''}
                        starFor={StarFor.Tag}
                        isStar={(option as Tag).isStarred}
                        stars={(option as Tag).stars}
                        onChange={(isStar) => { handleIsStarred(option.tag, isStar); }}
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