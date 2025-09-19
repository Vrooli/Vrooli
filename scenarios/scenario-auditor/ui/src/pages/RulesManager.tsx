import React, { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Shield, Plus, Terminal, X, Play, TestTube, Code, CheckCircle, XCircle, Clock, Eye, EyeOff, Brain, CircleStop } from 'lucide-react'
import { Highlight, themes } from 'prism-react-renderer'
import { AgentInfo } from '@/types/api'
import { apiService } from '../services/api'
import { CodeEditor } from '../components/CodeEditor'

// Code Block Component with syntax highlighting and line numbers
function CodeBlock({ code }: { code: string }) {
  if (!code) return <div className="text-gray-500 italic">No code available</div>
  
  return (
    <Highlight theme={themes.vsDark} code={code} language="go">
      {({ className, style, tokens, getLineProps, getTokenProps }) => (
        <div className="relative">
          <pre 
            className={`${className} text-sm rounded-lg p-4 overflow-auto max-h-[500px]`} 
            style={style}
          >
            <code className="block">
              {tokens.map((line, i) => (
                <div key={i} {...getLineProps({ line, key: i })} className="table-row">
                  <span className="table-cell text-right pr-4 select-none text-gray-500 text-xs" style={{ minWidth: '3ch' }}>
                    {i + 1}
                  </span>
                  <span className="table-cell">
                    {line.map((token, key) => (
                      <span key={key} {...getTokenProps({ token, key })} />
                    ))}
                  </span>
                </div>
              ))}
            </code>
          </pre>
        </div>
      )}
    </Highlight>
  )
}

// Rule Card Component
function RuleCard({ rule, onViewRule, onToggleRule }: { 
  rule: any, 
  onViewRule: (ruleId: string) => void,
  onToggleRule: (ruleId: string, enabled: boolean) => void 
}) {
  const handleToggle = (e: React.ChangeEvent<HTMLInputElement>) => {
    onToggleRule(rule.id, e.target.checked)
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow" data-testid={`rule-${rule.id}`}>
      <div className="p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-lg font-medium text-gray-900">{rule.name}</h3>
            <p className="mt-1 text-sm text-gray-500">{rule.description}</p>
          </div>
          <div className="ml-4">
            <button 
              className="text-gray-400 hover:text-gray-600 transition-colors"
              onClick={() => onViewRule(rule.id)}
              title="View rule file"
            >
              <Terminal className="h-5 w-5" />
            </button>
          </div>
        </div>
        
        <div className="mt-4 flex items-center justify-between">
          <div className="flex items-center space-x-2">
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
              rule.category === 'api' ? 'bg-blue-100 text-blue-800' :
              rule.category === 'cli' ? 'bg-green-100 text-green-800' :
              rule.category === 'config' ? 'bg-purple-100 text-purple-800' :
              rule.category === 'test' ? 'bg-yellow-100 text-yellow-800' :
              rule.category === 'ui' ? 'bg-pink-100 text-pink-800' :
              'bg-gray-100 text-gray-800'
            }`}>
              {rule.category}
            </span>
            <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
              rule.severity === 'critical' ? 'bg-red-100 text-red-800' :
              rule.severity === 'high' ? 'bg-orange-100 text-orange-800' :
              rule.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
              rule.severity === 'low' ? 'bg-green-100 text-green-800' :
              'bg-gray-100 text-gray-800'
            }`}>
              {rule.severity}
            </span>
          </div>
          <div className="flex items-center">
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                checked={rule.enabled}
                onChange={handleToggle}
                className="sr-only peer"
                data-testid={`rule-toggle-${rule.id}`}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
            </label>
          </div>
        </div>
      </div>
    </div>
  )
}

export default function RulesManager() {
  const [selectedCategory, setSelectedCategory] = useState<string>('')
  const [selectedRule, setSelectedRule] = useState<string | null>(null)
  const [localRules, setLocalRules] = useState<Record<string, any>>({})
  const [activeTab, setActiveTab] = useState<'implementation' | 'tests' | 'playground'>('implementation')
  const [testResults, setTestResults] = useState<any>(null)
  const [isRunningTests, setIsRunningTests] = useState(false)
  const [playgroundCode, setPlaygroundCode] = useState('')
  const [playgroundResult, setPlaygroundResult] = useState<any>(null)
  const [isRunningPlayground, setIsRunningPlayground] = useState(false)
  const [isLaunchingAgent, setIsLaunchingAgent] = useState(false)
  const [agentError, setAgentError] = useState<string | null>(null)
  const queryClient = useQueryClient()

  const { data: activeAgentsData } = useQuery({
    queryKey: ['activeAgents'],
    queryFn: () => apiService.getActiveAgents(),
    refetchInterval: 2000,
  })

  const ruleAgents = useMemo(() => {
    if (!selectedRule) return []
    return (activeAgentsData?.agents || []).filter((agent: AgentInfo) => agent.rule_id === selectedRule)
  }, [activeAgentsData, selectedRule])

  const isAgentRunning = ruleAgents.length > 0

  const formatDuration = (seconds: number) => {
    if (!seconds || seconds < 1) return '<1s'
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    if (mins > 0) {
      return `${mins}m ${secs}s`
    }
    return `${secs}s`
  }

  const describeAgentAction = (action: string) => {
    switch (action) {
      case 'add_rule_tests':
        return 'Add Test Cases'
      case 'fix_rule_tests':
        return 'Fix Test Cases'
      default:
        return action
    }
  }

  const [showExecutionOutput, setShowExecutionOutput] = useState<{[key: number]: boolean}>({})

  const { data: rulesData, isLoading, refetch } = useQuery({
    queryKey: ['rules', selectedCategory],
    queryFn: () => apiService.getRules(selectedCategory || undefined),
  })

  const { data: ruleDetail, isLoading: ruleDetailLoading } = useQuery({
    queryKey: ['rule', selectedRule],
    queryFn: () => selectedRule ? apiService.getRule(selectedRule) : null,
    enabled: !!selectedRule,
  })

  // Run tests when rule is selected and tests tab is active
  const runRuleTests = async () => {
    if (!selectedRule) return
    
    setIsRunningTests(true)
    try {
      const results = await apiService.testRule(selectedRule)
      setTestResults(results)
    } catch (error) {
      console.error('Failed to run tests:', error)
      setTestResults({ error: 'Failed to run tests' })
    } finally {
      setIsRunningTests(false)
    }
  }

  // Run playground code validation
  const runPlaygroundTest = async () => {
    if (!selectedRule || !playgroundCode.trim()) return
    
    setIsRunningPlayground(true)
    try {
      const result = await apiService.validateRule(selectedRule, playgroundCode, 'go')
      setPlaygroundResult(result)
    } catch (error) {
      console.error('Failed to run playground test:', error)
      setPlaygroundResult({ error: 'Failed to run playground test' })
    } finally {
      setIsRunningPlayground(false)
    }
  }

  const handleStartAgent = async (action: 'add_rule_tests' | 'fix_rule_tests') => {
    if (!selectedRule) return
    setAgentError(null)
    setIsLaunchingAgent(true)
    try {
      await apiService.startRuleAgent(selectedRule, action)
      await queryClient.invalidateQueries({ queryKey: ['activeAgents'] })
    } catch (error) {
      console.error('Failed to start agent:', error)
      setAgentError((error as Error).message)
    } finally {
      setIsLaunchingAgent(false)
    }
  }

  const handleStopAgent = async (agentId: string) => {
    setAgentError(null)
    setIsLaunchingAgent(true)
    try {
      await apiService.stopAgent(agentId)
      await queryClient.invalidateQueries({ queryKey: ['activeAgents'] })
    } catch (error) {
      console.error('Failed to stop agent:', error)
      setAgentError((error as Error).message)
    } finally {
      setIsLaunchingAgent(false)
    }
  }

  // Reset tab and test data when rule changes
  React.useEffect(() => {
    setAgentError(null)
    if (selectedRule) {
      setActiveTab('implementation')
      setTestResults(null)
      setPlaygroundCode('')
      setPlaygroundResult(null)
    }
  }, [selectedRule])

  // Sync local rules state with fetched data
  React.useEffect(() => {
    if (rulesData?.rules) {
      setLocalRules(rulesData.rules)
    }
  }, [rulesData])

  const handleToggleRule = async (ruleId: string, enabled: boolean) => {
    // Optimistically update the UI
    setLocalRules(prev => ({
      ...prev,
      [ruleId]: {
        ...prev[ruleId],
        enabled
      }
    }))

    try {
      // Call the API to toggle the rule
      await apiService.toggleRule(ruleId, enabled)
      // Refetch rules to ensure consistency
      refetch()
    } catch (error) {
      console.error('Failed to toggle rule:', error)
      // Revert on error
      setLocalRules(prev => ({
        ...prev,
        [ruleId]: {
          ...prev[ruleId],
          enabled: !enabled
        }
      }))
    }
  }

  if (isLoading) {
    return (
      <div className="px-6">
        <div className="animate-pulse">
          <div className="h-8 bg-gray-200 rounded w-1/4 mb-6"></div>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="bg-white p-6 rounded-lg shadow">
                <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
                <div className="h-3 bg-gray-200 rounded w-1/2"></div>
              </div>
            ))}
          </div>
        </div>
      </div>
    )
  }

  const rules = localRules
  const categories = rulesData?.categories || {}

  return (
    <div className="px-6">
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900" data-testid="rules-title">
            Rules Manager
          </h1>
          <p className="mt-1 text-sm text-gray-500">
            Manage and configure auditing rules
          </p>
        </div>
        <button 
          className="btn-primary"
          data-testid="create-rule-btn"
        >
          <Plus className="h-4 w-4 mr-2" />
          Create Rule
        </button>
      </div>

      {/* Filters */}
      <div className="mb-6 flex items-center space-x-4">
        <select
          value={selectedCategory}
          onChange={(e) => setSelectedCategory(e.target.value)}
          className="rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
          data-testid="category-filter"
        >
          <option value="">All Categories</option>
          {Object.values(categories).map((category: any) => (
            <option key={category.id} value={category.id}>
              {category.name}
            </option>
          ))}
        </select>
      </div>

      {/* Rules Display */}
      {selectedCategory ? (
        /* Single Category Grid */
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {Object.values(rules).map((rule: any) => (
            <RuleCard 
              key={rule.id} 
              rule={rule} 
              onViewRule={setSelectedRule} 
              onToggleRule={handleToggleRule}
            />
          ))}
        </div>
      ) : (
        /* Grouped by Category */
        <div className="space-y-8">
          {Object.values(categories).map((category: any) => {
            const categoryRules = Object.values(rules).filter((rule: any) => rule.category === category.id)
            
            if (categoryRules.length === 0) return null
            
            return (
              <div key={category.id} className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h2 className="text-xl font-semibold text-gray-900">{category.name}</h2>
                    <p className="text-sm text-gray-500">{category.description}</p>
                  </div>
                  <span className="text-sm text-gray-400">{categoryRules.length} rule{categoryRules.length !== 1 ? 's' : ''}</span>
                </div>
                
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {categoryRules.map((rule: any) => (
                    <RuleCard 
                      key={rule.id} 
                      rule={rule} 
                      onViewRule={setSelectedRule} 
                      onToggleRule={handleToggleRule}
                    />
                  ))}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {Object.keys(rules).length === 0 && (
        <div className="text-center py-12">
          <Shield className="mx-auto h-12 w-12 text-gray-400" />
          <h3 className="mt-2 text-sm font-medium text-gray-900">No rules found</h3>
          <p className="mt-1 text-sm text-gray-500">
            {selectedCategory ? 'No rules in this category.' : 'Get started by creating your first rule.'}
          </p>
        </div>
      )}

      {/* Rule Detail Modal */}
      {selectedRule && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-full items-end justify-center p-4 text-center sm:items-center sm:p-0">
            <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" onClick={() => setSelectedRule(null)} />
            
            <div className="relative transform overflow-hidden rounded-lg bg-white px-4 pb-4 pt-5 text-left shadow-xl transition-all sm:my-8 sm:w-full sm:max-w-4xl sm:p-6">
              <div className="absolute right-0 top-0 hidden pr-4 pt-4 sm:block">
                <button
                  type="button"
                  className="rounded-md bg-white text-gray-400 hover:text-gray-500 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
                  onClick={() => setSelectedRule(null)}
                >
                  <span className="sr-only">Close</span>
                  <X className="h-6 w-6" />
                </button>
              </div>
              
              <div className="sm:flex sm:items-start">
                <div className="mx-auto flex h-12 w-12 flex-shrink-0 items-center justify-center rounded-full bg-blue-100 sm:mx-0 sm:h-10 sm:w-10">
                  <Terminal className="h-6 w-6 text-blue-600" />
                </div>
                <div className="mt-3 text-center sm:ml-4 sm:mt-0 sm:text-left flex-1">
                  <h3 className="text-base font-semibold leading-6 text-gray-900">
                    {ruleDetail?.rule?.name || selectedRule}
                  </h3>
                  <div className="mt-2">
                    <p className="text-sm text-gray-500">
                      {ruleDetail?.rule?.description || 'Loading rule details...'}
                    </p>
                    {ruleDetail?.file_path && (
                      <p className="text-xs text-gray-400 mt-1">
                        {ruleDetail.file_path}
                      </p>
                    )}
                  </div>
                </div>
              </div>
              
              {/* Tab Navigation */}
              <div className="mt-5 border-b border-gray-200">
                <div className="flex space-x-8">
                  <button
                    onClick={() => setActiveTab('implementation')}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${
                      activeTab === 'implementation'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    <Code className="h-4 w-4 inline-block mr-2" />
                    Implementation
                  </button>
                  <button
                    onClick={() => {
                      setActiveTab('tests')
                      if (!testResults && !isRunningTests) runRuleTests()
                    }}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${
                      activeTab === 'tests'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    <TestTube className="h-4 w-4 inline-block mr-2" />
                    Test Cases
                    {testResults && (
                      <span className={`ml-2 inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium ${
                        testResults.error ? 'bg-red-100 text-red-800' :
                        testResults.failed > 0 ? 'bg-yellow-100 text-yellow-800' :
                        'bg-green-100 text-green-800'
                      }`}>
                        {testResults.error ? 'Error' : `${testResults.passed}/${testResults.total_tests} correct`}
                      </span>
                    )}
                  </button>
                  <button
                    onClick={() => setActiveTab('playground')}
                    className={`py-2 px-1 border-b-2 font-medium text-sm ${
                      activeTab === 'playground'
                        ? 'border-blue-500 text-blue-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                    }`}
                  >
                    <Play className="h-4 w-4 inline-block mr-2" />
                    Playground
                  </button>
                </div>
              </div>

              {/* Tab Content */}
              <div className="mt-5">
                {activeTab === 'implementation' && (
                  <div className="bg-gray-900 rounded-lg overflow-hidden">
                    <div className="bg-gray-800 px-4 py-2 border-b border-gray-700">
                      <h4 className="text-sm font-medium text-gray-200 flex items-center justify-between">
                        <span>Rule Implementation</span>
                        {ruleDetail?.file_path && (
                          <span className="text-xs text-gray-400 font-mono">
                            {ruleDetail.file_path.replace(/^.*\/rules\//, 'rules/')}
                          </span>
                        )}
                      </h4>
                    </div>
                    <div className="p-0">
                      {ruleDetailLoading ? (
                        <div className="p-4">
                          <div className="animate-pulse">
                            <div className="h-4 bg-gray-700 rounded w-3/4 mb-2"></div>
                            <div className="h-4 bg-gray-700 rounded w-1/2 mb-2"></div>
                            <div className="h-4 bg-gray-700 rounded w-2/3"></div>
                          </div>
                        </div>
                      ) : (
                        <CodeBlock code={ruleDetail?.file_content || ''} />
                      )}
                    </div>
                  </div>
                )}

                {activeTab === 'tests' && (
                  <div className="space-y-4">
                    <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                      <h4 className="text-lg font-medium text-gray-900">Test Cases</h4>
                      <div className="flex flex-wrap items-center gap-2">
                        <button
                          onClick={runRuleTests}
                          disabled={isRunningTests}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                        >
                          {isRunningTests ? (
                            <Clock className="h-4 w-4 mr-2 animate-spin" />
                          ) : (
                            <Play className="h-4 w-4 mr-2" />
                          )}
                          {isRunningTests ? 'Running...' : 'Run All Tests'}
                        </button>
                        <button
                          onClick={() => handleStartAgent('add_rule_tests')}
                          disabled={isLaunchingAgent || isAgentRunning || !selectedRule}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-purple-600 hover:bg-purple-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-purple-500 disabled:opacity-50"
                        >
                          <Brain className="h-4 w-4 mr-2" />
                          Add Tests (AI)
                        </button>
                        <button
                          onClick={() => handleStartAgent('fix_rule_tests')}
                          disabled={isLaunchingAgent || isAgentRunning || !selectedRule}
                          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-amber-600 hover:bg-amber-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-amber-500 disabled:opacity-50"
                        >
                          <Brain className="h-4 w-4 mr-2" />
                          Fix Tests (AI)
                        </button>
                      </div>
                    </div>

                    {agentError && (
                      <div className="bg-red-50 border border-red-200 rounded-md p-3 text-sm text-red-700">
                        {agentError}
                      </div>
                    )}

                    {isLaunchingAgent && !isAgentRunning && (
                      <div className="flex items-center gap-2 text-sm text-blue-700">
                        <Clock className="h-4 w-4 animate-spin" />
                        <span>Preparing AI agent…</span>
                      </div>
                    )}

                    {ruleAgents.map((agent: AgentInfo) => (
                      <div key={agent.id} className="border border-blue-200 bg-blue-50 rounded-lg p-4">
                        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                          <div className="flex items-center gap-3">
                            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-blue-600 text-white">
                              <Brain className="h-5 w-5" />
                            </div>
                            <div>
                              <p className="text-sm font-medium text-blue-900">{agent.label || agent.name}</p>
                              <p className="text-xs text-blue-700">{describeAgentAction(agent.action)} · {formatDuration(agent.duration_seconds)} elapsed</p>
                              {agent.metadata?.rule_file && (
                                <p className="text-xs text-blue-600">Rule file: {agent.metadata.rule_file}</p>
                              )}
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            <button
                              onClick={() => handleStopAgent(agent.id)}
                              disabled={isLaunchingAgent}
                              className="inline-flex items-center px-3 py-2 border border-red-200 text-sm font-medium rounded-md text-red-600 hover:bg-red-50 disabled:opacity-50"
                            >
                              <CircleStop className="h-4 w-4 mr-2" />
                              Stop
                            </button>
                          </div>
                        </div>
                      </div>
                    ))}

                    {isRunningTests && (
                      <div className="flex items-center justify-center py-8">
                        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
                        <span className="ml-3 text-gray-600">Running tests...</span>
                      </div>
                    )}

                    {testResults?.tests && (
                      <div className="space-y-4">
                        <div className="bg-gray-50 rounded-lg p-4">
                          <div className="flex items-center justify-between text-sm">
                            <span>
                              <strong>Results:</strong> {testResults.passed} behaving correctly, {testResults.failed} not behaving as expected out of {testResults.total_tests} tests
                            </span>
                            {testResults.cached && (
                              <span className="text-gray-500">(Cached)</span>
                            )}
                          </div>
                        </div>

                        {testResults.tests.map((test: any, index: number) => (
                          <div key={index} className="border border-gray-200 rounded-lg overflow-hidden">
                            <div className={`px-4 py-3 border-b border-gray-200 ${
                              test.passed ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'
                            }`}>
                              <div className="flex items-center justify-between">
                                <div className="flex items-center">
                                  {test.passed ? (
                                    <CheckCircle className="h-5 w-5 text-green-600 mr-2" />
                                  ) : (
                                    <XCircle className="h-5 w-5 text-red-600 mr-2" />
                                  )}
                                  <div>
                                    <h5 className="font-medium text-gray-900">{test.test_case.id}</h5>
                                    <p className="text-sm text-gray-600">{test.test_case.description}</p>
                                  </div>
                                </div>
                                <div className="flex items-center space-x-2">
                                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                                    test.test_case.should_fail ? 'bg-yellow-100 text-yellow-800' : 'bg-blue-100 text-blue-800'
                                  }`}>
                                    Expected: {test.test_case.should_fail ? 'Violations' : 'No Violations'}
                                  </span>
                                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                                    test.passed ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'
                                  }`}>
                                    {test.passed ? '✅ BEHAVED CORRECTLY' : '❌ UNEXPECTED BEHAVIOR'}
                                  </span>
                                </div>
                              </div>
                              {/* Show actual vs expected */}
                              <div className="mt-2 text-xs text-gray-600">
                                <span className="font-medium">Actual: </span>
                                {test.actual_violations && test.actual_violations.length > 0 ? (
                                  <span className="text-orange-600">{test.actual_violations.length} violation(s) found</span>
                                ) : (
                                  <span className="text-green-600">No violations found</span>
                                )}
                                {test.test_case.should_fail ? (
                                  test.actual_violations && test.actual_violations.length > 0 ? (
                                    <span className="ml-2 text-green-600">✓ As expected</span>
                                  ) : (
                                    <span className="ml-2 text-red-600">✗ Expected violations</span>
                                  )
                                ) : (
                                  test.actual_violations && test.actual_violations.length > 0 ? (
                                    <span className="ml-2 text-red-600">✗ Expected no violations</span>
                                  ) : (
                                    <span className="ml-2 text-green-600">✓ As expected</span>
                                  )
                                )}
                              </div>
                            </div>
                            
                            <div className="p-4 space-y-4">
                              <div>
                                <h6 className="text-sm font-medium text-gray-900 mb-2">Test Input</h6>
                                <div className="bg-gray-900 rounded-lg overflow-hidden">
                                  <CodeBlock code={test.test_case.input} />
                                </div>
                              </div>
                              
                              {test.actual_violations && test.actual_violations.length > 0 && (
                                <div>
                                  <h6 className="text-sm font-medium text-gray-900 mb-2">Violations Found</h6>
                                  <div className="space-y-2">
                                    {test.actual_violations.map((violation: any, vIndex: number) => (
                                      <div key={vIndex} className="bg-gray-50 rounded p-3 text-sm">
                                        <div className="flex items-start justify-between">
                                          <div>
                                            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                              violation.severity === 'critical' ? 'bg-red-100 text-red-800' :
                                              violation.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                                              violation.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                                              'bg-blue-100 text-blue-800'
                                            }`}>
                                              {violation.severity}
                                            </span>
                                          </div>
                                        </div>
                                        <p className="mt-2 text-gray-800">{violation.message}</p>
                                      </div>
                                    ))}
                                  </div>
                                </div>
                              )}

                              {/* Execution Output */}
                              {test.execution_output && (
                                <div>
                                  <div className="flex items-center justify-between mb-2">
                                    <h6 className="text-sm font-medium text-gray-900">Execution Output</h6>
                                    <button
                                      onClick={() => setShowExecutionOutput(prev => ({
                                        ...prev,
                                        [index]: !prev[index]
                                      }))}
                                      className="inline-flex items-center text-xs text-gray-600 hover:text-gray-800"
                                    >
                                      {showExecutionOutput[index] ? (
                                        <>
                                          <EyeOff className="h-3 w-3 mr-1" />
                                          Hide
                                        </>
                                      ) : (
                                        <>
                                          <Eye className="h-3 w-3 mr-1" />
                                          Show
                                        </>
                                      )}
                                    </button>
                                  </div>
                                  
                                  {showExecutionOutput[index] && (
                                    <div className="bg-gray-50 rounded p-3 text-xs space-y-2">
                                      <div className="flex items-center gap-2 text-gray-600">
                                        <span className="font-medium">Method:</span>
                                        <span className={`px-2 py-0.5 rounded text-xs ${
                                          test.execution_output.method === 'judge0' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'
                                        }`}>
                                          {test.execution_output.method}
                                        </span>
                                      </div>
                                      
                                      {test.execution_output.stdout && (
                                        <div>
                                          <span className="font-medium text-gray-700">Stdout:</span>
                                          <pre className="mt-1 bg-gray-900 text-gray-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.stdout}</pre>
                                        </div>
                                      )}
                                      
                                      {test.execution_output.stderr && (
                                        <div>
                                          <span className="font-medium text-gray-700">Stderr:</span>
                                          <pre className="mt-1 bg-red-900 text-red-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.stderr}</pre>
                                        </div>
                                      )}
                                      
                                      {test.execution_output.compile_output && (
                                        <div>
                                          <span className="font-medium text-gray-700">Compile Output:</span>
                                          <pre className="mt-1 bg-yellow-900 text-yellow-100 p-2 rounded text-xs whitespace-pre-wrap">{test.execution_output.compile_output}</pre>
                                        </div>
                                      )}
                                      
                                      {test.execution_output.exit_code !== undefined && (
                                        <div className="text-gray-600">
                                          <span className="font-medium">Exit Code:</span> {test.execution_output.exit_code}
                                        </div>
                                      )}
                                    </div>
                                  )}
                                </div>
                              )}
                            </div>
                          </div>
                        ))}
                      </div>
                    )}

                    {!isRunningTests && !testResults && (
                      <div className="text-center py-8 text-gray-500">
                        <TestTube className="h-12 w-12 mx-auto mb-3 text-gray-400" />
                        <p>Click "Run All Tests" to execute test cases for this rule</p>
                      </div>
                    )}
                  </div>
                )}

                {activeTab === 'playground' && (
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h4 className="text-lg font-medium text-gray-900">Test Playground</h4>
                      <button
                        onClick={runPlaygroundTest}
                        disabled={isRunningPlayground || !playgroundCode.trim()}
                        className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50"
                      >
                        {isRunningPlayground ? (
                          <Clock className="h-4 w-4 mr-2 animate-spin" />
                        ) : (
                          <Play className="h-4 w-4 mr-2" />
                        )}
                        {isRunningPlayground ? 'Testing...' : 'Test Code'}
                      </button>
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-gray-700 mb-2">
                        Enter Go code to test against this rule:
                      </label>
                      <div className="h-64">
                        <CodeEditor
                          value={playgroundCode}
                          onChange={setPlaygroundCode}
                          language="go"
                          placeholder="func HandleRequest(w http.ResponseWriter, r *http.Request) {
    // Your code here
}"
                          className="h-full"
                        />
                      </div>
                    </div>

                    {playgroundResult && (
                      <div className="border border-gray-200 rounded-lg overflow-hidden">
                        <div className={`px-4 py-3 border-b border-gray-200 ${
                          playgroundResult.error ? 'bg-red-50' :
                          playgroundResult.actual_violations?.length > 0 ? 'bg-yellow-50' : 'bg-green-50'
                        }`}>
                          <h5 className="font-medium text-gray-900">Test Result</h5>
                        </div>
                        
                        <div className="p-4">
                          {playgroundResult.error ? (
                            <div className="text-red-800">
                              <strong>Error:</strong> {playgroundResult.error}
                            </div>
                          ) : playgroundResult.actual_violations?.length > 0 ? (
                            <div className="space-y-3">
                              <p className="text-gray-800">
                                <strong>Found {playgroundResult.actual_violations.length} violation(s):</strong>
                              </p>
                              {playgroundResult.actual_violations.map((violation: any, index: number) => (
                                <div key={index} className="bg-gray-50 rounded p-3">
                                  <div className="flex items-start justify-between">
                                    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                                      violation.severity === 'critical' ? 'bg-red-100 text-red-800' :
                                      violation.severity === 'high' ? 'bg-orange-100 text-orange-800' :
                                      violation.severity === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                                      'bg-blue-100 text-blue-800'
                                    }`}>
                                      {violation.severity}
                                    </span>
                                  </div>
                                  <p className="mt-2 text-gray-800">{violation.message}</p>
                                </div>
                              ))}
                            </div>
                          ) : (
                            <div className="text-green-800">
                              <CheckCircle className="h-5 w-5 inline-block mr-2" />
                              <strong>No violations found!</strong> Your code follows this rule.
                            </div>
                          )}
                          
                          {/* Execution Output for Playground */}
                          {playgroundResult.test_result?.execution_output && (
                            <div className="mt-4 pt-4 border-t border-gray-200">
                              <h6 className="text-sm font-medium text-gray-900 mb-2">Execution Output</h6>
                              <div className="bg-gray-50 rounded p-3 text-xs space-y-2">
                                <div className="flex items-center gap-2 text-gray-600">
                                  <span className="font-medium">Method:</span>
                                  <span className={`px-2 py-0.5 rounded text-xs ${
                                    playgroundResult.test_result.execution_output.method === 'judge0' ? 'bg-green-100 text-green-700' : 'bg-blue-100 text-blue-700'
                                  }`}>
                                    {playgroundResult.test_result.execution_output.method}
                                  </span>
                                </div>
                                
                                {playgroundResult.test_result.execution_output.stdout && (
                                  <div>
                                    <span className="font-medium text-gray-700">Stdout:</span>
                                    <pre className="mt-1 bg-gray-900 text-gray-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.stdout}</pre>
                                  </div>
                                )}
                                
                                {playgroundResult.test_result.execution_output.stderr && (
                                  <div>
                                    <span className="font-medium text-gray-700">Stderr:</span>
                                    <pre className="mt-1 bg-red-900 text-red-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.stderr}</pre>
                                  </div>
                                )}
                                
                                {playgroundResult.test_result.execution_output.compile_output && (
                                  <div>
                                    <span className="font-medium text-gray-700">Compile Output:</span>
                                    <pre className="mt-1 bg-yellow-900 text-yellow-100 p-2 rounded text-xs whitespace-pre-wrap">{playgroundResult.test_result.execution_output.compile_output}</pre>
                                  </div>
                                )}
                                
                                {playgroundResult.test_result.execution_output.exit_code !== undefined && (
                                  <div className="text-gray-600">
                                    <span className="font-medium">Exit Code:</span> {playgroundResult.test_result.execution_output.exit_code}
                                  </div>
                                )}
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {!playgroundResult && (
                      <div className="text-center py-8 text-gray-500">
                        <Code className="h-12 w-12 mx-auto mb-3 text-gray-400" />
                        <p>Enter some Go code above and click "Test Code" to validate it against this rule</p>
                      </div>
                    )}
                  </div>
                )}
              </div>
              
              <div className="mt-5 sm:mt-4 sm:flex sm:flex-row-reverse">
                <button
                  type="button"
                  className="inline-flex w-full justify-center rounded-md bg-blue-600 px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-blue-500 sm:ml-3 sm:w-auto"
                  onClick={() => setSelectedRule(null)}
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}