const lifecycleManagedEnvVar = 'VROOLI_LIFECYCLE_MANAGED'

export interface LifecycleGuardOptions {
  serviceName?: string
  disableLifecycleGuard?: boolean
}

function isTestEnvironment(): boolean {
  return process.env.NODE_ENV === 'test' || process.env.VITEST === 'true'
}

export function assertLifecycleManagedUI(options: LifecycleGuardOptions = {}): void {
  if (options.disableLifecycleGuard || isTestEnvironment()) {
    return
  }

  if (process.env[lifecycleManagedEnvVar] === 'true') {
    return
  }

  const serviceName = options.serviceName?.trim() || '<scenario-name>'
  throw new Error(`This UI must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start ${serviceName}

The lifecycle system provides environment variables, port allocation,
and dependency management automatically. Direct execution is not supported.
`)
}
