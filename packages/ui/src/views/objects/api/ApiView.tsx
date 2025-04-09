import { ApiVersion, BookmarkFor, CodeLanguage, ResourceList as ResourceListType, Tag, endpointsApiVersion, getTranslation } from "@local/shared";
import {
    Avatar,
    Box,
    Button,
    Collapse,
    Container,
    Grid,
    IconButton,
    LinearProgress,
    Paper,
    Stack,
    TextField,
    Tooltip,
    Typography,
    useTheme
} from "@mui/material";
import { MouseEvent, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { PageContainer } from "../../../components/Page/Page.js";
import { BookmarkButton } from "../../../components/buttons/BookmarkButton.js";
import { ReportsLink } from "../../../components/buttons/ReportsLink/ReportsLink.js";
import { ShareButton } from "../../../components/buttons/ShareButton/ShareButton.js";
import { ObjectActionMenu } from "../../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { CodeInputBase } from "../../../components/inputs/CodeInput/CodeInput.js";
import { ResourceList } from "../../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { useWindowSize } from "../../../hooks/useWindowSize.js";
import { IconCommon, IconFavicon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { ScrollBox } from "../../../styles.js";
import { placeholderColor } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { ApiViewProps } from "./types.js";

// Function to parse OpenAPI/Swagger schema
const parseOpenAPISchema = (schemaText: string): any => {
    try {
        // For now, just try to parse as JSON
        // When js-yaml is installed, this can be replaced with proper YAML parsing
        if (typeof schemaText === 'string') {
            try {
                if (schemaText.trim().startsWith('{')) {
                    // It's JSON
                    return JSON.parse(schemaText);
                } else {
                    // It's YAML - for demo purposes, parse using mock OpenAPI data
                    // This is a placeholder until js-yaml is installed
                    const mockParsed = {
                        openapi: "3.0.0",
                        paths: {
                            "/comments": {
                                get: {
                                    summary: "List all comments",
                                    description: "Returns a list of comments",
                                    parameters: [
                                        {
                                            name: "limit",
                                            in: "query",
                                            description: "Maximum number of items to return",
                                            required: false
                                        },
                                        {
                                            name: "offset",
                                            in: "query",
                                            description: "Number of items to skip",
                                            required: false
                                        }
                                    ],
                                    responses: {
                                        "200": {
                                            description: "A list of comments"
                                        }
                                    }
                                },
                                post: {
                                    summary: "Create a new comment",
                                    description: "Creates a new comment in the database",
                                    responses: {
                                        "201": {
                                            description: "Comment created successfully"
                                        },
                                        "400": {
                                            description: "Invalid input data"
                                        }
                                    }
                                }
                            },
                            "/comments/{id}": {
                                get: {
                                    summary: "Get comment by ID",
                                    description: "Returns a single comment",
                                    parameters: [
                                        {
                                            name: "id",
                                            in: "path",
                                            description: "Comment ID",
                                            required: true
                                        }
                                    ],
                                    responses: {
                                        "200": {
                                            description: "A comment object"
                                        },
                                        "404": {
                                            description: "Comment not found"
                                        }
                                    }
                                },
                                put: {
                                    summary: "Update a comment",
                                    description: "Updates an existing comment",
                                    parameters: [
                                        {
                                            name: "id",
                                            in: "path",
                                            description: "Comment ID",
                                            required: true
                                        }
                                    ],
                                    responses: {
                                        "200": {
                                            description: "Comment updated successfully"
                                        },
                                        "404": {
                                            description: "Comment not found"
                                        }
                                    }
                                },
                                delete: {
                                    summary: "Delete a comment",
                                    description: "Deletes an existing comment",
                                    parameters: [
                                        {
                                            name: "id",
                                            in: "path",
                                            description: "Comment ID",
                                            required: true
                                        }
                                    ],
                                    responses: {
                                        "204": {
                                            description: "Comment deleted successfully"
                                        },
                                        "404": {
                                            description: "Comment not found"
                                        }
                                    }
                                }
                            }
                        }
                    };
                    return mockParsed;
                }
            } catch (e) {
                console.error('Error parsing schema:', e);
                return null;
            }
        }
        return schemaText;
    } catch (error) {
        console.error('Failed to parse schema:', error);
        return null;
    }
};

// Function to extract endpoints from parsed schema
const extractEndpoints = (schema: any): Array<{
    method: string;
    path: string;
    summary: string;
    description?: string;
    parameters?: any[];
    responses?: any;
}> => {
    if (!schema || !schema.paths) return [];

    const endpoints: Array<{
        method: string;
        path: string;
        summary: string;
        description?: string;
        parameters?: any[];
        responses?: any;
    }> = [];

    // Loop through all paths and methods
    Object.entries(schema.paths).forEach(([path, pathObj]: [string, any]) => {
        Object.entries(pathObj).forEach(([method, methodObj]: [string, any]) => {
            // Only include HTTP methods, not parameters like parameters, servers, etc.
            if (['get', 'post', 'put', 'delete', 'patch', 'options', 'head'].includes(method.toLowerCase())) {
                endpoints.push({
                    path,
                    method: method.toUpperCase(),
                    summary: methodObj.summary || '',
                    description: methodObj.description,
                    parameters: methodObj.parameters,
                    responses: methodObj.responses
                });
            }
        });
    });

    return endpoints;
};

export function ApiView({
    display,
    onClose,
}: ApiViewProps) {
    const session = useContext(SessionContext);
    const { palette, breakpoints } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const profileColors = useMemo(() => placeholderColor(), []);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const isMobile = useWindowSize(({ width }) => width <= breakpoints.values.sm);

    // State for endpoints and UI interaction
    const [parsedEndpoints, setParsedEndpoints] = useState<Array<{
        method: string;
        path: string;
        summary: string;
        description?: string;
        parameters?: any[];
        responses?: any;
    }>>([]);
    const [expandedEndpoint, setExpandedEndpoint] = useState<number | null>(null);
    const [selectedEndpoint, setSelectedEndpoint] = useState<any>(null);
    const [requestParams, setRequestParams] = useState<Record<string, string>>({});
    const [responseData, setResponseData] = useState<any>(null);

    const { id, isLoading, object: apiVersion, permissions, setObject: setApiVersion } = useManagedObject<ApiVersion>({
        ...endpointsApiVersion.findOne,
        objectType: "ApiVersion",
    });

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
            callLink: apiVersion?.callLink
        };
    }, [language, apiVersion]);

    // Parse schema when it changes
    useEffect(() => {
        if (!schemaText) return;

        try {
            const parsedSchema = parseOpenAPISchema(schemaText);
            if (parsedSchema) {
                const extractedEndpoints = extractEndpoints(parsedSchema);
                setParsedEndpoints(extractedEndpoints);
            }
        } catch (error) {
            console.error('Error parsing schema:', error);
        }
    }, [schemaText]);

    const resources = useMemo(() => (resourceList || permissions.canUpdate) ? (
        <ResourceList
            horizontal={false}
            list={resourceList as any}
            canUpdate={permissions.canUpdate}
            handleUpdate={(updatedList) => {
                if (!apiVersion) return;
                setApiVersion({
                    ...apiVersion,
                    resourceList: updatedList,
                });
            }}
            loading={isLoading}
            mutate={true}
            parent={{ __typename: "ApiVersion", id: apiVersion?.id ?? "" }}
        />
    ) : null, [resourceList, permissions.canUpdate, isLoading, apiVersion, setApiVersion]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
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
    const executeRequest = async (endpoint: any, params: Record<string, string>) => {
        if (!callLink) return;

        try {
            setResponseData({ loading: true });

            // Build the URL with path parameters
            let url = new URL(callLink).origin;
            let path = endpoint.path;

            // Replace path parameters with values from requestParams
            if (endpoint.parameters) {
                endpoint.parameters.forEach((param: any) => {
                    if (param.in === 'path' && params[param.name]) {
                        path = path.replace(`{${param.name}}`, params[param.name]);
                    }
                });
            }

            url += path;

            // Add query parameters
            const queryParams = new URLSearchParams();
            if (endpoint.parameters) {
                endpoint.parameters.forEach((param: any) => {
                    if (param.in === 'query' && params[param.name]) {
                        queryParams.append(param.name, params[param.name]);
                    }
                });
            }

            if (queryParams.toString()) {
                url += `?${queryParams.toString()}`;
            }

            // Mock response for now - in a real implementation, this would make an actual API call
            setTimeout(() => {
                setResponseData({
                    status: 200,
                    statusText: 'OK',
                    headers: {
                        'content-type': 'application/json'
                    },
                    data: {
                        success: true,
                        message: `Mock response for ${endpoint.method} ${endpoint.path}`,
                        params: params
                    }
                });
            }, 1000);
        } catch (error) {
            setResponseData({
                error: true,
                message: error instanceof Error ? error.message : 'Unknown error'
            });
        }
    };

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

    // Mock data for the view - in a real implementation, this would come from the API
    const mockMetrics = {
        totalViews: apiVersion?.root?.views || 2314,
        routinesCount: apiVersion?.root?.versionsCount || 392,
        lastUpdated: apiVersion?.versionLabel || "1.0"
    };

    // Get the tags from the API version's root
    const tags = useMemo(() => {
        return apiVersion?.root?.tags || [];
    }, [apiVersion?.root?.tags]);

    return (
        <PageContainer size="fullSize">
            <ScrollBox>
                <TopBar
                    display={display}
                    onClose={onClose}
                    title={firstString(name, t("Api", { count: 1 }))}
                />
                {/* Popup menu displayed when "More" ellipsis pressed */}
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={moreMenuAnchor}
                    object={apiVersion as any}
                    onClose={closeMoreMenu}
                />

                {/* Top section with API icon, title, and metadata */}
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
                            {/* API Icon - use IconFavicon if callLink is a valid URL, otherwise use IconCommon */}
                            <Avatar
                                sx={{
                                    backgroundColor: profileColors[0],
                                    color: profileColors[1],
                                    boxShadow: 2,
                                    width: { xs: 80, sm: 100 },
                                    height: { xs: 80, sm: 100 },
                                    fontSize: { xs: 40, sm: 50 },
                                    display: "flex",
                                    justifyContent: "center",
                                    alignItems: "center",
                                }}
                            >
                                {isValidUrl && callLink ? (
                                    <IconFavicon
                                        href={callLink}
                                        size={40}
                                        fallbackIcon={{ name: "Api", type: "Common" }}
                                        decorative
                                    />
                                ) : (
                                    <IconCommon
                                        decorative
                                        fill="white"
                                        name="Api"
                                        size={40}
                                    />
                                )}
                            </Avatar>

                            {/* API Title and Metadata */}
                            <Stack flex={1} spacing={1}>
                                {isLoading ? (
                                    <LinearProgress color="inherit" sx={{ width: "50%" }} />
                                ) : (
                                    <>
                                        <Stack direction="row" justifyContent="space-between" alignItems="center">
                                            <Typography variant="h4" fontWeight="bold" component="div">
                                                {name || "ApiView"}
                                            </Typography>
                                            <Tooltip title={t("MoreOptions")}>
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
                                            </Tooltip>
                                        </Stack>

                                        <Typography variant="body1" color="textSecondary">
                                            {summary || "Connect an API to read and comment."}
                                        </Typography>

                                        {/* Metrics */}
                                        <Stack
                                            direction={{ xs: "column", sm: "row" }}
                                            spacing={{ xs: 1, sm: 4 }}
                                            mt={1}
                                        >
                                            <Typography variant="body2" fontWeight="medium">
                                                TOTAL VIEWS <Typography component="span" variant="h6" fontWeight="bold">{mockMetrics.totalViews}</Typography>
                                            </Typography>
                                            <Typography variant="body2" fontWeight="medium">
                                                ROUTINES <Typography component="span" variant="h6" fontWeight="bold">{mockMetrics.routinesCount}</Typography>
                                            </Typography>
                                            <Typography variant="body2" fontWeight="medium">
                                                LAST UPDATED <Typography component="span" variant="h6" fontWeight="bold">{mockMetrics.lastUpdated}</Typography>
                                            </Typography>
                                        </Stack>

                                        {/* Category Tags - Using TagList component */}
                                        {tags.length > 0 && (
                                            <Box mt={1}>
                                                <TagList
                                                    tags={tags as Tag[]}
                                                    maxCharacters={200}
                                                    parentId={apiVersion?.id || ""}
                                                    sx={{
                                                        maxWidth: "100%",
                                                        flexWrap: "wrap"
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
                                                onChange={(isBookmarked: boolean) => { }}
                                            />
                                        </Stack>
                                    </>
                                )}
                            </Stack>
                        </Stack>
                    </Container>
                </Box>

                <Container maxWidth="lg" sx={{ py: 4 }}>
                    {/* On desktop: Schema and Endpoints side by side. On mobile: Stacked */}
                    <Grid container spacing={4}>
                        {/* Schema Section using CodeInputBase */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Schema
                            </Typography>
                            {isLoading ? (
                                <LinearProgress color="inherit" />
                            ) : (
                                <CodeInputBase
                                    codeLanguage={schemaLanguage || CodeLanguage.Yaml}
                                    content={schemaText || ""}
                                    disabled={true}
                                    handleCodeLanguageChange={() => { }}
                                    handleContentChange={() => { }}
                                    name="schema"
                                />
                            )}
                        </Grid>

                        {/* Endpoints Section */}
                        <Grid item xs={12} md={isMobile ? 12 : 6}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Endpoints
                            </Typography>
                            <Paper
                                elevation={0}
                                sx={{
                                    backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                    p: 3,
                                    borderRadius: 2,
                                    overflow: "auto",
                                    maxHeight: 500,
                                }}
                            >
                                {isLoading ? (
                                    <LinearProgress color="inherit" />
                                ) : parsedEndpoints.length > 0 ? (
                                    <Stack spacing={2}>
                                        {parsedEndpoints.map((endpoint, index) => (
                                            <Box
                                                key={index}
                                                sx={{
                                                    borderBottom: index < parsedEndpoints.length - 1 ? `1px solid ${palette.divider}` : 'none',
                                                    pb: 2
                                                }}
                                            >
                                                <Stack
                                                    direction="row"
                                                    spacing={1}
                                                    onClick={() => setExpandedEndpoint(expandedEndpoint === index ? null : index)}
                                                    sx={{ cursor: 'pointer', alignItems: 'center' }}
                                                >
                                                    <Box
                                                        component="span"
                                                        sx={{
                                                            color: getMethodColor(endpoint.method, palette.mode),
                                                            width: 80,
                                                            display: "inline-block",
                                                            fontFamily: "monospace",
                                                        }}
                                                    >
                                                        {endpoint.method}
                                                    </Box>
                                                    <Typography variant="body1" component="span" sx={{ fontFamily: "monospace", flex: 1 }}>
                                                        {endpoint.path}
                                                    </Typography>
                                                    <IconButton size="small">
                                                        <IconCommon
                                                            name={expandedEndpoint === index ? "ArrowUp" : "ArrowDown"}
                                                            decorative
                                                            size={16}
                                                        />
                                                    </IconButton>
                                                </Stack>

                                                <Collapse in={expandedEndpoint === index}>
                                                    <Box sx={{ ml: 10, mt: 1 }}>
                                                        {endpoint.summary && (
                                                            <Typography variant="body2" sx={{ mb: 1 }}>
                                                                {endpoint.summary}
                                                            </Typography>
                                                        )}

                                                        {endpoint.description && (
                                                            <Typography variant="body2" sx={{ mb: 1, color: 'text.secondary' }}>
                                                                {endpoint.description}
                                                            </Typography>
                                                        )}

                                                        {endpoint.parameters && endpoint.parameters.length > 0 && (
                                                            <>
                                                                <Typography variant="subtitle2" sx={{ mt: 1 }}>Parameters:</Typography>
                                                                {endpoint.parameters.map((param: any, pIndex: number) => (
                                                                    <Typography key={pIndex} variant="body2" sx={{ ml: 2 }}>
                                                                        <Box component="span" sx={{ fontWeight: 'bold' }}>{param.name}</Box>
                                                                        {param.required && <Box component="span" sx={{ color: 'error.main' }}> (required)</Box>}
                                                                        {param.description && ` - ${param.description}`}
                                                                    </Typography>
                                                                ))}
                                                            </>
                                                        )}

                                                        {endpoint.responses && Object.keys(endpoint.responses).length > 0 && (
                                                            <>
                                                                <Typography variant="subtitle2" sx={{ mt: 1 }}>Responses:</Typography>
                                                                {Object.entries(endpoint.responses).map(([code, response]: [string, any], rIndex: number) => (
                                                                    <Typography key={rIndex} variant="body2" sx={{ ml: 2 }}>
                                                                        <Box component="span" sx={{ fontWeight: 'bold' }}>{code}</Box>
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
                                                                setSelectedEndpoint(endpoint);
                                                                setRequestParams({});
                                                                setResponseData(null);
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
                                    <Typography variant="body2" color="text.secondary">
                                        No endpoints found in schema or schema could not be parsed.
                                    </Typography>
                                )}
                            </Paper>

                            {/* "Try It" Section */}
                            {selectedEndpoint && (
                                <Paper
                                    elevation={0}
                                    sx={{
                                        backgroundColor: palette.mode === "light" ? "#f5f7fa" : "#1e1e1e",
                                        p: 3,
                                        borderRadius: 2,
                                        mt: 3,
                                    }}
                                >
                                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                                        <Typography variant="h6">Test Endpoint</Typography>
                                        <IconButton
                                            size="small"
                                            onClick={() => setSelectedEndpoint(null)}
                                        >
                                            <IconCommon name="Close" size={16} decorative />
                                        </IconButton>
                                    </Box>
                                    <Typography
                                        variant="body1"
                                        sx={{
                                            mb: 2,
                                            fontFamily: 'monospace',
                                            color: getMethodColor(selectedEndpoint.method, palette.mode),
                                            fontWeight: 'bold'
                                        }}
                                    >
                                        {selectedEndpoint.method} {selectedEndpoint.path}
                                    </Typography>

                                    <form onSubmit={(e) => {
                                        e.preventDefault();
                                        executeRequest(selectedEndpoint, requestParams);
                                    }}>
                                        {selectedEndpoint.parameters?.filter((p: any) => p.in === 'path' || p.in === 'query').map((param: any, index: number) => (
                                            <TextField
                                                key={index}
                                                label={`${param.name}${param.required ? ' *' : ''}`}
                                                variant="outlined"
                                                size="small"
                                                fullWidth
                                                margin="dense"
                                                onChange={(e) => setRequestParams({
                                                    ...requestParams,
                                                    [param.name]: e.target.value
                                                })}
                                                helperText={param.description}
                                                required={param.required}
                                                value={requestParams[param.name] || ''}
                                            />
                                        ))}

                                        <Button
                                            type="submit"
                                            variant="contained"
                                            sx={{ mt: 2 }}
                                            disabled={responseData?.loading}
                                        >
                                            {responseData?.loading ? 'Executing...' : 'Execute'}
                                        </Button>
                                    </form>

                                    {responseData && !responseData.loading && (
                                        <Box sx={{ mt: 3 }}>
                                            <Typography variant="subtitle2" gutterBottom>Response:</Typography>
                                            <CodeInputBase
                                                codeLanguage={CodeLanguage.Json}
                                                content={JSON.stringify(responseData, null, 2)}
                                                disabled={true}
                                                handleCodeLanguageChange={() => { }}
                                                handleContentChange={() => { }}
                                                name="response"
                                            />
                                        </Box>
                                    )}
                                </Paper>
                            )}
                        </Grid>
                    </Grid>

                    {/* Resource List - if needed */}
                    {resources && (
                        <Box mt={4}>
                            <Typography variant="h5" component="h2" fontWeight="bold" mb={2}>
                                Resources
                            </Typography>
                            {resources}
                        </Box>
                    )}
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
        default:
            return mode === "light" ? "#9e9e9e" : "#e0e0e0"; // Gray
    }
}
