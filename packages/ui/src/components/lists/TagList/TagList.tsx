import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation } from '@apollo/client';
import { StarFor } from '@local/shared';
import { TagListProps } from '../types';
import { Autocomplete, Chip, TextField } from '@mui/material';
import { star } from 'graphql/generated/star';
import { StarButton } from 'components';
import { starMutation } from 'graphql/mutation';
import { Pubs } from 'utils';
import { Tag } from 'types';

export const TagList = ({
    session,
    parentId,
    tags,
}: TagListProps) => {
    const [inputValue, setInputValue] = useState<string>('');

    // Allows for favoriting tags
    const [star] = useMutation<star>(starMutation);
    const handleStar = useCallback((e: any, isStar: boolean, tag: Tag) => {
        // Prevent propagation of normal click event
        e.stopPropagation();
        // Send star mutation
        star({
            variables: {
                input: {
                    isStar,
                    starFor: StarFor.Tag,
                    forId: tag.id
                }
            }
        });
    }, []);

    return (
        // Uses an autocomplete to make displaying tags easier
        <Autocomplete
            id="tags-list"
            fullWidth
            multiple
            // Options aren't needed for this component, but it will give an error if not provided
            options={tags}
            inputValue={inputValue}
            limitTags={3}
            value={tags}
            renderTags={(value, getTagProps) =>
                value.map((option: Tag, index) => (
                    <Chip
                        variant="filled"
                        label={option.tag}
                        sx={{
                            backgroundColor: '#dc697d',
                            color: 'white',
                        }}
                    />
                )
                )}
            renderInput={(params) => (
                <TextField
                    sx={{ paddingRight: 0 }}
                    fullWidth
                    InputProps={params.InputProps}
                    inputProps={params.inputProps}
                />
            )}
        />
    )
}