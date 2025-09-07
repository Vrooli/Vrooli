const fs = require('fs');
const path = require('path');
const yaml = require('js-yaml');
const chokidar = require('chokidar');
const EventEmitter = require('events');

class RulesEngine extends EventEmitter {
  constructor(rulesPath) {
    super();
    this.rulesPath = rulesPath;
    this.rules = new Map();
    this.watchers = new Map();
    this.aiAnalyzer = null;
    this.cache = new Map();
    this.cacheTimeout = 900000; // 15 minutes
    
    this.loadRules();
    this.setupWatchers();
  }

  /**
   * Load all rule files from the rules directory
   */
  loadRules() {
    const ruleFiles = fs.readdirSync(this.rulesPath)
      .filter(file => file.endsWith('.yaml') || file.endsWith('.yml'));
    
    for (const file of ruleFiles) {
      this.loadRuleFile(path.join(this.rulesPath, file));
    }
    
    this.emit('rules-loaded', { count: this.rules.size });
  }

  /**
   * Load a single rule file
   */
  loadRuleFile(filePath) {
    try {
      const content = fs.readFileSync(filePath, 'utf8');
      const data = yaml.load(content);
      
      if (data && data.rules) {
        for (const rule of data.rules) {
          this.rules.set(rule.id, {
            ...rule,
            source: filePath
          });
        }
        
        console.log(`Loaded ${data.rules.length} rules from ${path.basename(filePath)}`);
      }
    } catch (error) {
      console.error(`Error loading rule file ${filePath}:`, error);
    }
  }

  /**
   * Setup file watchers for hot-reload
   */
  setupWatchers() {
    const watcher = chokidar.watch(this.rulesPath, {
      persistent: true,
      ignoreInitial: true,
      awaitWriteFinish: {
        stabilityThreshold: 1000,
        pollInterval: 100
      }
    });

    watcher
      .on('add', filePath => this.handleFileChange(filePath, 'added'))
      .on('change', filePath => this.handleFileChange(filePath, 'changed'))
      .on('unlink', filePath => this.handleFileRemoval(filePath));

    this.watchers.set('main', watcher);
  }

  /**
   * Handle rule file changes
   */
  handleFileChange(filePath, event) {
    if (!filePath.endsWith('.yaml') && !filePath.endsWith('.yml')) return;
    
    console.log(`Rule file ${event}: ${path.basename(filePath)}`);
    
    // Remove old rules from this file
    const oldRules = Array.from(this.rules.values())
      .filter(rule => rule.source === filePath);
    
    for (const rule of oldRules) {
      this.rules.delete(rule.id);
    }
    
    // Load new rules
    this.loadRuleFile(filePath);
    
    // Clear cache for affected files
    this.clearCache();
    
    this.emit('rules-reloaded', { 
      file: path.basename(filePath),
      event,
      totalRules: this.rules.size
    });
  }

  /**
   * Handle rule file removal
   */
  handleFileRemoval(filePath) {
    if (!filePath.endsWith('.yaml') && !filePath.endsWith('.yml')) return;
    
    console.log(`Rule file removed: ${path.basename(filePath)}`);
    
    // Remove rules from this file
    const removedCount = Array.from(this.rules.values())
      .filter(rule => rule.source === filePath)
      .reduce((count, rule) => {
        this.rules.delete(rule.id);
        return count + 1;
      }, 0);
    
    this.clearCache();
    
    this.emit('rules-removed', {
      file: path.basename(filePath),
      removedCount,
      totalRules: this.rules.size
    });
  }

  /**
   * Analyze a file against all enabled rules
   */
  async analyzeFile(filePath, options = {}) {
    const {
      rules: selectedRules = null,
      autoFix = false,
      riskLevel = 'safe'
    } = options;
    
    // Check cache
    const cacheKey = `${filePath}:${JSON.stringify(options)}`;
    if (this.cache.has(cacheKey)) {
      const cached = this.cache.get(cacheKey);
      if (Date.now() - cached.timestamp < this.cacheTimeout) {
        return cached.data;
      }
    }
    
    const violations = [];
    const fileContent = fs.readFileSync(filePath, 'utf8');
    const fileExt = path.extname(filePath).slice(1);
    const lines = fileContent.split('\n');
    
    // Filter rules to apply
    const rulesToApply = Array.from(this.rules.values()).filter(rule => {
      if (!rule.enabled !== false) return true;
      if (selectedRules && !selectedRules.includes(rule.id)) return false;
      if (rule.pattern?.file_types && !rule.pattern.file_types.includes(fileExt)) return false;
      return true;
    });
    
    // Apply each rule
    for (const rule of rulesToApply) {
      const ruleViolations = await this.applyRule(rule, filePath, fileContent, lines);
      violations.push(...ruleViolations);
    }
    
    // Auto-fix if requested
    let fixedContent = fileContent;
    const fixedViolations = [];
    
    if (autoFix) {
      for (const violation of violations) {
        if (violation.auto_fixable && this.canAutoFix(violation, riskLevel)) {
          const fixed = this.applyFix(fixedContent, violation);
          if (fixed !== fixedContent) {
            fixedContent = fixed;
            fixedViolations.push(violation);
            violation.status = 'fixed';
          }
        }
      }
      
      // Write fixed content if changes were made
      if (fixedContent !== fileContent) {
        fs.writeFileSync(filePath, fixedContent);
      }
    }
    
    const result = {
      file: filePath,
      violations: violations.filter(v => v.status !== 'fixed'),
      autoFixed: fixedViolations.length,
      totalViolations: violations.length
    };
    
    // Cache result
    this.cache.set(cacheKey, {
      data: result,
      timestamp: Date.now()
    });
    
    return result;
  }

  /**
   * Apply a single rule to file content
   */
  async applyRule(rule, filePath, content, lines) {
    const violations = [];
    
    switch (rule.pattern?.type) {
      case 'regex':
        violations.push(...this.applyRegexRule(rule, filePath, content, lines));
        break;
        
      case 'ai':
        violations.push(...await this.applyAIRule(rule, filePath, content));
        break;
        
      case 'line-count':
        violations.push(...this.applyLineCountRule(rule, filePath, lines));
        break;
        
      case 'file-existence':
        violations.push(...this.applyFileExistenceRule(rule, filePath));
        break;
        
      case 'custom':
        violations.push(...this.applyCustomRule(rule, filePath, content));
        break;
    }
    
    return violations;
  }

  /**
   * Apply regex-based rule
   */
  applyRegexRule(rule, filePath, content, lines) {
    const violations = [];
    const regex = new RegExp(rule.pattern.value, rule.pattern.multiline ? 'gm' : 'g');
    
    let match;
    while ((match = regex.exec(content)) !== null) {
      const position = this.getLineAndColumn(content, match.index);
      
      violations.push({
        id: `${rule.id}-${match.index}`,
        rule_id: rule.id,
        rule_name: rule.name,
        file_path: filePath,
        line_number: position.line,
        column_number: position.column,
        severity: this.getSeverity(rule),
        message: rule.message || rule.description,
        suggested_fix: rule.fix_template,
        auto_fixable: rule.category === 'auto-fix' && rule.auto_fix,
        match: match[0],
        status: 'pending',
        risk_level: rule.risk_level
      });
    }
    
    return violations;
  }

  /**
   * Apply AI-powered rule (using resource-claude-code)
   */
  async applyAIRule(rule, filePath, content) {
    const violations = [];
    
    try {
      const { exec } = require('child_process');
      const { promisify } = require('util');
      const execAsync = promisify(exec);
      
      const command = `resource-claude-code analyze --file "${filePath}" --prompt "${rule.pattern.prompt}"`;
      const { stdout } = await execAsync(command);
      
      const analysis = JSON.parse(stdout);
      if (analysis.violations) {
        for (const v of analysis.violations) {
          violations.push({
            id: `${rule.id}-${v.line}-${v.column}`,
            rule_id: rule.id,
            rule_name: rule.name,
            file_path: filePath,
            line_number: v.line,
            column_number: v.column || 0,
            severity: this.getSeverity(rule),
            message: v.message || rule.message,
            suggested_fix: v.fix || rule.fix_template,
            auto_fixable: false, // AI rules are never auto-fixable
            status: 'pending',
            risk_level: rule.risk_level
          });
        }
      }
    } catch (error) {
      console.error(`AI analysis failed for rule ${rule.id}:`, error);
    }
    
    return violations;
  }

  /**
   * Apply line count rule
   */
  applyLineCountRule(rule, filePath, lines) {
    const violations = [];
    
    if (lines.length > rule.pattern.threshold) {
      violations.push({
        id: `${rule.id}-file`,
        rule_id: rule.id,
        rule_name: rule.name,
        file_path: filePath,
        line_number: rule.pattern.threshold,
        column_number: 0,
        severity: this.getSeverity(rule),
        message: `${rule.message} (${lines.length} lines)`,
        suggested_fix: rule.fix_template,
        auto_fixable: false,
        status: 'pending',
        risk_level: rule.risk_level
      });
    }
    
    return violations;
  }

  /**
   * Apply file existence rule
   */
  applyFileExistenceRule(rule, filePath) {
    const violations = [];
    const dir = rule.pattern.scope === 'scenario-root' 
      ? path.dirname(path.dirname(filePath))
      : path.dirname(filePath);
    
    const requiredFile = path.join(dir, rule.pattern.required_file);
    
    if (!fs.existsSync(requiredFile)) {
      violations.push({
        id: `${rule.id}-missing`,
        rule_id: rule.id,
        rule_name: rule.name,
        file_path: dir,
        line_number: 0,
        column_number: 0,
        severity: this.getSeverity(rule),
        message: rule.message,
        suggested_fix: `Create ${rule.pattern.required_file}`,
        auto_fixable: false,
        status: 'pending',
        risk_level: rule.risk_level
      });
    }
    
    return violations;
  }

  /**
   * Apply custom validation rule
   */
  applyCustomRule(rule, filePath, content) {
    // This would call custom validator functions
    // For now, return empty
    return [];
  }

  /**
   * Check if a violation can be auto-fixed
   */
  canAutoFix(violation, riskLevel) {
    const riskLevels = ['safe', 'moderate', 'dangerous'];
    const violationRiskIndex = riskLevels.indexOf(violation.risk_level);
    const allowedRiskIndex = riskLevels.indexOf(riskLevel);
    
    return violationRiskIndex <= allowedRiskIndex;
  }

  /**
   * Apply a fix to content
   */
  applyFix(content, violation) {
    const rule = this.rules.get(violation.rule_id);
    if (!rule || !rule.auto_fix) return content;
    
    if (rule.auto_fix.search) {
      return content.replace(rule.auto_fix.search, rule.auto_fix.replace);
    }
    
    if (rule.auto_fix.search_regex) {
      const regex = new RegExp(rule.auto_fix.search_regex, 'g');
      return content.replace(regex, rule.auto_fix.replace);
    }
    
    if (rule.auto_fix.patterns) {
      let fixed = content;
      for (const pattern of rule.auto_fix.patterns) {
        if (pattern.search) {
          fixed = fixed.replace(pattern.search, pattern.replace);
        } else if (pattern.search_regex) {
          const regex = new RegExp(pattern.search_regex, 'g');
          fixed = fixed.replace(regex, pattern.replace);
        }
      }
      return fixed;
    }
    
    return content;
  }

  /**
   * Get line and column from character index
   */
  getLineAndColumn(content, index) {
    const lines = content.substring(0, index).split('\n');
    return {
      line: lines.length,
      column: lines[lines.length - 1].length + 1
    };
  }

  /**
   * Get severity level from rule
   */
  getSeverity(rule) {
    if (rule.risk_level === 'dangerous') return 'error';
    if (rule.risk_level === 'moderate') return 'warning';
    return 'info';
  }

  /**
   * Clear the analysis cache
   */
  clearCache() {
    this.cache.clear();
  }

  /**
   * Get all rules
   */
  getRules(filter = {}) {
    const rules = Array.from(this.rules.values());
    
    if (filter.category) {
      return rules.filter(r => r.category === filter.category);
    }
    
    if (filter.vrooli_specific !== undefined) {
      return rules.filter(r => r.vrooli_specific === filter.vrooli_specific);
    }
    
    return rules;
  }

  /**
   * Enable/disable a rule
   */
  setRuleEnabled(ruleId, enabled) {
    const rule = this.rules.get(ruleId);
    if (rule) {
      rule.enabled = enabled;
      this.clearCache();
      return true;
    }
    return false;
  }

  /**
   * Cleanup watchers
   */
  destroy() {
    for (const watcher of this.watchers.values()) {
      watcher.close();
    }
    this.watchers.clear();
    this.rules.clear();
    this.cache.clear();
  }
}

module.exports = RulesEngine;