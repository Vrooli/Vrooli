import { useMemo } from 'react';
import { TagListProps } from '../types';
import { Chip, Stack, Tooltip, Typography, useTheme } from '@mui/material';

export const TagList = ({
    session,
    parentId,
    sx,
    tags,
}: TagListProps) => {
    const { palette } = useTheme();

    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = 50;
        let chipResult: JSX.Element[] = [];
        for (let i = 0; i < tags.length; i++) {
            const tag = tags[i];
            if (tag?.tag && tag.tag.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= tag.tag.length;
                chipResult.push(
                    <Chip
                        key={tag.tag}
                        label={tag.tag}
                        size="small"
                        sx={{
                            backgroundColor: palette.mode === 'light' ? '#8148b0' : '#8148b0', //'#a068ce',
                            color: 'white',
                            width: 'fit-content',
                        }} />
                );
            }
        }
        // Check if any tags were cut off
        const numTagsCutOff = tags.length - chipResult.length;
        return [chipResult, numTagsCutOff];
    }, [palette.mode, tags]);

    return (
        <Tooltip title={tags.map(t => t.tag).join(', ')} placement="top">
            <Stack
                direction="row"
                spacing={1}
                justifyContent="left"
                alignItems="center"
                sx={{...(sx ?? {})}}
            >
                {chips}
                {numTagsCutOff > 0 && <Typography variant="body1">+{numTagsCutOff}</Typography>}
            </Stack>
        </Tooltip>
    )
}