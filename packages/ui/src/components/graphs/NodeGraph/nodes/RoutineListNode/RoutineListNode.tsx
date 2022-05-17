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
import { ChangeEvent, CSSProperties, MouseEvent, useCallback, useMemo, useRef, useState } from 'react';
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

    // Stores position of click/touch start, to cancel click event if drag occurs
    const clickStartPosition = useRef<{ x: number, y: number }>({ x: 0, y: 0 });
    // Stores if touch event was a drag
    const touchIsDrag = useRef(false);
    const [collapseOpen, setCollapseOpen] = useState<boolean>(false);

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
            label: getTranslation(node, 'title', [language], true),
        }
    }, [language, node]);

    const handleTouchMove = useCallback((e: any) => {
        if (!canExpand) return;
        touchIsDrag.current = true;
    }, [canExpand]);
    const handleTouchEnd = useCallback((e: any) => {
        if (!canExpand) return;
        if (!touchIsDrag.current) {
            setCollapseOpen(curr => !curr)
        }
        touchIsDrag.current = false;
    }, [canExpand]);
    const handleMouseDown = useCallback((e: any) => {
        if (!canExpand) return;
        clickStartPosition.current = { x: e.pageX, y: e.pageY };
    }, [canExpand]);
    const handleMouseUp = useCallback((e: any) => {
        if (!canExpand) return;
        const { x, y } = clickStartPosition.current;
        const { pageX, pageY } = e;
        if (Math.abs(pageX - x) < 5 && Math.abs(pageY - y) < 5) {
            setCollapseOpen(curr => !curr)
        }
    }, [canExpand]);

    const nodeSize = useMemo(() => `${NodeWidth.RoutineList * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.RoutineList * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `${NodeWidth.RoutineList * scale / 8}px`, [scale]);

    const confirmDelete = useCallback((event: any) => {
        event.preventDefault();
        PubSub.publish(Pubs.AlertDialog, {
            message: 'What would you like to do?',
            buttons: [
                { text: 'Unlink', onClick: handleNodeUnlink },
                { text: 'Remove', onClick: handleNodeDelete },
                { text: 'Cancel' }
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
        <DraggableNode className="handle" canDrag={canDrag} nodeId={node.id}
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
                    onMouseDown={handleMouseDown}
                    onMouseUp={handleMouseUp}
                    onTouchMove={handleTouchMove}
                    onTouchEnd={handleTouchEnd}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
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
                    {canExpand ?
                        collapseOpen ?
                            <ExpandLessIcon id={`${isLinked ? '' : 'unlinked-'}node-routinelist-shrink-icon-${node.id}`} /> :
                            <ExpandMoreIcon id={`${isLinked ? '' : 'unlinked-'}node-routinelist-expand-icon-${node.id}`} />
                        : null}
                    {labelObject}
                    {isEditing ? <DeleteIcon id={`${isLinked ? '' : 'unlinked-'}node-routinelist-delete-icon-${node.id}`} onClick={confirmDelete} /> : null}
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