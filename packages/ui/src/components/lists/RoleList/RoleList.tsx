import { Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { useMemo } from "react";
import { ListItemChip } from "../ObjectListItemBase/ObjectListItemBase";
import { RoleListProps } from "../types";

export const RoleList = ({
    maxCharacters = 50,
    roles,
    sx,
}: RoleListProps) => {
    const { palette } = useTheme();

    const [chips, numTagsCutOff] = useMemo(() => {
        let charactersBeforeCutoff = maxCharacters;
        const chipResult: JSX.Element[] = [];
        for (let i = 0; i < roles.length; i++) {
            const role = roles[i];
            if (role.name.length < charactersBeforeCutoff) {
                charactersBeforeCutoff -= role.name.length;
                chipResult.push(
                    <ListItemChip
                        color="Blue"
                        key={role.name}
                        label={role.name}
                    />,
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
                sx={{
                    overflowX: "auto",
                    ...sx,
                }}
            >
                {chips}
                {numTagsCutOff > 0 && <Typography variant="body1">+{numTagsCutOff}</Typography>}
            </Stack>
        </Tooltip>
    );
};
