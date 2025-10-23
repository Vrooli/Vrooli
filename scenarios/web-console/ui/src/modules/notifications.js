let snackbarModule = null;

export async function showToast(message, type = "info", duration = 3000) {
  try {
    if (!snackbarModule) {
      snackbarModule = await import("./mobile-toolbar/index.js");
    }
    if (snackbarModule && typeof snackbarModule.showSnackbar === "function") {
      snackbarModule.showSnackbar(message, type, duration);
    }
  } catch (error) {
    console.warn("Unable to display snackbar notification:", error, message);
  }
}
