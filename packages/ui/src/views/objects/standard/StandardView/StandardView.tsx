import { CommentFor, endpointGetStandardVersion, exists, noop, noopSubmit, StandardVersion } from "@local/shared";
import { Box, IconButton, Palette, Stack, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceList } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { StandardShape } from "utils/shape/models/standard";
import { TagShape } from "utils/shape/models/tag";
import { standardInitialValues } from "../StandardUpsert/StandardUpsert";
import { StandardViewProps } from "../types";

function containerProps(palette: Palette) {
    return {
        boxShadow: 1,
        background: palette.background.paper,
        borderRadius: 1,
        overflow: "overlay",
        marginTop: 4,
        marginBottom: 4,
        padding: 2,
    };
}

export function StandardView({
    display,
    onClose,
}: StandardViewProps) {
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
        objectType: "Standard",
        openAddCommentDialog,
        setLocation,
        setObject: setStandardVersion,
    });

    const initialValues = useMemo(() => standardInitialValues(session, existing), [existing, session]);
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
                    <Box sx={containerProps(palette)}>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isLoading}
                            loadingLines={2}
                        />
                    </Box>
                    {/* Box with standard */}
                    <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                        {/* <StandardInput
                            disabled={true}
                            fieldName="preview"
                        /> */}
                        TODO
                    </Stack>
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
                    <Box sx={containerProps(palette)}>
                        <CommentContainer
                            forceAddCommentOpen={isAddCommentOpen}
                            language={language}
                            objectId={existing?.id ?? ""}
                            objectType={CommentFor.StandardVersion}
                            onAddCommentClose={closeAddCommentDialog}
                        />
                    </Box>
                </Box>}
            </Formik>
            <SideActionsButtons display={display}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton aria-label={t("UpdateStandard")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} sx={{ background: palette.secondary.main }}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
}
