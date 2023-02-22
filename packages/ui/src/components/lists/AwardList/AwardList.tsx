import { useMemo } from 'react';
import { AwardListProps } from '../types';
import { Chip, Stack, useTheme } from '@mui/material';

export const AwardList = ({
    awards,
}: AwardListProps) => {
    const { palette } = useTheme();

    const chips = useMemo(() => {
        let chipResult: JSX.Element[] = [];
        for (let i = 0; i < awards.length; i++) {
            const award = awards[i];
            chipResult.push(
                <Chip
                    key={award.category}
                    label={award.earnedTier?.title ?? award.categoryTitle}
                    size="small"
                    sx={{
                        backgroundColor: palette.mode === 'light' ? '#8148b0' : '#8148b0', //'#a068ce',
                        color: 'white',
                        width: 'fit-content',
                    }} />
            );
        }
        return chipResult;
    }, [palette.mode, awards]);

    // // Description popup
    // const [anchorEl, setAnchorEl] = useState<any | null>(null);
    // const open = useCallback((target: EventTarget) => {
    //     setAnchorEl(target)
    // }, []);
    // const close = useCallback(() => setAnchorEl(null), []);
    // const pressEvents = usePress({
    //     onHover: open,
    //     onLongPress: open,
    //     onClick: open,
    // });

    return (
        <Stack
            direction="row"
            spacing={1}
            justifyContent="left"
            alignItems="center"
        >
            {chips}
        </Stack>
    )
}