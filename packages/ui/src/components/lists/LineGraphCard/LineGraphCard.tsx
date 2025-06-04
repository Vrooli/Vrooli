import { Card, CardContent, Typography, useTheme } from "@mui/material";
import { useDimensions } from "../../../hooks/useDimensions.js";
import { LineGraph } from "../../graphs/LineGraph/LineGraph.js";
import { type LineGraphCardProps } from "../types.js";

export function LineGraphCard({
    title,
    index,
    ...lineGraphProps
}: LineGraphCardProps) {
    const { breakpoints, palette } = useTheme();
    const { dimensions, ref } = useDimensions<HTMLDivElement>();

    return (
        <Card ref={ref} sx={{
            width: "100%",
            height: "100%",
            boxShadow: 0,
            background: palette.primary.light,
            color: palette.primary.contrastText,
            margin: 0,
            borderRadius: { xs: 0, sm: 2 },
            [breakpoints.down("sm")]: {
                borderBottom: `1px solid ${palette.divider}`,
            },
        }}>
            <CardContent
                sx={{
                    display: "contents",
                }}
            >
                <Typography
                    variant="h6"
                    component="h2"
                    textAlign="center"
                    sx={{
                        marginBottom: 2,
                        marginTop: 1,
                    }}
                >{title}</Typography>
                <LineGraph
                    dims={{
                        width: dimensions.width,
                        height: 250,
                    }}
                    {...lineGraphProps}
                />
            </CardContent>
        </Card>
    );
}
