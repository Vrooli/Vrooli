import { Theme } from '@mui/material';
import { Styles } from "@mui/styles";

export const nodeStyles: Styles<Theme, {}> = (theme: Theme) => ({
    // Disables text highlighting
    noSelect: {
        '-webkit-touch-callout': 'none', /* iOS Safari */
        '-webkit-user-select': 'none', /* Safari */
        '-ms-user-select': 'none', /* Internet Explorer/Edge */
        'user-select': 'none', /* Non-prefixed version, currently
                                  supported by Chrome, Edge, Opera and Firefox */
    },
    label: {
        position: 'absolute',
        textAlign: 'center',
        margin: '0',
        top: '50%',
        left: '50%',
        transform: 'translate(-50%, -50%)',
        width: '100%',
        overflow: 'hidden',
        textOverflow: 'ellipsis',
        lineBreak: 'anywhere',
        display:' -webkit-box',
        '-webkit-line-clamp': '3',
        '-webkit-box-orient': 'vertical',
        textShadow:
            `-0.5px -0.5px 0 black,  
            0.5px -0.5px 0 black,
            -0.5px 0.5px 0 black,
            0.5px 0.5px 0 black`
    },
    ignoreHover: {
        pointerEvents: 'none',
    }
});