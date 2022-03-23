import {
    Box,
    Checkbox,
    Container,
    FormControlLabel,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { ChangeEvent, CSSProperties, useCallback, useMemo } from 'react';
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
import { getTranslation, updateTranslationField } from 'utils';
import { MemberRole } from 'graphql/generated/globalTypes';
import { EditableLabel } from 'components/inputs';

export const RoutineSubnode = ({
    data,
    scale = 1,
    labelVisible = true,
    isOpen,
    isEditing = true,
    handleOpen,
    handleEdit,
    handleDelete,
    handleUpdate,
}: RoutineSubnodeProps) => {
    console.log('ROUTINE SUBNODE RENDER', data, data.id);
    const nodeSize = useMemo(() => `${220 * scale}px`, [scale]);
    const fontSize = useMemo(() => `min(${220 * scale / 5}px, 2em)`, [scale]);
    // Determines if the subroutine is one you can edit
    const canEdit = useMemo(() => data.routine.role && [MemberRole.Owner, MemberRole.Admin].includes(data.routine.role), [data.routine.role]);

    const { title } = useMemo(() => {
        const languages = navigator.languages;
        return {
            title: getTranslation(data, 'title', languages, true) ?? getTranslation(data.routine, 'title', languages, true)
        }
    }, [data]);

    const openSubnode = useMemo(() => () => handleOpen(data.id), [data.id, handleOpen]);
    const editSubnode = useMemo(() => (e) => { e.stopPropagation(); handleEdit(data.id) }, [data.id, handleEdit]);
    const deleteSubnode = useMemo(() => (e) => { e.stopPropagation(); handleDelete(data.id) }, [data.id, handleDelete]);

    const handleLabelUpdate = useCallback((newLabel: string) => {
        handleUpdate(data.id, {
            ...data,
            translations: updateTranslationField(data, 'title', newLabel, 'en') as any[],
        });
    }, [handleUpdate, data]);

    const onOptionalChange = useCallback((e: ChangeEvent<HTMLInputElement>, checked: boolean) => {
        handleUpdate(data.id, {
            ...data,
            isOptional: checked,
        });
    }, [handleUpdate, data]);

    const labelObject = useMemo(() => {
        if (!labelVisible) return null;
        return (
            <EditableLabel
                canEdit={isEditing && isOpen}
                handleUpdate={handleLabelUpdate}
                renderLabel={(t) => (
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
    }, [handleLabelUpdate, isEditing, isOpen, labelVisible, title]);

    return (
        <Box
            sx={{
                ...containerShadow,
                minWidth: nodeSize,
                fontSize: fontSize,
                position: 'relative',
                display: 'block',
                marginBottom: '8px',
                borderRadius: '12px',
                overflow: 'overlay',
                backgroundColor: (t) => t.palette.background.paper,
                color: (t) => t.palette.background.textPrimary,
            }}
        >
            <Container
                onClick={openSubnode}
                sx={{
                    display: 'flex',
                    alignItems: 'center',
                    backgroundColor: canEdit ? (t) => t.palette.primary.main : '#667899',
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
                {/* {isEditing && canEdit ? <EditIcon onClick={editSubnode} /> : null} */}
                {isEditing ? <DeleteIcon onClick={deleteSubnode} /> : null}
            </Container>
            <Stack direction="row" justifyContent="space-between" borderRadius={0}>
                <Tooltip placement={'top'} title='Routine can be skipped'>
                    <FormControlLabel
                        disabled={!isEditing}
                        label='Optional'
                        control={
                            <Checkbox
                                id={`${title ?? ''}-optional-option`}
                                size="small"
                                name='isOptionalCheckbox'
                                value='isOptionalCheckbox'
                                color='secondary'
                                checked={data?.isOptional}
                                onChange={onOptionalChange}
                                sx={{ ...routineNodeCheckboxOption }}
                            />
                        }
                        sx={{ ...routineNodeCheckboxLabel }}
                    />
                </Tooltip>
                {/* <Typography variant="body2">Steps: 4</Typography> */}
            </Stack>
        </Box>
    )
}