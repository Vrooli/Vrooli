import { FindByIdInput, FindVersionInput, LINKS, ListObject, Report, ReportFor, ReportSearchInput, ReportStatus, VisibilityType, endpointsApiVersion, endpointsChatMessage, endpointsCodeVersion, endpointsComment, endpointsIssue, endpointsNoteVersion, endpointsPost, endpointsProjectVersion, endpointsRoutineVersion, endpointsStandardVersion, endpointsTag, endpointsTeam, endpointsUser, getObjectUrl, noop, uuidValidate } from "@local/shared";
import { Box, Button, Typography, styled, useTheme } from "@mui/material";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ServerResponseParser } from "../../api/responseParser.js";
import { SortButton } from "../../components/buttons/SearchButtons/SearchButtons.js";
import { ListContainer } from "../../components/containers/ListContainer.js";
import { ObjectActionMenu } from "../../components/dialogs/ObjectActionMenu/ObjectActionMenu.js";
import { ObjectListItem } from "../../components/lists/ObjectList/ObjectList.js";
import { ReportListItem } from "../../components/lists/ReportListItem/ReportListItem.js";
import { ObjectListActions } from "../../components/lists/types.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { SessionContext } from "../../contexts/session.js";
import { goBack } from "../../hooks/forms.js";
import { useInfiniteScroll } from "../../hooks/gestures.js";
import { useObjectActions } from "../../hooks/objectActions.js";
import { useDimensions } from "../../hooks/useDimensions.js";
import { useFindMany } from "../../hooks/useFindMany.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { fetchDataUsingUrl } from "../../hooks/useManagedObject.js";
import { useObjectContextMenu } from "../../hooks/useObjectContextMenu.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { ScrollBox } from "../../styles.js";
import { ArgsType } from "../../types.js";
import { getCurrentUser } from "../../utils/authentication/session.js";
import { getDisplay, getYou } from "../../utils/display/listTools.js";
import { getCookiePartialData, removeCookiePartialData, setCookiePartialData } from "../../utils/localStorage.js";
import { UrlInfo, parseSingleItemUrl } from "../../utils/navigation/urlTools.js";
import { PubSub } from "../../utils/pubsub.js";
import { ReportUpsert } from "../../views/objects/report/ReportUpsert.js";

const scrollContainerId = "reports-search-scroll";
const display = "page";
const take = 20;

const reportForLinks: Record<ReportFor, string | string[]> = {
    ApiVersion: LINKS.Api,
    ChatMessage: LINKS.ChatMessage,
    CodeVersion: [LINKS.DataConverter, LINKS.SmartContract],
    Comment: LINKS.Comment,
    Issue: LINKS.Issue,
    NoteVersion: LINKS.Note,
    Post: LINKS.Post,
    ProjectVersion: LINKS.Project,
    RoutineVersion: [LINKS.RoutineMultiStep, LINKS.RoutineSingleStep],
    StandardVersion: [LINKS.DataStructure],
    Tag: LINKS.Tag,
    Team: LINKS.Team,
    User: LINKS.User,
};

type EndpointData = {
    endpoint: string;
    method: string;
};

const reportForEndpoints: Record<ReportFor, EndpointData> = {
    ApiVersion: endpointsApiVersion.findOne,
    ChatMessage: endpointsChatMessage.findOne,
    CodeVersion: endpointsCodeVersion.findOne,
    Comment: endpointsComment.findOne,
    Issue: endpointsIssue.findOne,
    NoteVersion: endpointsNoteVersion.findOne,
    Post: endpointsPost.findOne,
    ProjectVersion: endpointsProjectVersion.findOne,
    RoutineVersion: endpointsRoutineVersion.findOne,
    StandardVersion: endpointsStandardVersion.findOne,
    Tag: endpointsTag.findOne,
    Team: endpointsTeam.findOne,
    User: endpointsUser.findOne,
};

const reportForSearchFields: Record<ReportFor, keyof ReportSearchInput> = {
    ApiVersion: "apiVersionId",
    ChatMessage: "chatMessageId",
    CodeVersion: "codeVersionId",
    Comment: "commentId",
    Issue: "issueId",
    NoteVersion: "noteVersionId",
    Post: "postId",
    ProjectVersion: "projectVersionId",
    RoutineVersion: "routineVersionId",
    StandardVersion: "standardVersionId",
    Tag: "tagId",
    Team: "teamId",
    User: "userId",
};

function canNavigate() {
    return true; // Always allow navigation to object
}

const AlreadyReportedLabel = styled(Typography)(({ theme }) => ({
    marginLeft: "12px",
    marginBottom: "8px",
    display: "block",
    fontStyle: "italic",
    color: theme.palette.background.textSecondary,
}));

const reportedObjectBoxStyle = {
    marginTop: 2,
    marginBottom: 4,
} as const;

export function ReportsView() {
    const { t } = useTranslation();
    const { breakpoints, palette } = useTheme();
    const session = useContext(SessionContext);
    const { id: userId, languages } = useMemo(() => getCurrentUser(session), [session]);

    const { dimensions, ref: dimRef } = useDimensions();
    const isMobile = useMemo(() => dimensions.width <= breakpoints.values.md, [breakpoints, dimensions]);

    // Find object info from URL
    const [{ pathname }, setLocation] = useLocation();
    const {
        createdFor,
        endpointData,
        objectType,
        urlInfo,
    } = useMemo<{
        createdFor: { __typename: ReportFor, id: string } | undefined,
        endpointData: EndpointData | undefined,
        objectType: ReportFor | undefined,
        urlInfo: UrlInfo,
    }>(function parseUrlMemo() {
        const urlWithoutReports = pathname.replace(LINKS.Reports, "");
        // The rest of the URL should look like a typical single item URL
        const urlInfo = parseSingleItemUrl({ pathname: urlWithoutReports });
        // Find object type by checking if the link starts with reportForLinks[x] + "/"
        const objectType = Object.keys(reportForLinks).find(key => {
            const link = reportForLinks[key];
            return Array.isArray(link) ? link.some(l => urlWithoutReports.startsWith(l as string)) : urlWithoutReports.startsWith(link as string);
        }) as ReportFor | undefined;
        // Find endpoing to fetch object info
        const endpointData = objectType ? reportForEndpoints[objectType] : undefined;
        // Define createdFor for new reports
        const createdFor = objectType && urlInfo.id ? { __typename: objectType, id: urlInfo.id } : undefined;
        return {
            createdFor,
            endpointData,
            objectType,
            urlInfo,
        };
    }, [pathname]);

    // Fetch object info
    const [getData, { data: fetchedData, errors: fetchedErrors, loading: isLoadingObject }] = useLazyFetch<FindByIdInput | FindVersionInput, ListObject>({
        endpoint: endpointData?.endpoint,
        method: "GET",
    });
    const [object, setObject] = useState<ListObject | null>(function initializeObjectState() {
        if (!objectType || Object.keys(urlInfo).length === 0) return null;
        // Try to find object in cache
        const storedData = getCookiePartialData<ListObject>({ __typename: objectType, ...urlInfo });
        return storedData ?? null;
    });
    useEffect(function fetchObjectEffect() {
        // Try to fetch data using URL params
        const fetched = fetchDataUsingUrl(urlInfo, getData);
        if (fetched) return;
        // If fetch failed, display error
        PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
    }, [getData, urlInfo]);
    useEffect(function setObjectEffect() {
        if (!objectType) return;
        // If data was queried (i.e. object exists), store it in local state
        if (fetchedData) setCookiePartialData(fetchedData, "full");
        // If we didn't receive fetched data, and we received an "Unauthorized" error, 
        // we should clear the cookie data and set the object to its default value
        else if (ServerResponseParser.hasErrorCode({ errors: fetchedErrors }, "Unauthorized")) {
            removeCookiePartialData({ __typename: objectType, ...urlInfo });
            setObject(null);
            return;
        }
        // If we have fetched data or cached data, set the object   
        const knownData = fetchedData ?? getCookiePartialData<ListObject>({ __typename: objectType, ...urlInfo });
        if (knownData && typeof knownData === "object" && uuidValidate(knownData.id)) {
            setObject(knownData);
        }
    }, [fetchedData, fetchedErrors, objectType, urlInfo]);

    // Set up other info for the object being reported
    const onAction = useCallback((action: keyof ObjectListActions<ListObject>, ...data: unknown[]) => {
        switch (action) {
            case "Deleted": {
                // Determine fallback URL if previous page is not this app
                const fallbackUrl = object ? getObjectUrl(object) : LINKS.Home;
                // Go back to previous page if object is deleted
                goBack(setLocation, fallbackUrl);
                break;
            }
            case "Updated": {
                const [updatedItem] = data as ArgsType<ObjectListActions<ListObject>["Updated"]>;
                setObject(updatedItem);
                break;
            }
        }
    }, [object, setLocation]);
    const contextData = useObjectContextMenu();
    const actionData = useObjectActions({
        canNavigate, // Allow navigation to object
        isListReorderable: false, // Not a list
        objectType,
        onAction,
        setLocation,
        ...contextData,
    });

    const { canReport, title } = useMemo(() => {
        const { title } = getDisplay(object, languages, palette);
        const { canReport } = getYou(object);
        return {
            canReport,
            title,
        };
    }, [languages, object, palette]);

    const [statusFilter, setStatusFilter] = useState<ReportStatus | "All">("All");
    const [expandedReports, setExpandedReports] = useState<Set<string>>(new Set());
    // Fetch reports
    const findManyData = useFindMany<Report>({
        canSearch: () => typeof objectType === "string",
        controlsUrl: display === "page",
        searchType: "Report",
        take,
        where: objectType ? {
            [reportForSearchFields[objectType]]: object?.id,
            // status: statusFilter !== "All" ? [statusFilter] : undefined,
            visibility: VisibilityType.OwnOrPublic,
        } : {
            visibility: VisibilityType.OwnOrPublic,
        },
    });

    useInfiniteScroll({
        loading: findManyData.loading,
        loadMore: findManyData.loadMore,
        scrollContainerId,
    });

    // Fetch reports you've submtited separately
    const yourReportsData = useFindMany<Report>({
        canSearch: () => typeof objectType === "string" && !!object?.id && !!userId,
        controlsUrl: false,
        searchType: "Report",
        where: objectType && object?.id && userId ? {
            [reportForSearchFields[objectType]]: object?.id,
            sortBy: findManyData.sortBy,
            visibility: VisibilityType.Own,
        } : {},
    });
    const yourOpenReports = useMemo<Report[]>(function haveAlreadyReportedMemo() {
        return yourReportsData.allData.filter(report => report.you.isOwn === true && report.status === ReportStatus.Open);
    }, [yourReportsData.allData]);

    const combinedReports = useMemo<Report[]>(function combineReportsMemo() {
        return [...yourReportsData.allData, ...findManyData.allData.filter(report => report.you.isOwn === false)];
    }, [findManyData.allData, yourReportsData.allData]);


    const [showReportUpsert, setShowReportUpsert] = useState(false);
    const closeReportUpsert = useCallback(function closeReportUpsertCallback() {
        setShowReportUpsert(false);
    }, []);
    const completedReportUpsert = useCallback(function completedReportUpsertCallback(data: Report) {
        setShowReportUpsert(false);
        yourReportsData.addItem(data);
    }, []);
    function handleAddReport() {
        setShowReportUpsert(true);
    }

    return (
        <ScrollBox id={scrollContainerId} ref={dimRef}>
            <TopBar
                display={display}
                tabTitle={title ? `${t("Report", { count: 2 })} - ${title}` : t("Report", { count: 2 })}
                title={t("Report", { count: 2 })}
                titleBehaviorDesktop={"ShowBelow"}
            />
            <ListContainer
                borderRadius={2}
                isEmpty={false}
                sx={reportedObjectBoxStyle}
            >
                <ObjectActionMenu
                    actionData={actionData}
                    anchorEl={contextData.anchorEl}
                    object={contextData.object}
                    onClose={contextData.closeContextMenu}
                />
                {!isLoadingObject && object && <ObjectListItem
                    canNavigate={canNavigate}
                    data={object}
                    handleContextMenu={contextData.handleContextMenu}
                    handleToggleSelect={noop} // Disable selection
                    hideUpdateButton={false}
                    isMobile={isMobile}
                    isSelecting={false} // Disable selection
                    isSelected={false} // Disable selection
                    loading={false}
                    objectType={object.__typename}
                    onAction={onAction}
                />}
                {isLoadingObject && <ObjectListItem
                    data={null}
                    handleContextMenu={noop}
                    handleToggleSelect={noop}
                    hideUpdateButton={false}
                    isMobile={isMobile}
                    isSelecting={false}
                    isSelected={false}
                    loading={true}
                    objectType={"RoutineVersion"} // Can be any object type
                    onAction={noop}
                />}
            </ListContainer>
            <Box p={1}>
                {yourOpenReports.length > 0 && <AlreadyReportedLabel>You already have an open report on this object. You will not be able to submit another.</AlreadyReportedLabel>}
                {yourOpenReports.length === 0 && canReport && <Button
                    color="primary"
                    fullWidth
                    onClick={handleAddReport}
                    startIcon={<IconCommon
                        decorative
                        name="Report"
                    />}
                    variant="contained"
                >
                    {t("AddReport")}
                </Button>}
            </Box>

            {/* Sorting and Filtering */}
            <Box display="flex" justifyContent="center" alignItems="center" marginBottom={1}>
                <SortButton
                    options={findManyData.sortByOptions}
                    setSortBy={findManyData.setSortBy}
                    sortBy={findManyData.sortBy}
                />
                {/* <FormControl>
                    <InputLabel>{t("FilterByStatus")}</InputLabel>
                    <Select
                        value={statusFilter}
                        onChange={e => setStatusFilter(e.target.value as ReportStatus | "All")}
                    >
                        <MenuItem value="All">{t("AllStatuses")}</MenuItem>
                        <MenuItem value={ReportStatus.Open}>{t("Open")}</MenuItem>
                        <MenuItem value={ReportStatus.ClosedDeleted}>{t("ClosedDeleted")}</MenuItem>
                        <MenuItem value={ReportStatus.ClosedFalseReport}>{t("ClosedFalseReport")}</MenuItem>
                        <MenuItem value={ReportStatus.ClosedHidden}>{t("ClosedHidden")}</MenuItem>
                        <MenuItem value={ReportStatus.ClosedNonIssue}>{t("ClosedNonIssue")}</MenuItem>
                        <MenuItem value={ReportStatus.ClosedSuspended}>{t("ClosedSuspended")}</MenuItem>
                    </Select>
                </FormControl> */}
            </Box>

            {/* Reports List */}
            <ListContainer
                id={`${scrollContainerId}-list`}
                borderRadius={2}
                emptyText={t("NoResults", { ns: "error" })}
                isEmpty={combinedReports.length === 0 && !findManyData.loading}
            >
                {combinedReports.map(report => (
                    <ReportListItem
                        key={report.id}
                        canNavigate={canNavigate}
                        data={report}
                        handleContextMenu={noop} // Disable context menu
                        handleToggleSelect={noop} // Disable selection
                        hideUpdateButton={true}
                        isMobile={isMobile}
                        isSelecting={false} // Disable selection
                        isSelected={false} // Disable selection
                        loading={false}
                        objectType={"Report"}
                        onAction={noop} // Disable actions
                    />
                ))}
            </ListContainer>

            {/* Report Upsert Dialog */}
            {showReportUpsert && objectType && createdFor && (
                <ReportUpsert
                    createdFor={createdFor}
                    display={"dialog"}
                    isCreate={true}
                    isOpen={showReportUpsert}
                    onCancel={closeReportUpsert}
                    onClose={closeReportUpsert}
                    onCompleted={completedReportUpsert}
                    onDeleted={closeReportUpsert}
                />
            )}
        </ScrollBox>
    );
}
