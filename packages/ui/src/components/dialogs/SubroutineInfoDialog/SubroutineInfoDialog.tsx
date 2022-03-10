/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useMemo, useState } from 'react';
import {
    Close as CloseIcon,
    Info as InfoIcon,
} from '@mui/icons-material';
import {
    Box,
    Grid,
    IconButton,
    Link,
    SwipeableDrawer,
    Typography,
} from '@mui/material';
import { useLocation } from 'wouter';
import { User } from 'types';
import { SubroutineInfoDialogProps } from '../types';
import { getTranslation, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import Markdown from 'markdown-to-jsx';

export const SubroutineInfoDialog = ({
    open,
    language,
    subroutine,
    onClose,
}: SubroutineInfoDialogProps) => {
    const [, setLocation] = useLocation();

    /**
     * Name of user or organization that owns this routine
     */
    const ownedBy = useMemo<string | null>(() => {
        if (!subroutine?.owner) return null;
        return getTranslation(subroutine.owner, 'username', [language]) ?? getTranslation(subroutine.owner, 'name', [language]);
    }, [subroutine?.owner, language]);

    /**
     * Navigate to owner's profile
     */
    const toOwner = useCallback(() => {
        if (!subroutine?.owner) {
            PubSub.publish(Pubs.Snack, { message: 'Could not find owner.', severity: 'Error' });
            return;
        }
        // Check if user or organization
        if (subroutine?.owner.hasOwnProperty('username')) {
            setLocation(`${APP_LINKS.User}/${(subroutine?.owner as User).username}`);
        } else {
            setLocation(`${APP_LINKS.Organization}/${subroutine?.owner.id}`);
        }
    }, [subroutine?.owner, setLocation]);

    return (
        <SwipeableDrawer
            anchor="bottom"
            variant='persistent'
            open={open}
            onOpen={() => { }}
            onClose={onClose}
            sx={{
                '& .MuiDrawer-paper': {
                    background: (t) => t.palette.background.default,
                    borderLeft: `1px solid ${(t) => t.palette.text.primary}`,
                    maxHeight: 'min(300px, 100vh)',
                }
            }}
        >
            {/* Title bar with close icon */}
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                padding: 1,
            }}>
                {/* Close button */}
                <IconButton onClick={onClose} sx={{
                    background: (t) => t.palette.primary.light,
                    color: (t) => t.palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                {/* Subroutine title */}
                <Typography variant="h5">{getTranslation(subroutine, 'title', [language])}</Typography>
                {/* Owned by and version */}
                <Box>
                    {ownedBy ? (
                        <Link onClick={toOwner}>
                            <Typography variant="body1" sx={{ color: (t) => t.palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy} - </Typography>
                        </Link>
                    ) : null}
                    <Typography variant="body1">{subroutine?.version}</Typography>
                </Box>
                {/* Description and instructions */}
                <Grid container>
                    {/* Description */}
                    <Grid item xs={12} sm={6}>
                        <Box sx={{
                            padding: 1,
                            border: `1px solid ${(t) => t.palette.primary.dark}`,
                            borderRadius: 1,
                        }}>
                            <Typography variant="h6">Description</Typography>
                            {/* {
                                isEditing ? (
                                    <MarkdownInput
                                        id="description"
                                        placeholder="Description"
                                        value={formik.values.description}
                                        minRows={2}
                                        onChange={(newText: string) => formik.setFieldValue('description', newText)}
                                        error={formik.touched.description && Boolean(formik.errors.description)}
                                        helperText={formik.touched.description ? formik.errors.description as string : null}
                                    />
                                ) : (
                                    <Markdown>{getTranslation(routine, 'description', [language]) ?? ''}</Markdown>
                                )
                            } */}
                            <Markdown>{getTranslation(subroutine, 'description', [language]) ?? ''}</Markdown>
                        </Box>
                    </Grid>
                    {/* Instructions */}
                    <Grid item xs={12} sm={6}>
                        <Box sx={{
                            padding: 1,
                            border: `1px solid ${(t) => t.palette.background.paper}`,
                            borderRadius: 1,
                        }}>
                            <Typography variant="h6">Instructions</Typography>
                            {/* {
                                isEditing ? (
                                    <MarkdownInput
                                        id="instructions"
                                        placeholder="Instructions"
                                        value={formik.values.instructions}
                                        minRows={2}
                                        onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                                        error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                                        helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                                    />
                                ) : (
                                    <Markdown>{getTranslation(routine, 'instructions', [language]) ?? ''}</Markdown>
                                )
                            } */}
                            <Markdown>{getTranslation(subroutine, 'instructions', [language]) ?? ''}</Markdown>
                        </Box>
                    </Grid>
                </Grid>
            </Box>
        </SwipeableDrawer>
    );
}