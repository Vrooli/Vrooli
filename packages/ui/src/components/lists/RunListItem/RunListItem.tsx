import { RunStatus } from "@local/shared";
import { Stack } from "@mui/material";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ListItemChip, ListItemCompletionBar, ListItemStyleColor, ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase.js";
import { RunListItemProps } from "../types.js";

function statusToColor(status: RunStatus | undefined): ListItemStyleColor {
    if (!status) return "Default";
    switch (status) {
        case RunStatus.Completed: return "Green";
        case RunStatus.Failed: return "Red";
        default: return "Default";
    }
}

export function RunListItem({
    data,
    loading,
    ...props
}: RunListItemProps) {
    const { t } = useTranslation();

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        const completedComplexity = data?.completedComplexity ?? 0;
        const totalComplexity = data?.routineVersion?.complexity ?? null;
        const percentComplete = data?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0;
        return (<ListItemCompletionBar
            color={statusToColor(data?.status)}
            isLoading={loading}
            value={percentComplete}
        />);
    }, [data?.completedComplexity, data?.routineVersion?.complexity, data?.status, loading]);

    return (
        <ObjectListItemBase
            {...props}
            belowSubtitle={
                <Stack direction="row" spacing={1} sx={{
                    "& > .MuiBox-root:first-of-type": {
                        flex: "1 1 auto",
                    },
                }}>
                    {progressBar}
                    <ListItemChip
                        color={statusToColor(data?.status)}
                        label={t(data?.status ?? RunStatus.InProgress, { defaultValue: data?.status ?? RunStatus.InProgress })}
                    />
                </Stack>
            }
            data={data}
            loading={loading}
            objectType="Run"
        />
    );
}
