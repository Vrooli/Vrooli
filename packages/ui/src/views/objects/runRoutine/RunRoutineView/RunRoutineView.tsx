import { endpointGetRunRoutine, RunRoutine } from "@local/shared";
import { useTheme } from "@mui/material";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { getDisplay } from "utils/display/listTools";
import { toDisplay } from "utils/display/pageTools";
import { firstString } from "utils/display/stringTools";
import { RunRoutineViewProps } from "../types";

export const RunRoutineView = ({
    isOpen,
    onClose,
}: RunRoutineViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const display = toDisplay(isOpen);

    const { object: existing, isLoading, setObject: setRunRoutine } = useObjectFromUrl<RunRoutine>({
        ...endpointGetRunRoutine,
        objectType: "RunRoutine",
    });

    const { title } = useMemo(() => ({ title: getDisplay(existing).title ?? "" }), [existing]);

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
                onClose={onClose}
                title={firstString(title, t("Run", { count: 1 }))}
            />
            <>
                {/* TODO */}
            </>
        </>
    );
};
