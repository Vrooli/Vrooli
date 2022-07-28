import {
    Box,
    Checkbox,
    Container,
    FormControlLabel,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { CSSProperties, MouseEvent, useCallback, useMemo, useState } from 'react';
import { RoutineSubnodeProps } from '../types';
import {
    Close as DeleteIcon,
} from '@mui/icons-material';
import {
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
} from '../styles';
import { containerShadow, multiLineEllipsis, noSelect, textShadow } from 'styles';
import { BuildAction, getTranslation, updateTranslationField } from 'utils';
import { owns } from 'utils/authentication';
import { EditableLabel, NodeContextMenu } from 'components';

/**
 * Decides if a clicked element should trigger opening the subroutine dialog 
 * @param id ID of the clicked element
 */
const shouldOpen = (id: string | null | undefined): boolean => {
    console.log('shouldopen', id);
    // Only collapse if clicked on title bar or title
    return Boolean(id && (id.startsWith('subroutine-title-')));
}

export const RoutineSubnode = ({
    data,
    scale = 1,
    labelVisible = true,
    isOpen,
    isEditing = true,
    handleAction,
    handleUpdate,
    language,
    zIndex,
}: RoutineSubnodeProps) => {
    const { palette } = useTheme();

    const nodeSize = useMemo(() => `${220 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${220 * scale / 5}px, 2em)`, [scale]);
    // Determines if the subroutine is one you can edit
    const canEdit = useMemo<boolean>(() => (data?.routine?.isInternal ?? owns(data?.routine?.role)), [data.routine]);

    const { title } = useMemo(() => {
        const languages = navigator.languages;
        return {
            title: getTranslation(data, 'title', languages, true) ?? getTranslation(data.routine, 'title', languages, true) ?? ''
        }
    }, [data]);

    const onAction = useCallback((event: any | null, action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine) => {
        if (event && [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine].includes(action)) {
            event.stopPropagation();
        }
        handleAction(action, data.id);
    }, [data.id, handleAction]);
    const openSubroutine = useCallback((event) => {
        if (!shouldOpen(event.target?.id)) return;
        onAction(event, BuildAction.OpenSubroutine)
    }, [onAction]);
    const deleteSubroutine = useCallback((event) => { onAction(event, BuildAction.DeleteSubroutine) }, [onAction]);

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate(data.id, {
            ...data,
            translations: updateTranslationField(data, 'title', newLabel, language),
        });
    }, [handleUpdate, data, language]);

    const onOptionalChange = useCallback((checked: boolean) => {
        handleUpdate(data.id, {
            ...data,
            isOptional: checked,
        });
    }, [handleUpdate, data]);

    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canEdit={isEditing}
                handleUpdate={handleLabelUpdate}
                renderLabel={(t) => (
                    <Typography
                        id={`subroutine-title-${data.id}`}
                        variant="h6"
                        sx={{
                            ...noSelect,
                            ...textShadow,
                            ...multiLineEllipsis(1),
                            textAlign: 'center',
                            width: '100%',
                            lineBreak: 'anywhere' as any,
                            whiteSpace: 'pre' as any,
                        } as CSSProperties}
                    >{t ?? 'Untitled'}</Typography>
                )}
                sxs={{
                    stack: {
                        marginLeft: 'auto',
                        marginRight: 'auto',
                    }
                }}
                text={title}
            />
        )
    }, [labelVisible, isEditing, handleLabelUpdate, title, data.id]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `subroutine-context-menu-${data.id}`, [data?.id]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        // Ignore if not editing
        if (!isEditing) return;
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, [isEditing]);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={
                    isEditing ? 
                    [BuildAction.EditSubroutine, BuildAction.DeleteSubroutine] :
                    [BuildAction.OpenSubroutine, BuildAction.DeleteSubroutine]
                }
                handleClose={closeContext}
                handleSelect={(action) => { onAction(null, action) }}
                zIndex={zIndex + 1}
            />
            <Box
                sx={{
                    ...containerShadow,
                    minWidth: nodeSize,
                    fontSize: fontSize,
                    position: 'relative',
                    display: 'block',
                    marginBottom: '8px',
                    borderRadius: '12px',
                    overflow: 'hidden',
                    backgroundColor: palette.background.paper,
                    color: palette.background.textPrimary,
                }}
            >
                <Container
                    id={`subroutine-title-bar-${data.id}`}
                    onContextMenu={openContext}
                    aria-owns={contextOpen ? contextId : undefined}
                    onClick={openSubroutine}
                    sx={{
                        display: 'flex',
                        alignItems: 'center',
                        backgroundColor: canEdit ?
                            (palette.mode === 'light' ? palette.primary.dark : palette.secondary.dark) :
                            '#667899',
                        color: palette.mode === 'light' ? palette.primary.contrastText : palette.secondary.contrastText,
                        padding: '0.1em',
                        textAlign: 'center',
                        cursor: 'pointer',
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    {labelObject}
                    {isEditing ? <DeleteIcon
                        id={`subroutine-delete-icon-${data.id}`}
                        onClick={deleteSubroutine}
                        onTouchStart={deleteSubroutine}
                    /> : null}
                </Container>
                <Stack direction="row" justifyContent="space-between" borderRadius={0} sx={{ ...noSelect }}>
                    <Tooltip placement={'top'} title='Routine can be skipped'>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Optional'
                            control={
                                <Checkbox
                                    id={`${title ?? ''}-optional-option`}
                                    size="small"
                                    name='isOptionalCheckbox'
                                    color='secondary'
                                    checked={data?.isOptional}
                                    onChange={(_e, checked) => { onOptionalChange(checked) }}
                                    onTouchStart={() => { onOptionalChange(!data?.isOptional) }}
                                    sx={{ ...routineNodeCheckboxOption }}
                                />
                            }
                            sx={{ ...routineNodeCheckboxLabel }}
                        />
                    </Tooltip>
                    {/* <Typography variant="body2">Steps: 4</Typography> */}
                </Stack>
            </Box>
        </>
    )
}