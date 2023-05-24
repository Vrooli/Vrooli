import { RunStatus } from "@local/shared";
import { Chip, Stack } from "@mui/material";
import { CompletionBar } from "components/CompletionBar/CompletionBar";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { ObjectListItemBase } from "../ObjectListItemBase/ObjectListItemBase";
import { RunProjectListItemProps } from "../types";

const statusToColor = (status: RunStatus | undefined) => {
    if (!status) return "secondary";
    switch (status) {
        case RunStatus.Completed: return "success";
        case RunStatus.Failed: return "error";
        default: return "secondary";
    }
};

export function RunProjectListItem({
    data,
    loading,
    ...props
}: RunProjectListItemProps) {
    const { t } = useTranslation();

    /**
     * Run list items may get a progress bar
     */
    const progressBar = useMemo(() => {
        const completedComplexity = data?.completedComplexity ?? 0;
        const totalComplexity = data?.projectVersion?.complexity ?? null;
        const percentComplete = data?.status === RunStatus.Completed ? 100 :
            (completedComplexity && totalComplexity) ?
                Math.min(Math.round(completedComplexity / totalComplexity * 100), 100) :
                0;
        return (<CompletionBar
            color={statusToColor(data?.status)}
            variant={loading ? "indeterminate" : "determinate"}
            value={percentComplete}
            sx={{ height: "15px" }}
        />);
    }, [data?.completedComplexity, data?.projectVersion?.complexity, data?.status, loading]);

    return (
        <ObjectListItemBase
            {...props}
            belowSubtitle={
                <Stack direction="row" spacing={1} sx={{
                    "& > .MuiBox-root:first-child": {
                        flex: "1 1 auto",
                    },
                }}>
                    {progressBar}
                    <Chip
                        variant="filled"
                        color={statusToColor(data?.status)}
                        label={t(data?.status ?? RunStatus.InProgress, { defaultValue: data?.status ?? RunStatus.InProgress })}
                    />
                </Stack>
            }
            data={data}
            loading={loading}
            objectType="RunProject"
        />
    );
}
