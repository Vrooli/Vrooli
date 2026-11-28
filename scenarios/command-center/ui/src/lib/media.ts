import { useMediaQuery } from "@vrooli/react-component-library/useMediaQuery/1";

export { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1";

/** The portrait composition's query; styles.css switches layouts on the same one. */
export const PORTRAIT_QUERY = "(orientation: portrait), (max-width: 760px)";

/** The landscape wall composition, which never scrolls. Portrait is the desk view and may. */
export const useLandscapeRoom = (): boolean => !useMediaQuery(PORTRAIT_QUERY);
