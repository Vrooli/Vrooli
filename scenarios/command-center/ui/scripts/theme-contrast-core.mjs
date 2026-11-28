const TEXT_COLORS = [
  ["foreground", (theme) => theme.foreground],
  ["muted foreground", (theme) => theme.mutedForeground],
  ["primary", (theme) => theme.primary],
  ["accent", (theme) => theme.accent],
  ["warning", (theme) => theme.warning],
  ["danger", (theme) => theme.danger],
  ["gap", (theme) => theme.gap],
  ["success", (theme) => theme.success],
  ["live qualifier blend", (theme) => mix(theme.primary, theme.foreground, 0.8)],
  ["gap qualifier blend", (theme) => mix(theme.mutedForeground, theme.gap, 0.6)],
  ["room stamp blend", (theme) => mix(theme.foreground, theme.gap, 0.8)],
];

const FALLBACKS = {
  warning: "#f5b544",
  danger: "#ff8c9d",
  success: "#4ade80",
};

function color(value) {
  const input = value.trim();
  const hex = input.match(/^#([\da-f]{3}|[\da-f]{6})$/i);
  if (hex) {
    const expanded = hex[1].length === 3 ? [...hex[1]].map((part) => part + part).join("") : hex[1];
    return { r: Number.parseInt(expanded.slice(0, 2), 16), g: Number.parseInt(expanded.slice(2, 4), 16), b: Number.parseInt(expanded.slice(4, 6), 16), a: 1 };
  }
  const rgba = input.match(/^rgba?\(\s*([\d.]+)[, ]+([\d.]+)[, ]+([\d.]+)(?:\s*[,/]\s*([\d.]+))?\s*\)$/i);
  if (rgba) return { r: Number(rgba[1]), g: Number(rgba[2]), b: Number(rgba[3]), a: rgba[4] === undefined ? 1 : Number(rgba[4]) };
  throw new Error(`Unsupported theme color: ${value}`);
}

function opaqueOn(foreground, background) {
  const alpha = foreground.a;
  return {
    r: foreground.r * alpha + background.r * (1 - alpha),
    g: foreground.g * alpha + background.g * (1 - alpha),
    b: foreground.b * alpha + background.b * (1 - alpha),
    a: 1,
  };
}

function mix(first, second, firstWeight) {
  return {
    r: first.r * firstWeight + second.r * (1 - firstWeight),
    g: first.g * firstWeight + second.g * (1 - firstWeight),
    b: first.b * firstWeight + second.b * (1 - firstWeight),
    a: 1,
  };
}

function mixWithTransparency(first, firstWeight, second, secondWeight) {
  const alpha = firstWeight + secondWeight;
  return { ...mix(first, second, firstWeight / alpha), a: alpha };
}

function linear(channel) {
  const normalized = channel / 255;
  return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
}

function luminance(rgb) {
  return 0.2126 * linear(rgb.r) + 0.7152 * linear(rgb.g) + 0.0722 * linear(rgb.b);
}

export function contrastRatio(foreground, background) {
  const a = luminance(color(foreground));
  const b = luminance(color(background));
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

function cssColor(rgb) {
  return `rgb(${Math.round(rgb.r)}, ${Math.round(rgb.g)}, ${Math.round(rgb.b)})`;
}

function activeThemes(board, roomByID, themeByID) {
  const themeIDs = new Set();
  for (const setting of board.rooms.filter((room) => room.enabled)) {
    const room = roomByID.get(setting.id);
    if (!room) throw new Error(`Board references missing room ${setting.id}`);
    if (!themeByID.has(room.theme)) throw new Error(`Room ${room.id} references missing theme ${room.theme}`);
    themeIDs.add(room.theme);
  }
  return [...themeIDs].sort((a, b) => a.localeCompare(b)).map((id) => themeByID.get(id));
}

function buildTheme(themeFile) {
  const tokens = themeFile.tokens ?? {};
  const theme = {
    id: themeFile.id,
    background: color(tokens["--color-background"]),
    deep: color(tokens["--color-background-deep"]),
    surface: color(tokens["--color-surface"]),
    foreground: color(tokens["--color-foreground"]),
    mutedForeground: color(tokens["--color-muted-foreground"]),
    primary: color(tokens["--color-primary"]),
    accent: color(tokens["--color-accent"]),
    warning: color(tokens["--color-warning"] ?? FALLBACKS.warning),
    danger: color(tokens["--color-danger"] ?? FALLBACKS.danger),
    success: color(tokens["--color-success"] ?? FALLBACKS.success),
    gap: color(tokens["--color-gap"]),
  };
  theme.surfaceOnBackground = opaqueOn(theme.surface, theme.background);
  theme.panelOnBackground = opaqueOn({ ...theme.surface, a: theme.surface.a * 0.8 }, theme.background);
  theme.tileOnBackground = opaqueOn({ ...theme.surface, a: theme.surface.a * 0.65 }, theme.background);
  theme.heroTintOnBackground = opaqueOn({ ...theme.background, a: 0.88 }, theme.background);
  theme.heroTintOnDeep = opaqueOn({ ...theme.background, a: 0.88 }, theme.deep);
  theme.shellTintOnBackground = opaqueOn(mixWithTransparency(theme.background, 0.7, theme.primary, 0.06), theme.background);
  theme.shellTintOnDeep = opaqueOn(mixWithTransparency(theme.background, 0.7, theme.primary, 0.06), theme.deep);
  theme.sceneGlowOnBackground = opaqueOn({ ...theme.primary, a: 0.22 }, theme.background);
  theme.sceneGlowOnDeep = opaqueOn({ ...theme.primary, a: 0.22 }, theme.deep);
  theme.dockOnBackground = opaqueOn({ ...theme.background, a: 0.94 }, theme.background);
  theme.dockOnDeep = opaqueOn({ ...theme.background, a: 0.94 }, theme.deep);
  const helpScrimOnBackground = opaqueOn({ ...theme.deep, a: 0.7 }, theme.background);
  const helpScrimOnDeep = opaqueOn({ ...theme.deep, a: 0.7 }, theme.deep);
  const helpPanelMix = mixWithTransparency(theme.background, 0.92, theme.primary, 0.04);
  theme.helpPanelOnBackground = opaqueOn(helpPanelMix, helpScrimOnBackground);
  theme.helpPanelOnDeep = opaqueOn(helpPanelMix, helpScrimOnDeep);
  theme.degradedOnDeep = opaqueOn({ ...theme.background, a: 0.8 }, theme.deep);
  return theme;
}

export function auditThemeContrast({ board, rooms, themes, threshold = 4.5 }) {
  const active = activeThemes(board, new Map(rooms.map((room) => [room.id, room])), new Map(themes.map((theme) => [theme.id, theme])));
  return active.map((entry) => {
    const theme = buildTheme(entry);
    const backgrounds = [
      ["background", theme.background],
      ["deep background", theme.deep],
      ["surface", theme.surfaceOnBackground],
      ["panel surface", theme.panelOnBackground],
      ["tile surface", theme.tileOnBackground],
      ["hero tint on background", theme.heroTintOnBackground],
      ["hero tint on deep", theme.heroTintOnDeep],
      ["shell tint on background", theme.shellTintOnBackground],
      ["shell tint on deep", theme.shellTintOnDeep],
      ["scene glow on background", theme.sceneGlowOnBackground],
      ["scene glow on deep", theme.sceneGlowOnDeep],
      ["dock on background", theme.dockOnBackground],
      ["dock on deep", theme.dockOnDeep],
      ["help panel over background", theme.helpPanelOnBackground],
      ["help panel over deep", theme.helpPanelOnDeep],
      ["degraded surface", theme.degradedOnDeep],
    ];
    const pairs = TEXT_COLORS.flatMap(([role, pick]) => backgrounds.map(([surface, backdrop]) => {
      const foreground = pick(theme);
      const ratio = contrastRatio(cssColor(foreground), cssColor(backdrop));
      return { role, surface, ratio: Number(ratio.toFixed(2)), passes: ratio >= threshold };
    }));
    return { theme: theme.id, pairs, minimum: Math.min(...pairs.map((pair) => pair.ratio)), failures: pairs.filter((pair) => !pair.passes) };
  });
}
