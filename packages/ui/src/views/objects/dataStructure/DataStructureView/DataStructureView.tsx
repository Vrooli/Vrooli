import { CommentFor, ResourceListShape, StandardShape, StandardVersion, TagShape, endpointGetStandardVersion, exists, getTranslation, noop, noopSubmit } from "@local/shared";
import { Box, Stack, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TagList } from "components/lists/TagList/TagList";
import { ResourceList } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts";
import { Formik } from "formik";
import { useObjectActions } from "hooks/objectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { FormSection, SideActionsButton } from "styles";
import { ObjectAction } from "utils/actions/objectActions";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { dataStructureInitialValues } from "../DataStructureUpsert/DataStructureUpsert";
import { DataStructureViewProps } from "../types";

export function DataStructureView({
    display,
    onClose,
}: DataStructureViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setStandardVersion } = useObjectFromUrl<StandardVersion>({
        ...endpointGetStandardVersion,
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
                        parentId={existing?.id ?? ""}
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
                        exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
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
                    <SideActionsButton aria-label={t("UpdateDataStructure")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
