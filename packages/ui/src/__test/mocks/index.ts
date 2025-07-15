// Re-export all mocks for easy importing
export * from "./libraries/react-i18next.js";
export * from "./libraries/zxcvbn.js";
export * from "./libraries/mui.js";
export * from "./utils/authentication.js";
export * from "./utils/router.js";
export * from "./utils/api.js";
export * from "./utils/socket.js";
export * from "./components/icons.js";

// Pre-configured mock combinations for common scenarios
export const commonFormMocks = {
  "react-i18next": "./libraries/react-i18next.js",
  "zxcvbn": "./libraries/zxcvbn.js",
  "@mui/material/styles": "./libraries/mui.js",
};

export const commonAuthMocks = {
  "react-i18next": "./libraries/react-i18next.js",
  "../../utils/authentication/session.js": "./utils/authentication.js",
  "../../route/router.js": "./utils/router.js",
  "../../api/socket.js": "./utils/socket.js",
};

export const commonComponentMocks = {
  "../../icons/Icons.js": "./components/icons.js",
  "@mui/material/styles": "./libraries/mui.js",
  "react-i18next": "./libraries/react-i18next.js",
};
