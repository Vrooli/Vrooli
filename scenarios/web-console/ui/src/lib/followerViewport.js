// [REQ:P0-002e] Multi-Device Follower Presentation
export const MIN_LEGIBLE_FONT_PX = 9;
/** Height kept free for the follower label and Take over control. */
export function deviceControlsLane(archetype, tier) {
    if (tier === "strip")
        return 0;
    // Monitor-like silhouettes draw their stand outside their rectangular
    // bounds, so their caption needs more than a normal one-line gap.
    return archetype === "laptop" || archetype === "monitor" || archetype === "ultrawide" ? 64 : 44;
}
export function fitGrid(gridCols, gridRows, paneWidth, paneHeight, cellAspect) {
    const gridAspect = (gridCols * cellAspect) / gridRows;
    let width = Math.min(paneWidth, paneHeight * gridAspect);
    let height = width / gridAspect;
    if (height > paneHeight) {
        height = paneHeight;
        width = height * gridAspect;
    }
    const calculated = width / (gridCols * cellAspect);
    const fontSize = Math.max(MIN_LEGIBLE_FONT_PX, calculated);
    return { x: (paneWidth - width) / 2, y: (paneHeight - height) / 2, width, height, fontSize, scale: calculated >= MIN_LEGIBLE_FONT_PX ? 1 : calculated / MIN_LEGIBLE_FONT_PX };
}
// The terminal must fit the visible display, not the frame's outer bezel.
// Values are normalized to the outer silhouette and leave room for rounded
// corners, bezel, and the monitor/laptop base below the display.
export function screenAperture(archetype, tier) {
    if (tier === "strip")
        return { x: 0, y: 0, width: 1, height: 1 };
    if (archetype === "phone")
        return { x: 0.07, y: 0.1, width: 0.86, height: 0.78 };
    if (archetype === "tablet")
        return { x: 0.055, y: 0.075, width: 0.89, height: 0.83 };
    return { x: 0.04, y: 0.07, width: 0.92, height: 0.8 };
}
export function fitDeviceGrid(gridCols, gridRows, paneWidth, paneHeight, cellAspect, aperture) {
    const gridAspect = (gridCols * cellAspect) / gridRows;
    const frameAspect = gridAspect * aperture.height / aperture.width;
    let width = Math.min(paneWidth, paneHeight * frameAspect);
    let height = width / frameAspect;
    if (height > paneHeight) {
        height = paneHeight;
        width = height * frameAspect;
    }
    const screenWidth = width * aperture.width;
    const screenHeight = height * aperture.height;
    const calculated = screenWidth / (gridCols * cellAspect);
    const fontSize = Math.max(MIN_LEGIBLE_FONT_PX, calculated);
    const scale = calculated >= MIN_LEGIBLE_FONT_PX ? 1 : calculated / MIN_LEGIBLE_FONT_PX;
    const frame = { x: (paneWidth - width) / 2, y: (paneHeight - height) / 2, width, height, fontSize, scale };
    return { frame, screen: { x: frame.x + width * aperture.x, y: frame.y + height * aperture.y, width: screenWidth, height: screenHeight, fontSize, scale } };
}
// Fit the presentation into the area above its controls, rather than fitting
// the frame to the whole pane and hoping the controls fit after it. This is
// particularly important for a tall phone viewed in a short desktop pane.
export function fitDeviceGridWithControls(gridCols, gridRows, paneWidth, paneHeight, cellAspect, archetype, tier) {
    const availableHeight = Math.max(1, paneHeight - deviceControlsLane(archetype, tier));
    return fitDeviceGrid(gridCols, gridRows, paneWidth, availableHeight, cellAspect, screenAperture(archetype, tier));
}
export function surplusRatio(rect, paneWidth, paneHeight) {
    if (paneWidth <= 0 || paneHeight <= 0)
        return 1;
    return 1 - (rect.width * rect.height) / (paneWidth * paneHeight);
}
export function chromeTier(surplus, scale = 1) {
    // A scaled follower still needs a visible silhouette.  Only use the compact
    // caption strip when the fitted geometry itself has no spare presentation
    // room; otherwise retain the hairline frame around the overview.
    if (scale < 1)
        return surplus >= 0.95 ? "strip" : "hairline";
    // Keep the full frame for the canonical 45×30-in-16:9 follower case. Its
    // fitted bounds yield ~0.58 surplus; visually it still has ample chrome.
    if (surplus < 0.6)
        return "full";
    if (surplus < 0.95)
        return "hairline";
    return "strip";
}
