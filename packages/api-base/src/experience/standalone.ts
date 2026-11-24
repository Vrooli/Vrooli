/**
 * Experience telemetry for custom Express servers.
 *
 * `createScenarioServer` installs injection and ingest automatically. Servers
 * that predate it can call this once, before their static middleware, to gain
 * the same behaviour without adopting the full shared server:
 *
 * ```ts
 * import { installExperienceTelemetry } from '@vrooli/api-base/experience'
 * installExperienceTelemetry(app, { serviceName: 'my-scenario', distDir })
 * ```
 *
 * It wraps `res.send` and HTML `res.sendFile` responses, serves the agent, and
 * mounts ingest/status. Every route is loopback-gated and inert when disabled.
 */

import type { Express, NextFunction, Request, Response } from 'express'
import * as fs from 'node:fs'
import * as path from 'node:path'
import {
  experienceInjectionConfig,
  resolveExperienceTelemetry,
  shouldInstrumentRequest,
} from './gate.js'
import { injectExperienceTelemetry } from './inject.js'
import { createExperienceTelemetryRouter } from './router.js'
import type {
  ExperienceTelemetryOptions,
  ResolvedExperienceTelemetry,
} from './types.js'

export interface StandaloneExperienceOptions extends ExperienceTelemetryOptions {
  /** Scenario name used for the session directory and evidence identity. */
  serviceName: string
  /** Directory containing index.html and built assets. */
  distDir?: string
  /** Explicit entry document; wins over `distDir/index.html`. */
  indexFile?: string
}

function resolveIndexFile(options: StandaloneExperienceOptions): string | null {
  const candidates = options.indexFile
    ? [options.indexFile]
    : [
      ...(options.distDir ? [path.join(options.distDir, 'index.html')] : []),
      path.resolve(process.cwd(), 'dist', 'index.html'),
      path.resolve(process.cwd(), 'index.html'),
    ]
  for (const candidate of candidates) {
    try {
      if (fs.statSync(candidate).isFile()) return candidate
    } catch {
      // next
    }
  }
  return null
}

/**
 * Install telemetry on a custom Express app. Call this as early as possible
 * (immediately after creating the app, before `express.static` and routes).
 *
 * Returns the resolved settings, or null when telemetry is off.
 */
export function installExperienceTelemetry(
  app: Express,
  options: StandaloneExperienceOptions,
): ResolvedExperienceTelemetry | null {
  const telemetry = resolveExperienceTelemetry(
    { ...options, serviceName: undefined } as ExperienceTelemetryOptions,
    process.env,
    options.serviceName,
  )
  if (!telemetry) return null

  const indexFile = resolveIndexFile(options)
  const injectionConfig = experienceInjectionConfig(telemetry)

  // Specific routes first; they cannot collide with app routes.
  app.use(createExperienceTelemetryRouter(telemetry))

  // Wrap responses before registering the entry route: Express resolves routes
  // in order, and a route that sends before this middleware runs would bypass
  // the wrapper.
  app.use((req: Request, res: Response, next: NextFunction) => {
    const originalSend = res.send
    const originalSendFile = res.sendFile

    res.send = function patchedSend(body: unknown): Response {
      const contentType = res.getHeader('content-type')
      const isHtml = typeof contentType === 'string' && contentType.includes('text/html')
      if (isHtml && typeof body === 'string' && shouldInstrumentRequest(req, telemetry)) {
        return originalSend.call(this, injectExperienceTelemetry(body, injectionConfig))
      }
      return originalSend.call(this, body as string)
    }

    res.sendFile = function patchedSendFile(filePath: string, ...args: unknown[]): void {
      try {
        if (
          typeof filePath === 'string'
          && filePath.toLowerCase().endsWith('.html')
          && shouldInstrumentRequest(req, telemetry)
        ) {
          const html = fs.readFileSync(path.resolve(filePath), 'utf-8')
          res.type('html')
          res.send(html)
          return
        }
      } catch {
        // Fall through to the original so its own 404/500 behaviour is kept.
      }
      ;(originalSendFile as unknown as (...callArgs: unknown[]) => void).call(res, filePath, ...args)
    } as Response['sendFile']

    next()
  })

  // Serve the entry document through res.send so the wrapper above sees it.
  // `express.static` would stream it directly and bypass the wrapper.
  if (indexFile) {
    app.get(['/', '/index.html'], (_req: Request, res: Response, next: NextFunction) => {
      try {
        const html = fs.readFileSync(indexFile, 'utf-8')
        res.setHeader('Cache-Control', 'no-cache')
        res.type('html').send(html)
      } catch (error) {
        next(error)
      }
    })
  }

  return telemetry
}
