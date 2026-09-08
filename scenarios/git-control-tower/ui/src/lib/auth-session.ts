// Browser credentials are installed by the GCT API as an HttpOnly cookie.
// Keep this module credential-free so a future UI caller cannot accidentally
// reintroduce browser-readable bearer-token handling.
export {};
