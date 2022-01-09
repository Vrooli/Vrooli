import { SxProps } from "@mui/material"

export const centeredDiv: SxProps = {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center'
}

export const centeredText: SxProps = {
    textAlign: 'center',
}

/**
 * Lighthouse recommended size for clickable elements, to improve SEO
 */
export const clickSize: SxProps = {
    minHeight: '48px',
}

export const containerShadow: SxProps = {
    boxShadow: '0px 0px 12px gray',
}

export const multiLineEllipsis = (lines: number): SxProps => ({
    display: '-webkit-box',
    WebkitLineClamp: lines,
    WebkitBoxOrient: 'vertical',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
})