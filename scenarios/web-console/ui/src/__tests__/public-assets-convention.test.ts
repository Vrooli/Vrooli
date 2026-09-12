import { describe, expect, it } from "vitest";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";

/**
 * Enforces the fleet `/public/*` asset convention for web-console
 * (see docs/concepts/PUBLIC_ASSETS.md and the public-asset-access-bypass
 * plan). Branding / PWA / OG assets MUST serve under the URL path prefix
 * `/public/` so the tunnel-manager Cloudflare Access bypass (scoped to
 * `<host>/public`) can make them fetchable by anonymous clients (iOS
 * Add-to-Home-Screen, OG crawlers) without weakening Access elsewhere.
 *
 * The manifest's `start_url` / `scope` / `id` must point at the app root
 * (`/`), NOT into `/public/`, or the installed PWA would launch into the
 * asset folder instead of the app.
 */

const THIS_FILE: string = fileURLToPath(import.meta.url);
const UI_ROOT: string = resolve(THIS_FILE, "..", "..", "..");
const PUBLIC_DIR: string = resolve(UI_ROOT, "public", "public");

const INDEX_HTML: string = readFileSync(resolve(UI_ROOT, "index.html"), "utf8");

interface ManifestIcon {
  src: string;
  sizes?: string;
  type?: string;
  purpose?: string;
}

interface WebManifest {
  start_url?: string;
  scope?: string;
  id?: string;
  icons?: ManifestIcon[];
}

function readManifest(): WebManifest {
  return JSON.parse(readFileSync(resolve(PUBLIC_DIR, "site.webmanifest"), "utf8")) as WebManifest;
}

// A reference "resolves under /public/" if, after stripping a single
// leading `./` or `/`, it begins with `public/`. Vite's `base: './'`
// rewrites absolute `<link href="/public/x">` to relative `./public/x`
// at build time; both resolve to `/public/x` because index.html is served
// at the app root. Accept either form.
function resolvesUnderPublic(ref: string): boolean {
  const normalized = ref.replace(/^\.?\//, "");
  return normalized.startsWith("public/");
}

describe("public-assets convention: web-console branding/PWA/OG under /public/", () => {
  it("every <link> icon/apple-touch/manifest href resolves under /public/", () => {
    const re = /<link\b[^>]*\brel="(icon|apple-touch-icon|manifest)"[^>]*\bhref="([^"]+)"/g;
    const refs: Array<{ rel: string; href: string }> = [];
    let m: RegExpExecArray | null;
    while ((m = re.exec(INDEX_HTML)) !== null) {
      const [, rel, href] = m;
      if (rel === undefined || href === undefined) continue;
      refs.push({ rel, href });
    }
    // Sanity: we expect at least the svg icon, png favicons, apple-touch, manifest.
    expect(refs.length).toBeGreaterThanOrEqual(5);
    for (const { rel, href } of refs) {
      expect(resolvesUnderPublic(href), `${rel} href "${href}" must resolve under /public/`).toBe(
        true,
      );
    }
  });

  it("og:image / twitter:image meta resolve under /public/", () => {
    const re = /<meta\b[^>]*\b(?:property|name)="(og:image|twitter:image)"[^>]*\bcontent="([^"]+)"/g;
    const refs: Array<{ key: string; content: string }> = [];
    let m: RegExpExecArray | null;
    while ((m = re.exec(INDEX_HTML)) !== null) {
      const [, key, content] = m;
      if (key === undefined || content === undefined) continue;
      refs.push({ key, content });
    }
    expect(refs.map((r) => r.key).sort()).toEqual(["og:image", "twitter:image"]);
    for (const { key, content } of refs) {
      expect(
        resolvesUnderPublic(content),
        `${key} content "${content}" must resolve under /public/`,
      ).toBe(true);
    }
  });

  it("no branding asset is referenced from the URL root (anti-regression)", () => {
    // Catch a relapse where an icon/manifest/og ref drops the /public/ prefix.
    const rootRefs = [
      /href="\/(?:logo\.svg|favicon-\d+\.png|apple-touch-icon\.png|site\.webmanifest)"/,
      /content="\.?\/(?:og-image\.png)"/,
    ];
    for (const pattern of rootRefs) {
      expect(pattern.test(INDEX_HTML), `root-level asset ref matched ${pattern}`).toBe(false);
    }
  });

  it("the manifest lives at /public/site.webmanifest", () => {
    expect(/rel="manifest"[^>]*href="\.?\/public\/site\.webmanifest"/.test(INDEX_HTML)).toBe(true);
    expect(existsSync(resolve(PUBLIC_DIR, "site.webmanifest"))).toBe(true);
  });

  it("manifest start_url / scope / id point at the app root, not /public/", () => {
    const manifest = readManifest();
    // The installed PWA must launch into the app root. A value of "." would
    // resolve relative to the manifest URL (/public/site.webmanifest) and
    // wrongly scope the app to /public/.
    expect(manifest.start_url).toBe("/");
    expect(manifest.scope).toBe("/");
    expect(manifest.id).toBe("/");
  });

  it("manifest icon srcs are co-located in /public/ (resolve relative to the manifest URL)", () => {
    const manifest = readManifest();
    const icons = manifest.icons ?? [];
    expect(icons.length).toBeGreaterThan(0);
    for (const icon of icons) {
      const src: string = icon.src;
      // Bare relative srcs resolve against /public/site.webmanifest → /public/<src>.
      // They must NOT escape the public dir or use an absolute root path.
      expect(src.startsWith("/"), `icon src "${src}" must be relative to the manifest`).toBe(false);
      expect(src.includes(".."), `icon src "${src}" must not traverse up`).toBe(false);
      expect(
        existsSync(resolve(PUBLIC_DIR, src)),
        `icon src "${src}" must exist in /public/`,
      ).toBe(true);
    }
  });

  it("the manifest <link> carries crossorigin=use-credentials", () => {
    // Desktop Chromium installs carry the auth cookie when fetching the
    // manifest; without use-credentials the install can fail behind Access.
    expect(
      /rel="manifest"[^>]*crossorigin="use-credentials"/.test(INDEX_HTML),
    ).toBe(true);
  });
});
