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
import { useMemo, useState } from 'react';
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
    scale = 1,
    label = 'Routine List',
    labelVisible = true,
    isEditable = true,
    data,
    onAdd = () => { },
}: RoutineListNodeProps) => {
    const classes = useStyles();
    const [collapseOpen, setCollapseOpen] = useState(false);
    const toggleCollapse = () => setCollapseOpen(curr => !curr);

    const nodeSize = useMemo(() => `${250 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${250 * scale / 5}px, 2em)`, [scale]);
    const addSize = useMemo(() => `${250 * scale / 6}px`, [scale]);

    const addRoutine = () => {
        console.log('ADD ROUTINE CALLED')
    };

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.headerLabel} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.headerLabel, label]);

    const optionsCollapse = useMemo(() => (
        <Collapse className={classes.listOptions} in={collapseOpen}>
            <Tooltip placement={'top'} title='Must complete routines in order'>
                <FormControlLabel
                    className={classes.checkboxLabel}
                    disabled={!isEditable}
                    control={
                        <Checkbox
                            id={`${data?.title}-ordered-option`}
                            className={classes.routineOptionCheckbox}
                            size="small"
                            name='isOrderedCheckbox'
                            value='isOrderedCheckbox'
                            color='secondary'
                            checked={data?.isOrdered}
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
                            id={`${data?.title}-optional-option`}
                            className={classes.routineOptionCheckbox}
                            size="small"
                            name='isOptionalCheckbox'
                            value='isOptionalCheckbox'
                            color='secondary'
                            checked={data?.isOptional}
                            onChange={() => {}}
                        />
                    }
                    label='Optional'
                />
            </Tooltip>
        </Collapse>
    ), [classes.checkboxLabel, classes.listOptions, classes.routineOptionCheckbox, collapseOpen, data?.isOptional, data?.isOrdered, data?.title, isEditable]);

    const routines = useMemo(() => data?.routines?.map(routine => (
        <RoutineSubnode
            key={`${routine.id}`}
            scale={scale}
            labelVisible={labelVisible}
            data={routine}
        />
    )), [data, labelVisible, scale]);

    const addButton = useMemo(() => isEditable ? (
        <IconButton className={classes.addButton} style={{ width: addSize, height: addSize }} onClick={addRoutine}>
            <AddIcon className={classes.icon} />
        </IconButton>
    ) : null, [addSize, classes.addButton, classes.icon, isEditable]);

    return (
        <div className={classes.root} style={{ width: nodeSize, maxHeight: nodeSize, fontSize: fontSize }}>
            <Container className={classes.header} onClick={toggleCollapse}>
                {collapseOpen ? <ExpandLessIcon /> : <ExpandMoreIcon />}
                {labelObject}
                {isEditable ? <DeleteIcon /> : null}
            </Container>
            {optionsCollapse}
            <Collapse in={collapseOpen}>
                {routines}
                {addButton}
            </Collapse>
        </div>
    )
}