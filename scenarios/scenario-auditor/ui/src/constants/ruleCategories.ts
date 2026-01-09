/**
 * Rule category configuration and styling constants
 * Shared between RulesManager and ScenarioTestResults components
 */

export const TARGET_BADGE_CLASSES: Record<string, string> = {
  api: 'bg-blue-100 text-blue-800',
  main_go: 'bg-indigo-100 text-indigo-800',
  ui: 'bg-pink-100 text-pink-800',
  cli: 'bg-green-100 text-green-800',
  test: 'bg-yellow-100 text-yellow-800',
  service_json: 'bg-purple-100 text-purple-800',
  makefile: 'bg-orange-100 text-orange-800',
  structure: 'bg-slate-100 text-slate-800',
  documentation: 'bg-amber-100 text-amber-800',
  external: 'bg-sky-100 text-sky-800',
  misc: 'bg-gray-100 text-gray-700 border border-gray-300',
}

export const TARGET_CATEGORY_CONFIG: Array<{ id: string; label: string; description: string }> = [
  { id: 'api', label: 'API Files', description: 'Rules applied to files within the scenario\'s api/ directory.' },
  { id: 'main_go', label: 'main.go', description: 'Rules evaluated specifically against api/main.go or other lifecycle entrypoints.' },
  { id: 'ui', label: 'UI Files', description: 'Rules focused on assets within ui/ such as React components or static markup.' },
  { id: 'cli', label: 'CLI Files', description: 'Rules covering the CLI implementation under cli/.' },
  { id: 'test', label: 'Test Files', description: 'Rules that target files under test/ and other testing utilities.' },
  { id: 'service_json', label: 'service.json', description: 'Rules that run against .vrooli/service.json lifecycle configuration.' },
  { id: 'makefile', label: 'Makefile', description: 'Rules focused on the scenario Makefile lifecycle wrapper.' },
  { id: 'structure', label: 'Scenario Structure', description: 'Rules that validate high-level scenario layout and required assets.' },
  { id: 'documentation', label: 'Documentation', description: 'Rules that enforce PRDs, READMEs, and docs/ content quality.' },
  { id: 'external', label: 'External Providers', description: 'Rules executed in other scenarios (for example prd-control-tower).' },
  { id: 'misc', label: 'Miscellaneous', description: 'Rules missing targets; update the rule metadata so it runs during scans.' },
]
