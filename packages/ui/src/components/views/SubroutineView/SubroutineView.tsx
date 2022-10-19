import { Box, CircularProgress, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { ObjectActionMenu, OwnerLabel, ResourceListHorizontal, SnackSeverity, TextCollapse, VersionDisplay } from "components";
import { useCallback, useEffect, useMemo, useState } from "react";
import { formikToRunInputs, getTranslation, getUserLanguages, ObjectType, openObject, PubSub, runInputsToFormik, standardToFieldData, uuidToBase36 } from "utils";
import { useLocation } from '@shared/route';
import { SubroutineViewProps } from "../types";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from 'forms/generators';
import { useFormik } from "formik";
import { EllipsisIcon } from "@shared/icons";
import { ObjectAction, ObjectActionComplete } from "components/dialogs/types";
import { APP_LINKS } from "@shared/consts";

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
    console.log('subroutine view', owner, routine?.isInternal, routine?.owner)
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const [internalRoutine, setInternalRoutine] = useState(routine);
    useEffect(() => {
        setInternalRoutine(routine);
    }, [routine]);

    const { description, instructions, title } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions, title } = getTranslation(internalRoutine, languages, true);
        return {
            description,
            instructions,
            title,
        }
    }, [internalRoutine, session]);

    const confirmLeave = useCallback((callback: () => any) => {
        // Confirmation dialog for leaving routine
        PubSub.get().publishAlertDialog({
            message: 'Are you sure you want to stop this routine? You can continue it later.',
            buttons: [
                {
                    text: 'Yes', onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Trigger callback
                        callback();
                    }
                },
                { text: 'Cancel' },
            ]
        });
    }, [handleSaveProgress]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData }>(() => {
        if (!internalRoutine) return {};
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < internalRoutine.inputs?.length; i++) {
            const currInput = internalRoutine.inputs[i];
            if (!currInput.standard) continue;
            const currSchema = standardToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standard, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standard.props,
                name: currInput.name ?? currInput.standard.name,
                type: currInput.standard.type,
                yup: currInput.standard.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [internalRoutine, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    /**
     * Update formik values with the current user inputs, if any
     */
    useEffect(() => {
        console.log('useeffect1 calculating preview formik values', run)
        if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
        console.log('useeffect 1calling runInputsToFormik', run.inputs)
        const updatedValues = runInputsToFormik(run.inputs);
        console.log('useeffect1 updating formik, values', updatedValues)
        formik.setValues(updatedValues);
    },
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [formik.setValues, run?.inputs]);

    /**
     * Update run with updated user inputs
     */
    useEffect(() => {
        if (!formik.values) return;
        const updatedValues = formikToRunInputs(formik.values);
        handleUserInputsUpdate(updatedValues);
    }, [handleUserInputsUpdate, formik.values, run?.inputs]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ message: 'Copied to clipboard.', severity: SnackSeverity.Success });
        } else {
            PubSub.get().publishSnack({ message: 'Input is empty.', severity: SnackSeverity.Error });
        }
    }, [formik.values]);

    const resourceList = useMemo(() => {
        if (!internalRoutine ||
            !Array.isArray(internalRoutine.resourceLists) ||
            internalRoutine.resourceLists.length < 1 ||
            internalRoutine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(internalRoutine as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [internalRoutine, loading, session, zIndex]);

    const inputComponents = useMemo(() => {
        if (!internalRoutine?.inputs || !Array.isArray(internalRoutine?.inputs) || internalRoutine.inputs.length === 0) return null;
        return (
            <Box>
                {Object.values(formValueMap).map((fieldData: FieldData, index: number) => (
                    generateInputWithLabel({
                        copyInput,
                        disabled: false,
                        fieldData,
                        formik: formik,
                        index,
                        session,
                        textPrimary: palette.background.textPrimary,
                        onUpload: () => { },
                        zIndex,
                    })
                ))}
            </Box>
        )
    }, [copyInput, formValueMap, palette.background.textPrimary, formik, internalRoutine?.inputs, session, zIndex]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<HTMLElement | null>(null);
    const openMoreMenu = useCallback((ev: React.MouseEvent<HTMLElement>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const onEdit = useCallback(() => {
        setLocation(`${APP_LINKS.Routine}/edit/${uuidToBase36(internalRoutine?.id ?? '')}`);
    }, [internalRoutine?.id, setLocation]);

    const onMoreActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit]);

    const onMoreActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.success) {
                    setInternalRoutine({
                        ...internalRoutine,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success) {
                    setInternalRoutine({
                        ...internalRoutine,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.routine, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                openObject(data.routine, setLocation);
                window.location.reload();
                break;
        }
    }, [internalRoutine, setLocation]);

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
            // safe-area-inset-bottom is the iOS navigation bar
            marginBottom: 'calc(64px + env(safe-area-inset-bottom))',
            boxShadow: 12,
        }}>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                isUpvoted={internalRoutine?.isUpvoted}
                isStarred={internalRoutine?.isStarred}
                objectId={internalRoutine?.id ?? ''}
                objectName={title ?? ''}
                objectType={ObjectType.Routine}
                anchorEl={moreMenuAnchor}
                title='Subroutine Options'
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                permissions={{
                    ...(internalRoutine?.permissionsRoutine ?? {}),
                    canDelete: false,
                }}
                session={session}
                zIndex={zIndex + 1}
            />
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
                            onClick={openMoreMenu}
                            sx={{
                                display: 'block',
                                marginLeft: 'auto',
                                marginRight: 1,
                            }}
                        >
                            <EllipsisIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Tooltip>
                </Stack>
                <Stack direction="row" spacing={1}>
                    <OwnerLabel
                        confirmOpen={confirmLeave}
                        objectType={ObjectType.Routine}
                        owner={internalRoutine?.isInternal ? owner : internalRoutine?.owner}
                        session={session}
                    />
                    <VersionDisplay
                        confirmVersionChange={confirmLeave}
                        currentVersion={internalRoutine?.version}
                        prefix={" - "}
                        versions={internalRoutine?.versions}
                    />
                </Stack>
            </Stack>
            {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
            <Stack direction="column" spacing={2} padding={2}>
                {/* Resources */}
                {resourceList}
                {/* Description */}
                <TextCollapse title="Description" text={description} />
                {/* Instructions */}
                <TextCollapse title="Instructions" text={instructions} />
                {/* Inputs */}
                {inputComponents}
            </Stack>
        </Box>
    )
}