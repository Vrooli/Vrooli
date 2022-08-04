import { Box, CircularProgress, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import {
    ContentCopy as CopyIcon,
    MoreHoriz as EllipsisIcon,
} from "@mui/icons-material";
import { HelpButton, LinkButton, ResourceListHorizontal } from "components";
import Markdown from "markdown-to-jsx";
import { useCallback, useEffect, useMemo } from "react";
import { containerShadow } from "styles";
import { formikToRunInputs, getOwnedByString, getTranslation, getUserLanguages, PubSub, runInputsToFormik, standardToFieldData, toOwnedBy } from "utils";
import { useLocation } from "wouter";
import { SubroutineViewProps } from "../types";
import { FieldData } from "forms/types";
import { generateInputComponent } from 'forms/generators';
import { useFormik } from "formik";

export const SubroutineView = ({
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    owner,
    routine,
    run,
    session,
    zIndex,
}: SubroutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { description, instructions, title } = useMemo(() => {
        const languages = session.languages ?? navigator.languages;
        return {
            description: getTranslation(routine, 'description', languages, true),
            instructions: getTranslation(routine, 'instructions', languages, true),
            title: getTranslation(routine, 'title', languages, true),
        }
    }, [routine, session.languages]);

    const ownedBy = useMemo<string | null>(() => {
        if (!routine) return null;
        // If isInternal, owner is same as overall routine owner
        const ownerObject = routine.isInternal ? { owner } : routine;
        return getOwnedByString(ownerObject, getUserLanguages(session))
    }, [routine, owner, session]);
    const toOwner = useCallback(() => {
        // Confirmation dialog for leaving routine
        PubSub.get().publishAlertDialog({
            message: 'Are you sure you want to stop this routine? You can continue it later.',
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Navigate to owner
                        const ownerObject = routine?.isInternal ? { owner } : routine;
                        toOwnedBy(ownerObject, setLocation)
                    }
                },
                { text: 'Cancel' },
            ]
        });
    }, [routine, handleSaveProgress, owner, setLocation]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData } | null>(() => {
        if (!routine) return null;
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < routine.inputs?.length; i++) {
            const currInput = routine.inputs[i];
            const currSchema = standardToFieldData({
                description: getTranslation(currInput, 'description', getUserLanguages(session), false) ?? getTranslation(currInput.standard, 'description', getUserLanguages(session), false),
                fieldName: `inputs-${currInput.id}`,
                props: currInput.standard?.props ?? '',
                name: currInput.name ?? currInput.standard?.name ?? '',
                type: currInput.standard?.type ?? '',
                yup: currInput.standard?.yup ?? null,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [routine, session]);
    const previewFormik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    /**
     * Update formik values with the current user inputs
     */
    useEffect(() => {
        if (!run?.inputs || !Array.isArray(run?.inputs)) return;
        const updatedValues = runInputsToFormik(run.inputs);
        previewFormik.setValues(updatedValues);
    }, 
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [previewFormik.setValues, run?.inputs]);

    /**
     * Update run with updated user inputs
     */
    useEffect(() => {
        if (!previewFormik.values) return;
        const updatedValues = formikToRunInputs(previewFormik.values);
        handleUserInputsUpdate(updatedValues);
    }, [handleUserInputsUpdate, previewFormik.values, run?.inputs]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = previewFormik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ message: 'Copied to clipboard.', severity: 'success' });
        } else {
            PubSub.get().publishSnack({ message: 'Input is empty.', severity: 'error' });
        }
    }, [previewFormik]);

    const resourceList = useMemo(() => {
        if (!routine ||
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(routine as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [routine, loading, session, zIndex]);

    if (loading) return (
        <Box sx={{
            minHeight: 'min(300px, 25vh)',
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
        }}>
            <CircularProgress color="secondary" />
        </Box>
    )
    return (
        <Box sx={{
            background: palette.background.paper,
            overflowY: 'auto',
            width: 'min(96vw, 600px)',
            borderRadius: '8px',
            overflow: 'overlay',
            ...containerShadow
        }}>
            {/* Heading container */}
            <Stack direction="column" spacing={1} sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                padding: 2,
                marginBottom: 1,
                background: palette.primary.dark,
                color: palette.primary.contrastText,
            }}>
                {/* Show more ellipsis next to title */}
                <Stack direction="row" spacing={1} alignItems="center">
                    <Typography variant="h5">{title}</Typography>
                    <Tooltip title="Options">
                        <IconButton
                            aria-label="More"
                            size="small"
                            onClick={() => { }} //TODO
                            sx={{
                                display: 'block',
                                marginLeft: 'auto',
                                marginRight: 1,
                            }}
                        >
                            <EllipsisIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1}>
                    {ownedBy ? (
                        <LinkButton
                            onClick={toOwner}
                            text={ownedBy}
                        />
                    ) : null}
                    <Typography variant="body1"> - {routine?.version}</Typography>
                </Stack>
            </Stack>
            {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
            <Stack direction="column" spacing={2} padding={1}>
                {/* Resources */}
                {resourceList}
                {/* Description */}
                {Boolean(description) && <Box sx={{
                    padding: 1,
                    borderRadius: 1,
                    color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary,
                }}>
                    <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Description</Typography>
                    <Typography variant="body1">{description}</Typography>
                </Box>}
                {/* Instructions */}
                <Box sx={{
                    padding: 1,
                    borderRadius: 1,
                    color: Boolean(instructions) ? palette.background.textPrimary : palette.background.textSecondary
                }}>
                    <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Instructions</Typography>
                    <Markdown>{instructions ?? 'No instructions'}</Markdown>
                </Box>
                {/* Auto-generated inputs */}
                {
                    Object.keys(previewFormik.values).length > 0 && <Box sx={{
                        padding: 1,
                        borderRadius: 1,
                    }}>
                        {
                            Object.values(formValueMap ?? {}).map((field: FieldData, i: number) => (
                                <Box key={i} sx={{
                                    padding: 1,
                                    borderRadius: 1,
                                }}>
                                    {/* Label, help button, and copy iput icon */}
                                    <Stack direction="row" spacing={0} sx={{ alignItems: 'center' }}>
                                        <Tooltip title="Copy to clipboard">
                                            <IconButton onClick={() => copyInput(field.fieldName)}>
                                                <CopyIcon />
                                            </IconButton>
                                        </Tooltip>
                                        <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>{field.label ?? `Input ${i + 1}`}</Typography>
                                        {field.description && <HelpButton markdown={field.description} />}
                                    </Stack>
                                    {
                                        generateInputComponent({
                                            data: field,
                                            disabled: false,
                                            formik: previewFormik,
                                            session,
                                            onUpload: () => { },
                                            zIndex,
                                        })
                                    }
                                </Box>
                            ))
                        }
                    </Box>
                }
            </Stack>
        </Box>
    )
}