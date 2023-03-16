import { Box, useTheme } from "@mui/material";
import { FindByIdInput, Resource } from "@shared/consts";
import { useLocation } from '@shared/route';
import { resourceFindOne } from "api/generated/endpoints/resource_findOne";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useEffect, useMemo, useState } from "react";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { ResourceViewProps } from "../types";

export const ResourceView = ({
    display = 'dialog',
    partialData,
    session,
    zIndex = 200,
}: ResourceViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { id, isLoading, object: resource, permissions, setObject: setResource } = useObjectFromUrl<Resource, FindByIdInput>({
        query: resourceFindOne,
        partialData,
    });

    const availableLanguages = useMemo<string[]>(() => (resource?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [resource?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { description, name } = useMemo(() => {
        const { description, name } = getTranslation(resource ?? partialData, [language]);
        return {
            description: description && description.trim().length > 0 ? description : undefined,
            name,
        };
    }, [language, resource, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const actionData = useObjectActions({
        object: resource,
        objectType: 'Resource',
        session,
        setLocation,
        setObject: setResource,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={() => {}}
                session={session}
                titleData={{
                    titleKey: 'Resource',
                }}
            />
            <Box sx={{
                background: palette.mode === 'light' ? "#b2b3b3" : "#303030",
                display: 'flex',
                paddingTop: 5,
                paddingBottom: { xs: 0, sm: 2, md: 5 },
                position: "relative",
            }}>
                {/* Language display/select */}
                <Box sx={{
                    position: 'absolute',
                    top: 8,
                    right: 8,
                }}>
                    <SelectLanguageMenu
                        currentLanguage={language}
                        handleCurrent={setLanguage}
                        session={session}
                        translations={resource?.translations ?? partialData?.translations ?? []}
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
            {/* TODO */}
        </>
    )
}