/**
 * The banner visual base is the library's — see `react-component-library:Banner`.
 *
 * Re-exported as a default so the app's existing call sites and tests keep
 * importing it the way they always have.
 */
import { Banner } from "@vrooli/react-component-library/Banner";

export type { BannerProps } from "@vrooli/react-component-library/Banner";
export default Banner;
