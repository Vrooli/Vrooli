import { makeStyles } from '@mui/styles';
import { Collapse, Container, Theme, Tooltip, Typography } from '@mui/material';
import { useMemo, useState } from 'react';
import { RoutineSubnodeProps } from '../types';
import { nodeStyles } from '../styles';
import { combineStyles } from 'utils';

const componentStyles = (theme: Theme) => ({
    root: {
        position: 'relative',
        display: 'block',
        borderRadius: '12px',
        backgroundColor: theme.palette.background.paper,
        color: theme.palette.background.textPrimary,
        boxShadow: '0px 0px 12px gray',
        '&:hover': {
            filter: `brightness(120%)`,
            transition: 'filter 0.2s',
        },
    },
    header: {
        borderRadius: '12px 12px 0 0',
        backgroundColor: theme.palette.primary.main,
        color: theme.palette.primary.contrastText,
        padding: '0.1em',
        textAlign: 'center'
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
    }
});

const useStyles = makeStyles(combineStyles(nodeStyles, componentStyles));

export const RoutineSubnode = ({
    scale = 1,
    label = 'Subroutine Item',
    labelVisible = true,
    data,
}: RoutineSubnodeProps) => {
    const classes = useStyles();
    const [collapseOpen, setCollapseOpen] = useState(false);
    const toggleCollapse = () => setCollapseOpen(curr => !curr);

    const labelObject = useMemo(() => labelVisible ? (
        <Typography className={classes.headerLabel} variant="h6">{label}</Typography>
    ) : null, [labelVisible, classes.headerLabel, label]);

    const nodeSize = useMemo(() => `${200 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${200 * scale / 5}px, 2em)`, [scale]);

    return (
        <div className={classes.root} style={{ width: nodeSize, height: nodeSize, fontSize: fontSize }}>
            <Container className={classes.header} onClick={toggleCollapse}>
                {labelObject}
            </Container>
            <Collapse style={{ height: '20%' }} in={collapseOpen}>

            </Collapse>
        </div>
    )
}