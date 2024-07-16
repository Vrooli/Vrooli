import { CommentFor, exists, noop, ResourceList as ResourceListType, RoutineVersion, Tag } from "@local/shared";
import { Box, Button, LinearProgress, Stack, Typography, useTheme } from "@mui/material";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { FormInput } from "components/inputs/form";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceList } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { useFormik } from "formik";
import { createFormInput } from "forms/generators";
import { FormInputType } from "forms/types";
import { useObjectActions } from "hooks/useObjectActions";
import { SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { formikToRunInputs, runInputsToFormik } from "utils/runUtils";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { RoutineShape } from "utils/shape/models/routine";
import { TagShape } from "utils/shape/models/tag";
import { routineInitialValues } from "views/objects/routine";
import { SubroutineViewProps } from "../types";

//TODO update to latest from RoutineView
export function SubroutineView({
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    onClose,
    owner,
    routineVersion,
    run,
}: SubroutineViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = "dialog";
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
        PubSub.get().publish("alertDialog", {
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
    const formValueMap = useMemo<{ [fieldName: string]: FormInputType }>(() => {
        if (!internalRoutineVersion) return {};
        const schemas: { [fieldName: string]: FormInputType } = {};
        for (let i = 0; i < internalRoutineVersion.inputs?.length; i++) {
            const currInput = internalRoutineVersion.inputs[i];
            if (!currInput.standardVersion) continue;
            const currSchema = createFormInput({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                label: currInput.name ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).name ?? "",
                type: currInput.standardVersion.standardType as FormInputType["type"],
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
        onSubmit: noop,
    });

    /**
     * Update formik values with the current user inputs, if any
     */
    useEffect(() => {
        if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
        const updatedValues = runInputsToFormik(run.inputs);
        formik.setValues(updatedValues);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [formik.setValues, run?.inputs]);

    /**
     * Update run with updated user inputs
     */
    useEffect(() => {
        if (!formik.values) return;
        const updatedValues = formikToRunInputs(formik.values);
        handleUserInputsUpdate(updatedValues);
    }, [handleUserInputsUpdate, formik.values, run?.inputs]);

    const inputComponents = useMemo(() => {
        if (!internalRoutineVersion?.inputs || !Array.isArray(internalRoutineVersion?.inputs) || internalRoutineVersion.inputs.length === 0) return null;
        return (
            <Box>
                {Object.values(formValueMap).map((fieldData: FormInputType, index: number) => (
                    <FormInput
                        key={fieldData.id}
                        fieldData={fieldData}
                        index={index}
                        isEditing={false}
                        onConfigUpdate={noop}
                        onDelete={noop}
                        textPrimary={palette.background.textPrimary}
                    />
                ))}
            </Box>
        );
    }, [formValueMap, palette.background.textPrimary, internalRoutineVersion?.inputs]);

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
                {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                    horizontal
                    title={"Resources"}
                    list={resourceList as unknown as ResourceListType}
                    canUpdate={false}
                    handleUpdate={noop}
                    loading={loading}
                    parent={{ __typename: "RoutineVersion", id: routineVersion?.id ?? "" }}
                />}
                {/* Box with description and instructions */}
                <FormSection>
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
                </FormSection>
                <FormSection>
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
                </FormSection>
                {/* Action buttons */}
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                    object={internalRoutineVersion}
                />
                <FormSection>
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
                                timestamp={internalRoutineVersion?.created_at}
                            />
                            <VersionDisplay
                                currentVersion={internalRoutineVersion}
                                prefix={" - v"}
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
                </FormSection>
                {/* Comments */}
                <FormSection>
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        isOpen={false}
                        language={language}
                        objectId={internalRoutineVersion?.id ?? ""}
                        objectType={CommentFor.RoutineVersion}
                        onAddCommentClose={closeAddCommentDialog}
                    />
                </FormSection>
            </Box>
        </>
    );
}
