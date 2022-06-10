import { Box, CircularProgress, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import {
    ContentCopy as CopyIcon,
    MoreHoriz as EllipsisIcon,
} from "@mui/icons-material";
import { HelpButton, LinkButton, ResourceListHorizontal } from "components";
import Markdown from "markdown-to-jsx";
import { useCallback, useMemo } from "react";
import { containerShadow } from "styles";
import { getOwnedByString, getTranslation, getUserLanguages, Pubs, standardToFieldData, toOwnedBy } from "utils";
import { useLocation } from "wouter";
import { SubroutineViewProps } from "../types";
import { FieldData } from "forms/types";
import { generateInputComponent } from 'forms/generators';
import { useFormik } from "formik";

export const SubroutineView = ({
    loading,
    data,
    handleSaveProgress,
    owner,
    session,
    zIndex,
}: SubroutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { description, instructions, title } = useMemo(() => {
        const languages = session.languages ?? navigator.languages;
        return {
            description: getTranslation(data, 'description', languages, true),
            instructions: getTranslation(data, 'instructions', languages, true),
            title: getTranslation(data, 'title', languages, true),
        }
    }, [data, session.languages]);

    const ownedBy = useMemo<string | null>(() => {
        if (!data) return null;
        // If isInternal, owner is same as overall routine owner
        const ownerObject = data.isInternal ? { owner } : data;
        return getOwnedByString(ownerObject, getUserLanguages(session))
    }, [data, owner, session]);
    const toOwner = useCallback(() => {
        // Confirmation dialog for leaving routine
        PubSub.publish(Pubs.AlertDialog, {
            message: 'Are you sure you want to stop this routine? You can continue it later.',
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Navigate to owner
                        const ownerObject = data?.isInternal ? { owner } : data;
                        toOwnedBy(ownerObject, setLocation)
                    }
                },
                { text: 'Cancel' },
            ]
        });
    }, [data, handleSaveProgress, owner, setLocation]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData } | null>(() => {
        if (!data) return null;
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < data.inputs?.length; i++) {
            const currSchema = standardToFieldData({
                description: getTranslation(data.inputs[i], 'description', getUserLanguages(session), false) ?? getTranslation(data.inputs[i].standard, 'description', getUserLanguages(session), false),
                fieldName: `inputs-${i}`,
                props: data.inputs[i].standard?.props ?? '',
                name: data.inputs[i].name ?? data.inputs[i].standard?.name ?? '',
                type: data.inputs[i].standard?.type ?? '',
                yup: data.inputs[i].standard?.yup ?? null,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [data, session]);
    const previewFormik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = previewFormik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.publish(Pubs.Snack, { message: 'Copied to clipboard.', severity: 'success' });
        } else {
            PubSub.publish(Pubs.Snack, { message: 'Input is empty.', severity: 'error' });
        }
    }, [previewFormik]);

    const resourceList = useMemo(() => {
        if (!data ||
            !Array.isArray(data.resourceLists) ||
            data.resourceLists.length < 1 ||
            data.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(data as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [data, loading, session, zIndex]);

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
                    <Typography variant="body1"> - {data?.version}</Typography>
                </Stack>
            </Stack>
            {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
            <Stack direction="column" spacing={2} padding={1}>
                {/* Resources */}
                {resourceList}
                {/* Description */}
                <Box sx={{
                    padding: 1,
                    borderRadius: 1,
                    color: Boolean(description) ? palette.background.textPrimary : palette.background.textSecondary,
                }}>
                    <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Description</Typography>
                    <Typography variant="body1">{description ?? 'No description set'}</Typography>
                </Box>
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
                        <Typography variant="h6" sx={{ color: palette.background.textPrimary }}>Inputs</Typography>
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