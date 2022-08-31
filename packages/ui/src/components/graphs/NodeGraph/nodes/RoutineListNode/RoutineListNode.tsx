import {
    Checkbox,
    Collapse,
    Container,
    FormControlLabel,
    IconButton,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import React, { CSSProperties, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { RoutineListNodeProps } from '../types';
import { DraggableNode, RoutineSubnode } from '..';
import {
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { NodeContextMenu, NodeWidth } from '../..';
import {
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
} from '../styles';
import { multiLineEllipsis, noSelect, textShadow } from 'styles';
import { NodeDataRoutineList, NodeDataRoutineListItem } from 'types';
import { getTranslation, BuildAction, updateTranslationField, PubSub, useLongPress } from 'utils';
import { EditableLabel } from 'components/inputs';
import { AddIcon, CloseIcon } from '@shared/icons';

/**
 * Distance before a click is considered a drag
 */
const DRAG_THRESHOLD = 10;

/**
 * Decides if a clicked element should trigger a collapse/expand. 
 * @param id ID of the clicked element
 */
const shouldCollapse = (id: string | null | undefined): boolean => {
    // Only collapse if clicked on shrink/expand icon, title bar, or title
    return Boolean(id && (
        id.startsWith('toggle-expand-icon-') ||
        id.startsWith('node-')
    ));
}

export const RoutineListNode = ({
    canDrag,
    canExpand,
    handleAction,
    handleUpdate,
    isLinked,
    labelVisible,
    language,
    linksIn,
    linksOut,
    isEditing,
    node,
    scale = 1,
    zIndex,
}: RoutineListNodeProps) => {
    const { palette } = useTheme();

    // When fastUpdate is triggered, context menu should never open
    const fastUpdateRef = useRef<boolean>(false);
    const fastUpdateTimeout = useRef<NodeJS.Timeout | null>(null);
    useEffect(() => {
        let fastSub = PubSub.get().subscribeFastUpdate(({ on, duration }) => {
            if (!on) {
                fastUpdateRef.current = false;
                if (fastUpdateTimeout.current) clearTimeout(fastUpdateTimeout.current);
            } else {
                fastUpdateRef.current = true;
                fastUpdateTimeout.current = setTimeout(() => {
                    fastUpdateRef.current = false;
                }, duration);
            }
        });
        return () => { PubSub.get().unsubscribe(fastSub); };
    }, []);

    // Default to open if editing and empty
    const [collapseOpen, setCollapseOpen] = useState<boolean>(isEditing && (node?.data as NodeDataRoutineList)?.routines?.length === 0);
    const handleNodeClick = useCallback((event: any) => {
        if (isLinked && (!canDrag || shouldCollapse(event.target.id))) {
            PubSub.get().publishFastUpdate({ duration: 1000 });
            setCollapseOpen(!collapseOpen);
        }
    }, [canDrag, collapseOpen, isLinked]);
    /**
     * When not dragging, DraggableNode onClick will not be triggered. So 
     * we must handle this ourselves.
     */
    const handleNodeMouseUp = useCallback((event: any) => {
        if (isLinked && !canDrag && shouldCollapse(event.target.id)) {
            PubSub.get().publishFastUpdate({ duration: 1000 });
            setCollapseOpen(!collapseOpen);
        }
    }, [canDrag, isLinked, collapseOpen]);

    const handleNodeUnlink = useCallback(() => { handleAction(BuildAction.UnlinkNode, node.id); }, [handleAction, node.id]);
    const handleNodeDelete = useCallback(() => { handleAction(BuildAction.DeleteNode, node.id); }, [handleAction, node.id]);

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate({
            ...node,
            translations: updateTranslationField(node, 'title', newLabel, language),
        });
    }, [handleUpdate, language, node]);

    const onOrderedChange = useCallback((checked: boolean) => {
        handleUpdate({
            ...node,
            data: { ...node.data, isOrdered: checked } as any,
        });
    }, [handleUpdate, node]);

    const onOptionalChange = useCallback((checked: boolean) => {
        handleUpdate({
            ...node,
            data: { ...node.data, isOptional: checked } as any,
        });
    }, [handleUpdate, node]);

    const handleSubroutineAction = useCallback((
        action: BuildAction.OpenSubroutine | BuildAction.EditSubroutine | BuildAction.DeleteSubroutine,
        subroutineId: string,
    ) => {
        handleAction(action, node.id, subroutineId);
    }, [handleAction, node.id]);

    // Opens dialog to add a new subroutine, so no suroutineId is passed
    const handleSubroutineAdd = useCallback(() => {
        handleAction(BuildAction.AddSubroutine, node.id);
    }, [handleAction, node.id]);
    const handleSubroutineUpdate = useCallback((subroutineId: string, newData: NodeDataRoutineListItem) => {
        handleUpdate({
            ...node,
            data: {
                ...node.data,
                routines: (node.data as NodeDataRoutineList)?.routines?.map((subroutine) => {
                    if (subroutine.id === subroutineId) {
                        return { ...subroutine, ...newData };
                    }
                    return subroutine;
                }) ?? [],
            },
        } as any);
    }, [handleUpdate, node]);

    const { label } = useMemo(() => {
        return {
            label: getTranslation(node, 'title', [language], true) ?? '',
        }
    }, [language, node]);

    const minNodeSize = useMemo(() => `${NodeWidth.RoutineList * scale}px`, [scale]);
    const maxNodeSize = useMemo(() => `${NodeWidth.RoutineList * scale * 3}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.RoutineList * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `max(${NodeWidth.RoutineList * scale / 8}px, 48px)`, [scale]);

    const confirmDelete = useCallback((event: any) => {
        PubSub.get().publishAlertDialog({
            message: 'What would you like to do?',
            buttons: [
                { text: 'Unlink', onClick: handleNodeUnlink },
                { text: 'Remove', onClick: handleNodeDelete },
            ]
        });
    }, [handleNodeDelete, handleNodeUnlink])

    const isLabelDialogOpen = useRef<boolean>(false);
    const onLabelDialogOpen = useCallback((isOpen: boolean) => {
        isLabelDialogOpen.current = isOpen;
    }, []);
    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canEdit={isEditing && collapseOpen}
                handleUpdate={handleLabelUpdate}
                onDialogOpen={onLabelDialogOpen}
                renderLabel={(t) => (
                    <Typography
                        id={`node-routinelist-title-${node.id}`}
                        variant="h6"
                        sx={{
                            ...noSelect,
                            ...textShadow,
                            ...(!collapseOpen ? multiLineEllipsis(1) : multiLineEllipsis(4)),
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
                text={label}
            />
        )
    }, [labelVisible, isEditing, collapseOpen, handleLabelUpdate, onLabelDialogOpen, label, node.id]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{
            ...noSelect,
            background: palette.mode === 'light' ? '#b0bbe7' : '#384164',
        }}>
            <Tooltip placement={'top'} title='Must complete routines in order'>
                <FormControlLabel
                    disabled={!isEditing}
                    label='Ordered'
                    control={
                        <Checkbox
                            id={`${label ?? ''}-ordered-option`}
                            size="small"
                            name='isOrderedCheckbox'
                            color='secondary'
                            checked={(node?.data as NodeDataRoutineList)?.isOrdered}
                            onChange={(_e, checked) => { onOrderedChange(checked) }}
                            onTouchStart={() => { onOrderedChange(!(node?.data as NodeDataRoutineList)?.isOrdered) }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
            <Tooltip placement={'top'} title='Routine can be skipped'>
                <FormControlLabel
                    disabled={!isEditing}
                    label='Optional'
                    control={
                        <Checkbox
                            id={`${label ?? ''}-optional-option`}
                            size="small"
                            name='isOptionalCheckbox'
                            color='secondary'
                            checked={(node?.data as NodeDataRoutineList)?.isOptional}
                            onChange={(_e, checked) => { onOptionalChange(checked) }}
                            onTouchStart={() => { onOptionalChange(!(node?.data as NodeDataRoutineList)?.isOptional) }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
        </Collapse>
    ), [collapseOpen, palette.mode, isEditing, label, node?.data, onOrderedChange, onOptionalChange]);

    /** 
     * Subroutines, sorted from lowest to highest index
     * */
    const routines = useMemo(() => [...((node?.data as NodeDataRoutineList)?.routines ?? [])].sort((a, b) => a.index - b.index).map(routine => (
        <RoutineSubnode
            key={`${routine.id}`}
            data={routine}
            handleAction={handleSubroutineAction}
            handleUpdate={handleSubroutineUpdate}
            isEditing={isEditing}
            isOpen={collapseOpen}
            labelVisible={labelVisible}
            language={language}
            scale={scale}
            zIndex={zIndex}
        />
    )), [node?.data, handleSubroutineAction, handleSubroutineUpdate, isEditing, collapseOpen, labelVisible, language, scale, zIndex]);

    /**
     * Border color indicates status of node.
     * Default (grey) for valid or unlinked, 
     * Yellow for missing subroutines,
     * Red for not fully connected (missing in or out links)
     */
    const borderColor = useMemo(() => {
        if (!isLinked) return 'gray';
        if (linksIn.length === 0 || linksOut.length === 0) return 'red';
        if (routines.length === 0) return 'yellow';
        return 'gray';
    }, [linksIn, isLinked, linksOut, routines]);


    const addButton = useMemo(() => isEditing ? (
        <IconButton
            onClick={handleSubroutineAdd}
            onTouchStart={handleSubroutineAdd}
            sx={{
                boxShadow: '0px 0px 12px gray',
                width: addSize,
                height: addSize,
                position: 'relative',
                padding: '0',
                margin: '5px auto',
                display: 'flex',
                alignItems: 'center',
                backgroundColor: '#6daf72',
                color: 'white',
                borderRadius: '100%',
                '&:hover': {
                    backgroundColor: '#6daf72',
                    filter: `brightness(110%)`,
                    transition: 'filter 0.2s',
                },
            }}
        >
            <AddIcon />
        </IconButton>
    ) : null, [addSize, handleSubroutineAdd, isEditing]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        // Ignore if not linked, not editing, or in the middle of an event (drag, collapse, move, etc.)
        if (!canDrag || !isLinked || !isEditing || isLabelDialogOpen.current || fastUpdateRef.current) return;
        ev.preventDefault();
        setContextAnchor(ev.currentTarget ?? ev.target)
    }, [canDrag, isEditing, isLinked, isLabelDialogOpen]);
    const closeContext = useCallback(() => setContextAnchor(null), []);
    const longPressEvent = useLongPress({ onLongPress: openContext });

    return (
        <DraggableNode
            className="handle"
            canDrag={canDrag}
            nodeId={node.id}
            onClick={handleNodeClick}
            dragThreshold={DRAG_THRESHOLD}
            sx={{
                zIndex: 5,
                minWidth: minNodeSize,
                maxWidth: collapseOpen ? maxNodeSize : minNodeSize,
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                borderRadius: '12px',
                overflow: 'hidden',
                backgroundColor: palette.background.paper,
                color: palette.background.textPrimary,
                boxShadow: `0px 0px 12px ${borderColor}`,
            }}
        >
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                availableActions={[BuildAction.AddListBeforeNode, BuildAction.AddListAfterNode, BuildAction.AddEndAfterNode, BuildAction.MoveNode, BuildAction.UnlinkNode, BuildAction.AddIncomingLink, BuildAction.AddOutgoingLink, BuildAction.DeleteNode, BuildAction.AddSubroutine]}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id) }}
                zIndex={zIndex + 1}
            />
            <Tooltip placement={'top'} title={label ?? 'Routine List'}>
                <Container
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    {...longPressEvent}
                    onMouseUp={handleNodeMouseUp}
                    sx={{
                        display: 'flex',
                        height: '48px', // Lighthouse SEO requirement
                        alignItems: 'center',
                        backgroundColor: palette.mode === 'light' ? palette.primary.dark : palette.secondary.dark,
                        color: palette.mode === 'light' ? palette.primary.contrastText : palette.secondary.contrastText,
                        paddingLeft: '0.1em!important',
                        paddingRight: '0.1em!important',
                        textAlign: 'center',
                        cursor: isEditing ? 'grab' : 'pointer',
                        '&:active': {
                            cursor: isEditing ? 'grabbing' : 'pointer',
                        },
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    {
                        canExpand && (
                            <IconButton
                                id={`toggle-expand-icon-button-${node.id}`}
                                aria-label={collapseOpen ? 'Collapse' : 'Expand'}
                                color="inherit"
                            >
                                {collapseOpen ? <ExpandLessIcon id={`toggle-expand-icon-${node.id}`} /> : <ExpandMoreIcon id={`toggle-expand-icon-${node.id}`} />}
                            </IconButton>
                        )
                    }
                    {labelObject}
                    {
                        isEditing && (
                            <IconButton
                                id={`${isLinked ? '' : 'unlinked-'}delete-node-icon-${node.id}`}
                                onClick={confirmDelete}
                                onTouchStart={confirmDelete}
                                color="inherit"
                            >
                                <CloseIcon id={`delete-node-icon-button-${node.id}`} />
                            </IconButton>
                        )
                    }
                </Container>
            </Tooltip>
            {optionsCollapse}
            <Collapse
                in={collapseOpen}
                sx={{
                    padding: collapseOpen ? '0.5em' : '0'
                }}
            >
                {routines}
                {addButton}
            </Collapse>
        </DraggableNode>
    )
}