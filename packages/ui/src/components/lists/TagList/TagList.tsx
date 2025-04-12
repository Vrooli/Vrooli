import { Box, Chip, Stack, useTheme } from "@mui/material";
import { useCallback, useMemo, useState } from "react";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { ListItemChip } from "../ObjectListItemBase/ObjectListItemBase.js";
import { TagListProps } from "../types.js";

export function TagList({
    maxCharacters = 50,
    sx,
    tags,
}: TagListProps) {
    const { palette, spacing } = useTheme();
    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);

    const [displayedTags, hiddenTags] = useMemo(() => {
        let charactersBeforeCutoff = maxCharacters;
        const visibleTags: typeof tags = [];
        const hiddenTagsList: typeof tags = [];

        for (let i = 0; i < tags.length; i++) {
            const tag = tags[i];
            if (tag?.tag && tag.tag.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= tag.tag.length;
                visibleTags.push(tag);
            } else {
                hiddenTagsList.push(tag);
            }
        }

        return [visibleTags, hiddenTagsList];
    }, [maxCharacters, tags]);

    const handleMoreClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
        setAnchorEl(event.currentTarget);
    }, []);

    const handleClose = useCallback(() => {
        setAnchorEl(null);
    }, []);

    return (
        <>
            <Box
                sx={{
                    display: 'flex',
                    flexWrap: 'wrap',
                    gap: 0.75,
                    alignItems: 'center',
                    ...sx,
                }}
            >
                {displayedTags.map((tag) => (
                    <ListItemChip
                        color="Purple"
                        key={tag.tag}
                        label={tag.tag}
                        size="small"
                        sx={{
                            borderRadius: '16px',
                            transition: 'all 0.2s ease',
                            backgroundColor: palette.primary.light,
                            color: palette.primary.contrastText,
                            '&:hover': {
                                transform: 'translateY(-2px)',
                                boxShadow: 1,
                                backgroundColor: palette.primary.main,
                            },
                        }}
                    />
                ))}

                {hiddenTags.length > 0 && (
                    <Chip
                        label={`+${hiddenTags.length}`}
                        size="small"
                        onClick={(e) => handleMoreClick(e as any)}
                        sx={{
                            backgroundColor: palette.secondary.light,
                            color: palette.secondary.contrastText,
                            fontWeight: 'bold',
                            cursor: 'pointer',
                            borderRadius: '16px',
                            transition: 'all 0.2s ease',
                            '&:hover': {
                                backgroundColor: palette.secondary.main,
                                transform: 'translateY(-2px)',
                                boxShadow: 1,
                            },
                        }}
                    />
                )}
            </Box>

            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={handleClose}
                title="All Tags"
            >
                <Box sx={{ p: 2, maxWidth: '300px' }}>
                    <Stack direction="row" spacing={1} flexWrap="wrap" gap={1}>
                        {tags.map((tag) => (
                            <ListItemChip
                                color="Purple"
                                key={tag.tag}
                                label={tag.tag}
                                size="small"
                                sx={{
                                    backgroundColor: palette.primary.light,
                                    color: palette.primary.contrastText,
                                    '&:hover': {
                                        backgroundColor: palette.primary.main,
                                    },
                                }}
                            />
                        ))}
                    </Stack>
                </Box>
            </PopoverWithArrow>
        </>
    );
}
