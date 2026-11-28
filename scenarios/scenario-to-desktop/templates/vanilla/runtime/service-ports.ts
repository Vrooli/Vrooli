/** Select the root app's API port from the bundled runtime port inventory. */
export function selectBundledAPIPort(
    services: Record<string, Record<string, number>>,
    uiServiceID: string,
): number | undefined {
    const rootServices = Object.entries(services).filter(([id, ports]) =>
        id !== uiServiceID && !id.includes("--") && (ports.http || ports.api || ports.https));
    const selected = rootServices.find(([id]) => id === "api")
        ?? rootServices.find(([id]) => /api|server|backend/i.test(id))
        ?? rootServices[0];
    const ports = selected?.[1];
    return ports?.http || ports?.api || ports?.https;
}
