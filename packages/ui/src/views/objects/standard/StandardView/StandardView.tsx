import { CommentFor, EditIcon, endpointGetStandardVersion, StandardVersion, useLocation } from "@local/shared";
import { Box, Palette, Stack, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { StandardInput } from "components/inputs/standards/StandardInput/StandardInput";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { standardInitialValues } from "forms/StandardForm/StandardForm";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ObjectAction } from "utils/actions/objectActions";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { SessionContext } from "utils/SessionContext";
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
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: StandardViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { isLoading, object: existing, permissions, setObject: setStandardVersion } = useObjectFromUrl<StandardVersion>({
        ...endpointGetStandardVersion,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(existing ?? partialData, [language]);
        return { description, name };
    }, [existing, partialData, language]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

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

    const initialValues = useMemo(() => standardInitialValues(session, (existing ?? partialData as any)), [existing, partialData, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Standard",
                }}
            />
            <Box sx={{
                marginLeft: "auto",
                marginRight: "auto",
                width: "min(100%, 700px)",
                padding: 2,
            }}>
                {/* Edit button, positioned at bottom corner of screen */}
                <Stack direction="row" spacing={2} sx={{
                    position: "fixed",
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "center",
                    zIndex: zIndex + 2,
                    bottom: 0,
                    right: 0,
                    // Accounts for BottomNav
                    marginBottom: {
                        xs: "calc(56px + 16px + env(safe-area-inset-bottom))",
                        md: "calc(16px + env(safe-area-inset-bottom))",
                    },
                    marginLeft: "calc(16px + env(safe-area-inset-left))",
                    marginRight: "calc(16px + env(safe-area-inset-right))",
                    height: "calc(64px + env(safe-area-inset-bottom))",
                }}>
                    {/* Edit button */}
                    {permissions.canUpdate ? (
                        <ColorIconButton aria-label="confirm-title-change" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                            <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                        </ColorIconButton>
                    ) : null}
                </Stack>
                <ObjectTitle
                    language={language}
                    languages={availableLanguages}
                    loading={isLoading}
                    title={name}
                    setLanguage={setLanguage}
                    translations={existing?.translations ?? []}
                    zIndex={zIndex}
                />
                {/* Relationships */}
                <RelationshipList
                    isEditing={false}
                    objectType={"Routine"}
                    zIndex={zIndex}
                />
                {/* Resources */}
                {Array.isArray(resourceList?.resources) && resourceList!.resources.length > 0 && <ResourceListHorizontal
                    title={"Resources"}
                    list={resourceList as any}
                    canUpdate={false}
                    // eslint-disable-next-line @typescript-eslint/no-empty-function
                    handleUpdate={() => { }} // Intentionally blank
                    loading={isLoading}
                    zIndex={zIndex}
                />}
                {/* Box with description */}
                <Box sx={containerProps(palette)}>
                    <TextCollapse title="Description" text={description} loading={isLoading} loadingLines={2} />
                </Box>
                {/* Box with standard */}
                <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                    <StandardInput
                        disabled={true}
                        fieldName="preview"
                        zIndex={zIndex}
                    />
                </Stack>
                {/* Tags */}
                {Array.isArray(tags) && tags!.length > 0 && <TagList
                    maxCharacters={30}
                    parentId={existing?.id ?? ""}
                    tags={tags as any[]}
                    sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
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
                    zIndex={zIndex}
                />
                {/* Comments */}
                <Box sx={containerProps(palette)}>
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        language={language}
                        objectId={existing?.id ?? ""}
                        objectType={CommentFor.StandardVersion}
                        onAddCommentClose={closeAddCommentDialog}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    );
};
