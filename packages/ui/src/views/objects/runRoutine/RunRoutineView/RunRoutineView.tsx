import { FindByIdInput, RunRoutine, useLocation } from "@local/shared";
import { useTheme } from "@mui/material";
import { runRoutineFindOne } from "api/generated/endpoints/runRoutine_findOne";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { getDisplay } from "utils/display/listTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { RunRoutineViewProps } from "../types";

export const RunRoutineView = ({
    display = "page",
    partialData,
    zIndex = 200,
}: RunRoutineViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { object: existing, isLoading, setObject: setRunRoutine } = useObjectFromUrl<RunRoutine, FindByIdInput>({
        query: runRoutineFindOne,
        partialData,
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RunRoutine",
        setLocation,
        setObject: setRunRoutine,
    });

    return (
        <>
            <TopBar
                display={display}
                onClose={() => { }}
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
