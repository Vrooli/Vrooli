import { Box, Button, Collapse, Container, Grid, IconButton, LinearProgress, Paper, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { BookmarkFor, CodeLanguage, CodeVersionConfig, CommentFor, endpointsCodeVersion, getTranslation, type CodeVersion, type ResourceList as ResourceListType, type Tag } from "@vrooli/shared";
import { useCallback, useContext, useEffect, useMemo, useState, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink.js";
import { ShareButton } from "../../../components/buttons/ShareButton.js";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { CodeInputBase } from "../../../components/inputs/CodeInput/CodeInput.js";
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
import { type DataConverterViewProps } from "./types.js";

const RANDOM_PASS_THRESHOLD = 0.3;
const RANDOM_ERROR_THRESHOLD = 0.7;
const TEST_SIMULATION_DELAY = 1000;
const TEST_BATCH_DELAY = 500;
const TAGS_MAX_CHARS = 200;

const contextActionsExcluded = [
    ObjectAction.Edit,
    ObjectAction.Delete,
    ObjectAction.Fork,
    ObjectAction.Report,
    ObjectAction.Bookmark,
    ObjectAction.Comment,
];

// Only include the languages that exist in the CodeLanguage enum
const testDefaultInputs = {
    [CodeLanguage.Javascript]: "{}",
    [CodeLanguage.Python]: "{}",
    [CodeLanguage.Go]: "{}",
    [CodeLanguage.Rust]: "{}",
    [CodeLanguage.Swift]: "{}",
    [CodeLanguage.Solidity]: "{}",
    [CodeLanguage.Cpp]: "{}",
    [CodeLanguage.Ruby]: "{}",
    [CodeLanguage.Java]: "{}",
    [CodeLanguage.Haskell]: "{}",
};

// Style constants
const paperStyle = {
    p: 3,
    borderRadius: 2,
} as const;

const testOutputStyle = {
    p: 1,
    maxHeight: 150,
    overflow: "auto",
} as const;

const preStyle = {
    whiteSpace: "pre-wrap",
    margin: 0,
} as const;

export function DataConverterView({
    display,
    onClose,
}: DataConverterViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const { palette, breakpoints } = useTheme();
    const [, setLocation] = useLocation();
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    // State for test cases and results
    const [testResults, setTestResults] = useState<any[]>([]);
    const [isRunningTest, setIsRunningTest] = useState<boolean>(false);
    const [expandedTestCase, setExpandedTestCase] = useState<number | null>(null);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [isAddCommentOpen, setIsAddCommentOpen] = useState<boolean>(false);

    // Use managed object hook to handle API calls and caching
    const { id, isLoading, object: codeVersion, permissions, setObject: setCodeVersion } = useManagedObject<CodeVersion>({
        ...endpointsCodeVersion.findOne,
        objectType: "CodeVersion",
    });

    // Get available languages from translations
    const availableLanguages = useMemo<string[]>(() =>
        (codeVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []),
        [codeVersion?.translations]);

    // Set preferred language
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    // Extract data from code version
    const { canBookmark, description, name, resourceList, tags, codeLimitTo } = useMemo(() => {
        const { canBookmark } = codeVersion?.root?.you ?? {};
        const resourceList: ResourceListType | null | undefined = codeVersion?.resourceList;
        const { description, name } = getTranslation(codeVersion, [language]);
        const tags = codeVersion?.root?.tags || [];
        return {
            description,
            canBookmark,
            name,
            resourceList,
            tags,
            codeLimitTo: codeVersion?.codeLanguage as CodeLanguage | undefined,
        };
    }, [language, codeVersion]);

    // Configuration for the data converter
    const codeConfig = useMemo(() => {
        if (!codeVersion) return null;

        return CodeVersionConfig.parse({
            data: codeVersion.data,
            content: codeVersion.content,
            codeLanguage: codeVersion.codeLanguage,
        }, console);
    }, [codeVersion]);

    const testCases = useMemo(() => {
        return codeConfig?.testCases || [];
    }, [codeConfig]);

    // Callbacks
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    // Handle resources display
    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceList
            horizontal
            list={resourceList as any}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!codeVersion) return;
                setCodeVersion({
                    ...codeVersion,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            parent={{ __typename: "CodeVersion", id: codeVersion?.id ?? "" }}
        />
    ) : null, [resourceList, permissions.canUpdate, isLoading, codeVersion, setCodeVersion]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<null | HTMLElement>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<HTMLElement>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: codeVersion,
        objectType: "CodeVersion",
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

    // Get the background color for code/output blocks based on the current theme
    const getCodeBgColor = useCallback(() => {
        return palette.mode === "light" ? "#f0f0f0" : "#2a2a2a";
    }, [palette.mode]);

    // Function to run a test case
    const runTest = useCallback(async (testCase: any, index: number) => {
        if (!codeConfig) return;

        setIsRunningTest(true);

        try {
            // Simulation of running a test in sandbox
            // In a real implementation, this would call the sandbox API
            await new Promise(resolve => setTimeout(resolve, TEST_SIMULATION_DELAY));

            const result = {
                description: testCase.description,
                passed: Math.random() > RANDOM_PASS_THRESHOLD, // Simulate random pass/fail for demo
                actualOutput: testCase.expectedOutput,
                error: Math.random() > RANDOM_ERROR_THRESHOLD ? "Mock error message" : undefined,
            };

            setTestResults(prev => {
                const newResults = [...prev];
                newResults[index] = result;
                return newResults;
            });
        } catch (error) {
            console.error("Test execution error:", error);
            setTestResults(prev => {
                const newResults = [...prev];
                newResults[index] = {
                    description: testCase.description,
                    passed: false,
                    error: error instanceof Error ? error.message : "Unknown error",
                };
                return newResults;
            });
        } finally {
            setIsRunningTest(false);
        }
    }, [codeConfig]);

    // Function to run all test cases
    const runAllTests = useCallback(async () => {
        if (!codeConfig || !testCases.length) return;

        setIsRunningTest(true);
        setTestResults([]);

        try {
            // Run all tests in sequence
            const results: any[] = [];
            for (let i = 0; i < testCases.length; i++) {
                // Simulate test execution
                await new Promise(resolve => setTimeout(resolve, TEST_BATCH_DELAY));

                const result = {
                    description: testCases[i].description,
                    passed: Math.random() > RANDOM_PASS_THRESHOLD, // Simulate random pass/fail for demo
                    actualOutput: testCases[i].expectedOutput,
                    error: Math.random() > RANDOM_ERROR_THRESHOLD ? "Mock error message" : undefined,
                };

                results.push(result);
            }

            setTestResults(results);
        } catch (error) {
            console.error("Tests execution error:", error);
        } finally {
            setIsRunningTest(false);
        }
    }, [codeConfig, testCases]);

    // Create handlers ahead of time to avoid inline functions in JSX
    const handleEditClick = useCallback(() => actionData.onActionStart(ObjectAction.Edit), [actionData]);

    // Prepare test case run handlers to avoid inline function creation
    const testCaseRunHandlers = useMemo(() =>
        testCases.map((_, index) => () => runTest(testCases[index], index)),
        [testCases, runTest]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={firstString(name, t("DataConverter", { count: 1 }))}
                    titleBehavior="Hide"
                />
                {/* Popup menu displayed when "More" ellipsis pressed */}
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={moreMenuAnchor}
                    object={codeVersion as any}
                    onClose={closeMoreMenu}
                />

                {/* Top section with Data Converter icon, title, and metadata */}
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
                            {/* Data Converter Icon */}
                            <IconCommon
                                decorative
                                name="Terminal"
                                size={80}
                                style={{
                                    margin: 10,
                                    filter: "drop-shadow(0px 2px 4px rgba(0, 0, 0, 0.2))",
                                }}
                            />

                            {/* Data Converter Title and Metadata */}
                            <Stack flex={1} spacing={1}>
                                {isLoading ? (
                                    <LinearProgress color="inherit" sx={{ width: "50%" }} />
                                ) : (
                                    <>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                                            <Typography variant="h4" fontWeight="bold" component="div">
                                                {name || "Data Converter"}
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
                                            A data converter transforms data from one format to another
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
                                        </Stack>

                                        {/* Tags */}
                                        {tags.length > 0 && (
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
                    {/* Details Section */}
                    {description && (
                        <Box mt={4}>
                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: getPanelBgColor(),
                                    p: 3,
                                    borderRadius: 2,
                                }}
                            >
                                <MarkdownDisplay content={description} headingLevelOffset={2} />
                            </Paper>
                        </Box>
                    )}

                    {/* Resources Section */}
                    {resources && (
                        <Box mt={4} mx={2}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Resources
                            </Typography>
                            {resources}
                        </Box>
                    )}
                    {/* Code and Test Cases in grid layout */}
                    <Grid container spacing={4} mt={4} mb={4}>
                        {/* Code Section */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Code
                            </Typography>
                            {isLoading ? (
                                <LinearProgress color="inherit" />
                            ) : (
                                <CodeInputBase
                                    codeLanguage={codeLimitTo || CodeLanguage.Javascript}
                                    content={codeVersion?.content || ""}
                                    disabled={true}
                                    handleCodeLanguageChange={() => { }}
                                    handleContentChange={() => { }}
                                    name="content"
                                />
                            )}
                        </Grid>

                        {/* Test Cases Section */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2}>
                                <Typography variant="h5" component="h2" fontWeight="bold">
                                    Test Cases
                                </Typography>
                                {testCases.length > 0 && (
                                    <Button
                                        variant="contained"
                                        color="primary"
                                        startIcon={<IconCommon name="Play" />}
                                        onClick={runAllTests}
                                        disabled={isRunningTest}
                                    >
                                        {t("RunAllTests")}
                                    </Button>
                                )}
                            </Stack>

                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: getPanelBgColor(),
                                    p: 3,
                                    borderRadius: 2,
                                    overflow: "auto",
                                    maxHeight: 500,
                                }}
                            >
                                {isLoading ? (
                                    <LinearProgress color="inherit" />
                                ) : testCases.length > 0 ? (
                                    <Stack spacing={2}>
                                        {testCases.map((testCase, index) => (
                                            <Box
                                                key={index}
                                                sx={{
                                                    borderBottom: index < testCases.length - 1 ? `1px solid ${palette.divider}` : "none",
                                                    pb: 2,
                                                }}
                                            >
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    onClick={() => setExpandedTestCase(expandedTestCase === index ? null : index)}
                                                    sx={{
                                                        cursor: "pointer",
                                                        alignItems: "center",
                                                        bgcolor: testResults[index]?.passed !== undefined
                                                            ? testResults[index]?.passed
                                                                ? "success.light"
                                                                : "error.light"
                                                            : "transparent",
                                                        p: 1,
                                                        borderRadius: 1,
                                                    }}
                                                >
                                                    <Typography
                                                        variant="subtitle1"
                                                        fontWeight="bold"
                                                        sx={{ flex: 1 }}
                                                    >
                                                        {testCase.description || `Test Case ${index + 1}`}
                                                    </Typography>
                                                    {testResults[index] && (
                                                        <Typography
                                                            variant="body2"
                                                            color={testResults[index].passed ? "success.main" : "error.main"}
                                                            sx={{ mr: 1, fontWeight: "bold" }}
                                                        >
                                                            {testResults[index].passed ? "PASSED" : "FAILED"}
                                                        </Typography>
                                                    )}
                                                    <IconButton size="small">
                                                        <IconCommon
                                                            name={expandedTestCase === index ? "ArrowDropUp" : "ArrowDropDown"}
                                                            decorative
                                                            size={16}
                                                        />
                                                    </IconButton>
                                                </Stack>

                                                <Collapse in={expandedTestCase === index}>
                                                    <Box sx={{ mt: 2 }}>
                                                        <Grid container spacing={2}>
                                                            <Grid item xs={12} md={6}>
                                                                <Typography variant="subtitle2">
                                                                    Input:
                                                                </Typography>
                                                                <Paper
                                                                    variant="outlined"
                                                                    sx={{
                                                                        ...testOutputStyle,
                                                                        backgroundColor: getCodeBgColor(),
                                                                    }}
                                                                >
                                                                    <Typography
                                                                        variant="body2"
                                                                        component="pre"
                                                                        sx={preStyle}
                                                                    >
                                                                        {JSON.stringify(testCase.input, null, 2)}
                                                                    </Typography>
                                                                </Paper>
                                                            </Grid>
                                                            <Grid item xs={12} md={6}>
                                                                <Typography variant="subtitle2">
                                                                    Expected Output:
                                                                </Typography>
                                                                <Paper
                                                                    variant="outlined"
                                                                    sx={{
                                                                        ...testOutputStyle,
                                                                        backgroundColor: getCodeBgColor(),
                                                                    }}
                                                                >
                                                                    <Typography
                                                                        variant="body2"
                                                                        component="pre"
                                                                        sx={preStyle}
                                                                    >
                                                                        {JSON.stringify(testCase.expectedOutput, null, 2)}
                                                                    </Typography>
                                                                </Paper>
                                                            </Grid>
                                                        </Grid>

                                                        {testResults[index] && (
                                                            <Box mt={2}>
                                                                {testResults[index].error && (
                                                                    <Typography variant="body2" color="error.main">
                                                                        {testResults[index].error}
                                                                    </Typography>
                                                                )}

                                                                {testResults[index].actualOutput && !testResults[index].passed && (
                                                                    <Box mt={1}>
                                                                        <Typography variant="subtitle2">
                                                                            Actual Output:
                                                                        </Typography>
                                                                        <Paper
                                                                            variant="outlined"
                                                                            sx={{
                                                                                ...testOutputStyle,
                                                                                backgroundColor: getCodeBgColor(),
                                                                            }}
                                                                        >
                                                                            <Typography
                                                                                variant="body2"
                                                                                component="pre"
                                                                                sx={preStyle}
                                                                            >
                                                                                {JSON.stringify(testResults[index].actualOutput, null, 2)}
                                                                            </Typography>
                                                                        </Paper>
                                                                    </Box>
                                                                )}
                                                            </Box>
                                                        )}
                                                    </Box>
                                                </Collapse>
                                            </Box>
                                        ))}
                                    </Stack>
                                ) : (
                                    <Stack spacing={2} alignItems="center" p={2}>
                                        <Typography variant="body1" color="text.secondary" align="center">
                                            No test cases available.
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" align="center">
                                            This could be because:
                                        </Typography>
                                        <Box component="ul" sx={{ color: "text.secondary", mt: 0 }}>
                                            <li>The data converter doesn't have any test cases defined</li>
                                            <li>The test cases are defined in an unsupported format</li>
                                            <li>There was an error loading the test cases</li>
                                        </Box>
                                        <Typography variant="body2" color="text.secondary" align="center">
                                            Add test cases when editing the data converter.
                                        </Typography>
                                    </Stack>
                                )}
                            </Paper>
                        </Grid>
                    </Grid>
                    {/* Comments Section */}
                    <Box mb={4} mt={4}>
                        <Paper
                            elevation={0}
                            sx={{
                                backgroundColor: getPanelBgColor(),
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
                                objectId={codeVersion?.id ?? ""}
                                objectType={CommentFor.CodeVersion}
                                onAddCommentClose={closeAddCommentDialog}
                            />
                        </Paper>
                    </Box>
                </Container>
            </ScrollBox>

            {/* Edit button (if canUpdate), positioned at bottom corner of screen */}
            <SideActionsButtons display={display}>
                {permissions.canUpdate && (
                    <IconButton
                        aria-label={t("UpdateDataConverter")}
                        onClick={() => actionData.onActionStart(ObjectAction.Edit)}
                    >
                        <IconCommon name="Edit" />
                    </IconButton>
                )}
            </SideActionsButtons>
        </PageContainer>
    );
}
