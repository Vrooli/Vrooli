import { useCallback, useEffect, useMemo, useState } from 'react';
import { useMutation } from '@apollo/client';
import { StarFor } from '@local/shared';
import { TagListProps } from '../types';
import { Autocomplete, Chip, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { Pubs } from 'utils';
import { Tag } from 'types';

export const TagList = ({
    session,
    parentId,
    tags,
}: TagListProps) => {
    const [inputValue, setInputValue] = useState<string>('');

    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = 50;
        let chipResult: JSX.Element[] = [];
        for (let i = 0; i < tags.length; i++) {
            const tag = tags[i];
            if (tag?.tag && tag.tag.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= tag.tag.length;
                chipResult.push(
                    <Chip
                        key={tag.id}
                        label={tag.tag}
                        size="small"
                        sx={{
                            backgroundColor: '#dc697d',
                            color: 'white',
                            width: 'fit-content',
                        }} />
                );
            }
        }
        // Check if any tags were cut off
        const numTagsCutOff = tags.length - chipResult.length;
        return [chipResult, numTagsCutOff];
    }, [tags]);

    return (
        <Tooltip title={tags.map(t => t.tag).join(', ')} placement="top">
            <Stack direction="row" spacing={1} justifyContent="left" alignItems="center">
                {chips}
                {numTagsCutOff > 0 && <Typography variant="body1">+{numTagsCutOff}</Typography>}
            </Stack>
        </Tooltip>
    )
}