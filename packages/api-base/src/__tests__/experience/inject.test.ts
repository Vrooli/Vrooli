import { describe, expect, it } from 'vitest'
import * as fs from 'node:fs'
import * as path from 'node:path'
import { fileURLToPath } from 'node:url'
import { buildExperienceScriptTag, injectExperienceTelemetry } from '../../experience/inject.js'
import { EXPERIENCE_AGENT_ATTRIBUTE } from '../../experience/types.js'

const config = {
  assetPath: '/__vrooli/experience/agent.js',
  ingestPath: '/__vrooli/experience/events',
  scenario: 'git-control-tower',
  sample: 1,
  captureLabels: true,
  performance: true,
  maxEvents: 5000,
}

describe('buildExperienceScriptTag', () => {
  it('emits a module script with data-attribute configuration', () => {
    const tag = buildExperienceScriptTag({ ...config, sample: 0.25, build: '1.2.3' })
    expect(tag).toContain('type="module"')
    expect(tag).toContain('src="/__vrooli/experience/agent.js"')
    expect(tag).toContain(`${EXPERIENCE_AGENT_ATTRIBUTE}="1"`)
    expect(tag).toContain('data-endpoint="/__vrooli/experience/events"')
    expect(tag).toContain('data-scenario="git-control-tower"')
    expect(tag).toContain('data-sample="0.25"')
    expect(tag).toContain('data-build="1.2.3"')
  })

  it('escapes attribute values', () => {
    const tag = buildExperienceScriptTag({ ...config, scenario: 'a"b<c>&d' })
    expect(tag).toContain('data-scenario="a&quot;b&lt;c&gt;&amp;d"')
  })

  it('records opt-outs for labels and performance', () => {
    const tag = buildExperienceScriptTag({ ...config, captureLabels: false, performance: false })
    expect(tag).toContain('data-labels="0"')
    expect(tag).toContain('data-performance="0"')
  })
})

describe('injectExperienceTelemetry', () => {
  it('injects before </head>', () => {
    const html = '<!doctype html><html><head><title>x</title></head><body></body></html>'
    const result = injectExperienceTelemetry(html, config)
    expect(result.indexOf(EXPERIENCE_AGENT_ATTRIBUTE)).toBeLessThan(result.indexOf('</head>'))
    expect(result).toContain('<title>x</title>')
  })

  it('falls back to </body> and then the start of the document', () => {
    expect(injectExperienceTelemetry('<html><body>x</body></html>', config))
      .toContain('x<script')
    expect(injectExperienceTelemetry('<div>x</div>', config).startsWith('<script')).toBe(true)
  })

  it('is idempotent so stacked injection layers cannot double-install', () => {
    const html = '<html><head></head><body></body></html>'
    const once = injectExperienceTelemetry(html, config)
    const twice = injectExperienceTelemetry(once, config)
    expect(twice).toBe(once)
  })

  it('keeps the served agent in sync with the injection attribute', () => {
    const here = path.dirname(fileURLToPath(import.meta.url))
    const source = fs.readFileSync(path.resolve(here, '../../experience/agent.ts'), 'utf-8')
    expect(source).toContain(`'${EXPERIENCE_AGENT_ATTRIBUTE}'`)
  })
})
