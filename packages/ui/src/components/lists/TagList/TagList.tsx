import { Box, Chip, Stack, styled, useTheme } from "@mui/material";
import { type SyntheticEvent, useCallback, useMemo, useState } from "react";
import { PopoverWithArrow } from "../../dialogs/PopoverWithArrow/PopoverWithArrow.js";
import { ListItemChip } from "../ObjectListItemBase/ObjectListItemBase.js";
import { type TagListProps } from "../types.js";

const DEFAULT_MAX_CHARACTERS = 50;

const OuterBox = styled(Box)(({ theme }) => ({
    alignItems: "center",
    display: "flex",
    flexWrap: "wrap",
    gap: theme.spacing(1),
}));
const TagChip = styled(Chip)(({ theme }) => ({
    borderRadius: theme.spacing(2),
    transition: "all 0.2s ease",
    backgroundColor: theme.palette.primary.light,
    color: theme.palette.primary.contrastText,
    cursor: "pointer",
    "&:hover": {
        boxShadow: theme.shadows[1],
        backgroundColor: theme.palette.primary.main,
    },
}));

export function TagList({
    maxCharacters = DEFAULT_MAX_CHARACTERS,
    tags,
}: TagListProps) {
    const { palette } = useTheme();
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

    const handleMoreClick = useCallback((event: SyntheticEvent) => {
        setAnchorEl(event.currentTarget as HTMLElement);
    }, []);

    const handleClose = useCallback(() => {
        setAnchorEl(null);
    }, []);

    return (
        <>
            <OuterBox>
                {displayedTags.map((tag) => (
                    <TagChip
                        key={tag.tag}
                        label={tag.tag}
                        size="small"
                    />
                ))}

                {hiddenTags.length > 0 && (
                    <Chip
                        label={`+${hiddenTags.length}`}
                        size="small"
                        onClick={handleMoreClick}
                        sx={{
                            backgroundColor: palette.secondary.light,
                            color: palette.secondary.contrastText,
                            fontWeight: "bold",
                            cursor: "pointer",
                            borderRadius: "16px",
                            transition: "all 0.2s ease",
                            "&:hover": {
                                backgroundColor: palette.secondary.main,
                                transform: "translateY(-2px)",
                                boxShadow: 1,
                            },
                        }}
                    />
                )}
            </OuterBox>
            <PopoverWithArrow
                anchorEl={anchorEl}
                handleClose={handleClose}
                title="All Tags"
            >
                <Box p={2} maxWidth="300px">
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
                                    "&:hover": {
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
