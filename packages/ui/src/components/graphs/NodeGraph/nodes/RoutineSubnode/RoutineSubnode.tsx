import {
    Box,
    Checkbox,
    Container,
    FormControlLabel,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { CSSProperties, useMemo } from 'react';
import { RoutineSubnodeProps } from '../types';
import {
    Close as DeleteIcon,
    Edit as EditIcon,
} from '@mui/icons-material';
import {
    routineNodeCheckboxOption,
    routineNodeCheckboxLabel,
} from '../styles';
import { containerShadow, multiLineEllipsis, noSelect, textShadow } from 'styles';
import { OrchestrationDialogOption } from 'utils';

export const RoutineSubnode = ({
    nodeId,
    data,
    scale = 1,
    label = 'Routine Item',
    labelVisible = true,
    isEditable = true,
    handleDialogOpen,
}: RoutineSubnodeProps) => {
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
            } as CSSProperties}
        >
            {label}
        </Typography>
    ) : null, [labelVisible, label]);

    return (
        <Box
            borderRadius={'12px 12px 0 0'}
            sx={{
                ...containerShadow,
                minWidth: nodeSize,
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                marginBottom: '8px',
                backgroundColor: (t) => t.palette.background.paper,
                color: (t) => t.palette.background.textPrimary,
            }}
        >
            <Container
                onClick={() => handleDialogOpen(nodeId, OrchestrationDialogOption.ViewRoutineItem)}
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
                {labelObject}
                {isEditable ? <EditIcon /> : null}
                {isEditable ? <DeleteIcon /> : null}
            </Container>
            <Stack direction="row" justifyContent="space-between" borderRadius={0}>
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
                                sx={{ ...routineNodeCheckboxOption }}
                            />
                        }
                        sx={{ ...routineNodeCheckboxLabel }}
                    />
                </Tooltip>
                <Typography variant="body2">Steps: 4</Typography>
            </Stack>
        </Box>
    )
}