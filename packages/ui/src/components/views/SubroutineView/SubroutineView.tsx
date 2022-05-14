import { Box, CircularProgress, IconButton, Link, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import {
    MoreHoriz as EllipsisIcon,
} from "@mui/icons-material";
import { ResourceListHorizontal } from "components";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo, useState } from "react";
import { containerShadow } from "styles";
import { getOwnedByString, getTranslation, getUserLanguages, toOwnedBy } from "utils";
import { useLocation } from "wouter";
import { SubroutineViewProps } from "../types";

export const SubroutineView = ({
    loading,
    data,
    session,
}: SubroutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { description, instructions, title } = useMemo(() => {
        const languages = navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            instructions: getTranslation(data, 'instructions', languages, true),
            title: getTranslation(data, 'title', languages, true),
        }
    }, [data]);

    const ownedBy = useMemo<string | null>(() => getOwnedByString(data, getUserLanguages(session)), [data, session]);
    const toOwner = useCallback(() => { toOwnedBy(data, setLocation) }, [data, setLocation]);

    // The schema for the form
    const [schema, setSchema] = useState<any>();

    const resourceList = useMemo(() => {
        if (!data ||
            !Array.isArray(data.resourceLists) ||
            data.resourceLists.length < 1 ||
            data.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(data as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            session={session}
        />
    }, [data, session]);

    if (loading) return (
        <Box sx={{
            minHeight: 'min(300px, 25vh)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        }}>
            <CircularProgress color="secondary" />
        </Box>
    )
    return (
        <Box sx={{
            background: palette.background.paper,
            overflowY: 'auto',
            width: 'min(96vw, 600px)',
            borderRadius: '8px',
            overflow: 'overlay',
            ...containerShadow
        }}>
            {/* Heading container */}
            <Stack direction="column" spacing={1} sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: 2,
                marginBottom: 1,
                background: palette.primary.main,
                color: palette.primary.contrastText,
            }}>
                {/* Show more ellipsis next to title */}
                <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="h5">{title}</Typography>
                    <Tooltip title="Options">
                        <IconButton
                            aria-label="More"
                            size="small"
                            onClick={() => { }}
                            sx={{
                                display: 'block',
                                marginLeft: 'auto',
                                marginRight: 1,
                            }}
                        >
                            <EllipsisIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1}>
                    {ownedBy ? (
                        <Link onClick={toOwner}>
                            <Typography variant="body1" sx={{ color: palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy}</Typography>
                        </Link>
                    ) : null}
                    <Typography variant="body1"> - {data?.version}</Typography>
                </Stack>
            </Stack>
            {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
            <Stack direction="column" spacing={2} padding={1}>
                {/* Resources */}
                {resourceList}
                {/* Description */}
                <Box sx={{
                    padding: 1,
                    borderRadius: 1,
                    color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary,
                }}>
                    <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Description</Typography>
                    <Typography variant="body1">{description ?? 'No description set'}</Typography>
                </Box>
                {/* Instructions */}
                <Box sx={{
                    padding: 1,
                    borderRadius: 1,
                    color: Boolean(instructions) ? palette.background.textPrimary : palette.background.textSecondary
                }}>
                    <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Instructions</Typography>
                    <Markdown>{instructions ?? 'No instructions'}</Markdown>
                </Box>
            </Stack>
        </Box>
    )
}