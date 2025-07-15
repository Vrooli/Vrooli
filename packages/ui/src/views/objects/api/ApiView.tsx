// AI_CHECK: TYPE_SAFETY=fixed-api-schema-types | LAST: 2025-06-28
import $RefParser from "@apidevtools/json-schema-ref-parser";
import Avatar from "@mui/material/Avatar";
import Box from "@mui/material/Box";
import Button from "@mui/material/Button";
import Chip from "@mui/material/Chip";
import Collapse from "@mui/material/Collapse";
import Container from "@mui/material/Container";
import Grid from "@mui/material/Grid";
import LinearProgress from "@mui/material/LinearProgress";
import Paper from "@mui/material/Paper";
import Stack from "@mui/material/Stack";
import TextField from "@mui/material/TextField";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { BookmarkFor, CodeLanguage, getTranslation, type ResourceList as ResourceListType, type ResourceVersion, type Tag } from "@vrooli/shared";
import yaml from "js-yaml";
import { useCallback, useContext, useEffect, useMemo, useState, type MouseEvent } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { Tooltip } from "../../../components/Tooltip/Tooltip.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { IconButton } from "../../../components/buttons/IconButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink.js";
import { ShareButton } from "../../../components/buttons/ShareButton.js";
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
import { IconCommon, IconFavicon, IconRoutine } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ScrollBox } from "../../../styles.js";
import { placeholderColor } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { type ApiEndpoint, type ApiParameter, type ApiRequestParams, type ApiResponseData, type ApiSchema, type ApiViewProps } from "./types.js";

/**
 * Function to parse and process API schemas in JSON or YAML format
 * Supports OpenAPI/Swagger, AsyncAPI, and other schema formats
 * Handles schema references ($ref) through json-schema-ref-parser
 */
async function parseOpenAPISchema(schemaText: string): Promise<ApiSchema | null> {
    try {
        if (!schemaText || typeof schemaText !== "string") {
            console.error("Invalid schema text provided");
            return null;
        }

        // Parse the schema based on format (JSON or YAML)
        let parsedSchema;
        try {
            // Try parsing as JSON first
            if (schemaText.trim().startsWith("{")) {
                parsedSchema = JSON.parse(schemaText);
            } else {
                // If not JSON, try parsing as YAML
                parsedSchema = yaml.load(schemaText);
            }
        } catch (e) {
            console.error("Error parsing schema as JSON or YAML:", e);
            return null;
        }

        if (!parsedSchema) {
            console.error("Failed to parse schema");
            return null;
        }

        // Identify schema type (OpenAPI, AsyncAPI, etc.)
        const schemaType = identifySchemaType(parsedSchema);

        // Dereference JSON schema references
        try {
            // This will resolve all $ref pointers in the schema
            const dereferencedSchema = await $RefParser.dereference(parsedSchema, {
                continueOnError: true, // Continue even if some references can't be resolved
                dereference: {
                    circular: true, // Handle circular references
                },
            });

            console.log(`Successfully dereferenced ${schemaType} schema`);
            return dereferencedSchema;
        } catch (e) {
            console.error("Error dereferencing schema references:", e);
            // If dereferencing fails, return the original parsed schema
            return parsedSchema;
        }
    } catch (error) {
        console.error("Failed to process schema:", error);
        return null;
    }
}

/**
 * Identifies the type of API schema (OpenAPI/Swagger, AsyncAPI, etc.)
 */
function identifySchemaType(schema: ApiSchema): string {
    if (!schema) return "Unknown";

    if (schema.openapi) {
        return `OpenAPI ${schema.openapi}`;
    } else if (schema.swagger) {
        return `Swagger ${schema.swagger}`;
    } else if (schema.asyncapi) {
        return `AsyncAPI ${schema.asyncapi}`;
    } else if (schema.jsonSchema) {
        return `JSON Schema ${schema.jsonSchema}`;
    } else {
        return "Generic API Schema";
    }
}

// Function to extract endpoints from parsed schema
function extractEndpoints(schema: ApiSchema): ApiEndpoint[] {
    // Return empty array if schema is not valid
    if (!schema) return [];

    const endpoints: ApiEndpoint[] = [];

    // Handle OpenAPI/Swagger format
    if (schema.paths) {
        // Loop through all paths and methods
        Object.entries(schema.paths).forEach(([path, pathObj]) => {
            if (typeof pathObj === "object" && pathObj !== null) {
                Object.entries(pathObj).forEach(([method, methodObj]) => {
                    // Only include HTTP methods, not parameters like parameters, servers, etc.
                    if (["get", "post", "put", "delete", "patch", "options", "head"].includes(method.toLowerCase()) &&
                        typeof methodObj === "object" && methodObj !== null) {
                        const operation = methodObj as {
                            summary?: string;
                            description?: string;
                            parameters?: ApiParameter[];
                            responses?: Record<string, unknown>
                        };
                        endpoints.push({
                            path,
                            method: method.toUpperCase(),
                            summary: operation.summary || "",
                            description: operation.description,
                            parameters: operation.parameters,
                            responses: operation.responses,
                        });
                    }
                });
            }
        });
    }

    // Handle AsyncAPI format
    else if (schema.channels) {
        // Loop through all channels
        Object.entries(schema.channels).forEach(([path, channelObj]) => {
            // Check for publish operations (subscriber perspective)
            if (typeof channelObj === "object" && channelObj !== null && "publish" in channelObj) {
                const channel = channelObj as { publish?: { summary?: string; description?: string; message?: { payload?: unknown } } };
                if (channel.publish) {
                    endpoints.push({
                        path,
                        method: "SUBSCRIBE",
                        summary: channel.publish.summary || "",
                        description: channel.publish.description,
                        parameters: extractAsyncApiParameters(channelObj),
                        responses: {
                            "message": {
                                description: "Received message",
                                payload: channel.publish.message?.payload,
                            },
                        },
                    });
                }
            }

            // Check for subscribe operations (publisher perspective)
            if (typeof channelObj === "object" && channelObj !== null && "subscribe" in channelObj) {
                const channel = channelObj as { subscribe?: { summary?: string; description?: string; message?: { payload?: unknown } } };
                if (channel.subscribe) {
                    endpoints.push({
                        path,
                        method: "PUBLISH",
                        summary: channel.subscribe.summary || "",
                        description: channel.subscribe.description,
                        parameters: extractAsyncApiParameters(channelObj),
                        responses: {
                            "message": {
                                description: "Published message",
                                payload: channel.subscribe.message?.payload,
                            },
                        },
                    });
                }
            }
        });
    }

    return endpoints;
}

/**
 * Helper function to extract parameters from AsyncAPI channels
 */
function extractAsyncApiParameters(channelObj: unknown): ApiParameter[] {
    const parameters: ApiParameter[] = [];

    // Extract parameters from channel parameters
    if (typeof channelObj === "object" && channelObj !== null && "parameters" in channelObj) {
        const channel = channelObj as { parameters?: Record<string, unknown> };
        const channelParams = channel.parameters;
        if (typeof channelParams === "object" && channelParams !== null) {
            Object.entries(channelParams).forEach(([name, paramObj]) => {
                if (typeof paramObj === "object" && paramObj !== null) {
                    const param = paramObj as { description?: unknown; schema?: unknown };
                    parameters.push({
                        name,
                        in: "path" as const,
                        description: typeof param.description === "string" ? param.description : undefined,
                        required: true,
                        schema: param.schema as Record<string, unknown> | undefined,
                    });
                }
            });
        }
    }

    return parameters;
}

const GRID_SIZES = {
    FULL_WIDTH: 12,
    HALF_WIDTH: 6,
} as const;

const ICON_SIZES = {
    AVATAR_XS: 80,
    AVATAR_SM: 100,
    ICON_AVATAR_XS: 40,
    ICON_AVATAR_SM: 50,
    ICON_STANDARD: 80,
    ICON_SMALL: 20,
    ICON_TINY: 16,
} as const;

const UI_CONSTANTS = {
    MAX_ENDPOINTS_HEIGHT: 500,
} as const;

export function ApiView({
    display,
    onClose,
}: ApiViewProps) {
    const session = useContext(SessionContext);
    const { palette, breakpoints } = useTheme();
    const { t } = useTranslation();
    const [{ pathname }, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    // State for schema type and endpoints
    const [schemaType, setSchemaType] = useState<string>("Unknown");
    const [parsedEndpoints, setParsedEndpoints] = useState<ApiEndpoint[]>([]);
    const [expandedEndpoint, setExpandedEndpoint] = useState<number | null>(null);
    const [selectedEndpoint, setSelectedEndpoint] = useState<ApiEndpoint | null>(null);
    const [requestParams, setRequestParams] = useState<ApiRequestParams>({});
    const [responseData, setResponseData] = useState<ApiResponseData | null>(null);

    const { isLoading, object: apiVersion, permissions, setObject: setApiVersion } = useManagedObject<ResourceVersion>({ pathname });

    const availableLanguages = useMemo<string[]>(() => (apiVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [apiVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canBookmark, details, name, resourceList, summary, schemaText, schemaLanguage, callLink } = useMemo(() => {
        const { canBookmark } = apiVersion?.root?.you ?? {};
        const resourceList: ResourceListType | null | undefined = apiVersion?.resourceList;
        const { details, name, summary } = getTranslation(apiVersion, [language]);
        return {
            details,
            canBookmark,
            name,
            resourceList,
            summary,
            schemaText: apiVersion?.schemaText,
            schemaLanguage: apiVersion?.schemaLanguage as CodeLanguage | undefined,
            callLink: apiVersion?.callLink,
        };
    }, [language, apiVersion]);

    // Parse schema when it changes
    useEffect(() => {
        if (!schemaText) return;

        // Create async function to use inside useEffect
        async function parseSchema() {
            try {
                // Call the async parse function
                const parsedSchema = await parseOpenAPISchema(schemaText);
                if (parsedSchema) {
                    const extractedEndpoints = extractEndpoints(parsedSchema);
                    setParsedEndpoints(extractedEndpoints);

                    // Determine and set the schema type
                    const detectedType = identifySchemaType(parsedSchema);
                    setSchemaType(detectedType);
                }
            } catch (error) {
                console.error("Error parsing schema:", error);
            }
        }

        // Call the async function
        parseSchema();
    }, [schemaText]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceList
            horizontal
            list={resourceList ?? undefined}
            canUpdate={permissions.canUpdate}
            handleUpdate={handleResourceListUpdate}
            loading={isLoading}
            mutate={true}
            parent={{ __typename: "ApiVersion", id: apiVersion?.id ?? "" }}
        />
    ) : null, [resourceList, permissions.canUpdate, handleResourceListUpdate, isLoading, apiVersion?.id]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<HTMLElement | null>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<HTMLElement>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const actionData = useObjectActions({
        object: apiVersion,
        objectType: "ApiVersion",
        setLocation,
        setObject: setApiVersion,
    });

    // Function to execute an API request
    const executeRequest = useCallback(async (endpoint: ApiEndpoint, params: ApiRequestParams) => {
        if (!callLink) return;

        try {
            setResponseData({ loading: true });

            // Build the URL with path parameters
            const _url = new URL(callLink).origin;
            let path = endpoint.path;

            // Replace path parameters with values from requestParams
            if (endpoint.parameters) {
                endpoint.parameters.forEach((param) => {
                    if (param.in === "path" && params[param.name]) {
                        path = path.replace(`{${param.name}}`, params[param.name]);
                    }
                });
            }

            url += path;

            // Add query parameters
            const queryParams = new URLSearchParams();
            if (endpoint.parameters) {
                endpoint.parameters.forEach((param) => {
                    if (param.in === "query" && params[param.name]) {
                        queryParams.append(param.name, params[param.name]);
                    }
                });
            }

            if (queryParams.toString()) {
                url += `?${queryParams.toString()}`;
            }

            // Mock response for now - in a real implementation, this would make an actual API call
            const MOCK_DELAY_MS = 1000;
            setTimeout(() => {
                setResponseData({
                    status: 200,
                    statusText: "OK",
                    headers: {
                        "content-type": "application/json",
                    },
                    data: {
                        success: true,
                        message: `Mock response for ${endpoint.method} ${endpoint.path}`,
                        params,
                    },
                });
            }, MOCK_DELAY_MS);
        } catch (error) {
            setResponseData({
                error: true,
                message: error instanceof Error ? error.message : "Unknown error",
            });
        }
    }, [callLink]);

    // Check if we can extract a valid URL from the callLink
    const isValidUrl = useMemo(() => {
        if (!callLink) return false;
        try {
            new URL(callLink);
            return true;
        } catch (e) {
            return false;
        }
    }, [callLink]);

    // Get the tags from the API version's root
    const tags = useMemo(() => {
        return apiVersion?.root?.tags || [];
    }, [apiVersion?.root?.tags]);

    // Memoized style objects
    const headerBoxStyle = useMemo(() => ({
        background: palette.mode === "light" ? "#b2b3b3" : "#303030",
        paddingY: 3,
        position: "relative" as const,
    }), [palette.mode]);

    const avatarStyle = useMemo(() => ({
        backgroundColor: profileColors[0],
        color: profileColors[1],
        boxShadow: 2,
        width: { xs: ICON_SIZES.AVATAR_XS, sm: ICON_SIZES.AVATAR_SM },
        height: { xs: ICON_SIZES.AVATAR_XS, sm: ICON_SIZES.AVATAR_SM },
        fontSize: { xs: ICON_SIZES.ICON_AVATAR_XS, sm: ICON_SIZES.ICON_AVATAR_SM },
        display: "flex",
        justifyContent: "center",
        alignItems: "center",
        "& svg": {
            width: "85%",
            height: "85%",
        },
    }), [profileColors]);

    const paperStyle = useMemo(() => ({
        backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
        p: 3,
        borderRadius: 2,
    }), [palette.mode]);

    const endpointsPaperStyle = useMemo(() => ({
        ...paperStyle,
        overflow: "auto",
        maxHeight: UI_CONSTANTS.MAX_ENDPOINTS_HEIGHT,
    }), [paperStyle]);

    // More memoized styles
    const stackStyle = useMemo(() => ({ cursor: "pointer", alignItems: "center" }), []);
    const monoFontStyle = useMemo(() => ({ fontFamily: "monospace", flex: 1 }), []);
    const endpointTestHeaderStyle = useMemo(() => ({
        display: "flex",
        justifyContent: "space-between",
        alignItems: "center",
        mb: 2,
    }), []);
    const langSelectorBoxStyle = useMemo(() => ({
        position: "absolute" as const,
        top: 8,
        right: 8,
    }), []);
    const linearProgressStyle = useMemo(() => ({ width: "50%" }), []);
    const colorTextSecondaryStyle = useMemo(() => ({ color: "text.secondary", mt: 0 }), []);
    const mb1Style = useMemo(() => ({ mb: 1 }), []);
    const mb1ColorSecondaryStyle = useMemo(() => ({ mb: 1, color: "text.secondary" }), []);
    const ml10Mt1Style = useMemo(() => ({ ml: 10, mt: 1 }), []);
    const ml2Style = useMemo(() => ({ ml: 2 }), []);
    const mt1Style = useMemo(() => ({ mt: 1 }), []);
    const mt2Style = useMemo(() => ({ mt: 2 }), []);
    const mt3Style = useMemo(() => ({ mt: 3 }), []);
    const mt4Style = useMemo(() => ({ mt: 4 }), []);
    const m2Style = useMemo(() => ({ m: 2 }), []);
    const monospaceStyle = useMemo(() => ({ fontFamily: "monospace" }), []);

    // Create a function to get method color style
    const getMethodColorStyle = useCallback((method: string) => ({
        color: getMethodColor(method, palette.mode),
        width: 80,
        display: "inline-block",
        fontFamily: "monospace",
    }), [palette.mode]);

    // Create endpoint divider style function
    const getEndpointDividerStyle = useCallback((index: number, length: number) => ({
        borderBottom: index < length - 1 ? `1px solid ${palette.divider}` : "none",
        pb: 2,
    }), [palette.divider]);

    // Memoized handlers
    const handleExpandEndpoint = useCallback((index: number) => {
        setExpandedEndpoint(expandedEndpoint === index ? null : index);
    }, [expandedEndpoint]);

    const handleSelectEndpoint = useCallback((endpoint: ApiEndpoint) => {
        setSelectedEndpoint(endpoint);
        setRequestParams({});
        setResponseData(null);
    }, []);

    const handleCloseEndpoint = useCallback(() => {
        setSelectedEndpoint(null);
    }, []);

    const handleResourceListUpdate = useCallback((updatedList: ResourceListType) => {
        if (!apiVersion) return;
        setApiVersion({
            ...apiVersion,
            resourceList: updatedList,
        });
    }, [apiVersion, setApiVersion]);

    const handleBookmarkChange = useCallback(() => {
        // Bookmark change is handled internally by BookmarkButton
    }, []);

    const handleCodeLanguageChange = useCallback(() => {
        // Language change is disabled for this read-only view
    }, []);

    const handleContentChange = useCallback(() => {
        // Content change is disabled for this read-only view
    }, []);

    const handleRequestParamChange = useCallback((paramName: string, value: string) => {
        setRequestParams({
            ...requestParams,
            [paramName]: value,
        });
    }, [requestParams]);

    const handleFormSubmit = useCallback((e: React.FormEvent) => {
        e.preventDefault();
        if (selectedEndpoint) {
            executeRequest(selectedEndpoint, requestParams);
        }
    }, [selectedEndpoint, requestParams, executeRequest]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={firstString(name, t("Api", { count: 1 }))}
                    titleBehavior="Hide"
                />
                {/* Popup menu displayed when "More" ellipsis pressed */}
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={moreMenuAnchor}
                    object={apiVersion}
                    onClose={closeMoreMenu}
                />

                {/* Top section with API icon, title, and metadata */}
                <Box sx={headerBoxStyle}>
                    {/* Language selector */}
                    <Box sx={langSelectorBoxStyle}>
                        {availableLanguages.length > 1 && <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            languages={availableLanguages}
                        />}
                    </Box>

                    <Container maxWidth="lg">
                        <Stack direction={{ xs: "column", sm: "row" }} spacing={3} alignItems="center">
                            {/* API Icon - use IconFavicon directly if callLink is valid, otherwise use IconCommon in an Avatar */}
                            {isValidUrl && callLink ? (
                                <IconFavicon
                                    href={callLink}
                                    size={ICON_SIZES.ICON_STANDARD}
                                    fallbackIcon={{ name: "Api", type: "Common" }}
                                    decorative
                                    style={{
                                        margin: 2, // Add a small margin to maintain consistent spacing with avatar
                                    }}
                                />
                            ) : (
                                <Avatar
                                    sx={avatarStyle}
                                >
                                    <IconCommon
                                        decorative
                                        fill="white"
                                        name="Api"
                                        size={ICON_SIZES.ICON_STANDARD}
                                    />
                                </Avatar>
                            )}

                            {/* API Title and Metadata */}
                            <Stack flex={1} spacing={1}>
                                {isLoading ? (
                                    <LinearProgress color="inherit" sx={linearProgressStyle} />
                                ) : (
                                    <>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                                            <Typography variant="h4" fontWeight="bold" component="div">
                                                {name || "ApiView"}
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
                                            {summary || "Connect an API to read and comment."}
                                        </Typography>

                                        {/* Metrics */}
                                        <Stack
                                            direction="row"
                                            spacing={{ xs: 1, sm: 4 }}
                                            mt={1}
                                        >
                                            <Tooltip title={t("View", { count: apiVersion?.root?.views || 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconCommon
                                                        decorative
                                                        name="Visible"
                                                        size={ICON_SIZES.ICON_SMALL}
                                                        fill={palette.background.textSecondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {apiVersion?.root?.views || 0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                            <Tooltip title={t("Routine", { count: 0 })}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <IconRoutine
                                                        decorative
                                                        name="Routine"
                                                        size={ICON_SIZES.ICON_SMALL}
                                                        fill={palette.background.textSecondary}
                                                    />
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {apiVersion?.calledByRoutineVersionsCount || 0}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                            <Tooltip title={t("Version")}>
                                                <Stack direction="row" alignItems="center" spacing={1}>
                                                    <Typography variant="body2" fontWeight="medium">v</Typography>
                                                    <Typography variant="body1" fontWeight="bold">
                                                        {apiVersion?.versionLabel || "1.0"}
                                                    </Typography>
                                                </Stack>
                                            </Tooltip>
                                        </Stack>

                                        {/* Category Tags - Using TagList component */}
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
                                            <ShareButton object={apiVersion} />
                                            <ReportsLink object={apiVersion} />
                                            <BookmarkButton
                                                disabled={!canBookmark}
                                                objectId={apiVersion?.id ?? ""}
                                                bookmarkFor={BookmarkFor.Api}
                                                isBookmarked={apiVersion?.root?.you?.isBookmarked ?? false}
                                                bookmarks={apiVersion?.root?.bookmarks ?? 0}
                                                onChange={handleBookmarkChange}
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
                    {details && (
                        <Box sx={mt4Style}>
                            <Paper
                                elevation={0}
                                sx={paperStyle}
                            >
                                <MarkdownDisplay content={details} />
                            </Paper>
                        </Box>
                    )}

                    {resources && (
                        <Box sx={{ ...mt4Style, ...m2Style }}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Resources
                            </Typography>
                            {resources}
                        </Box>
                    )}
                    {/* On desktop: Schema and Endpoints side by side. On mobile: Stacked */}
                    <Grid container spacing={4} mt={4} mb={4}>
                        {/* Schema Section using CodeInputBase */}
                        <Grid item xs={GRID_SIZES.FULL_WIDTH} md={isMobile ? GRID_SIZES.FULL_WIDTH : GRID_SIZES.HALF_WIDTH}>
                            <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
                                <Typography variant="h5" component="h2" fontWeight="bold">
                                    Schema
                                </Typography>
                                <Chip
                                    label={schemaType}
                                    color={schemaType.includes("Unknown") ? "default" : "primary"}
                                    size="small"
                                />
                            </Stack>
                            {isLoading ? (
                                <LinearProgress color="inherit" />
                            ) : (
                                <CodeInputBase
                                    codeLanguage={schemaLanguage || CodeLanguage.Yaml}
                                    content={schemaText || ""}
                                    disabled={true}
                                    handleCodeLanguageChange={handleCodeLanguageChange}
                                    handleContentChange={handleContentChange}
                                    name="schema"
                                />
                            )}
                        </Grid>

                        {/* Endpoints Section */}
                        <Grid item xs={GRID_SIZES.FULL_WIDTH} md={isMobile ? GRID_SIZES.FULL_WIDTH : GRID_SIZES.HALF_WIDTH}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Endpoints
                            </Typography>
                            <Paper
                                elevation={0}
                                sx={endpointsPaperStyle}
                            >
                                {isLoading ? (
                                    <LinearProgress color="inherit" />
                                ) : parsedEndpoints.length > 0 ? (
                                    <Stack spacing={2}>
                                        {parsedEndpoints.map((endpoint, index) => (
                                            <Box
                                                key={index}
                                                sx={getEndpointDividerStyle(index, parsedEndpoints.length)}
                                            >
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    onClick={() => handleExpandEndpoint(index)}
                                                    sx={stackStyle}
                                                >
                                                    <Box
                                                        component="span"
                                                        sx={getMethodColorStyle(endpoint.method)}
                                                    >
                                                        {endpoint.method}
                                                    </Box>
                                                    <Typography variant="body1" component="span" sx={monoFontStyle}>
                                                        {endpoint.path}
                                                    </Typography>
                                                    <IconButton size="small">
                                                        <IconCommon
                                                            name={expandedEndpoint === index ? "ArrowDropUp" : "ArrowDropDown"}
                                                            decorative
                                                            size={ICON_SIZES.ICON_TINY}
                                                        />
                                                    </IconButton>
                                                </Stack>

                                                <Collapse in={expandedEndpoint === index}>
                                                    <Box sx={ml10Mt1Style}>
                                                        {endpoint.summary && (
                                                            <Typography variant="body2" sx={mb1Style}>
                                                                {endpoint.summary}
                                                            </Typography>
                                                        )}

                                                        {endpoint.description && (
                                                            <Typography variant="body2" sx={mb1ColorSecondaryStyle}>
                                                                {endpoint.description}
                                                            </Typography>
                                                        )}

                                                        {endpoint.parameters && endpoint.parameters.length > 0 && (
                                                            <>
                                                                <Typography variant="subtitle2" sx={{ mt: 1 }}>Parameters:</Typography>
                                                                {endpoint.parameters.map((param, pIndex: number) => (
                                                                    <Typography key={pIndex} variant="body2" sx={{ ml: 2 }}>
                                                                        <Box component="span" sx={{ fontWeight: "bold" }}>{param.name}</Box>
                                                                        {param.required && <Box component="span" sx={{ color: "error.main" }}> (required)</Box>}
                                                                        {param.description && ` - ${param.description}`}
                                                                    </Typography>
                                                                ))}
                                                            </>
                                                        )}

                                                        {endpoint.responses && Object.keys(endpoint.responses).length > 0 && (
                                                            <>
                                                                <Typography variant="subtitle2" sx={{ mt: 1 }}>Responses:</Typography>
                                                                {Object.entries(endpoint.responses).map(([code, response], rIndex: number) => (
                                                                    <Typography key={rIndex} variant="body2" sx={{ ml: 2 }}>
                                                                        <Box component="span" sx={{ fontWeight: "bold" }}>{code}</Box>
                                                                        {response.description && ` - ${response.description}`}
                                                                    </Typography>
                                                                ))}
                                                            </>
                                                        )}

                                                        <Button
                                                            variant="outlined"
                                                            size="small"
                                                            sx={{ mt: 2 }}
                                                            onClick={(e) => {
                                                                e.stopPropagation();
                                                                handleSelectEndpoint(endpoint);
                                                            }}
                                                        >
                                                            Try it
                                                        </Button>
                                                    </Box>
                                                </Collapse>
                                            </Box>
                                        ))}
                                    </Stack>
                                ) : (
                                    <Stack spacing={2} alignItems="center" p={2}>
                                        <Typography variant="body1" color="text.secondary" align="center">
                                            No endpoints found in the schema.
                                        </Typography>
                                        <Typography variant="body2" color="text.secondary" align="center">
                                            This could be because:
                                        </Typography>
                                        <Box component="ul" sx={{ color: "text.secondary", mt: 0 }}>
                                            <li>The schema format is not recognized (supported: OpenAPI, Swagger, AsyncAPI)</li>
                                            <li>The schema doesn&apos;t contain any path or channel definitions</li>
                                            <li>The schema contains syntax errors or invalid references</li>
                                        </Box>
                                        <Typography variant="body2" color="text.secondary" align="center">
                                            Check the schema content and format to ensure it&apos;s valid.
                                        </Typography>
                                    </Stack>
                                )}
                            </Paper>

                            {/* "Try It" Section */}
                            {selectedEndpoint && (
                                <Paper
                                    elevation={0}
                                    sx={{ ...paperStyle, mt: 3 }}
                                >
                                    <Box sx={{ display: "flex", justifyContent: "space-between", alignItems: "center", mb: 2 }}>
                                        <Typography variant="h6">Test Endpoint</Typography>
                                        <IconButton
                                            size="small"
                                            onClick={handleCloseEndpoint}
                                        >
                                            <IconCommon name="Close" size={ICON_SIZES.ICON_TINY} decorative />
                                        </IconButton>
                                    </Box>
                                    <Typography
                                        variant="body1"
                                        sx={{
                                            mb: 2,
                                            fontFamily: "monospace",
                                            color: getMethodColor(selectedEndpoint.method, palette.mode),
                                            fontWeight: "bold",
                                        }}
                                    >
                                        {selectedEndpoint.method} {selectedEndpoint.path}
                                    </Typography>

                                    <form onSubmit={handleFormSubmit}>
                                        {selectedEndpoint.parameters?.filter((p) => p.in === "path" || p.in === "query").map((param, index: number) => (
                                            <TextField
                                                key={index}
                                                label={`${param.name}${param.required ? " *" : ""}`}
                                                variant="outlined"
                                                size="small"
                                                fullWidth
                                                margin="dense"
                                                onChange={(e) => handleRequestParamChange(param.name, e.target.value)}
                                                helperText={param.description}
                                                required={param.required}
                                                value={requestParams[param.name] || ""}
                                            />
                                        ))}

                                        <Button
                                            type="submit"
                                            variant="contained"
                                            sx={{ mt: 2 }}
                                            disabled={responseData?.loading}
                                        >
                                            {responseData?.loading ? "Executing..." : "Execute"}
                                        </Button>
                                    </form>

                                    {responseData && !responseData.loading && (
                                        <Box sx={{ mt: 3 }}>
                                            <Typography variant="subtitle2" gutterBottom>Response:</Typography>
                                            <CodeInputBase
                                                codeLanguage={CodeLanguage.Json}
                                                content={JSON.stringify(responseData, null, 2)}
                                                disabled={true}
                                                handleCodeLanguageChange={handleCodeLanguageChange}
                                                handleContentChange={handleContentChange}
                                                name="response"
                                            />
                                        </Box>
                                    )}
                                </Paper>
                            )}
                        </Grid>
                    </Grid>
                </Container>
            </ScrollBox>
        </PageContainer>
    );
}

// Helper function to get color for HTTP method
function getMethodColor(method: string, mode: string): string {
    switch (method) {
        case "GET":
            return mode === "light" ? "#4caf50" : "#81c784"; // Green
        case "POST":
            return mode === "light" ? "#2196f3" : "#64b5f6"; // Blue
        case "PUT":
            return mode === "light" ? "#ff9800" : "#ffb74d"; // Orange
        case "DELETE":
            return mode === "light" ? "#f44336" : "#e57373"; // Red
        case "PATCH":
            return mode === "light" ? "#9c27b0" : "#ba68c8"; // Purple
        case "PUBLISH":
            return mode === "light" ? "#3f51b5" : "#7986cb"; // Indigo
        case "SUBSCRIBE":
            return mode === "light" ? "#009688" : "#4db6ac"; // Teal
        default:
            return mode === "light" ? "#9e9e9e" : "#e0e0e0"; // Gray
    }
}
