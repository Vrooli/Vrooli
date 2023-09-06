import { CommentFor, exists, ResourceList, RoutineVersion, Tag } from "@local/shared";
import { Box, Button, LinearProgress, Palette, Stack, Typography, useTheme } from "@mui/material";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { GeneratedInputComponentWithLabel } from "components/inputs/generated";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { useFormik } from "formik";
import { routineInitialValues } from "forms/RoutineForm/RoutineForm";
import { FieldData } from "forms/types";
import { useObjectActions } from "hooks/useObjectActions";
import { SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { toDisplay } from "utils/display/pageTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { formikToRunInputs, runInputsToFormik } from "utils/runUtils";
import { standardVersionToFieldData } from "utils/shape/general";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { RoutineShape } from "utils/shape/models/routine";
import { TagShape } from "utils/shape/models/tag";
import { SubroutineViewProps } from "../types";

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});

export const SubroutineView = ({
    isOpen,
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    onClose,
    owner,
    routineVersion,
    run,
}: SubroutineViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const [internalRoutineVersion, setInternalRoutineVersion] = useState(routineVersion);
    useEffect(() => {
        setInternalRoutineVersion(routineVersion);
    }, [routineVersion]);
    const updateRoutine = useCallback((routineVersion: RoutineVersion) => { setInternalRoutineVersion(routineVersion); }, [setInternalRoutineVersion]);

    const availableLanguages = useMemo<string[]>(() => (internalRoutineVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [internalRoutineVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, instructions, name } = useMemo(() => {
        const languages = getUserLanguages(session);
        const { description, instructions, name } = getTranslation(internalRoutineVersion, languages, true);
        return {
            description,
            instructions,
            name,
        };
    }, [internalRoutineVersion, session]);

    const confirmLeave = useCallback((callback: () => any) => {
        // Confirmation dialog for leaving routine
        PubSub.get().publishAlertDialog({
            messageKey: "RunStopConfirm",
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        // Save progress
                        handleSaveProgress();
                        // Trigger callback
                        callback();
                    },
                },
                { labelKey: "Cancel" },
            ],
        });
    }, [handleSaveProgress]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData }>(() => {
        if (!internalRoutineVersion) return {};
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < internalRoutineVersion.inputs?.length; i++) {
            const currInput = internalRoutineVersion.inputs[i];
            if (!currInput.standardVersion) continue;
            const currSchema = standardVersionToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                name: currInput.name ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).name ?? "",
                standardType: currInput.standardVersion.standardType,
                yup: currInput.standardVersion.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [internalRoutineVersion, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? "";
            return acc;
        }, {}),
        enableReinitialize: true,
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        onSubmit: () => { },
    });

    /**
     * Update formik values with the current user inputs, if any
     */
    useEffect(() => {
        console.log("useeffect1 calculating preview formik values", run);
        if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
        console.log("useeffect 1calling runInputsToFormik", run.inputs);
        const updatedValues = runInputsToFormik(run.inputs);
        console.log("useeffect1 updating formik, values", updatedValues);
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
            PubSub.get().publishSnack({ messageKey: "CopiedToClipboard", severity: "Success" });
        } else {
            PubSub.get().publishSnack({ messageKey: "InputEmpty", severity: "Error" });
        }
    }, [formik.values]);

    const inputComponents = useMemo(() => {
        if (!internalRoutineVersion?.inputs || !Array.isArray(internalRoutineVersion?.inputs) || internalRoutineVersion.inputs.length === 0) return null;
        return (
            <Box>
                {Object.values(formValueMap).map((fieldData: FieldData, index: number) => (
                    <GeneratedInputComponentWithLabel
                        copyInput={copyInput}
                        disabled={false}
                        fieldData={fieldData}
                        index={index}
                        textPrimary={palette.background.textPrimary}
                        // eslint-disable-next-line @typescript-eslint/no-empty-function
                        onUpload={() => { }}
                    />
                ))}
            </Box>
        );
    }, [copyInput, formValueMap, palette.background.textPrimary, internalRoutineVersion?.inputs]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: internalRoutineVersion,
        objectType: "RoutineVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setInternalRoutineVersion,
    });

    const initialValues = useMemo(() => routineInitialValues(session, internalRoutineVersion), [internalRoutineVersion, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    // Display title or loading bar
    const titleComponent = loading ? <LinearProgress color="inherit" sx={{
        borderRadius: 1,
        width: "50vw",
        height: 8,
        marginTop: "12px !important",
        marginBottom: "12px !important",
        maxWidth: "300px",
    }} /> : <Typography
        component="h1"
        variant="h3"
        sx={{
            textAlign: "center",
            sx: { marginTop: 2, marginBottom: 2 },
            fontSize: { xs: "1.5rem", sm: "2rem", md: "2.5rem" },
        }}
    >{name}</Typography>;

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
            />
            <Box sx={{
                marginLeft: "auto",
                marginRight: "auto",
                width: "min(100%, 700px)",
                padding: 2,
            }}>
                <Stack
                    direction="row"
                    justifyContent="center"
                    alignItems="center"
                    spacing={2}
                    sx={{
                        marginTop: 2,
                        marginBottom: 2,
                        marginLeft: "auto",
                        marginRight: "auto",
                        maxWidth: "700px",
                    }}
                >
                    {titleComponent}
                    {availableLanguages.length > 1 && <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        languages={availableLanguages}
                    />}
                </Stack>
                {/* Resources */}
                {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                    title={"Resources"}
                    list={resourceList as unknown as ResourceList}
                    canUpdate={false}
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    handleUpdate={() => { }} // Intentionally blank
                    loading={loading}
                    parent={{ __typename: "RoutineVersion", id: routineVersion?.id ?? "" }}
                />}
                {/* Box with description and instructions */}
                <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                    {/* Description */}
                    <TextCollapse
                        title="Description"
                        text={description}
                        loading={loading}
                        loadingLines={2}
                    />
                    {/* Instructions */}
                    <TextCollapse
                        title="Instructions"
                        text={instructions}
                        loading={loading}
                        loadingLines={4}
                    />
                </Stack>
                <Box sx={containerProps(palette)}>
                    <ContentCollapse title="Inputs">
                        {inputComponents}
                        <Button
                            startIcon={<SuccessIcon />}
                            fullWidth
                            onClick={() => { }}
                            color="secondary"
                            sx={{ marginTop: 2 }}
                            variant="contained"
                        >{t("Submit")}</Button>
                    </ContentCollapse>
                </Box>
                {/* Action buttons */}
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                    object={internalRoutineVersion}
                />
                <Box sx={containerProps(palette)}>
                    <ContentCollapse
                        isOpen={false}
                        title="Additional Information"
                    >
                        {/* Relationships */}
                        <RelationshipList
                            isEditing={false}
                            objectType={"Routine"}
                        />
                        {/* Tags */}
                        {exists(tags) && tags.length > 0 && <TagList
                            maxCharacters={30}
                            parentId={internalRoutineVersion?.id ?? ""}
                            tags={tags as Tag[]}
                            sx={{ marginTop: 4 }}
                        />}
                        {/* Date and version labels */}
                        <Stack direction="row" spacing={1} mt={2} mb={1}>
                            {/* Date created */}
                            <DateDisplay
                                loading={loading}
                                showIcon={true}
                                timestamp={internalRoutineVersion?.created_at}
                            />
                            <VersionDisplay
                                currentVersion={internalRoutineVersion}
                                prefix={" - "}
                                versions={internalRoutineVersion?.root?.versions}
                            />
                        </Stack>
                        {/* Votes, reports, and other basic stats */}
                        {/* <StatsCompact
                        handleObjectUpdate={updateRoutine}
                        loading={loading}
                        object={internalRoutineVersion ?? null}
                    /> */}
                    </ContentCollapse>
                </Box>
                {/* Comments */}
                <Box sx={containerProps(palette)}>
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        isOpen={false}
                        language={language}
                        objectId={internalRoutineVersion?.id ?? ""}
                        objectType={CommentFor.RoutineVersion}
                        onAddCommentClose={closeAddCommentDialog}
                    />
                </Box>
            </Box>
        </>
    );
};
