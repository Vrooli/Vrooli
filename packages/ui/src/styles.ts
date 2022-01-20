import { SxProps } from "@mui/material"
import { CSSProperties } from "@mui/styles";

//==============================================================
/* #region Centering */
//==============================================================
export const centeredDiv: SxProps = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center'
} as CSSProperties;
//==============================================================
/* #endregion Centering */
//==============================================================

//==============================================================
/* #region Shadows */
//==============================================================
export const containerShadow: SxProps = {
    boxShadow: '0px 0px 12px gray',
} as CSSProperties;

export const textShadow: SxProps = {
    textShadow:
        `-0.5px -0.5px 0 black,  
        0.5px -0.5px 0 black,
        -0.5px 0.5px 0 black,
        0.5px 0.5px 0 black`
} as CSSProperties;
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
} as CSSProperties;

export const multiLineEllipsis = (lines: number): SxProps => ({
    display: '-webkit-box',
    WebkitLineClamp: lines,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
}) as CSSProperties;
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
} as CSSProperties;
/**
 * 
 */
//==============================================================
/* #endregion Actions */
//==============================================================