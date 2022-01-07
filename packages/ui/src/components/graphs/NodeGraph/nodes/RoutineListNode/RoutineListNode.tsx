import { makeStyles } from '@mui/styles';
import {
    Checkbox,
    Collapse,
    Container,
    FormControlLabel,
    IconButton,
    Theme,
    Tooltip,
    Typography
} from '@mui/material';
import { MouseEvent, useCallback, useMemo, useState } from 'react';
import { RoutineListNodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';
import { RoutineSubnode } from '..';
import {
    Add as AddIcon,
    Close as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { RoutineListNodeData } from '@local/shared';
import { NodeContextMenu } from '../..';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        borderRadius: '12px',
        backgroundColor: theme.palette.background.paper,
        color: theme.palette.background.textPrimary,
        boxShadow: '0px 0px 12px gray',
    },
    header: {
        display: 'flex',
        alignItems: 'center',
        borderRadius: '12px 12px 0 0',
        backgroundColor: theme.palette.primary.dark,
        color: theme.palette.primary.contrastText,
        padding: '0.1em',
        textAlign: 'center',
        cursor: 'pointer',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
    headerLabel: {
        textAlign: 'center',
        width: '100%',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        lineBreak: 'anywhere',
        whiteSpace: 'pre',
        textShadow:
            `-0.5px -0.5px 0 black,  
            0.5px -0.5px 0 black,
            -0.5px 0.5px 0 black,
            0.5px 0.5px 0 black`
    },
    listOptions: {
        background: '#b0bbe7',
    },
    checkboxLabel: {
        marginLeft: '0'
    },
    routineOptionCheckbox: {
        padding: '4px',
    },
    addButton: {
        position: 'relative',
        padding: '0',
        margin: '5px auto',
        display: 'flex',
        alignItems: 'center',
        backgroundColor: '#6daf72',
        color: 'white',
        borderRadius: '100%',
        boxShadow: '0px 0px 12px gray',
        '&:hover': {
            backgroundColor: '#6daf72',
            filter: `brightness(110%)`,
            transition: 'filter 0.2s',
        },
    },
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const RoutineListNode = ({
    node,
    scale = 1,
    label = 'Routine List',
    labelVisible = true,
    isEditable = true,
    onAdd = () => { },
}: RoutineListNodeProps) => {
    const classes = useStyles();
    const [collapseOpen, setCollapseOpen] = useState(false);
    const toggleCollapse = () => setCollapseOpen(curr => !curr);

    const nodeSize = useMemo(() => `${350 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${350 * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `${350 * scale / 8}px`, [scale]);

    const addRoutine = () => {
        console.log('ADD ROUTINE CALLED')
    };

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={`${classes.headerLabel} ${classes.noSelect}`} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.headerLabel, classes.noSelect, label]);

    const optionsCollapse = useMemo(() => (
        <Collapse className={classes.listOptions} in={collapseOpen}>
            <Tooltip placement={'top'} title='Must complete routines in order'>
                <FormControlLabel
                    className={classes.checkboxLabel}
                    disabled={!isEditable}
                    control={
                        <Checkbox
                            id={`${label ?? ''}-ordered-option`}
                            className={classes.routineOptionCheckbox}
                            size="small"
                            name='isOrderedCheckbox'
                            value='isOrderedCheckbox'
                            color='secondary'
                            checked={(node?.data as RoutineListNodeData)?.isOrdered}
                            onChange={() => {}}
                        />
                    }
                    label='Ordered'
                />
            </Tooltip>
            <Tooltip placement={'top'} title='Routine can be skipped'>
                <FormControlLabel
                    className={classes.checkboxLabel}
                    disabled={!isEditable}
                    control={
                        <Checkbox
                            id={`${label ?? ''}-optional-option`}
                            className={classes.routineOptionCheckbox}
                            size="small"
                            name='isOptionalCheckbox'
                            value='isOptionalCheckbox'
                            color='secondary'
                            checked={(node?.data as RoutineListNodeData)?.isOptional}
                            onChange={() => {}}
                        />
                    }
                    label='Optional'
                />
            </Tooltip>
        </Collapse>
    ), [classes.checkboxLabel, classes.listOptions, classes.routineOptionCheckbox, collapseOpen, node?.data, isEditable, label]);

    const routines = useMemo(() => (node?.data as RoutineListNodeData)?.routines?.map(routine => (
        <RoutineSubnode
            key={`${routine.id}`}
            scale={scale}
            labelVisible={labelVisible}
            data={routine}
        />
    )), [node?.data, labelVisible, scale]);

    const addButton = useMemo(() => isEditable ? (
        <IconButton className={classes.addButton} style={{ width: addSize, height: addSize }} onClick={addRoutine}>
            <AddIcon className={classes.icon} />
        </IconButton>
    ) : null, [addSize, classes.addButton, classes.icon, isEditable]);

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
        <div className={classes.root} style={{ width: nodeSize, fontSize: fontSize }}>
            <NodeContextMenu
                id={contextId}
                anchorEl={contextAnchor}
                open={contextOpen}
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
                    className={classes.header} 
                    onClick={toggleCollapse}
                    aria-owns={contextOpen ? contextId : undefined}
                    onContextMenu={openContext}
                >
                    {collapseOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                    {labelObject}
                    {isEditable ? <DeleteIcon /> : null}
                </Container>
            </Tooltip>
            {optionsCollapse}
            <Collapse className={classes.collapse} style={{padding: collapseOpen ? '0.5em' : '0'}} in={collapseOpen}>
                {routines}
                {addButton}
            </Collapse>
        </div>
    )
}