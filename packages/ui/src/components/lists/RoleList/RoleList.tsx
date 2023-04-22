import { Chip, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
import { RoleListProps } from "../types";

export const RoleList = ({
    maxCharacters = 50,
    roles,
    sx,
}: RoleListProps) => {
    const { palette } = useTheme();

    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = maxCharacters;
        let chipResult: JSX.Element[] = [];
        for (let i = 0; i < roles.length; i++) {
            const role = roles[i];
            if (role.name.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= role.name.length;
                chipResult.push(
                    <Chip
                        key={role.name}
                        label={role.name}
                        size="small"
                        sx={{
                            backgroundColor: palette.mode === "light" ? "#1d7691" : "#016d97",
                            color: "white",
                            width: "fit-content",
                        }} />
                );
            }
        }
        // Check if any roles were cut off
        const numTagsCutOff = roles.length - chipResult.length;
        return [chipResult, numTagsCutOff];
    }, [maxCharacters, palette.mode, roles]);

    return (
        <Tooltip title={roles.map(t => t.name).join(", ")} placement="top">
            <Stack
                direction="row"
                spacing={1}
                justifyContent="left"
                alignItems="center"
                sx={{ ...(sx ?? {}) }}
            >
                {chips}
                {numTagsCutOff > 0 && <Typography variant="body1">+{numTagsCutOff}</Typography>}
            </Stack>
        </Tooltip>
    )
}