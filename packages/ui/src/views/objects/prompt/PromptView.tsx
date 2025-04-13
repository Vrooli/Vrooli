import { BookmarkFor, CommentFor, ResourceList as ResourceListType, StandardShape, StandardVersion, Tag, endpointsStandardVersion, getTranslation } from "@local/shared";
import {
    Box,
    Button,
    Container,
    Grid,
    IconButton,
    LinearProgress,
    Paper,
    Stack,
    Tooltip,
    Typography,
    useTheme
} from "@mui/material";
import { Formik } from "formik";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink.js";
import { ShareButton } from "../../../components/buttons/ShareButton.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { AdvancedInput } from "../../../components/inputs/AdvancedInput/AdvancedInput.js";
import { ResourceList } from "../../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { MarkdownDisplay } from "../../../components/text/MarkdownDisplay.js";
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
import { PromptViewProps } from "./types.js";

export function PromptView({
    display,
    onClose,
}: PromptViewProps) {
    const session = useContext(SessionContext);
    const { palette, breakpoints } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(false);
    const [startMessage, setStartMessage] = useState<string>("");

    const { id, isLoading, object: standardVersion, permissions, setObject: setStandardVersion } = useManagedObject<StandardVersion>({
        ...endpointsStandardVersion.findOne,
        objectType: "StandardVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (standardVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [standardVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canBookmark, name, resourceList, root } = useMemo(() => {
        const { canBookmark } = standardVersion?.root?.you ?? {};
        const resourceList: ResourceListType | null | undefined = standardVersion?.resourceList;
        const { name } = getTranslation(standardVersion, [language]);
        return {
            canBookmark,
            name,
            resourceList,
            root: standardVersion?.root as StandardShape,
        };
    }, [language, standardVersion]);

    // Get description and content separately to avoid linting errors
    const description = useMemo(() => {
        const translations = getTranslation(standardVersion, [language]);
        return translations.description ?? "";
    }, [standardVersion, language]);

    const content = useMemo(() => {
        return standardVersion?.props ?? ""; // Using props instead of content which doesn't exist
    }, [standardVersion]);

    // Callbacks for comment section
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: standardVersion,
        objectType: "StandardVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setStandardVersion,
    });

    // Resources section
    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceList
            horizontal
            list={resourceList as any}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!standardVersion) return;
                setStandardVersion({
                    ...standardVersion,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            parent={{ __typename: "StandardVersion", id: standardVersion?.id ?? "" }}
        />
    ) : null, [resourceList, permissions.canUpdate, isLoading, standardVersion, setStandardVersion]);

    // Tags
    const tags = useMemo(() => {
        return root?.tags || [];
    }, [root?.tags]);

    // Handle message input changes
    const handleStartMessageChange = useCallback((value: string) => {
        setStartMessage(value);
    }, []);

    // Handle the create chat action
    const handleCreateChat = useCallback(() => {
        // Placeholder for actual implementation
        console.log("Creating new chat with prompt:", content);
        console.log("Starting message:", startMessage);
        // Would implement actual chat creation/navigation here
    }, [content, startMessage]);

    // Handle the add to system message action
    const handleAddToSystemMessage = useCallback(() => {
        // Placeholder for actual implementation
        console.log("Adding to system message in new chat:", content);
        // Would implement actual system message addition here
    }, [content]);

    // Handle the customize (fork) action
    const handleCustomize = useCallback(() => {
        // Placeholder for actual implementation
        console.log("Customizing prompt (forking):", content);
        // Would implement actual forking/customization here
        actionData.onActionStart(ObjectAction.Fork);
    }, [content, actionData]);

    // Handle the add as context action
    const handleAddAsContext = useCallback(() => {
        // Placeholder for actual implementation
        console.log("Adding as context to active chat:", content);
        // Would implement actual context addition here
    }, [content]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={firstString(name, t("Prompt", { count: 1 }))}
                    titleBehavior="Hide"
                />
                {/* Popup menu displayed when "More" ellipsis pressed */}
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={moreMenuAnchor}
                    object={standardVersion as any}
                    onClose={closeMoreMenu}
                />

                {/* Top section with Prompt icon, title, and metadata */}
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
                            {/* Prompt Icon */}
                            <IconCommon
                                decorative
                                name="Article"
                                size={80}
                                style={{
                                    margin: 10,
                                    filter: `drop-shadow(0px 2px 4px rgba(0, 0, 0, 0.2))`,
                                }}
                            />

                            {/* Prompt Title and Metadata */}
                            <Stack flex={1} spacing={1}>
                                {isLoading ? (
                                    <LinearProgress color="inherit" sx={{ width: "50%" }} />
                                ) : (
                                    <>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                                            <Typography variant="h4" fontWeight="bold" component="div">
                                                {name || "Prompt"}
                                            </Typography>
                                            {actionData.availableActions.length > 0 && <Tooltip title={t("MoreOptions")}>
                                                <IconButton
                                                    aria-label={t("MoreOptions")}
                                                    size="small"
                                                    onClick={openMoreMenu}
                                                >
                                                    <IconCommon
                                                        decorative
                                                        fill={palette.text.secondary}
                                                        name="Ellipsis"
                                                    />
                                                </IconButton>
                                            </Tooltip>}
                                        </Stack>

                                        <Typography variant="body1" color="textSecondary">
                                            {"A prompt for generating text with AI models."}
                                        </Typography>

                                        {/* Metrics */}
                                        <Stack
                                            direction="row"
                                            spacing={{ xs: 1, sm: 4 }}
                                            mt={1}
                                        >
                                            <Tooltip title={t("View", { count: standardVersion?.root?.views || 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconCommon
                                                        decorative
                                                        name="Visible"
                                                        size={20}
                                                        fill={palette.text.secondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {standardVersion?.root?.views || 0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                            <Tooltip title={t("Routine", { count: 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconRoutine
                                                        decorative
                                                        name="Routine"
                                                        size={20}
                                                        fill={palette.text.secondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                            <Tooltip title={t("Version")}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <Typography variant="body2" fontWeight="medium">v</Typography>
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {standardVersion?.versionLabel || "1.0"}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                        </Stack>

                                        {/* Tags */}
                                        {tags.length > 0 && (
                                            <Box mt={1}>
                                                <TagList
                                                    tags={tags as Tag[]}
                                                    maxCharacters={200}
                                                    sx={{
                                                        maxWidth: "100%",
                                                        flexWrap: "wrap",
                                                    }}
                                                />
                                            </Box>
                                        )}

                                        {/* Action Buttons */}
                                        <Stack direction="row" spacing={2} mt={1}>
                                            <ShareButton object={standardVersion} />
                                            <ReportsLink object={standardVersion} />
                                            <BookmarkButton
                                                disabled={!canBookmark}
                                                objectId={standardVersion?.id ?? ""}
                                                bookmarkFor={BookmarkFor.Standard}
                                                isBookmarked={standardVersion?.root?.you?.isBookmarked ?? false}
                                                bookmarks={standardVersion?.root?.bookmarks ?? 0}
                                                onChange={() => { }}
                                            />
                                        </Stack>
                                    </>
                                )}
                            </Stack>
                        </Stack>
                    </Container>
                </Box>

                <Container maxWidth="lg">
                    {/* Description Section */}
                    {description && (
                        <Box mt={4}>
                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                    p: 3,
                                    borderRadius: 2,
                                }}
                            >
                                <MarkdownDisplay content={description} />
                            </Paper>
                        </Box>
                    )}

                    {/* Resources Section */}
                    {resources && (
                        <Box mt={4} m={2}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Resources
                            </Typography>
                            {resources}
                        </Box>
                    )}
                    {/* Prompt Content and Start a Conversation Sections in side-by-side layout */}
                    <Grid container spacing={4} mt={4} mb={4}>
                        {/* Prompt Content Section */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Prompt Content
                            </Typography>
                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                    p: 3,
                                    borderRadius: 2,
                                    height: "100%",
                                }}
                            >
                                <MarkdownDisplay content={content} />
                            </Paper>
                        </Grid>

                        {/* Start Message Input Section */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Start a Conversation
                            </Typography>
                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                    p: 3,
                                    borderRadius: 2,
                                    height: "100%",
                                    display: "flex",
                                    flexDirection: "column",
                                }}
                            >
                                <Typography variant="body1" mb={2}>
                                    Enter a message to start the conversation with this prompt:
                                </Typography>
                                <Box sx={{ flex: 1, minHeight: "150px" }}>
                                    <Formik
                                        initialValues={{ startMessage: "" }}
                                        onSubmit={() => { }}
                                    >
                                        {() => (
                                            <AdvancedInput
                                                name="startMessage"
                                                features={{
                                                    allowFormatting: true,
                                                    allowSubmit: false,
                                                }}
                                            />
                                        )}
                                    </Formik>
                                </Box>
                                <Stack
                                    direction={{ xs: "column", sm: "row" }}
                                    spacing={2}
                                    mt={3}
                                    justifyContent="flex-end"
                                    flexWrap="wrap"
                                >
                                    <Button
                                        variant="outlined"
                                        startIcon={<IconCommon name="Add" />}
                                        onClick={handleAddToSystemMessage}
                                        sx={{ mb: { xs: 1, sm: 0 } }}
                                    >
                                        Add to System Message
                                    </Button>
                                    <Button
                                        variant="outlined"
                                        startIcon={<IconCommon name="Add" />}
                                        onClick={handleAddAsContext}
                                        sx={{ mb: { xs: 1, sm: 0 } }}
                                    >
                                        Add as Context
                                    </Button>
                                    <Button
                                        variant="outlined"
                                        startIcon={<IconCommon name="Edit" />}
                                        onClick={handleCustomize}
                                        sx={{ mb: { xs: 1, sm: 0 } }}
                                    >
                                        Customize
                                    </Button>
                                    <Button
                                        variant="contained"
                                        color="primary"
                                        startIcon={<IconCommon name="Message" />}
                                        onClick={handleCreateChat}
                                    >
                                        Create Chat
                                    </Button>
                                </Stack>
                            </Paper>
                        </Grid>
                    </Grid>
                    {/* Comments Section */}
                    <Box mb={4} mt={4}>
                        <Paper
                            elevation={0}
                            sx={{
                                backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                p: 3,
                                borderRadius: 2,
                            }}
                        >
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Comments
                            </Typography>
                            <CommentContainer
                                forceAddCommentOpen={isAddCommentOpen}
                                language={language}
                                objectId={standardVersion?.id ?? ""}
                                objectType={CommentFor.StandardVersion}
                                onAddCommentClose={closeAddCommentDialog}
                            />
                        </Paper>
                    </Box>
                </Container>
            </ScrollBox>

            {/* Edit button - fix position to bottom right */}
            {permissions.canUpdate && (
                <Box
                    sx={{
                        position: 'fixed',
                        bottom: 16,
                        right: 16,
                        zIndex: 10,
                    }}
                >
                    <IconButton
                        aria-label={t("UpdatePrompt")}
                        onClick={() => actionData.onActionStart(ObjectAction.Edit)}
                        sx={{
                            backgroundColor: palette.primary.main,
                            color: palette.primary.contrastText,
                            '&:hover': {
                                backgroundColor: palette.primary.dark,
                            },
                            width: 56,
                            height: 56,
                        }}
                    >
                        <IconCommon name="Edit" />
                    </IconButton>
                </Box>
            )}
        </PageContainer>
    );
}
