import {
    Box,
    Checkbox,
    Collapse,
    Container,
    FormControlLabel,
    Theme,
    Tooltip,
    Typography
} from '@mui/material';
import { useMemo, useState } from 'react';
import { RoutineSubnodeProps } from '../types';
import {
    Close as DeleteIcon,
    Edit as EditIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { 
    multiLineEllipsis,
    noSelect,
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
    routineNodeListOptions,
    containerShadow,
    textShadow,
} from 'styles';

export const RoutineSubnode = ({
    data,
    scale = 1,
    label = 'Routine Item',
    labelVisible = true,
    isEditable = true,
}: RoutineSubnodeProps) => {
    const [collapseOpen, setCollapseOpen] = useState(false);
    const toggleCollapse = () => setCollapseOpen(curr => !curr);

    const nodeSize = useMemo(() => `${220 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${220 * scale / 5}px, 2em)`, [scale]);

    const labelObject = useMemo(() => labelVisible ? (
        <Typography
            variant="h6"
            sx={{
                ...noSelect,
                ...textShadow,
                ...multiLineEllipsis(1),
                textAlign: 'center',
                width: '100%',
                lineBreak: 'anywhere',
                whiteSpace: 'pre',
            }}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label]);

    const optionsCollapse = useMemo(() => (
        <Collapse in={collapseOpen} sx={{...routineNodeListOptions}}>
            <Tooltip placement={'top'} title='Routine can be skipped'>
                <FormControlLabel
                    disabled={!isEditable}
                    label='Optional'
                    control={
                        <Checkbox
                            id={`${data?.title}-optional-option`}
                            size="small"
                            name='isOptionalCheckbox'
                            value='isOptionalCheckbox'
                            color='secondary'
                            checked={data?.isOptional}
                            onChange={() => { }}
                            sx={{...routineNodeCheckboxOption}}
                        />
                    }
                    sx={{...routineNodeCheckboxLabel}}
                />
            </Tooltip>
        </Collapse>
    ), [collapseOpen, data?.isOptional, data?.title, isEditable]);

    return (
        <Box 
            sx={{ 
                ...containerShadow,
                minWidth: nodeSize, 
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                borderRadius: '12px',
                marginBottom: '8px',
                backgroundColor: (t) => t.palette.background.paper,
                color: (t) => t.palette.background.textPrimary,
            }}
        >
            <Container 
                onClick={toggleCollapse}
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    borderRadius: '12px 12px 0 0',
                    backgroundColor: (t) => t.palette.primary.main,
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
                {isEditable ? <EditIcon /> : null}
                {isEditable ? <DeleteIcon /> : null}
            </Container>
            {optionsCollapse}
        </Box>
    )
}