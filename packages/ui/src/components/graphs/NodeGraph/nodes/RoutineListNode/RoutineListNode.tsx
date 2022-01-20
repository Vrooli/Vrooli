import {
    Box,
    Checkbox,
    Collapse,
    Container,
    FormControlLabel,
    IconButton,
    Tooltip,
    Typography
} from '@mui/material';
import { CSSProperties, MouseEvent, useCallback, useEffect, useMemo, useState } from 'react';
import { RoutineListNodeProps } from '../types';
import { RoutineSubnode } from '..';
import {
    Add as AddIcon,
    Close as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { RoutineListNodeData } from '@local/shared';
import { NodeContextMenu, NodeWidth } from '../..';
import {
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
    routineNodeListOptions,
} from '../styles';
import { containerShadow, multiLineEllipsis, noSelect, textShadow } from 'styles';

export const RoutineListNode = ({
    node,
    scale = 1,
    label = 'Routine List',
    labelVisible = true,
    isEditable = true,
    onAdd = () => { },
    dragIsOver,
}: RoutineListNodeProps) => {
    const [collapseOpen, setCollapseOpen] = useState(false);
    const toggleCollapse = () => setCollapseOpen(curr => !curr);

    const nodeSize = useMemo(() => `${NodeWidth.RoutineList * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${NodeWidth.RoutineList * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `${NodeWidth.RoutineList * scale / 8}px`, [scale]);

    /**
     * Pass up size of node to parent
     */
    useEffect(() => {

    }, [collapseOpen])

    const addRoutine = () => {
        console.log('ADD ROUTINE CALLED')
    };

    const labelObject = useMemo(() => labelVisible ? (
        <Typography
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
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{ ...routineNodeListOptions }}>
            <Tooltip placement={'top'} title='Must complete routines in order'>
                <FormControlLabel
                    disabled={!isEditable}
                    label='Ordered'
                    control={
                        <Checkbox
                            id={`${label ?? ''}-ordered-option`}
                            size="small"
                            name='isOrderedCheckbox'
                            value='isOrderedCheckbox'
                            color='secondary'
                            checked={(node?.data as RoutineListNodeData)?.isOrdered}
                            onChange={() => { }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
            <Tooltip placement={'top'} title='Routine can be skipped'>
                <FormControlLabel
                    disabled={!isEditable}
                    label='Optional'
                    control={
                        <Checkbox
                            id={`${label ?? ''}-optional-option`}
                            size="small"
                            name='isOptionalCheckbox'
                            value='isOptionalCheckbox'
                            color='secondary'
                            checked={(node?.data as RoutineListNodeData)?.isOptional}
                            onChange={() => { }}
                            sx={{ ...routineNodeCheckboxOption }}
                        />
                    }
                    sx={{ ...routineNodeCheckboxLabel }}
                />
            </Tooltip>
        </Collapse>
    ), [collapseOpen, node?.data, isEditable, label]);

    const routines = useMemo(() => (node?.data as RoutineListNodeData)?.routines?.map(routine => (
        <RoutineSubnode
            key={`${routine.id}`}
            scale={scale}
            labelVisible={labelVisible}
            data={routine}
        />
    )), [node?.data, labelVisible, scale]);

    const addButton = useMemo(() => isEditable ? (
        <IconButton
            onClick={addRoutine}
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
    ) : null, [addSize, isEditable]);

    // Right click context menu
    const [contextAnchor, setContextAnchor] = useState<any>(null);
    const contextId = useMemo(() => `node-context-menu-${node.id}`, [node]);
    const contextOpen = Boolean(contextAnchor);
    const openContext = useCallback((ev: MouseEvent<HTMLDivElement>) => {
        setContextAnchor(ev.currentTarget)
        ev.preventDefault();
    }, []);
    const closeContext = useCallback(() => setContextAnchor(null), []);

    return (
        <Box className="handle"
            sx={{
                width: nodeSize,
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                borderRadius: '12px',
                backgroundColor: (t) => t.palette.background.paper,
                color: (t) => t.palette.background.textPrimary,
                boxShadow: '0px 0px 12px gray',
                opacity: dragIsOver ? 0.5 : 1,
            }}
        >
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                node={node}
                onClose={closeContext}
                onAddBefore={() => { }}
                onAddAfter={() => { }}
                onDelete={() => { }}
                onEdit={() => { }}
                onMove={() => { }}
            />
            <Tooltip placement={'top'} title={label ?? 'Routine List'}>
                <Container
                    onClick={toggleCollapse}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                    sx={{
                        display: 'flex',
                        alignItems: 'center',
                        borderRadius: '12px 12px 0 0',
                        backgroundColor: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        padding: '0.1em',
                        textAlign: 'center',
                        cursor: 'pointer',
                        '&:hover': {
                            filter: `brightness(120%)`,
                            transition: 'filter 0.2s',
                        },
                    }}
                >
                    {collapseOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    {labelObject}
                    {isEditable ? <DeleteIcon /> : null}
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
        </Box>
    )
}