/**
 * HTML injection for the experience telemetry agent.
 *
 * The injected tag is a module script. Configuration travels in data
 * attributes rather than an inline JSON script so a strict Content-Security
 * Policy has nothing inline to refuse, and so the module can find its own
 * settings without a second request.
 */

import { EXPERIENCE_AGENT_ATTRIBUTE } from './types.js'

export interface ExperienceInjectionConfig {
  assetPath: string
  ingestPath: string
  scenario: string
  sample: number
  captureLabels: boolean
  performance: boolean
  maxEvents: number
  build?: string
}

function escapeAttribute(value: string): string {
  return value
    .replace(/&/g, '&amp;')
    .replace(/"/g, '&quot;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

/** Build the script tag. Exported for tests and for host-level injectors. */
export function buildExperienceScriptTag(config: ExperienceInjectionConfig): string {
  const attributes = [
    `type="module"`,
    `src="${escapeAttribute(config.assetPath)}"`,
    `${EXPERIENCE_AGENT_ATTRIBUTE}="1"`,
    `data-endpoint="${escapeAttribute(config.ingestPath)}"`,
    `data-scenario="${escapeAttribute(config.scenario)}"`,
    `data-sample="${config.sample}"`,
    `data-labels="${config.captureLabels ? '1' : '0'}"`,
    `data-performance="${config.performance ? '1' : '0'}"`,
    `data-max-events="${Math.max(1, Math.floor(config.maxEvents))}"`,
  ]
  if (config.build) {
    attributes.push(`data-build="${escapeAttribute(config.build)}"`)
  }
  return `<script ${attributes.join(' ')}></script>`
}

/**
 * Inject the agent into an HTML document. Idempotent: a document that already
 * carries the attribute is returned unchanged, so stacked injection layers
 * (UI server plus hosting proxy) cannot double-install.
 */
export function injectExperienceTelemetry(html: string, config: ExperienceInjectionConfig): string {
  if (!html) return html
  if (new RegExp(`${EXPERIENCE_AGENT_ATTRIBUTE}\\s*=`).test(html)) return html

  const tag = buildExperienceScriptTag(config)
  const lower = html.toLowerCase()

  const headEnd = lower.indexOf('</head>')
  if (headEnd !== -1) {
    return `${html.slice(0, headEnd)}${tag}\n${html.slice(headEnd)}`
  }

  const bodyEnd = lower.indexOf('</body>')
  if (bodyEnd !== -1) {
    return `${html.slice(0, bodyEnd)}${tag}\n${html.slice(bodyEnd)}`
  }

  return `${tag}\n${html}`
}
