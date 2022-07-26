import { SxProps } from "@mui/material"

//==============================================================
/* #region Centering */
//==============================================================
export const centeredDiv: SxProps = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center'
};
//==============================================================
/* #endregion Centering */
//==============================================================

//==============================================================
/* #region Shadows */
//==============================================================
export const containerShadow = {
    boxShadow: '0px 0px 12px gray',
};

export const textShadow: SxProps = {
    textShadow:
        `-0.5px -0.5px 0 black,  
        0.5px -0.5px 0 black,
        -0.5px 0.5px 0 black,
        0.5px 0.5px 0 black`
};
//==============================================================
/* #endregion Shadows*/
//==============================================================

//==============================================================
/* #region Sizing */
//==============================================================
/**
 * Lighthouse recommended size for clickable elements, to improve SEO
 */
export const clickSize: SxProps = {
    minHeight: '48px',
};

export const multiLineEllipsis = (lines: number): SxProps => ({
    display: '-webkit-box',
    WebkitLineClamp: lines,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
});
//==============================================================
/* #endregion Sizing */
//==============================================================

//==============================================================
/* #region Actions */
//==============================================================
/**
 * Disables text highlighting
 */
export const noSelect: SxProps = {
    WebkitTouchCallout: 'none', /* iOS Safari */
    WebkitUserSelect: 'none', /* Safari */
    MozUserSelect: 'none',
    msUserSelect: 'none', /* Internet Explorer/Edge */
    userSelect: 'none', /* Non-prefixed version, currently
    supported by Chrome, Edge, Opera and Firefox */
};
/**
 * 
 */
//==============================================================
/* #endregion Actions */
//==============================================================