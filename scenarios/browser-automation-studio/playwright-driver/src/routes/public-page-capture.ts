import type { IncomingMessage, ServerResponse } from 'node:http';
import type { SessionManager } from '../session';
import type { Config } from '../config';
import { parseJsonBody, sendError, sendJson } from '../middleware';
import { capturePublicPageFromURL, captureScenarioPage } from '../capture/public-page';
import { captureHTMLFile } from '../capture/public-page';
import { PublicEgressError } from '../capture/public-egress';
import { InvalidInstructionError } from '../utils';

type PublicCaptureBody = {
  url?: unknown;
  width?: unknown;
  height?: unknown;
  theme?: unknown;
};
type FileCaptureBody = {
  html?: unknown;
  filename?: unknown;
  width?: unknown;
  height?: unknown;
  theme?: unknown;
};
type ScenarioCaptureBody = {
  url?: unknown;
  scenario?: unknown;
  route?: unknown;
  width?: unknown;
  height?: unknown;
  steps?: unknown;
  wait_selector?: unknown;
  wait_hidden_selector?: unknown;
  theme?: unknown;
  strict_tokens?: unknown;
  allowed_origins?: unknown;
  evidence_profile?: unknown;
  region_selector?: unknown;
  full_page_pixels?: unknown;
};

function positiveDimension(value: unknown, field: string): number | undefined {
  if (value === undefined) return undefined;
  const lower = field === 'width' ? 320 : 240;
  const upper = field === 'width' ? 2560 : 1600;
  if (typeof value !== 'number' || !Number.isInteger(value) || value < lower || value > upper)
    throw new InvalidInstructionError(`${field} must be an integer between ${lower} and ${upper}`);
  return value;
}

/** POST /ux-capture/file: bounded HTML bytes only, isolated script-disabled context, deny-all network. */
export async function handleHTMLFileCapture(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const body = (await parseJsonBody(req, config)) as FileCaptureBody;
    const unknown = Object.keys(body).filter(
      (key) => !['html', 'filename', 'width', 'height', 'theme'].includes(key)
    );
    if (unknown.length)
      throw new InvalidInstructionError(
        `unsupported capture request field(s): ${unknown.join(', ')}`
      );
    if (
      typeof body.html !== 'string' ||
      Buffer.byteLength(body.html, 'utf8') < 1 ||
      Buffer.byteLength(body.html, 'utf8') > 2 * 1024 * 1024
    )
      throw new InvalidInstructionError('html must contain between 1 byte and 2 MiB');
    if (typeof body.filename !== 'string' || !body.filename.trim())
      throw new InvalidInstructionError('filename is required');
    const width = positiveDimension(body.width, 'width'),
      height = positiveDimension(body.height, 'height');
    const theme = captureTheme(body.theme);
    const result = await sessionManager.withPublicCaptureBrowser((browser) =>
      captureHTMLFile(browser, {
        url: '',
        html: body.html as string,
        filename: body.filename as string,
        width,
        height,
        theme,
      })
    );
    sendJson(res, 200, result);
  } catch (error) {
    sendError(res, error as Error, '/ux-capture/file');
  }
}

/** POST /ux-capture/scenario: capture one resolved loopback scenario page and explicit state steps. */
export async function handleScenarioPageCapture(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const body = (await parseJsonBody(req, config)) as ScenarioCaptureBody;
    const allowedFields = [
      'url',
      'scenario',
      'route',
      'width',
      'height',
      'steps',
      'wait_selector',
      'wait_hidden_selector',
      'theme',
      'strict_tokens',
      'allowed_origins',
      'evidence_profile',
      'region_selector',
      'full_page_pixels',
    ];
    const unknown = Object.keys(body).filter((key) => !allowedFields.includes(key));
    if (unknown.length)
      throw new InvalidInstructionError(
        `unsupported capture request field(s): ${unknown.join(', ')}`
      );
    if (typeof body.url !== 'string' || !body.url.trim())
      throw new InvalidInstructionError('resolved scenario url is required');
    if (typeof body.scenario !== 'string' || !/^[a-z0-9][a-z0-9-]{0,62}$/.test(body.scenario))
      throw new InvalidInstructionError('scenario must be a valid scenario slug');
    if (
      typeof body.route !== 'string' ||
      !body.route.startsWith('/') ||
      body.route.startsWith('//')
    )
      throw new InvalidInstructionError('route must be a local path beginning with one slash');
    const width = positiveDimension(body.width, 'width'),
      height = positiveDimension(body.height, 'height');
    const steps = body.steps === undefined ? [] : body.steps;
    if (
      !Array.isArray(steps) ||
      steps.length > 32 ||
      steps.some(
        (step) => typeof step !== 'string' || step.length > 256 || !/^click:.+/i.test(step.trim())
      )
    )
      throw new InvalidInstructionError('steps must be up to 32 click:<accessible label> actions');
    const allowedOrigins = body.allowed_origins === undefined ? [] : body.allowed_origins;
    const waitSelector = body.wait_selector === undefined ? '' : body.wait_selector;
    if (typeof waitSelector !== 'string' || waitSelector.length > 512)
      throw new InvalidInstructionError(
        'wait_selector must be a CSS selector up to 512 characters'
      );
    const waitHiddenSelector = body.wait_hidden_selector === undefined ? '' : body.wait_hidden_selector;
    if (typeof waitHiddenSelector !== 'string' || waitHiddenSelector.length > 512)
      throw new InvalidInstructionError(
        'wait_hidden_selector must be a CSS selector up to 512 characters'
      );
    const theme = body.theme === undefined ? 'light' : body.theme;
    if (theme !== 'light' && theme !== 'dark' && theme !== 'no-preference')
      throw new InvalidInstructionError('theme must be light, dark, or no-preference');
    if (body.strict_tokens !== undefined && typeof body.strict_tokens !== 'boolean')
      throw new InvalidInstructionError('strict_tokens must be a boolean');
    if (body.full_page_pixels !== undefined && typeof body.full_page_pixels !== 'boolean')
      throw new InvalidInstructionError('full_page_pixels must be a boolean');
    if (body.evidence_profile !== undefined && body.evidence_profile !== 'componenttests')
      throw new InvalidInstructionError('evidence_profile must be componenttests');
    if (
      body.region_selector !== undefined &&
      (typeof body.region_selector !== 'string' ||
        body.region_selector.length > 256 ||
        body.region_selector.startsWith('//'))
    )
      throw new InvalidInstructionError(
        'region_selector must be a CSS selector up to 256 characters'
      );
    if (
      !Array.isArray(allowedOrigins) ||
      allowedOrigins.length > 4 ||
      allowedOrigins.some((value) => typeof value !== 'string')
    )
      throw new InvalidInstructionError('allowed_origins must contain at most four URLs');
    const result = await sessionManager.withPublicCaptureBrowser((browser) =>
      captureScenarioPage(browser, {
        url: body.url as string,
        scenario: body.scenario as string,
        route: body.route as string,
        width,
        height,
        steps: steps as string[],
        waitSelector,
        waitHiddenSelector,
        theme,
        strictTokens: body.strict_tokens === true,
        fullPagePixels: body.full_page_pixels === true,
        evidenceProfile: body.evidence_profile as 'componenttests' | undefined,
        regionSelector: body.region_selector as string | undefined,
        allowedOrigins: allowedOrigins as string[],
      })
    );
    sendJson(res, 200, result);
  } catch (error) {
    sendError(res, error as Error, '/ux-capture/scenario');
  }
}

/** POST /ux-capture/public: one fresh, unauthenticated public-page session. */
export async function handlePublicPageCapture(
  req: IncomingMessage,
  res: ServerResponse,
  sessionManager: SessionManager,
  config: Config
): Promise<void> {
  try {
    const body = (await parseJsonBody(req, config)) as PublicCaptureBody;
    const unknownFields = Object.keys(body).filter(
      (key) => !['url', 'width', 'height', 'theme'].includes(key)
    );
    if (unknownFields.length)
      throw new InvalidInstructionError(
        `unsupported capture request field(s): ${unknownFields.join(', ')}`
      );
    const url = body.url;
    if (typeof url !== 'string' || !url.trim())
      throw new InvalidInstructionError('url is required');
    const width = positiveDimension(body.width, 'width');
    const height = positiveDimension(body.height, 'height');
    const theme = captureTheme(body.theme);
    const result = await sessionManager.withPublicCaptureBrowser((browser) =>
      capturePublicPageFromURL(browser, {
        url: url.trim(),
        width,
        height,
        theme,
      })
    );
    sendJson(res, 200, result);
  } catch (error) {
    if (error instanceof PublicEgressError) {
      sendJson(res, 403, { error: { code: 'PUBLIC_DESTINATION_BLOCKED', message: error.message } });
      return;
    }
    sendError(res, error as Error, '/ux-capture/public');
  }
}

function captureTheme(value: unknown): 'light' | 'dark' | 'no-preference' {
  if (value === undefined) return 'light';
  if (value === 'light' || value === 'dark' || value === 'no-preference') return value;
  throw new InvalidInstructionError('theme must be light, dark, or no-preference');
}
