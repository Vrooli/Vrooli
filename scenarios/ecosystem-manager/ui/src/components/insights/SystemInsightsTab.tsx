/**
 * System-level Insights Tab
 * Displays cross-task patterns, system-wide statistics, and actionable suggestions
 */

import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { BarChart3, TrendingUp, AlertTriangle, CheckCircle2, RefreshCw, Sparkles } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { SystemInsightsDeepView } from './SystemInsightsDeepView';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { InsightReport } from '@/types/api';

export function SystemInsightsTab() {
  const [sinceDays, setSinceDays] = useState(7);
  const [activeView, setActiveView] = useState<'summary' | 'deep'>('summary');

  const { data, isLoading, refetch, error } = useQuery({
    queryKey: queryKeys.insights.system(sinceDays),
    queryFn: () => api.getSystemInsights(sinceDays),
    staleTime: 30000,
  });

  const formatDateTime = (value?: string) => {
    if (!value) return '--';
    try {
      return new Date(value).toLocaleString();
    } catch {
      return value;
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'critical':
        return 'bg-red-500/20 text-red-100 border-red-400/60';
      case 'high':
        return 'bg-orange-500/20 text-orange-100 border-orange-400/60';
      case 'medium':
        return 'bg-yellow-500/20 text-yellow-100 border-yellow-400/60';
      case 'low':
        return 'bg-blue-500/20 text-blue-100 border-blue-400/60';
      default:
        return 'bg-slate-500/20 text-slate-200 border-slate-500/40';
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'critical':
        return 'bg-red-500/20 text-red-100 border-red-400/60';
      case 'high':
        return 'bg-orange-500/20 text-orange-100 border-orange-400/60';
      case 'medium':
        return 'bg-yellow-500/20 text-yellow-100 border-yellow-400/60';
      case 'low':
        return 'bg-green-500/20 text-green-100 border-green-400/60';
      default:
        return 'bg-slate-500/20 text-slate-200 border-slate-500/40';
    }
  };

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-slate-400">
        <AlertTriangle className="h-12 w-12 text-red-400" />
        <p>Failed to load system insights</p>
        <Button variant="outline" size="sm" onClick={() => refetch()}>
          Retry
        </Button>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-400">
        <RefreshCw className="h-8 w-8 animate-spin" />
        <span className="ml-3">Loading system insights...</span>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="flex flex-col items-center justify-center h-full gap-4 text-slate-400">
        <BarChart3 className="h-12 w-12" />
        <p>No system insights available</p>
        <p className="text-sm text-slate-500">Generate task-level insights first</p>
      </div>
    );
  }

  const summary = (data as any).summary || {};
  const reports = ((data as any).reports || []) as InsightReport[];
  const timeWindow = (data as any).time_window || {};

  // Aggregate patterns and suggestions across all reports
  const allPatterns = reports.flatMap(r => r.patterns || []);
  const allSuggestions = reports.flatMap(r => r.suggestions || []);

  // Group patterns by type
  const patternsByType = allPatterns.reduce((acc, pattern) => {
    const type = pattern.type || 'unknown';
    if (!acc[type]) acc[type] = [];
    acc[type].push(pattern);
    return acc;
  }, {} as Record<string, typeof allPatterns>);

  // Group suggestions by priority
  const suggestionsByPriority = allSuggestions.reduce((acc, suggestion) => {
    const priority = suggestion.priority || 'low';
    if (!acc[priority]) acc[priority] = [];
    acc[priority].push(suggestion);
    return acc;
  }, {} as Record<string, typeof allSuggestions>);

  // Calculate success rate (if we have that data)
  const successRate = summary.success_rate ?? null;

  return (
    <div className="flex flex-col gap-4 flex-1">
      {/* Header with time window and view tabs */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <BarChart3 className="h-5 w-5 text-emerald-400" />
          <div>
            <h3 className="text-lg font-semibold text-white">System-Wide Insights</h3>
            <p className="text-xs text-slate-400">
              Last {sinceDays} days • {summary.total_reports || 0} reports analyzed
            </p>
          </div>
        </div>
        <Button variant="outline" size="sm" onClick={() => refetch()} disabled={isLoading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${isLoading ? 'animate-spin' : ''}`} />
          Refresh
        </Button>
      </div>

      {/* View Tabs */}
      <Tabs value={activeView} onValueChange={(v) => setActiveView(v as 'summary' | 'deep')} className="flex-1 flex flex-col">
        <TabsList className="bg-slate-900/70 border border-white/5 mb-4">
          <TabsTrigger value="summary" className="gap-2">
            <BarChart3 className="h-4 w-4" />
            Quick Summary
          </TabsTrigger>
          <TabsTrigger value="deep" className="gap-2">
            <Sparkles className="h-4 w-4" />
            Deep Analysis
          </TabsTrigger>
        </TabsList>

        <TabsContent value="summary" className="flex-1 mt-0 overflow-auto">
          <div className="flex flex-col gap-4">
            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
              <Card className="bg-slate-900/70 border-white/5 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Unique Tasks</div>
                <div className="text-2xl font-semibold text-white">{summary.unique_tasks || 0}</div>
              </Card>
              <Card className="bg-slate-900/70 border-white/5 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Total Executions</div>
                <div className="text-2xl font-semibold text-white">{summary.total_executions || 0}</div>
              </Card>
              <Card className="bg-slate-900/70 border-white/5 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Patterns Identified</div>
                <div className="text-2xl font-semibold text-amber-300">{allPatterns.length}</div>
              </Card>
              <Card className="bg-slate-900/70 border-white/5 p-4">
                <div className="text-xs uppercase text-slate-400 mb-1">Suggestions</div>
                <div className="text-2xl font-semibold text-emerald-300">{allSuggestions.length}</div>
              </Card>
            </div>

            {/* Main Content */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Patterns by Type */}
              <Card className="bg-slate-900/70 border-white/5 overflow-hidden flex flex-col">
                <div className="p-4 border-b border-white/5">
                  <h4 className="font-semibold text-white flex items-center gap-2">
                    <AlertTriangle className="h-4 w-4 text-amber-400" />
                    Common Patterns
                  </h4>
                </div>
                <div className="flex-1 overflow-y-auto p-4 space-y-3">
                  {Object.keys(patternsByType).length === 0 ? (
                    <div className="text-center text-slate-500 py-8">
                      No patterns identified
                    </div>
                  ) : (
                    Object.entries(patternsByType).map(([type, patterns]) => (
                      <div key={type} className="border border-white/5 rounded-lg p-3 bg-slate-800/50">
                        <div className="flex items-center justify-between mb-2">
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs">
                              {type.replace('_', ' ')}
                            </Badge>
                            <span className="text-xs text-slate-400">{patterns.length} occurrences</span>
                          </div>
                        </div>
                        <div className="space-y-2">
                          {patterns.slice(0, 3).map((pattern, idx) => (
                            <div key={`${pattern.id}-${idx}`} className="text-sm">
                              <div className="flex items-start gap-2">
                                <Badge variant="outline" className={`text-xs ${getSeverityColor(pattern.severity)}`}>
                                  {pattern.severity}
                                </Badge>
                                <span className="text-slate-200 flex-1">{pattern.description}</span>
                              </div>
                              {pattern.frequency && (
                                <div className="text-xs text-slate-500 ml-16 mt-1">
                                  Seen in {pattern.frequency} executions
                                </div>
                              )}
                            </div>
                          ))}
                          {patterns.length > 3 && (
                            <div className="text-xs text-slate-500 ml-16">
                              +{patterns.length - 3} more...
                            </div>
                          )}
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </Card>

              {/* Top Suggestions */}
              <Card className="bg-slate-900/70 border-white/5 overflow-hidden flex flex-col">
                <div className="p-4 border-b border-white/5">
                  <h4 className="font-semibold text-white flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-emerald-400" />
                    Recommended Actions
                  </h4>
                </div>
                <div className="flex-1 overflow-y-auto p-4 space-y-3">
                  {allSuggestions.length === 0 ? (
                    <div className="text-center text-slate-500 py-8">
                      No suggestions available
                    </div>
                  ) : (
                    ['critical', 'high', 'medium', 'low']
                      .filter(priority => suggestionsByPriority[priority]?.length > 0)
                      .map(priority => (
                        <div key={priority} className="space-y-2">
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className={getPriorityColor(priority)}>
                              {priority}
                            </Badge>
                            <span className="text-xs text-slate-400">
                              {suggestionsByPriority[priority].length} suggestions
                            </span>
                          </div>
                          {suggestionsByPriority[priority].slice(0, 2).map((suggestion, idx) => (
                            <div
                              key={`${suggestion.id}-${idx}`}
                              className="border border-white/5 rounded-lg p-3 bg-slate-800/50"
                            >
                              <div className="font-medium text-sm text-white mb-1">
                                {suggestion.title}
                              </div>
                              <div className="text-xs text-slate-300 mb-2">
                                {suggestion.description}
                              </div>
                              <div className="flex items-center gap-2 text-xs text-slate-500">
                                <Badge variant="outline" className="text-xs">
                                  {suggestion.type}
                                </Badge>
                                {suggestion.impact?.confidence && (
                                  <span>Confidence: {suggestion.impact.confidence}</span>
                                )}
                              </div>
                            </div>
                          ))}
                        </div>
                      ))
                  )}
                </div>
              </Card>
            </div>

            {/* Pattern Type Breakdown */}
            {summary.patterns_by_type && Object.keys(summary.patterns_by_type).length > 0 && (
              <Card className="bg-slate-900/70 border-white/5 p-4">
                <h4 className="font-semibold text-white mb-3 flex items-center gap-2">
                  <TrendingUp className="h-4 w-4 text-blue-400" />
                  Pattern Distribution
                </h4>
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  {Object.entries(summary.patterns_by_type as Record<string, number>).map(([type, count]) => (
                    <div key={type} className="flex items-center justify-between p-2 bg-slate-800/50 rounded border border-white/5">
                      <span className="text-sm text-slate-300">{type.replace('_', ' ')}</span>
                      <Badge variant="outline">{count}</Badge>
                    </div>
                  ))}
                </div>
              </Card>
            )}
          </div>
        </TabsContent>

        <TabsContent value="deep" className="flex-1 overflow-auto mt-0">
          <SystemInsightsDeepView sinceDays={sinceDays} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
