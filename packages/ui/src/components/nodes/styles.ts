import { Theme } from "@material-ui/core";
import { Styles } from "@material-ui/styles";

export const nodeStyles: Styles<Theme, {}> = (theme: Theme) => ({
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
        textShadow:
            `-0.5px -0.5px 0 black,  
            0.5px -0.5px 0 black,
            -0.5px 0.5px 0 black,
            0.5px 0.5px 0 black`
    }
});