import { CommentFor, ResourceListShape, StandardShape, StandardVersion, TagShape, endpointsStandardVersion, exists, getTranslation, noop, noopSubmit } from "@local/shared";
import { Box, IconButton, Stack, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { TextCollapse } from "../../../components/containers/TextCollapse.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceList } from "../../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { DateDisplay } from "../../../components/text/DateDisplay.js";
import { VersionDisplay } from "../../../components/text/VersionDisplay.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { FormSection } from "../../../styles.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { dataStructureInitialValues } from "./DataStructureUpsert.js";
import { DataStructureViewProps } from "./types.js";

const contextActionsExcluded = [ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp] as const;

export function DataStructureView({
    display,
    onClose,
}: DataStructureViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setStandardVersion } = useManagedObject<StandardVersion>({
        ...endpointsStandardVersion.findOne,
        objectType: "StandardVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(existing, [language]);
        return { description, name };
    }, [existing, language]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "StandardVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setStandardVersion,
    });

    const initialValues = useMemo(() => dataStructureInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as StandardShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("Standard"))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                />}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
            >
                {(formik) => <Box sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 800px)",
                    padding: 2,
                }}>
                    {/* Relationships */}
                    <RelationshipList
                        isEditing={false}
                        objectType={"Routine"}
                    />
                    {/* Resources */}
                    {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                        horizontal
                        title={"Resources"}
                        list={resourceList as any}
                        canUpdate={false}
                        handleUpdate={noop}
                        loading={isLoading}
                        parent={{ __typename: "StandardVersion", id: existing?.id ?? "" }}
                    />}
                    {/* Box with description */}
                    <FormSection>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isLoading}
                            loadingLines={2}
                        />
                    </FormSection>
                    {/* Box with standard */}
                    <FormSection>
                        {/* <StandardInput
                            disabled={true}
                            fieldName="preview"
                        /> */}
                        TODO
                    </FormSection>
                    {/* Tags */}
                    {Array.isArray(tags) && tags!.length > 0 && <TagList
                        maxCharacters={30}
                        tags={tags as any[]}
                        sx={{ marginTop: 4 }}
                    />}
                    {/* Date and version labels */}
                    <Stack direction="row" spacing={1} mt={2} mb={1}>
                        {/* Date created */}
                        <DateDisplay
                            loading={isLoading}
                            timestamp={existing?.created_at}
                        />
                        <VersionDisplay
                            currentVersion={existing}
                            prefix={" - v"}
                            versions={existing?.root?.versions}
                        />
                    </Stack>
                    {/* Votes, reports, and other basic stats */}
                    {/* <StatsCompact
                handleObjectUpdate={updateStandard}
                loading={loading}
                object={existing}
            /> */}
                    {/* Action buttons */}
                    <ObjectActionsRow
                        actionData={actionData}
                        exclude={contextActionsExcluded} // Handled elsewhere
                        object={existing}
                    />
                    {/* Comments */}
                    <FormSection>
                        <CommentContainer
                            forceAddCommentOpen={isAddCommentOpen}
                            language={language}
                            objectId={existing?.id ?? ""}
                            objectType={CommentFor.StandardVersion}
                            onAddCommentClose={closeAddCommentDialog}
                        />
                    </FormSection>
                </Box>}
            </Formik>
            <SideActionsButtons display={display}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton aria-label={t("UpdateDataStructure")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}>
                        <IconCommon name="Edit" />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
