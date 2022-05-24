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
import { InputOutputContainer, LinkButton, MarkdownInput } from 'components';
import { useFormik } from 'formik';
import { RoutineInputList, RoutineOutputList } from 'types';

export const SubroutineInfoDialog = ({
    handleUpdate,
    handleViewFull,
    isEditing,
    open,
    language,
    session,
    subroutine,
    onClose,
}: SubroutineInfoDialogProps) => {
    console.log('boop isediting', isEditing)
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const ownedBy = useMemo<string | null>(() => getOwnedByString(subroutine, [language]), [subroutine, language]);
    const toOwner = useCallback(() => { toOwnedBy(subroutine, setLocation) }, [subroutine, setLocation]);

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
        setInputsList(subroutine?.inputs ?? []);
        setOutputsList(subroutine?.outputs ?? []);
    }, [subroutine]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            description: getTranslation(subroutine, 'description', [language]) ?? '',
            instructions: getTranslation(subroutine, 'instructions', [language]) ?? '',
            isInternal: subroutine?.isInternal ?? false,
            title: getTranslation(subroutine, 'title', [language]) ?? '',
            version: subroutine?.version ?? '',
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
                    maxHeight: 'min(300px, 100vh)',
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
                {/* Close button */}
                <IconButton onClick={onClose} sx={{
                    color: palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                {/* Subroutine title */}
                <Typography variant="h5">{getTranslation(subroutine, 'title', [language])}</Typography>
                {/* Owned by and version */}
                <Stack direction="row" sx={{ marginLeft: 'auto' }}>
                    {ownedBy ? (
                        <LinkButton
                            onClick={toOwner}
                            text={ownedBy}
                        />
                    ) : null}
                    <Typography variant="body1"> - {subroutine?.version}</Typography>
                </Stack>
            </Box>
            {/* Main content */}
            <Box sx={{
                padding: 2,
                overflowY: 'auto',
            }}>
                {/* Description and instructions */}
                <Grid container>
                    {/* Description */}
                    <Grid item xs={12} sm={6}>
                        <Box sx={{
                            padding: 1,
                            border: `1px solid ${palette.primary.dark}`,
                            borderRadius: 1,
                        }}>
                            <Typography variant="h6">Description</Typography>
                            {
                                isEditing ? (
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
                            border: `1px solid ${palette.background.paper}`,
                            borderRadius: 1,
                        }}>
                            <Typography variant="h6">Instructions</Typography>
                            {
                                isEditing ? (
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
                    <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={isEditing}
                            handleUpdate={handleInputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                            isInput={true}
                            language={language}
                            list={inputsList}
                            session={session}
                        />
                    </Grid>
                    {/* Outputs */}
                    <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={isEditing}
                            handleUpdate={handleOutputsUpdate as (updatedList: RoutineInputList | RoutineOutputList) => void}
                            isInput={false}
                            language={language}
                            list={outputsList}
                            session={session}
                        />
                    </Grid>
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