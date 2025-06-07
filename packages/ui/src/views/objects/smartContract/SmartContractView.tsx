import { Box, Button, Container, IconButton, LinearProgress, Paper, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkFor, CodeLanguage, CommentFor, LINKS, SearchVersionPageTabOption, endpointsResource, exists, getTranslation, noopSubmit, type Resource, type ResourceVersion, type ResourceListShape, type ResourceList as ResourceListType, type Tag, type TagShape } from "@vrooli/shared";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useState, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink.js";
import { SearchExistingButton } from "../../../components/buttons/SearchExistingButton.js";
import { ShareButton } from "../../../components/buttons/ShareButton.js";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { CodeInput } from "../../../components/inputs/CodeInput/CodeInput.js";
import { ResourceList } from "../../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { DateDisplay } from "../../../components/text/DateDisplay.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
import { StatsCompact } from "../../../components/text/StatsCompact.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { IconCommon, IconRoutine } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ScrollBox } from "../../../styles.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { smartContractInitialValues } from "./SmartContractUpsert.js";
import { type SmartContractViewProps } from "./types.js";

const TAGS_MAX_CHARS = 200;
const codeLimitTo = [CodeLanguage.Javascript] as const;

const contextActionsExcluded = [
    ObjectAction.Edit,
    ObjectAction.Delete,
    ObjectAction.Fork,
    ObjectAction.Report,
    ObjectAction.Bookmark,
    ObjectAction.Comment,
];

// Style constants
const paperStyle = {
    p: 3,
    borderRadius: 2,
} as const;

export function SmartContractView({
    display,
    isOpen,
    onClose,
}: SmartContractViewProps) {
    const session = useContext(SessionContext);
    const { palette, breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(false);

    const { isLoading, object: codeVersion, permissions, setObject: setCodeVersion } = useManagedObject<ResourceVersion>({
        ...endpointsResource.findOne,
        objectType: "ResourceVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (codeVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [codeVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, canBookmark } = useMemo(() => {
        const { canBookmark } = codeVersion?.root?.you ?? {};
        const { description, name } = getTranslation(codeVersion, [language]);
        return { name, description, canBookmark };
    }, [codeVersion, language]);

    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const initialValues = useMemo(() => smartContractInitialValues(session, codeVersion), [codeVersion, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() =>
        initialValues.resourceList as ResourceListShape | null | undefined,
        [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() =>
        (initialValues.root as Resource)?.tags as TagShape[] | null | undefined,
        [initialValues]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<null | HTMLElement>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<HTMLElement>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: codeVersion,
        objectType: "ResourceVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setCodeVersion,
    });

    // Create a noop function for handlers that don't need to do anything
    const handleNoop = useCallback(() => undefined, []);

    // Get the background color for panels based on the current theme
    const getPanelBgColor = useCallback(() => {
        return palette.mode === "light" ? "#f5f7fa" : "#1e1e1e";
    }, [palette.mode]);

    // Create handlers ahead of time to avoid inline functions in JSX
    const handleEditClick = useCallback(() => actionData.onActionStart(ObjectAction.Edit), [actionData]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={firstString(name, t("SmartContract", { count: 1 }))}
                    titleBehavior="Hide"
                />
                {/* Popup menu displayed when "More" ellipsis pressed */}
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={moreMenuAnchor}
                    object={codeVersion as any}
                    onClose={closeMoreMenu}
                />

                {/* Top section with Smart Contract icon, title, and metadata */}
                <Box sx={{
                    background: palette.mode === "light" ? "#b2b3b3" : "#303030",
                    paddingY: 3,
                    position: "relative",
                }}>
                    {/* Language selector */}
                    <Box
                        position="absolute"
                        top={8}
                        right={8}
                    >
                        {availableLanguages.length > 1 && <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            languages={availableLanguages}
                        />}
                    </Box>

                    <Container maxWidth="lg">
                        <Stack direction={{ xs: "column", sm: "row" }} spacing={3} alignItems="center">
                            {/* Smart Contract Icon */}
                            <IconCommon
                                decorative
                                name="SmartContract"
                                size={80}
                                style={{
                                    margin: 10,
                                    filter: "drop-shadow(0px 2px 4px rgba(0, 0, 0, 0.2))",
                                }}
                            />

                            {/* Smart Contract Title and Metadata */}
                            <Stack flex={1} spacing={1}>
                                {isLoading ? (
                                    <LinearProgress color="inherit" sx={{ width: "50%" }} />
                                ) : (
                                    <>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                                            <Typography variant="h4" fontWeight="bold" component="div">
                                                {name || t("SmartContract", { count: 1 })}
                                            </Typography>
                                            {actionData.availableActions.length > 0 && <Tooltip title={t("MoreOptions")}>
                                                <IconButton
                                                    aria-label={t("MoreOptions")}
                                                    size="small"
                                                    onClick={openMoreMenu}
                                                >
                                                    <IconCommon
                                                        decorative
                                                        fill={palette.background.textSecondary}
                                                        name="Ellipsis"
                                                    />
                                                </IconButton>
                                            </Tooltip>}
                                        </Stack>

                                        <Typography variant="body1" color="textSecondary">
                                            {t("SmartContractDescription")}
                                        </Typography>

                                        {/* Metrics */}
                                        <Stack
                                            direction="row"
                                            spacing={{ xs: 1, sm: 4 }}
                                            mt={1}
                                        >
                                            <Tooltip title={t("View", { count: codeVersion?.root?.views || 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconCommon
                                                        decorative
                                                        name="Visible"
                                                        size={20}
                                                        fill={palette.background.textSecondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {codeVersion?.root?.views || 0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>

                                            <Tooltip title={t("Routine", { count: codeVersion?.calledByRoutineVersionsCount || 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconRoutine
                                                        decorative
                                                        name="Routine"
                                                        size={20}
                                                        fill={palette.background.textSecondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {codeVersion?.calledByRoutineVersionsCount || 0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>

                                            <Tooltip title={t("Version")}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <Typography variant="body2" fontWeight="medium">v</Typography>
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {codeVersion?.versionLabel || "1.0"}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>

                                            <DateDisplay
                                                loading={isLoading}
                                                timestamp={codeVersion?.createdAt}
                                            />
                                        </Stack>

                                        {/* Tags */}
                                        {exists(tags) && tags.length > 0 && (
                                            <Box mt={1}>
                                                <TagList
                                                    tags={tags as Tag[]}
                                                    maxCharacters={TAGS_MAX_CHARS}
                                                    sx={{
                                                        maxWidth: "100%",
                                                        flexWrap: "wrap",
                                                    }}
                                                />
                                            </Box>
                                        )}

                                        {/* Action Buttons */}
                                        <Stack direction="row" spacing={2} mt={1}>
                                            <ShareButton object={codeVersion} />
                                            <ReportsLink object={codeVersion} />
                                            <BookmarkButton
                                                disabled={!canBookmark}
                                                objectId={codeVersion?.id ?? ""}
                                                bookmarkFor={BookmarkFor.Code}
                                                isBookmarked={codeVersion?.root?.you?.isBookmarked ?? false}
                                                bookmarks={codeVersion?.root?.bookmarks ?? 0}
                                                onChange={handleNoop}
                                            />
                                        </Stack>
                                    </>
                                )}
                            </Stack>
                        </Stack>
                    </Container>
                </Box>

                <Container maxWidth="lg">
                    <Formik
                        enableReinitialize={true}
                        initialValues={initialValues}
                        onSubmit={noopSubmit}
                    >
                        {() => (
                            <>
                                {/* Description Section */}
                                {description && (
                                    <Box mt={4}>
                                        <Paper
                                            elevation={0}
                                            sx={{
                                                backgroundColor: getPanelBgColor(),
                                                ...paperStyle,
                                            }}
                                        >
                                            <MarkdownDisplay content={description} headingLevelOffset={2} />
                                        </Paper>
                                    </Box>
                                )}

                                {/* Resources Section */}
                                {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && (
                                    <Box mt={4} mx={2}>
                                        <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                            {t("Resources")}
                                        </Typography>
                                        <ResourceList
                                            horizontal
                                            list={resourceList as unknown as ResourceListType}
                                            canUpdate={permissions.canUpdate}
                                            handleUpdate={(updatedList) => {
                                                if (!codeVersion) return;
                                                setCodeVersion({
                                                    ...codeVersion,
                                                    resourceList: updatedList,
                                                });
                                            }}
                                            loading={isLoading}
                                            parent={{ __typename: "CodeVersion", id: codeVersion?.id ?? "" }}
                                        />
                                    </Box>
                                )}

                                {/* Code Section */}
                                <Box mt={4}>
                                    <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                        {t("Contract")}
                                    </Typography>
                                    {isLoading ? (
                                        <LinearProgress color="inherit" />
                                    ) : (
                                        <Paper
                                            elevation={0}
                                            sx={{
                                                backgroundColor: getPanelBgColor(),
                                                ...paperStyle,
                                            }}
                                        >
                                            <CodeInput
                                                disabled={true}
                                                limitTo={codeLimitTo}
                                                name="content"
                                            />
                                        </Paper>
                                    )}
                                </Box>

                                {/* Statistics Section */}
                                <Box mt={4}>
                                    <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                        {t("StatisticsShort")}
                                    </Typography>
                                    <Paper
                                        elevation={0}
                                        sx={{
                                            backgroundColor: getPanelBgColor(),
                                            ...paperStyle,
                                        }}
                                    >
                                        <Stack spacing={2}>
                                            <SearchExistingButton
                                                href={`${LINKS.SearchVersion}?type="${SearchVersionPageTabOption.SmartContractVersion}"&codeVersionId="${codeVersion?.id}"`}
                                                text={t("RoutinesConnected", { count: codeVersion?.calledByRoutineVersionsCount ?? 0 })}
                                            />
                                            {permissions.canUpdate && (
                                                <Button
                                                    fullWidth
                                                    onClick={handleNoop}
                                                    startIcon={<IconCommon name="Add" />}
                                                    variant="contained"
                                                    color="primary"
                                                >
                                                    {t("CreateRoutine")}
                                                </Button>
                                            )}
                                            <StatsCompact
                                                handleObjectUpdate={handleNoop}
                                                object={codeVersion}
                                            />
                                        </Stack>
                                    </Paper>
                                </Box>

                                {/* Comments Section */}
                                <Box mt={4} mb={4}>
                                    <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                        {t("Comments")}
                                    </Typography>
                                    <Paper
                                        elevation={0}
                                        sx={{
                                            backgroundColor: getPanelBgColor(),
                                            ...paperStyle,
                                        }}
                                    >
                                        <CommentContainer
                                            forceAddCommentOpen={isAddCommentOpen}
                                            language={language}
                                            objectId={codeVersion?.id ?? ""}
                                            objectType={CommentFor.ResourceVersion}
                                            onAddCommentClose={closeAddCommentDialog}
                                        />
                                    </Paper>
                                </Box>
                            </>
                        )}
                    </Formik>
                </Container>
            </ScrollBox>

            {/* Edit button (if canUpdate), positioned at bottom corner of screen */}
            <SideActionsButtons display={display}>
                {permissions.canUpdate && (
                    <IconButton
                        aria-label={t("UpdateSmartContract")}
                        onClick={handleEditClick}
                    >
                        <IconCommon name="Edit" />
                    </IconButton>
                )}
            </SideActionsButtons>
        </PageContainer>
    );
}
