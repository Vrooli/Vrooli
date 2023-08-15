/**
 * Search page for organizations, projects, routines, standards, and users
 */
import { CommonKey, GqlModelType, LINKS, VisibilityType } from "@local/shared";
import { Box, Button, IconButton, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { SearchList } from "components/lists/SearchList/SearchList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { PageTabs } from "components/PageTabs/PageTabs";
import { PageTab } from "components/types";
import { AddIcon } from "icons";
import { useCallback, useContext, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, parseSearchParams, useLocation } from "route";
import { centeredDiv } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { toDisplay } from "utils/display/pageTools";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { MyStuffPageTabOption, SearchType } from "utils/search/objectToSearch";
import { SessionContext } from "utils/SessionContext";
import { MyStuffViewProps } from "../types";

// Tab data type
type BaseParams = {
    searchType: SearchType;
    tabType: MyStuffPageTabOption;
    where: (userId: string) => { [x: string]: any };
}

// Data for each tab TODO add bot tab
const tabParams: BaseParams[] = [{
    searchType: SearchType.Routine,
    tabType: MyStuffPageTabOption.Routine,
    where: () => ({ isInternal: false, visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Project,
    tabType: MyStuffPageTabOption.Project,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Schedule,
    tabType: MyStuffPageTabOption.Schedule,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Reminder,
    tabType: MyStuffPageTabOption.Reminder,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Note,
    tabType: MyStuffPageTabOption.Note,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Question,
    tabType: MyStuffPageTabOption.Question,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Organization,
    tabType: MyStuffPageTabOption.Organization,
    where: (userId) => ({ memberUserIds: [userId] }),
}, {
    searchType: SearchType.Standard,
    tabType: MyStuffPageTabOption.Standard,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.Api,
    tabType: MyStuffPageTabOption.Api,
    where: () => ({ visibility: VisibilityType.Own }),
}, {
    searchType: SearchType.SmartContract,
    tabType: MyStuffPageTabOption.SmartContract,
    where: () => ({ visibility: VisibilityType.Own }),
}];

export const MyStuffView = ({
    isOpen,
    onClose,
    zIndex,
}: MyStuffViewProps) => {
    const session = useContext(SessionContext);
    const [, setLocation] = useLocation();
    const { palette } = useTheme();
    const { t } = useTranslation();
    const {
        id: userId,
        apisCount,
        membershipsCount,
        questionsAskedCount,
        smartContractsCount,
        standardsCount,
    } = useMemo(() => getCurrentUser(session), [session]);
    const display = toDisplay(isOpen);

    // Popup button, which opens either an add or invite dialog
    const [popupButton, setPopupButton] = useState<boolean>(false);

    // Handle tabs
    const tabs = useMemo<(PageTab<MyStuffPageTabOption> & { where: any, searchType: any, tabType: any })[]>(() => {
        // Filter out certain tabs that we don't have any data for, 
        // so user isn't overwhelmed with tabs for objects they never worked with. 
        // Always keeps routines, projects, and notes
        const filteredTabParams = tabParams.filter(tab => {
            switch (tab.tabType) {
                case MyStuffPageTabOption.Api:
                    return Boolean(apisCount);
                case MyStuffPageTabOption.Organization:
                    return Boolean(membershipsCount);
                case MyStuffPageTabOption.Question:
                    return Boolean(questionsAskedCount);
                case MyStuffPageTabOption.SmartContract:
                    return Boolean(smartContractsCount);
                case MyStuffPageTabOption.Standard:
                    return Boolean(standardsCount);
            }
            return true;
        });
        return filteredTabParams.map((tab, i) => ({
            index: i,
            label: t(tab.searchType, { count: 2, defaultValue: tab.searchType }),
            searchType: tab.searchType,
            tabType: tab.tabType,
            value: tab.tabType,
            where: tab.where,
        }));
    }, [apisCount, membershipsCount, questionsAskedCount, smartContractsCount, standardsCount, t]);
    const [currTab, setCurrTab] = useState<PageTab<MyStuffPageTabOption>>(() => {
        const searchParams = parseSearchParams();
        const index = tabs.findIndex(tab => tab.tabType === searchParams.type);
        // Default to routine tab
        if (index === -1) return tabs[0];
        // Return tab
        return tabs[index];
    });
    const handleTabChange = useCallback((e: any, tab: PageTab<MyStuffPageTabOption>) => {
        e.preventDefault();
        // Update search params
        addSearchParams(setLocation, { type: tab.value });
        // Update curr tab
        setCurrTab(tab);
    }, [setLocation]);

    // On tab change, update BaseParams, document title, where, and URL
    const { searchType, where } = useMemo<Omit<BaseParams, "where"> & { where: { [x: string]: any } }>(() => {
        const params = tabs[currTab.index];
        return {
            ...params,
            where: params.where(userId!),
        } as any;
    }, [currTab.index, tabs, userId]);

    const onAddClick = useCallback((ev: any) => {
        const addUrl = `${getObjectUrlBase({ __typename: searchType as `${GqlModelType}` })}/add`;
        // If not logged in, redirect to login page
        if (!userId) {
            PubSub.get().publishSnack({ messageKey: "MustBeLoggedIn", severity: "Error" });
            setLocation(LINKS.Start, { searchParams: { redirect: addUrl } });
            return;
        }
        // If search type is a routine, open create routine page
        if (searchType === SearchType.Routine) {
            setLocation(`${LINKS.Routine}/add`);
        }
        // If search type is a user, open start page
        else if (searchType === SearchType.User) {
            setLocation(`${LINKS.Start}`);
        }
        // Otherwise, navigate to add page
        else {
            setLocation(addUrl);
        }
    }, [searchType, setLocation, userId]);

    const handleScrolledFar = useCallback(() => { setPopupButton(true); }, []);
    const popupButtonContainer = useMemo(() => (
        <Box sx={{ ...centeredDiv, paddingTop: 1 }}>
            <Tooltip title={t("AddTooltip")}>
                <Button
                    onClick={onAddClick}
                    size="large"
                    variant="contained"
                    startIcon={<AddIcon />}
                    sx={{
                        zIndex: 100,
                        minWidth: "min(100%, 200px)",
                        height: "48px",
                        borderRadius: 3,
                        position: "fixed",
                        bottom: "calc(5em + env(safe-area-inset-bottom))",
                        transform: popupButton ? "translateY(0)" : "translateY(calc(10em + env(safe-area-inset-bottom)))",
                        transition: "transform 1s ease-in-out",
                    }}
                >
                    {t("Add")}
                </Button>
            </Tooltip>
        </Box>
    ), [onAddClick, popupButton, t]);

    return (
        <>
            <TopBar
                display={display}
                hideTitleOnDesktop={true}
                onClose={onClose}
                title={t("MyStuff")}
                below={<PageTabs
                    ariaLabel="my-stuff-tabs"
                    fullWidth
                    id="my-stuff-tabs"
                    currTab={currTab}
                    onChange={handleTabChange}
                    tabs={tabs}
                />}
                zIndex={zIndex}
            />
            <Stack direction="row" alignItems="center" justifyContent="center" sx={{ paddingTop: 2 }}>
                <Typography component="h2" variant="h4">{t(searchType as CommonKey, { count: 2, defaultValue: searchType })}</Typography>
                <Tooltip title="Add new" placement="top">
                    <IconButton
                        size="medium"
                        onClick={onAddClick}
                        sx={{
                            padding: 1,
                        }}
                    >
                        <AddIcon fill={palette.secondary.main} width='1.5em' height='1.5em' />
                    </IconButton>
                </Tooltip>
            </Stack>
            {searchType && <SearchList
                id="main-search-page-list"
                dummyLength={display === "page" ? 5 : 3}
                take={20}
                searchType={searchType}
                onScrolledFar={handleScrolledFar}
                zIndex={zIndex}
                where={where}
            />}
            {popupButtonContainer}
        </>
    );
};
