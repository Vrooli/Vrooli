import { APP_LINKS } from "@local/shared";
import { Box, CircularProgress, IconButton, Link, Stack, Tooltip, Typography } from "@mui/material";
import {
    MoreHoriz as EllipsisIcon,
} from "@mui/icons-material";
import { ResourceListHorizontal } from "components";
import { BaseForm } from "forms";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo, useState } from "react";
import { containerShadow } from "styles";
import { ResourceList, Routine, User } from "types";
import { getTranslation, getUserLanguages, Pubs } from "utils";
import { useLocation } from "wouter";
import { SubroutineViewProps } from "../types";

export const SubroutineView = ({
    loading,
    data,
    session,
}: SubroutineViewProps) => {
    const [, setLocation] = useLocation();

    const { description, instructions, title } = useMemo(() => {
        const languages = navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            instructions: getTranslation(data, 'instructions', languages, true),
            title: getTranslation(data, 'title', languages, true),
        }
    }, [data]);

    /**
     * Name of user or organization that owns this routine
     */
     const ownedBy = useMemo<string | null>(() => {
        if (!data?.owner) return null;
        const languages = getUserLanguages(session);
        return getTranslation(data.owner, 'username', languages) ?? getTranslation(data.owner, 'name', languages);
    }, [data?.owner, session]);

    // The schema for the form
    const [schema, setSchema] = useState<any>();

    /**
     * Navigate to owner's profile
     */
     const toOwner = useCallback(() => {
        if (!data?.owner) {
            PubSub.publish(Pubs.Snack, { message: 'Could not find owner.', severity: 'Error' });
            return;
        }
        // Check if user or organization
        if (data?.owner.hasOwnProperty('username')) {
            setLocation(`${APP_LINKS.User}/${(data?.owner as User).username}`);
        } else {
            setLocation(`${APP_LINKS.Organization}/${data?.owner.id}`);
        }
    }, [data?.owner, setLocation]);

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
            background: (t) => t.palette.background.paper,
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
                background: (t) => t.palette.primary.main,
                color: (t) => t.palette.primary.contrastText,
            }}>
                {/* Show more ellipsis next to title */}
                <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="h5">{title}</Typography>
                    <Tooltip title="Options">
                        <IconButton
                            aria-label="More"
                            size="small"
                            onClick={() => {}}
                            sx={{
                                display: 'block',
                                marginLeft: 'auto',
                                marginRight: 1,
                            }}
                        >
                            <EllipsisIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1}>
                    {ownedBy ? (
                        <Link onClick={toOwner}>
                            <Typography variant="body1" sx={{ color: (t) => t.palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy} - </Typography>
                        </Link>
                    ) : null}
                    <Typography variant="body1">{data?.version}</Typography>
                </Stack>
            </Stack>
            {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
            <Stack direction="column" spacing={2} padding={1}>
                {/* Resources */}
                {Array.isArray(data?.resourceLists) && (data?.resourceLists as ResourceList[]).length > 0 ? <ResourceListHorizontal
                    title={'Resources'}
                    list={(data as Routine).resourceLists[0]}
                    canEdit={false}
                    handleUpdate={() => { }}
                    session={session}
                /> : null}
                {/* Description */}
                <Box sx={{
                    padding: 1,
                    border: `1px solid ${(t) => t.palette.primary.dark}`,
                    borderRadius: 1,
                    color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                }}>
                    <Typography variant="h6">Description</Typography>
                    <Typography variant="body1" sx={{ color: description ? 'black' : 'gray' }}>{description ?? 'No description set'}</Typography>
                </Box>
                {/* Instructions */}
                <Box sx={{
                    padding: 1,
                    border: `1px solid ${(t) => t.palette.background.paper}`,
                    borderRadius: 1,
                    color: Boolean(instructions) ? 'text.primary' : 'text.secondary',
                }}>
                    <Typography variant="h6">Instructions</Typography>
                    <Markdown>{instructions ?? 'No instructions'}</Markdown>
                </Box>
            </Stack>
        </Box>
    )
}