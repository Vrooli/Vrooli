/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    AccountTree as GraphIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import {
    Box,
    Button,
    Grid,
    IconButton,
    Stack,
    SwipeableDrawer,
    Typography,
    useTheme,
} from '@mui/material';
import { useLocation } from 'wouter';
import { SubroutineInfoDialogProps } from '../types';
import { getOwnedByString, getTranslation, toOwnedBy } from 'utils';
import Markdown from 'markdown-to-jsx';
import { routineUpdateForm as validationSchema } from '@local/shared';
import { InputOutputContainer, LinkButton, MarkdownInput, QuantityBox } from 'components';
import { useFormik } from 'formik';
import { RoutineInputList, RoutineOutputList } from 'types';
import { owns } from 'utils/authentication';
import { routine_routine_nodes_data_NodeRoutineList_routines } from 'graphql/generated/routine';

export const SubroutineInfoDialog = ({
    data,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    open,
    language,
    session,
    onClose,
}: SubroutineInfoDialogProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    console.log('SUBROUTINE INFO DIALOG', data)

    const subroutine = useMemo<routine_routine_nodes_data_NodeRoutineList_routines | undefined>(() => {
        if (!data?.node || !data?.routineId) return undefined;
        return data.node.routines.find(r => r.id === data.routineId);
    }, [data]);
    console.log("SUBROUTINE DATAAAAAA IN INFO DIALOG", subroutine)

    const ownedBy = useMemo<string | null>(() => getOwnedByString(subroutine?.routine, [language]), [subroutine, language]);
    const toOwner = useCallback(() => { toOwnedBy(subroutine?.routine, setLocation) }, [subroutine, setLocation]);
    const canEdit = useMemo<boolean>(() => isEditing && (subroutine?.routine?.isInternal ?? owns(subroutine?.routine?.role)), [isEditing, subroutine?.routine?.isInternal, subroutine?.routine?.role]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<RoutineInputList>([]);
    const handleInputsUpdate = useCallback((updatedList: RoutineInputList) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<RoutineOutputList>([]);
    const handleOutputsUpdate = useCallback((updatedList: RoutineOutputList) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    useEffect(() => {
        setInputsList(subroutine?.routine?.inputs ?? []);
        setOutputsList(subroutine?.routine?.outputs ?? []);
    }, [subroutine]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            description: getTranslation(subroutine?.routine, 'description', [language]) ?? '',
            instructions: getTranslation(subroutine?.routine, 'instructions', [language]) ?? '',
            isInternal: subroutine?.routine?.isInternal ?? false,
            title: getTranslation(subroutine?.routine, 'title', [language]) ?? '',
            version: subroutine?.routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            handleUpdate({
                ...subroutine,
                isInternal: values.isInternal,
                version: values.version,
                translations: [{
                    language,
                    title: values.title,
                    description: values.description,
                    instructions: values.instructions,
                }],
            } as any);
        },
    });

    /**
     * Navigate to the subroutine's build page
     */
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);

    return (
        <SwipeableDrawer
            anchor="bottom"
            variant='persistent'
            open={open}
            onOpen={() => { }}
            onClose={onClose}
            sx={{
                '& .MuiDrawer-paper': {
                    background: palette.background.default,
                }
            }}
        >
            {/* Title bar with close icon */}
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 1,
            }}>
                {/* Subroutine title and position */}
                <Typography variant="h5">{formik.values.title}</Typography>
                <Typography variant="h6" ml={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routines?.length ?? 1)})`}</Typography>
                {/* Owned by and version */}
                <Stack direction="row" sx={{ marginLeft: 'auto' }}>
                    {ownedBy ? (
                        <LinkButton
                            onClick={toOwner}
                            text={`${ownedBy} - `}
                        />
                    ) : null}
                    <Typography variant="body1">{subroutine?.routine?.version}</Typography>
                </Stack>
                {/* Close button */}
                <IconButton onClick={onClose} sx={{
                    color: palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
            </Box>
            {/* Main content */}
            <Box sx={{
                padding: 2,
                overflowY: 'auto',
            }}>
                {/* Position, description and instructions */}
                <Grid container>
                    {/* Position */}
                    <Grid item xs={12}>
                        <Box sx={{
                            padding: 1,
                        }}>
                            <Typography variant="h6">Order</Typography>
                            <QuantityBox
                                id="subroutine-position"
                                disabled={!canEdit}
                                label="Order"
                                min={1}
                                max={data?.node?.routines?.length ?? 1}
                                tooltip="The order of this subroutine in its parent routine"
                                value={(subroutine?.index ?? 0) + 1}
                                handleChange={(value: number) => { handleReorder(data?.node?.id ?? '', subroutine?.index ?? 0, value - 1) }}
                            />
                        </Box>
                    </Grid>
                    {/* Description */}
                    <Grid item xs={12} sm={6}>
                        <Box sx={{
                            padding: 1,
                        }}>
                            <Typography variant="h6">Description</Typography>
                            {
                                canEdit ? (
                                    <MarkdownInput
                                        id="description"
                                        placeholder="Description"
                                        value={formik.values.description}
                                        minRows={2}
                                        onChange={(newText: string) => formik.setFieldValue('description', newText)}
                                        error={formik.touched.description && Boolean(formik.errors.description)}
                                        helperText={formik.touched.description ? formik.errors.description as string : null}
                                    />
                                ) : (
                                    <Markdown>{getTranslation(subroutine, 'description', [language]) ?? ''}</Markdown>
                                )
                            }
                        </Box>
                    </Grid>
                    {/* Instructions */}
                    <Grid item xs={12} sm={6}>
                        <Box sx={{
                            padding: 1,
                        }}>
                            <Typography variant="h6">Instructions</Typography>
                            {
                                canEdit ? (
                                    <MarkdownInput
                                        id="instructions"
                                        placeholder="Instructions"
                                        value={formik.values.instructions}
                                        minRows={2}
                                        onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                                        error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                                        helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                                    />
                                ) : (
                                    <Markdown>{getTranslation(subroutine, 'instructions', [language]) ?? ''}</Markdown>
                                )
                            }
                        </Box>
                    </Grid>
                    {/* Inputs */}
                    {(canEdit || inputsList.length > 0) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canEdit}
                            handleUpdate={handleInputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                            isInput={true}
                            language={language}
                            list={inputsList}
                            session={session}
                        />
                    </Grid>}
                    {/* Outputs */}
                    {(canEdit || outputsList.length > 0) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canEdit}
                            handleUpdate={handleOutputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                            isInput={false}
                            language={language}
                            list={outputsList}
                            session={session}
                        />
                    </Grid>}
                </Grid>
            </Box>
            {/* Bottom nav container */}

            {/* If subroutine has its own subroutines, display button to switch to that graph */}
            {(subroutine as any)?.nodesCount > 0 && (
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.primary.dark,
                }}>
                    <Button
                        color="secondary"
                        startIcon={<GraphIcon />}
                        onClick={toGraph}
                        sx={{
                            marginLeft: 'auto'
                        }}
                    >View Graph</Button>
                </Box>
            )}
        </SwipeableDrawer>
    );
}