import { AddIcon, BookmarkList, EditIcon, endpointGetBookmarkList, useLocation } from "@local/shared";
import { Box, useTheme } from "@mui/material";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { SiteSearchBar } from "components/inputs/search";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ObjectAction } from "utils/actions/objectActions";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { getObjectUrlBase } from "utils/navigation/openObject";
import { BookmarkListViewProps } from "../types";

export const BookmarkListView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: BookmarkListViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setBookmarkList } = useObjectFromUrl<BookmarkList>({
        ...endpointGetBookmarkList,
        partialData,
    });

    const { label } = useMemo(() => ({ label: existing?.label ?? "" }), [existing]);

    useEffect(() => {
        document.title = `${label} | Vrooli`;
    }, [label]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "BookmarkList",
        setLocation,
        setObject: setBookmarkList,
    });

    const onAddClick = useCallback(() => {
        const addUrl = `${getObjectUrlBase({ __typename: "BookmarkList" })}/add`;
        setLocation(addUrl);
    }, [setLocation]);

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((newString: string) => {
        setSearchString(newString);
    }, []);

    const onBookmarkSelect = useCallback((data: any) => {
        console.log("onBookmarkSelect", data);
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "BookmarkList",
                    titleVariables: { count: 1 },
                }}
                below={<Box sx={{
                    width: "min(100%, 700px)",
                    margin: "auto",
                    marginTop: 2,
                    marginBottom: 2,
                    paddingLeft: 2,
                    paddingRight: 2,
                }}>
                    <SiteSearchBar
                        id={"history-search-bar"}
                        placeholder={"SearchBookmark"}
                        loading={isLoading}
                        value={searchString}
                        onChange={updateSearchString}
                        onInputChange={onBookmarkSelect}
                        sxs={{ root: { width: "min(100%, 600px)", paddingLeft: 2, paddingRight: 2 } }}
                        zIndex={zIndex}
                    />
                </Box>}
            />
            <>
                <SideActionButtons display={display} zIndex={zIndex + 1}>
                    {/* Edit button */}
                    <ColorIconButton aria-label="Edit list" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                    {/* Add button */}
                    <ColorIconButton aria-label="Add bookmark" background={palette.secondary.main} onClick={onAddClick} >
                        <AddIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                </SideActionButtons>
                <Box
                    sx={{
                        display: "flex",
                        justifyContent: "center",
                        alignItems: "center",
                    }}
                >
                    {/* Bookmarks */}
                    {/* TODO */}
                </Box>
            </>
        </>
    );
};
