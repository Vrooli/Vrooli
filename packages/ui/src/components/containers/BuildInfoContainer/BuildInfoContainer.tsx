/**
 * Displays metadata about the routine.
 * On the left is a status indicator, which lets you know if the routine is valid or not.
 * In the middle is the title of the routine. Once clicked, the information bar converts to 
 * a text input field, which allows you to edit the title of the routine.
 * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
 * title, descriptions, instructions, inputs, outputs, tags, etc.
 */
import { Box, Chip, IconButton, Menu, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import { useCallback, useMemo, useState } from 'react';
import {
    AddLink as AddLinkIcon,
    Close as CloseIcon,
    Compress as CleanUpIcon,
    Edit as EditIcon,
    Mood as ValidIcon,
    MoodBad as InvalidIcon,
    SentimentDissatisfied as IncompleteIcon,
} from '@mui/icons-material';
import { getTranslation, BuildStatus } from 'utils';
import { BuildInfoContainerProps } from '../types';
import { HelpButton, BuildInfoDialog, SelectLanguageDialog, UnlinkedNodesDialog } from 'components';
import Markdown from 'markdown-to-jsx';
import { noSelect } from 'styles';
import { EditableLabel } from 'components/inputs';

const helpText =
    `## What am I looking at?
Lorem ipsum dolor sit amet consectetur adipisicing elit. 


## How does it work?
Lorem ipsum dolor sit amet consectetur adipisicing elit.
`

/**
 * Status indicator and slider change color to represent routine's status
 */
const STATUS_COLOR = {
    [BuildStatus.Incomplete]: '#82970e', // Yellow
    [BuildStatus.Invalid]: '#802a2d', // Red
    [BuildStatus.Valid]: '#01a918', // Green
}
const STATUS_LABEL = {
    [BuildStatus.Incomplete]: 'Incomplete',
    [BuildStatus.Invalid]: 'Invalid',
    [BuildStatus.Valid]: 'Valid',
}
const STATUS_ICON = {
    [BuildStatus.Incomplete]: IncompleteIcon,
    [BuildStatus.Invalid]: InvalidIcon,
    [BuildStatus.Valid]: ValidIcon,
}

const TERTIARY_COLOR = '#95f3cd';

export const BuildInfoContainer = ({
    canEdit,
    handleAddLink,
    handleLanguageUpdate,
    handleNodeDelete,
    handleRoutineAction,
    handleRoutineUpdate,
    handleStartEdit,
    handleTitleUpdate,
    isEditing,
    language,
    loading,
    nodesOffGraph,
    routine,
    session,
    status,
}: BuildInfoContainerProps) => {
    const { palette } = useTheme();
    /**
     * List of status messages converted to markdown. 
     * If one message, no bullet points. If multiple, bullet points.
     */
    const statusMarkdown = useMemo(() => {
        if (status.messages.length === 0) return 'Routine is valid.';
        if (status.messages.length === 1) {
            return status.messages[0];
        }
        return status.messages.map((s) => {
            return `* ${s}`;
        }).join('\n');
    }, [status]);

    const [statusMenuAnchorEl, setStatusMenuAnchorEl] = useState(null);
    const statusMenuOpen = Boolean(statusMenuAnchorEl);
    const openStatusMenu = useCallback((event) => {
        if (!statusMenuAnchorEl) setStatusMenuAnchorEl(event.currentTarget);
    }, [statusMenuAnchorEl])
    const closeStatusMenu = () => {
        setStatusMenuAnchorEl(null);
    };

    /**
     * Menu displayed when status is clicked
     */
    const statusMenu = useMemo(() => {
        return (
            <Box>
                <Box sx={{ background: palette.primary.dark }}>
                    <IconButton edge="start" color="inherit" onClick={closeStatusMenu} aria-label="close">
                        <CloseIcon sx={{ fill: 'white', marginLeft: '0.5em' }} />
                    </IconButton>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Markdown>{statusMarkdown}</Markdown>
                </Box>
            </Box>
        )
    }, [palette.primary.dark, statusMarkdown])

    const StatusIcon = useMemo(() => STATUS_ICON[status.code], [status]);

    const title = useMemo(() => (
        <EditableLabel
            canEdit={isEditing}
            handleUpdate={handleTitleUpdate}
            placeholder={loading ? 'Loading...' : 'Enter title...'}
            renderLabel={(t) => (
                <Typography
                    component="h2"
                    variant="h5"
                    textAlign="center"
                    sx={{
                        fontSize: { xs: '1em', sm: '1.25em', md: '1.5em' },
                    }}
                >{t ?? (loading ? 'Loading...' : 'Enter title')}</Typography>
            )}
            text={getTranslation(routine, 'title', [language], false) ?? ''}
        />
    ), [handleTitleUpdate, isEditing, language, loading, routine]);

    // Open/close unlinked nodes drawer
    const [isUnlinkedNodesOpen, setIsUnlinkedNodesOpen] = useState<boolean>(false);
    const toggleUnlinkedNodes = useCallback(() => setIsUnlinkedNodesOpen(curr => !curr), []);

    /**
     * Cleans up graph by removing empty columns and row gaps within columns.
     * Also adds end nodes to the end of each unfinished path
     */
    const cleanUpGraph = useCallback(() => {
        //TODO
    }, []);

    return (
        <>
            {/* Display title first on small screens */}
            <Stack
                id="routine-title-and-language"
                direction="row"
                justifyContent="center"
                sx={{
                    zIndex: 2,
                    background: palette.mode === 'light' ? '#19487a' : '#383844',
                    color: palette.primary.contrastText,
                    height: '64px',
                    display: { xs: 'flex', lg: 'none' },
                    marginTop: { xs: '64px', md: '80px', lg: '0' },
                }}>
                {title}
            </Stack>
            <Stack
                id="build-routine-information-bar"
                direction="row"
                spacing={2}
                width="100%"
                justifyContent="space-between"
                sx={{
                    zIndex: 2,
                    height: '48px',
                    background: palette.primary.light,
                    color: palette.primary.contrastText,
                    marginTop: { xs: '0', lg: '80px' },
                }}
            >
                {/* Status indicator */}
                <Tooltip title='Press for details'>
                    <Chip
                        icon={<StatusIcon sx={{ fill: 'white' }} />}
                        label={STATUS_LABEL[status.code]}
                        onClick={openStatusMenu}
                        sx={{
                            ...noSelect,
                            background: STATUS_COLOR[status.code],
                            color: 'white',
                            cursor: isEditing ? 'pointer' : 'default',
                            marginTop: 'auto',
                            marginBottom: 'auto',
                            marginLeft: 2,
                            // Hide label on small screens
                            '& .MuiChip-label': {
                                display: { xs: 'none', sm: 'block' },
                            },
                            // Hiding label messes up spacing with icon
                            '& .MuiSvgIcon-root': {
                                marginLeft: '4px',
                                marginRight: { xs: '4px', sm: '-4px' },
                            },
                        }}
                    />
                </Tooltip>
                <Menu
                    id='status-menu'
                    open={statusMenuOpen}
                    disableScrollLock={true}
                    anchorEl={statusMenuAnchorEl}
                    onClose={closeStatusMenu}
                    anchorOrigin={{
                        vertical: 'bottom',
                        horizontal: 'center',
                    }}
                    transformOrigin={{
                        vertical: 'top',
                        horizontal: 'center',
                    }}
                    sx={{
                        '& .MuiPopover-paper': {
                            background: palette.background.paper,
                            maxWidth: 'min(100vw, 400px)',
                        },
                        '& .MuiMenu-list': {
                            padding: 0,
                        }
                    }}
                >
                    {statusMenu}
                </Menu>
                {/* Display title between icons on large screen */}
                <Stack
                    id="routine-title-and-language"
                    direction="row"
                    justifyContent="center"
                    sx={{ display: { xs: 'none', lg: 'flex' } }}>
                    {title}
                </Stack>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                    {/* Clean up graph */}
                    {isEditing && <Tooltip title='Clean up graph'>
                        <IconButton
                            id="clean-graph-button"
                            edge="end"
                            onClick={cleanUpGraph}
                            aria-label='Clean up graph'
                            sx={{
                                background: '#ab9074',
                                marginLeft: 'auto',
                                marginRight: 1,
                                transition: 'brightness 0.2s ease-in-out',
                                '&:hover': {
                                    filter: `brightness(105%)`,
                                    background: '#ab9074',
                                },
                            }}
                        >
                            <CleanUpIcon id="clean-up-button-icon" sx={{ fill: 'white' }} />
                        </IconButton>
                    </Tooltip>}
                    {/* Add new links to the routine */}
                    {isEditing && <Tooltip title='Add new link'>
                        <IconButton
                            id="add-link-button"
                            edge="end"
                            onClick={handleAddLink}
                            aria-label='Add link'
                            sx={{
                                background: '#9e3984',
                                marginRight: 1,
                                transition: 'brightness 0.2s ease-in-out',
                                '&:hover': {
                                    filter: `brightness(105%)`,
                                    background: '#9e3984',
                                },
                            }}
                        >
                            <AddLinkIcon id="add-link-button-icon" sx={{ fill: 'white' }} />
                        </IconButton>
                    </Tooltip>}
                    {/* Displays unlinked nodes */}
                    {isEditing && <UnlinkedNodesDialog
                        handleNodeDelete={handleNodeDelete}
                        handleToggleOpen={toggleUnlinkedNodes}
                        language={language}
                        nodes={nodesOffGraph}
                        open={isUnlinkedNodesOpen}
                    />}
                    {/* Language select */}
                    <SelectLanguageDialog
                        handleSelect={handleLanguageUpdate}
                        language={language}
                        availableLanguages={routine?.translations.map(t => t.language) ?? []}
                        session={session}
                        sxs={{
                            root: {
                                marginTop: 'auto',
                                marginBottom: 'auto',
                                height: 'fit-content',
                            }
                        }}
                    />
                    {/* Edit button */}
                    {canEdit && !isEditing ? (
                        <IconButton aria-label="confirm-title-change" onClick={handleStartEdit} >
                            <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                        </IconButton>
                    ) : null}
                    {/* Help button */}
                    <HelpButton markdown={helpText} sxRoot={{ margin: "auto", marginRight: 1 }} sx={{ color: TERTIARY_COLOR }} />
                    {/* Display routine description, insturctionss, etc. */}
                    <BuildInfoDialog
                        handleAction={handleRoutineAction}
                        handleUpdate={handleRoutineUpdate}
                        isEditing={isEditing}
                        language={language}
                        loading={loading}
                        routine={routine}
                        session={session}
                        sxs={{ icon: { fill: TERTIARY_COLOR, marginRight: 1 } }}
                    />
                </Box>
            </Stack>
        </>
    )
};