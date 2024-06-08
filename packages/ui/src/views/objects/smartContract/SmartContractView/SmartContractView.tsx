import { CodeVersion, CommentFor, LINKS, ResourceList as ResourceListType, Tag, endpointGetCodeVersion, exists, noopSubmit } from "@local/shared";
import { Box, Button, Divider, IconButton, Stack, useTheme } from "@mui/material";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { CodeInput, CodeLanguage } from "components/inputs/CodeInput/CodeInput";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TagList } from "components/lists/TagList/TagList";
import { ResourceList } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { AddIcon, EditIcon, SearchIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { SearchVersionPageTabOption } from "utils/search/objectToSearch";
import { CodeShape } from "utils/shape/models/code";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { TagShape } from "utils/shape/models/tag";
import { smartContractInitialValues } from "../SmartContractUpsert/SmartContractUpsert";
import { SmartContractViewProps } from "../types";

export const SmartContractView = ({
    display,
    isOpen,
    onClose,
}: SmartContractViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading, object: existing, permissions, setObject: setCodeVersion } = useObjectFromUrl<CodeVersion>({
        ...endpointGetCodeVersion,
        objectType: "CodeVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description } = useMemo(() => {
        const { description, name } = getTranslation(existing, [language]);
        return { name, description };
    }, [existing, language]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Code",
        openAddCommentDialog,
        setLocation,
        setObject: setCodeVersion,
    });

    const initialValues = useMemo(() => smartContractInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as CodeShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("SmartContract", { count: 1 }))}
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
                {(formik) => <Stack direction="column" spacing={4} sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 700px)",
                    padding: 2,
                }}>
                    {/* Relationships */}
                    <RelationshipList
                        isEditing={false}
                        objectType={"Code"}
                    />
                    {/* Resources */}
                    {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                        horizontal
                        list={resourceList as unknown as ResourceListType}
                        canUpdate={false}
                        // eslint-disable-next-line @typescript-eslint/no-empty-function
                        handleUpdate={() => { }}
                        loading={isLoading}
                        parent={{ __typename: "CodeVersion", id: existing?.id ?? "" }}
                    />}
                    {!!description && <TextCollapse
                        title="Description"
                        text={description}
                        loading={isLoading}
                        loadingLines={2}
                    />}
                    {/* Tags */}
                    {exists(tags) && tags.length > 0 && <TagList
                        maxCharacters={30}
                        parentId={existing?.id ?? ""}
                        tags={tags as Tag[]}
                        sx={{ marginTop: 4 }}
                    />}
                    <ContentCollapse
                        title="Contract"
                        titleVariant="h4"
                        isOpen={true}
                        sxs={{ titleContainer: { marginBottom: 1 } }}
                    >
                        <CodeInput
                            disabled={true}
                            limitTo={[CodeLanguage.Javascript]}
                            name="content"
                        />
                    </ContentCollapse>
                    <Divider />
                    <ContentCollapse
                        title={t("StatisticsShort")}
                        titleVariant="h4"
                        isOpen={true}
                        sxs={{ titleContainer: { marginBottom: 1 } }}
                    >
                        <Button
                            href={`${LINKS.SearchVersion}?type=${SearchVersionPageTabOption.RoutineVersion}&codeVersionId=${existing.id}`}
                            sx={{
                                color: palette.background.textSecondary,
                                display: "flex",
                                marginBottom: 2,
                                textAlign: "center",
                                textTransform: "none",
                            }}
                            variant="text"
                            endIcon={<SearchIcon />}
                        >
                            {t("RoutinesConnected", { count: existing?.calledByRoutineVersionsCount ?? 0 })}
                        </Button>
                        {permissions.canUpdate && <Button
                            fullWidth
                            onClick={() => { }}
                            startIcon={<AddIcon />}
                            variant="outlined"
                        >
                            {t("CreateRoutine")}
                        </Button>}
                    </ContentCollapse>
                    <Box>
                        {/* Date and version labels */}
                        <Stack direction="row" spacing={1} mt={2} mb={1}>
                            {/* Date created */}
                            <DateDisplay
                                loading={isLoading}
                                showIcon={true}
                                timestamp={existing?.created_at}
                            />
                            <VersionDisplay
                                currentVersion={existing}
                                prefix={" - "}
                                versions={existing?.root?.versions}
                            />
                        </Stack>
                        {/* Votes, reports, and other basic stats */}
                        <StatsCompact
                            handleObjectUpdate={() => { }}
                            object={existing}
                        />
                        {/* Action buttons */}
                        <ObjectActionsRow
                            actionData={actionData}
                            exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                            object={existing}
                        />
                    </Box>
                    <Divider />
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        language={language}
                        objectId={existing?.id ?? ""}
                        objectType={CommentFor.RoutineVersion}
                        onAddCommentClose={closeAddCommentDialog}
                    />
                </Stack>}
            </Formik>
            {/* Edit button (if canUpdate), positioned at bottom corner of screen */}
            <SideActionsButtons display={display}>
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton aria-label={t("UpdateSmartContract")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} sx={{ background: palette.secondary.main }}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
            </SideActionsButtons>
        </>
    );
};
