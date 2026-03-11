// Layout types - shared by SettingsModal and other components
export type LayoutPreset = "classic" | "split" | "bottom";
export type LayoutSection = "changes" | "history" | "commit" | "diff" | "review";

// Note: The LayoutSettingsModal component has been replaced by SettingsModal
// which contains a Layout tab (SettingsTabLayout) and Credentials tab.
// These types are kept for backwards compatibility with existing imports.
