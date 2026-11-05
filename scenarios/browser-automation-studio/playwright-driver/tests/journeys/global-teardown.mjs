export default async function globalTeardown() {
  await globalThis.__BAS_JOURNEY_SITE__?.close();
}
