import { CommentFor, endpointGetStandardVersion, StandardVersion } from "@local/shared";
import { Box, Palette, Stack, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { standardInitialValues } from "forms/StandardForm/StandardForm";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { RoutineShape } from "utils/shape/models/routine";
import { TagShape } from "utils/shape/models/tag";
import { StandardViewProps } from "../types";

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: "overlay",
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
});

export const StandardView = ({
    isOpen,
    onClose,
}: StandardViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

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
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

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
            <Box sx={{
                marginLeft: "auto",
                marginRight: "auto",
                width: "min(100%, 700px)",
                padding: 2,
            }}>
                {/* Relationships */}
                <RelationshipList
                    isEditing={false}
                    objectType={"Routine"}
                />
                {/* Resources */}
                {Array.isArray(resourceList?.resources) && resourceList!.resources.length > 0 && <ResourceListHorizontal
                    title={"Resources"}
                    list={resourceList as any}
                    canUpdate={false}
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    handleUpdate={() => { }} // Intentionally blank
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
                    <StandardInput
                        disabled={true}
                        fieldName="preview"
                    />
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
            </Box>
            <SideActionButtons
                display={display}
                sx={{ position: "fixed" }}
            >
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <ColorIconButton aria-label="confirm-title-change" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
            </SideActionButtons>
        </>
    );
};
