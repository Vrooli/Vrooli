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
import { ChangeEvent, CSSProperties, MouseEvent, useCallback, useMemo, useState } from 'react';
import { RoutineListNodeProps } from '../types';
import { DraggableNode, RoutineSubnode } from '..';
import {
    Add as AddIcon,
    Close as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { NodeContextMenu, NodeWidth } from '../..';
import {
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
} from '../styles';
import { containerShadow, multiLineEllipsis, noSelect, textShadow } from 'styles';
import { NodeDataRoutineList, NodeDataRoutineListItem } from 'types';
import { getTranslation, BuildAction, Pubs, updateTranslationField } from 'utils';
import { EditableLabel } from 'components/inputs';

/**
 * Distance before a click is considered a drag
 */
const DRAG_THRESHOLD = 10;

/**
 * Decides if a clicked element should trigger a collapse/expand. 
 * @param id ID of the clicked element
 */
const shouldCollapse = (id: string | null | undefined): boolean => {
    // Don't collapse if clicking on the edit or delete buttons, or if unlinked
    if (id && 
        (id.startsWith('unlinked') ||
        id.startsWith('edit-label-icon') || 
        id.startsWith('node-delete'))) return false;
    // In any other case, collapse
    return true;
}

export const RoutineListNode = ({
    canDrag,
    canExpand,
    handleAction,
    handleUpdate,
    isLinked,
    labelVisible,
    language,
    isEditing,
    node,
    scale = 1,
}: RoutineListNodeProps) => {
    const { palette } = useTheme();
    const [collapseOpen, setCollapseOpen] = useState<boolean>(false);
    const handleNodeClick = useCallback((event: any) => {
        console.log('NODE CLICKED NOT DRAGGED!!!!!!!', event.target.id);
        if (!canDrag || shouldCollapse(event.target.id)) setCollapseOpen(!collapseOpen);
    }, [canDrag, collapseOpen]);
    /**
     * When not dragging, DraggableNode onClick will not be triggered. So 
     * we must handle this ourselves.
     */
    const handleNodeMouseUp = useCallback((event: any) => {
        console.log('MOUSE UP', !canDrag, event.target.id, shouldCollapse(event.target.id));
        if (!canDrag && shouldCollapse(event.target.id)) {
            setCollapseOpen(!collapseOpen);
        }
    }, [collapseOpen, canDrag]);

    const handleNodeUnlink = useCallback(() => { handleAction(BuildAction.UnlinkNode, node.id); }, [handleAction, node.id]);
    const handleNodeDelete = useCallback(() => { handleAction(BuildAction.DeleteNode, node.id); }, [handleAction, node.id]);

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate({
            ...node,
            translations: updateTranslationField(node, 'title', newLabel, language) as any[],
        });
    }, [handleUpdate, language, node]);

    const onOrderedChange = useCallback((e: ChangeEvent<HTMLInputElement>, checked: boolean) => {
        handleUpdate({
            ...node,
            data: { ...node.data, isOrdered: checked } as any,
        });
    }, [handleUpdate, node]);

    const onOptionalChange = useCallback((e: ChangeEvent<HTMLInputElement>, checked: boolean) => {
        handleUpdate({
            ...node,
            data: { ...node.data, isOptional: checked } as any,
        });
    }, [handleUpdate, node]);

    const handleSubroutineOpen = useCallback((subroutineId: string) => {
        handleAction(BuildAction.OpenSubroutine, node.id, subroutineId);
    }, [handleAction, node.id]);
    // Opens dialog to add a new subroutine, so no suroutineId is passed
    const handleSubroutineAdd = useCallback(() => {
        handleAction(BuildAction.AddSubroutine, node.id);
    }, [handleAction, node.id]);
    const handleSubroutineEdit = useCallback((subroutineId: string) => {
        handleAction(BuildAction.EditSubroutine, node.id, subroutineId);
    }, [handleAction, node.id]);
    const handleSubroutineDelete = useCallback((subroutineId: string) => {
        handleAction(BuildAction.DeleteSubroutine, node.id, subroutineId);
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

    const nodeSize = useMemo(() => `${NodeWidth.RoutineList * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.RoutineList * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `${NodeWidth.RoutineList * scale / 8}px`, [scale]);

    const confirmDelete = useCallback((event: any) => {
        console.log('DELETE CONFIRMED', event);
        PubSub.publish(Pubs.AlertDialog, {
            message: 'What would you like to do?',
            buttons: [
                { text: 'Unlink', onClick: handleNodeUnlink },
                { text: 'Remove', onClick: handleNodeDelete },
            ]
        });
    }, [handleNodeDelete, handleNodeUnlink])

    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canEdit={isEditing && collapseOpen}
                handleUpdate={handleLabelUpdate}
                renderLabel={(t) => (
                    <Typography
                        id={`${isLinked ? '' : 'unlinked-'}node-routinelist-title-${node.id}`}
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
                text={label}
            />
        )
    }, [collapseOpen, label, labelVisible, isEditing, node.id, handleLabelUpdate, isLinked]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{
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
                            onChange={onOrderedChange}
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
                            onChange={onOptionalChange}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
        </Collapse>
    ), [collapseOpen, palette.mode, isEditing, label, node?.data, onOrderedChange, onOptionalChange]);

    const routines = useMemo(() => (node?.data as NodeDataRoutineList)?.routines?.map(routine => (
        <RoutineSubnode
            key={`${routine.id}`}
            data={routine}
            handleOpen={handleSubroutineOpen}
            handleEdit={handleSubroutineEdit}
            handleDelete={handleSubroutineDelete}
            handleUpdate={handleSubroutineUpdate}
            isEditing={isEditing}
            isOpen={collapseOpen}
            labelVisible={labelVisible}
            language={language}
            scale={scale}
        />
    )), [node?.data, handleSubroutineOpen, handleSubroutineEdit, handleSubroutineDelete, handleSubroutineUpdate, isEditing, collapseOpen, labelVisible, language, scale]);

    const addButton = useMemo(() => isEditing ? (
        <IconButton
            onClick={handleSubroutineAdd}
            sx={{
                ...containerShadow,
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
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        // Ignore if not linked or editing
        if (!canDrag || !isLinked) return;
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, [canDrag, isLinked]);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <DraggableNode
            className="handle"
            canDrag={canDrag}
            nodeId={node.id}
            onClick={handleNodeClick}
            dragThreshold={DRAG_THRESHOLD}
            sx={{
                zIndex: 5,
                width: nodeSize,
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                borderRadius: '12px',
                overflow: 'hidden',
                backgroundColor: palette.background.paper,
                color: palette.background.textPrimary,
                boxShadow: '0px 0px 12px gray',
            }}
        >
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                handleClose={closeContext}
                handleSelect={(option) => { handleAction(option, node.id) }}
            />
            <Tooltip placement={'top'} title={label ?? 'Routine List'}>
                <Container
                    id={`${isLinked ? '' : 'unlinked-'}node-${node.id}`}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    onMouseUp={handleNodeMouseUp}
                    sx={{
                        display: 'flex',
                        height: '48px', // Lighthouse SEO requirement
                        alignItems: 'center',
                        backgroundColor: palette.mode === 'light' ? palette.primary.dark : palette.secondary.dark,
                        color: palette.mode === 'light' ? palette.primary.contrastText : palette.secondary.contrastText,
                        padding: '0.1em',
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
                                color="inherit"
                            >
                                {collapseOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                            </IconButton>
                        )
                    }
                    {labelObject}
                    {
                        isEditing && (
                            <IconButton
                                id={`${isLinked ? '' : 'unlinked-'}node-delete-icon-${node.id}`}
                                onClick={confirmDelete}
                                color="inherit"
                            >
                                <DeleteIcon id={`node-delete-icon-button-${node.id}`}/>
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