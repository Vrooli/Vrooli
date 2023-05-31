import { FindByIdInput, RunProject, runProjectFindOne, useLocation } from "@local/shared";
import { useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { RunProjectViewProps } from "../types";

export const RunProjectView = ({
    display = "page",
    onClose,
    partialData,
    zIndex = 200,
}: RunProjectViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunProject } = useObjectFromUrl<RunProject, FindByIdInput>({
        query: runProjectFindOne,
        partialData,
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RunProject",
        setLocation,
        setObject: setRunProject,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Run",
                    titleVariables: { count: 1 },
                }}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
