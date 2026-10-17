export const ARTIFACT_VIEWER_MIN_WIDTH = 360;
export const ARTIFACT_WORKSPACE_MIN_WIDTH = 480;

/** The viewer enters focus mode before either side becomes unusably narrow. */
export function shouldFocusArtifactViewer(availableWidth: number, viewerWidth: number): boolean {
  return availableWidth < ARTIFACT_WORKSPACE_MIN_WIDTH + Math.max(ARTIFACT_VIEWER_MIN_WIDTH, viewerWidth);
}
