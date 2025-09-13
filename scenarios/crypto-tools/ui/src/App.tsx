import { useState, useEffect } from 'react'
import axios from 'axios'
import { Shield, Lock, Key, Hash, FileCheck, AlertCircle, CheckCircle } from 'lucide-react'

interface HealthStatus {
  status: string
  service: string
  timestamp: string
  readiness: boolean
  dependencies?: Record<string, any>
  crypto_stats?: {
    total_operations: number
    hash_operations: number
    encryption_operations: number
    signature_operations: number
    key_operations: number
  }
}

function App() {
  const [apiHealth, setApiHealth] = useState<HealthStatus | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const response = await axios.get<HealthStatus>('/api/health')
        setApiHealth(response.data)
        setError(null)
      } catch (err) {
        setError('Failed to connect to API')
        console.error('Health check failed:', err)
      }
    }

    checkHealth()
    const interval = setInterval(checkHealth, 30000)
    return () => clearInterval(interval)
  }, [])

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-green-500'
      case 'degraded':
        return 'text-yellow-500'
      case 'unhealthy':
        return 'text-red-500'
      default:
        return 'text-gray-500'
    }
  }

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <div className="container mx-auto px-4 py-8">
        <header className="mb-8">
          <div className="flex items-center gap-3 mb-4">
            <Shield className="w-10 h-10 text-blue-400" />
            <h1 className="text-4xl font-bold">Crypto Tools</h1>
          </div>
          <p className="text-gray-400">Cryptography & Blockchain Utilities</p>
        </header>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Health Status Card */}
          <div className="bg-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold">API Health</h2>
              {apiHealth ? (
                <CheckCircle className={`w-6 h-6 ${getStatusColor(apiHealth.status)}`} />
              ) : error ? (
                <AlertCircle className="w-6 h-6 text-red-500" />
              ) : (
                <div className="w-6 h-6 border-2 border-blue-400 border-t-transparent rounded-full animate-spin" />
              )}
            </div>
            {apiHealth ? (
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-400">Status:</span>
                  <span className={getStatusColor(apiHealth.status)}>{apiHealth.status}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Service:</span>
                  <span>{apiHealth.service}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Ready:</span>
                  <span>{apiHealth.readiness ? 'Yes' : 'No'}</span>
                </div>
              </div>
            ) : error ? (
              <p className="text-red-400">{error}</p>
            ) : (
              <p className="text-gray-400">Checking health...</p>
            )}
          </div>

          {/* Crypto Operations Stats */}
          {apiHealth?.crypto_stats && (
            <div className="bg-gray-800 rounded-lg p-6">
              <h2 className="text-xl font-semibold mb-4">Operations Stats</h2>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-gray-400">Total Operations:</span>
                  <span>{apiHealth.crypto_stats.total_operations}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Hash Operations:</span>
                  <span>{apiHealth.crypto_stats.hash_operations}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Encryptions:</span>
                  <span>{apiHealth.crypto_stats.encryption_operations}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Signatures:</span>
                  <span>{apiHealth.crypto_stats.signature_operations}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-gray-400">Key Operations:</span>
                  <span>{apiHealth.crypto_stats.key_operations}</span>
                </div>
              </div>
            </div>
          )}

          {/* Features */}
          <div className="bg-gray-800 rounded-lg p-6">
            <h2 className="text-xl font-semibold mb-4">Features</h2>
            <div className="space-y-3">
              <div className="flex items-center gap-2">
                <Hash className="w-5 h-5 text-blue-400" />
                <span>Hashing Algorithms</span>
              </div>
              <div className="flex items-center gap-2">
                <Lock className="w-5 h-5 text-blue-400" />
                <span>Encryption/Decryption</span>
              </div>
              <div className="flex items-center gap-2">
                <Key className="w-5 h-5 text-blue-400" />
                <span>Key Generation</span>
              </div>
              <div className="flex items-center gap-2">
                <FileCheck className="w-5 h-5 text-blue-400" />
                <span>Digital Signatures</span>
              </div>
            </div>
          </div>

          {/* Dependencies Status */}
          {apiHealth?.dependencies && (
            <div className="bg-gray-800 rounded-lg p-6 md:col-span-2 lg:col-span-3">
              <h2 className="text-xl font-semibold mb-4">Dependencies</h2>
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                {Object.entries(apiHealth.dependencies).map(([name, dep]: [string, any]) => (
                  <div key={name} className="bg-gray-700 rounded p-3">
                    <div className="flex items-center justify-between mb-2">
                      <span className="font-medium capitalize">{name}</span>
                      <span className={`text-sm ${getStatusColor(dep.status)}`}>
                        {dep.status}
                      </span>
                    </div>
                    {dep.checks && (
                      <div className="text-xs text-gray-400">
                        {Object.entries(dep.checks).slice(0, 2).map(([check, value]) => (
                          <div key={check} className="flex justify-between">
                            <span>{check}:</span>
                            <span>{String(value)}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default App