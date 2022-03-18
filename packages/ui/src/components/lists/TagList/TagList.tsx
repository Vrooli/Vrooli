import { useMemo, useState } from 'react';
import { TagListProps } from '../types';
import { Chip, Stack, Tooltip, Typography } from '@mui/material';

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