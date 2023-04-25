import { AddIcon, BookmarkList, EditIcon, FindByIdInput, useLocation } from "@local/shared";
import { Box, TextField, useTheme } from "@mui/material";
import { bookmarkListFindOne } from "api/generated/endpoints/bookmarkList_findOne";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useCallback, useEffect, useMemo, useState } from "react";
import { ObjectAction } from "utils/actions/objectActions";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { BookmarkListViewProps } from "../types";

export const BookmarkListView = ({
    display = "page",
    partialData,
    zIndex = 200,
}: BookmarkListViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { object: existing, setObject: setBookmarkList } = useObjectFromUrl<BookmarkList, FindByIdInput>({
        query: bookmarkListFindOne,
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

    const [searchString, setSearchString] = useState("");
    const updateSearchString = useCallback((event: React.ChangeEvent<HTMLInputElement>) => {
        setSearchString(event.target.value);
    }, []);

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: "BookmarkList",
                    titleVariables: { count: 1 },
                }}
                below={<TextField
                    placeholder="Search bookmarks..."
                    autoFocus={true}
                    value={searchString}
                    onChange={updateSearchString}
                />}
            />
            <>
                <SideActionButtons display={display} zIndex={zIndex + 1}>
                    {/* Edit button */}
                    <ColorIconButton aria-label="Edit list" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                    {/* Add button */}
                    <ColorIconButton aria-label="Add bookmark" background={palette.secondary.main} onClick={() => { }} >
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
