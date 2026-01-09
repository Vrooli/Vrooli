import { Scenario } from '@/types/api'
import { Shield, X, Lock, Unlock, Search } from 'lucide-react'
import { useState, useMemo, useEffect } from 'react'

interface ManageProtectedScenariosDialogProps {
  isOpen: boolean
  onClose: () => void
  scenarios: Scenario[]
  protectedScenarios: string[]
  onSave: (scenarios: string[]) => Promise<void>
}

export default function ManageProtectedScenariosDialog({
  isOpen,
  onClose,
  scenarios,
  protectedScenarios,
  onSave,
}: ManageProtectedScenariosDialogProps) {
  const [selectedScenarios, setSelectedScenarios] = useState<Set<string>>(new Set())
  const [searchTerm, setSearchTerm] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Initialize selection from existing protected scenarios
  useEffect(() => {
    if (isOpen) {
      setSelectedScenarios(new Set(protectedScenarios))
      setSearchTerm('')
      setError(null)
    }
  }, [isOpen, protectedScenarios])

  const filteredScenarios = useMemo(() => {
    const term = searchTerm.trim().toLowerCase()
    if (!term) return scenarios

    return scenarios.filter(s =>
      s.name.toLowerCase().includes(term) ||
      (s.description && s.description.toLowerCase().includes(term))
    )
  }, [scenarios, searchTerm])

  const handleToggleScenario = (scenarioName: string) => {
    const newSelection = new Set(selectedScenarios)
    if (newSelection.has(scenarioName)) {
      newSelection.delete(scenarioName)
    } else {
      newSelection.add(scenarioName)
    }
    setSelectedScenarios(newSelection)
  }

  const handleSelectAll = () => {
    if (selectedScenarios.size === filteredScenarios.length && filteredScenarios.length > 0) {
      // Deselect all filtered scenarios
      const newSelection = new Set(selectedScenarios)
      filteredScenarios.forEach(s => newSelection.delete(s.name))
      setSelectedScenarios(newSelection)
    } else {
      // Select all filtered scenarios
      const newSelection = new Set(selectedScenarios)
      filteredScenarios.forEach(s => newSelection.add(s.name))
      setSelectedScenarios(newSelection)
    }
  }

  const handleSave = async () => {
    setError(null)
    setIsSaving(true)

    try {
      await onSave(Array.from(selectedScenarios))
      onClose()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save protected scenarios')
    } finally {
      setIsSaving(false)
    }
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[70] overflow-y-auto">
      <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
        <div
          className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity"
          onClick={onClose}
        />

        <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-2xl sm:p-6">
          <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
            <button
              type="button"
              className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
              onClick={onClose}
              disabled={isSaving}
            >
              <span className="sr-only">Close</span>
              <X className="h-6 w-6" />
            </button>
          </div>

          <div className="sm:flex sm:items-start">
            <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10">
              <Shield className="h-6 w-6 text-blue-600" />
            </div>
            <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left flex-1">
              <h3 className="text-base font-semibold leading-6 text-gray-900">
                Manage Protected Scenarios
              </h3>
              <p className="mt-2 text-sm text-gray-500">
                Protected scenarios are automatically excluded from rule testing and issue reporting operations. This prevents accidental modifications to critical scenarios like ecosystem-manager and app-issue-tracker.
              </p>
            </div>
          </div>

          <div className="mt-5 space-y-4">
            {/* Search */}
            <div className="relative">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search scenarios..."
                className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-md leading-5 bg-white placeholder-gray-500 focus:outline-none focus:placeholder-gray-400 focus:ring-1 focus:ring-blue-500 focus:border-blue-500 text-sm"
              />
            </div>

            {/* Select All Toggle */}
            {filteredScenarios.length > 0 && (
              <div className="flex items-center justify-between px-3 py-2 bg-gray-50 rounded-md">
                <span className="text-sm text-gray-600">
                  {selectedScenarios.size} of {scenarios.length} scenarios protected
                </span>
                <button
                  type="button"
                  onClick={handleSelectAll}
                  className="text-xs font-medium text-blue-600 hover:text-blue-700 hover:underline"
                >
                  {selectedScenarios.size === filteredScenarios.length && filteredScenarios.length > 0
                    ? 'Deselect All Filtered'
                    : 'Select All Filtered'}
                </button>
              </div>
            )}

            {/* Scenario List */}
            <div className="max-h-96 overflow-y-auto rounded-md border border-gray-200 bg-white">
              {filteredScenarios.length === 0 ? (
                <div className="px-4 py-8 text-center text-sm text-gray-500">
                  {scenarios.length === 0
                    ? 'No scenarios available'
                    : 'No scenarios match your search'}
                </div>
              ) : (
                <ul className="divide-y divide-gray-200">
                  {filteredScenarios.map(scenario => {
                    const isProtected = selectedScenarios.has(scenario.name)
                    return (
                      <li
                        key={scenario.name}
                        className="px-4 py-3 hover:bg-gray-50 cursor-pointer transition-colors"
                        onClick={() => handleToggleScenario(scenario.name)}
                      >
                        <div className="flex items-center gap-3">
                          <input
                            type="checkbox"
                            checked={isProtected}
                            onChange={() => handleToggleScenario(scenario.name)}
                            onClick={(e) => e.stopPropagation()}
                            className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="text-sm font-medium text-gray-900">{scenario.name}</span>
                              {isProtected && (
                                <Lock className="h-3.5 w-3.5 text-blue-600" aria-label="Protected" />
                              )}
                              {!isProtected && (
                                <Unlock className="h-3.5 w-3.5 text-gray-400" aria-label="Not protected" />
                              )}
                            </div>
                            {scenario.description && (
                              <p className="text-xs text-gray-500 mt-0.5 truncate">{scenario.description}</p>
                            )}
                          </div>
                        </div>
                      </li>
                    )
                  })}
                </ul>
              )}
            </div>

            {/* Error Display */}
            {error && (
              <div className="rounded-md bg-red-50 border border-red-200 p-3">
                <p className="text-sm text-red-700">{error}</p>
              </div>
            )}
          </div>

          {/* Action Buttons */}
          <div className="mt-5 sm:mt-6 sm:flex sm:flex-row-reverse gap-3">
            <button
              type="button"
              onClick={handleSave}
              disabled={isSaving}
              className="inline-flex w-full justify-center rounded-md bg-blue-600 px-4 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600 disabled:opacity-50 disabled:cursor-not-allowed sm:w-auto"
            >
              {isSaving ? 'Saving...' : 'Save'}
            </button>
            <button
              type="button"
              onClick={onClose}
              disabled={isSaving}
              className="mt-3 inline-flex w-full justify-center rounded-md bg-white px-4 py-2 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50 disabled:opacity-50 sm:mt-0 sm:w-auto"
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
