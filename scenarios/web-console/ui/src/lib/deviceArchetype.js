export function aspectForGrid(cols, rows, cellAspect) { return (cols * cellAspect) / rows; }
export function archetypeForGrid(cols, rows, cellAspect) {
    const aspect = aspectForGrid(cols, rows, cellAspect);
    if (aspect < 1.1)
        return "phone";
    if (aspect < 1.6 && cols <= 110)
        return "tablet";
    if (aspect < 2.1)
        return "laptop";
    if (aspect < 3)
        return "monitor";
    return "ultrawide";
}
export function orientationForGrid(cols, rows, cellAspect) { return aspectForGrid(cols, rows, cellAspect) < 1 ? "portrait" : "landscape"; }
