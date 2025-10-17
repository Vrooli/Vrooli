#!/usr/bin/env python3
"""
Custom Blocking Rules Interface for Splink
Allows user-defined blocking strategies for performance optimization
"""

import os
import json
import logging
from typing import Dict, List, Optional, Any, Callable
from datetime import datetime
from enum import Enum
import re

from pydantic import BaseModel, Field, validator

# Configure logging
logger = logging.getLogger(__name__)

class BlockingStrategy(str, Enum):
    """Available blocking strategies"""
    EXACT = "exact"  # Exact match on field
    SOUNDEX = "soundex"  # Phonetic matching
    FIRST_N = "first_n"  # First N characters
    LAST_N = "last_n"  # Last N characters
    SUBSTRING = "substring"  # Substring match
    REGEX = "regex"  # Regular expression
    DATE_RANGE = "date_range"  # Date within range
    NUMERIC_RANGE = "numeric_range"  # Number within range
    TOKEN_SET = "token_set"  # Set of tokens match
    CUSTOM_SQL = "custom_sql"  # Custom SQL expression

class BlockingRule(BaseModel):
    """Individual blocking rule definition"""
    name: str = Field(description="Name of the blocking rule")
    field: str = Field(description="Field to apply blocking on")
    strategy: BlockingStrategy = Field(description="Blocking strategy to use")
    parameters: Dict[str, Any] = Field(default_factory=dict, description="Strategy-specific parameters")
    enabled: bool = Field(default=True, description="Whether the rule is enabled")
    priority: int = Field(default=5, ge=1, le=10, description="Rule priority (higher = more important)")
    
    @validator('parameters')
    def validate_parameters(cls, v, values):
        """Validate parameters based on strategy"""
        if 'strategy' not in values:
            return v
        
        strategy = values['strategy']
        
        if strategy == BlockingStrategy.FIRST_N:
            if 'n' not in v or not isinstance(v['n'], int) or v['n'] < 1:
                raise ValueError("FIRST_N strategy requires 'n' parameter as positive integer")
        
        elif strategy == BlockingStrategy.LAST_N:
            if 'n' not in v or not isinstance(v['n'], int) or v['n'] < 1:
                raise ValueError("LAST_N strategy requires 'n' parameter as positive integer")
        
        elif strategy == BlockingStrategy.SUBSTRING:
            if 'start' not in v or 'end' not in v:
                raise ValueError("SUBSTRING strategy requires 'start' and 'end' parameters")
        
        elif strategy == BlockingStrategy.REGEX:
            if 'pattern' not in v:
                raise ValueError("REGEX strategy requires 'pattern' parameter")
            # Validate regex pattern
            try:
                re.compile(v['pattern'])
            except re.error as e:
                raise ValueError(f"Invalid regex pattern: {e}")
        
        elif strategy == BlockingStrategy.DATE_RANGE:
            if 'days' not in v or not isinstance(v['days'], (int, float)):
                raise ValueError("DATE_RANGE strategy requires 'days' parameter")
        
        elif strategy == BlockingStrategy.NUMERIC_RANGE:
            if 'tolerance' not in v or not isinstance(v['tolerance'], (int, float)):
                raise ValueError("NUMERIC_RANGE strategy requires 'tolerance' parameter")
        
        elif strategy == BlockingStrategy.CUSTOM_SQL:
            if 'expression' not in v:
                raise ValueError("CUSTOM_SQL strategy requires 'expression' parameter")
        
        return v

class BlockingRuleSet(BaseModel):
    """Collection of blocking rules"""
    name: str = Field(description="Name of the rule set")
    description: Optional[str] = Field(None, description="Description of the rule set")
    rules: List[BlockingRule] = Field(default_factory=list, description="List of blocking rules")
    combination_logic: str = Field(default="OR", pattern="^(AND|OR|CUSTOM)$", description="How to combine rules")
    custom_logic: Optional[str] = Field(None, description="Custom SQL logic for combining rules")
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)

class BlockingRulesEngine:
    """Engine for managing and applying custom blocking rules"""
    
    def __init__(self, data_dir: str = "/data"):
        self.data_dir = data_dir
        self.rule_sets: Dict[str, BlockingRuleSet] = {}
        self.load_rule_sets()
    
    def load_rule_sets(self):
        """Load saved rule sets from disk"""
        rule_sets_file = os.path.join(self.data_dir, "blocking_rule_sets.json")
        if os.path.exists(rule_sets_file):
            try:
                with open(rule_sets_file, 'r') as f:
                    data = json.load(f)
                    for name, rule_set_data in data.items():
                        self.rule_sets[name] = BlockingRuleSet(**rule_set_data)
                logger.info(f"Loaded {len(self.rule_sets)} blocking rule sets")
            except Exception as e:
                logger.error(f"Failed to load rule sets: {e}")
    
    def save_rule_sets(self):
        """Save rule sets to disk"""
        rule_sets_file = os.path.join(self.data_dir, "blocking_rule_sets.json")
        try:
            data = {name: rule_set.dict() for name, rule_set in self.rule_sets.items()}
            # Convert datetime objects to strings
            for rule_set_data in data.values():
                rule_set_data['created_at'] = rule_set_data['created_at'].isoformat()
                rule_set_data['updated_at'] = rule_set_data['updated_at'].isoformat()
            
            with open(rule_sets_file, 'w') as f:
                json.dump(data, f, indent=2)
            logger.info(f"Saved {len(self.rule_sets)} blocking rule sets")
        except Exception as e:
            logger.error(f"Failed to save rule sets: {e}")
    
    def create_rule_set(self, rule_set: BlockingRuleSet) -> Dict[str, Any]:
        """Create a new blocking rule set"""
        if rule_set.name in self.rule_sets:
            return {
                "success": False,
                "error": f"Rule set '{rule_set.name}' already exists"
            }
        
        self.rule_sets[rule_set.name] = rule_set
        self.save_rule_sets()
        
        return {
            "success": True,
            "message": f"Created rule set '{rule_set.name}' with {len(rule_set.rules)} rules"
        }
    
    def update_rule_set(self, name: str, rule_set: BlockingRuleSet) -> Dict[str, Any]:
        """Update an existing blocking rule set"""
        if name not in self.rule_sets:
            return {
                "success": False,
                "error": f"Rule set '{name}' not found"
            }
        
        rule_set.updated_at = datetime.utcnow()
        self.rule_sets[name] = rule_set
        self.save_rule_sets()
        
        return {
            "success": True,
            "message": f"Updated rule set '{name}'"
        }
    
    def delete_rule_set(self, name: str) -> Dict[str, Any]:
        """Delete a blocking rule set"""
        if name not in self.rule_sets:
            return {
                "success": False,
                "error": f"Rule set '{name}' not found"
            }
        
        del self.rule_sets[name]
        self.save_rule_sets()
        
        return {
            "success": True,
            "message": f"Deleted rule set '{name}'"
        }
    
    def get_rule_set(self, name: str) -> Optional[BlockingRuleSet]:
        """Get a specific rule set"""
        return self.rule_sets.get(name)
    
    def list_rule_sets(self) -> List[Dict[str, Any]]:
        """List all rule sets"""
        return [
            {
                "name": name,
                "description": rule_set.description,
                "rules_count": len(rule_set.rules),
                "combination_logic": rule_set.combination_logic,
                "created_at": rule_set.created_at.isoformat(),
                "updated_at": rule_set.updated_at.isoformat()
            }
            for name, rule_set in self.rule_sets.items()
        ]
    
    def generate_sql_blocking(self, rule: BlockingRule, table_alias: str = "a") -> str:
        """Generate SQL blocking clause for a rule"""
        field = f"{table_alias}.{rule.field}"
        
        if rule.strategy == BlockingStrategy.EXACT:
            return f"{field} = b.{rule.field}"
        
        elif rule.strategy == BlockingStrategy.SOUNDEX:
            return f"SOUNDEX({field}) = SOUNDEX(b.{rule.field})"
        
        elif rule.strategy == BlockingStrategy.FIRST_N:
            n = rule.parameters['n']
            return f"LEFT({field}, {n}) = LEFT(b.{rule.field}, {n})"
        
        elif rule.strategy == BlockingStrategy.LAST_N:
            n = rule.parameters['n']
            return f"RIGHT({field}, {n}) = RIGHT(b.{rule.field}, {n})"
        
        elif rule.strategy == BlockingStrategy.SUBSTRING:
            start = rule.parameters['start']
            end = rule.parameters['end']
            return f"SUBSTRING({field}, {start}, {end}) = SUBSTRING(b.{rule.field}, {start}, {end})"
        
        elif rule.strategy == BlockingStrategy.REGEX:
            pattern = rule.parameters['pattern']
            return f"REGEXP_MATCHES({field}, '{pattern}')"
        
        elif rule.strategy == BlockingStrategy.DATE_RANGE:
            days = rule.parameters['days']
            return f"ABS(DATEDIFF('day', {field}, b.{rule.field})) <= {days}"
        
        elif rule.strategy == BlockingStrategy.NUMERIC_RANGE:
            tolerance = rule.parameters['tolerance']
            return f"ABS({field} - b.{rule.field}) <= {tolerance}"
        
        elif rule.strategy == BlockingStrategy.TOKEN_SET:
            # This would need more complex SQL or UDF
            return f"TOKEN_OVERLAP({field}, b.{rule.field}) > 0.5"
        
        elif rule.strategy == BlockingStrategy.CUSTOM_SQL:
            return rule.parameters['expression'].replace("{{field}}", field)
        
        else:
            return f"{field} = b.{rule.field}"  # Default to exact match
    
    def apply_rule_set(self, rule_set_name: str, dataset_info: Dict[str, Any]) -> Dict[str, Any]:
        """Apply a rule set to generate blocking SQL"""
        if rule_set_name not in self.rule_sets:
            return {
                "success": False,
                "error": f"Rule set '{rule_set_name}' not found"
            }
        
        rule_set = self.rule_sets[rule_set_name]
        
        # Filter enabled rules and sort by priority
        enabled_rules = [r for r in rule_set.rules if r.enabled]
        enabled_rules.sort(key=lambda x: x.priority, reverse=True)
        
        if not enabled_rules:
            return {
                "success": False,
                "error": "No enabled rules in rule set"
            }
        
        # Generate SQL clauses
        sql_clauses = []
        for rule in enabled_rules:
            try:
                sql_clause = self.generate_sql_blocking(rule)
                sql_clauses.append(f"({sql_clause})")
            except Exception as e:
                logger.warning(f"Failed to generate SQL for rule '{rule.name}': {e}")
        
        if not sql_clauses:
            return {
                "success": False,
                "error": "Failed to generate any SQL clauses"
            }
        
        # Combine clauses based on logic
        if rule_set.combination_logic == "AND":
            combined_sql = " AND ".join(sql_clauses)
        elif rule_set.combination_logic == "OR":
            combined_sql = " OR ".join(sql_clauses)
        elif rule_set.combination_logic == "CUSTOM" and rule_set.custom_logic:
            # Replace placeholders in custom logic
            combined_sql = rule_set.custom_logic
            for i, clause in enumerate(sql_clauses):
                combined_sql = combined_sql.replace(f"$${i+1}$$", clause)
        else:
            combined_sql = " OR ".join(sql_clauses)  # Default to OR
        
        return {
            "success": True,
            "sql": combined_sql,
            "rules_applied": len(sql_clauses),
            "message": f"Applied {len(sql_clauses)} blocking rules"
        }
    
    def validate_rule(self, rule: BlockingRule, sample_data: Dict[str, Any]) -> Dict[str, Any]:
        """Validate a blocking rule against sample data"""
        try:
            # Check if field exists in sample data
            if rule.field not in sample_data:
                return {
                    "success": False,
                    "error": f"Field '{rule.field}' not found in sample data"
                }
            
            # Test the SQL generation
            sql = self.generate_sql_blocking(rule)
            
            return {
                "success": True,
                "sql": sql,
                "message": "Rule validation successful"
            }
            
        except Exception as e:
            return {
                "success": False,
                "error": str(e)
            }
    
    def optimize_rules(self, rule_set_name: str, performance_data: Dict[str, Any]) -> Dict[str, Any]:
        """Optimize blocking rules based on performance data"""
        if rule_set_name not in self.rule_sets:
            return {
                "success": False,
                "error": f"Rule set '{rule_set_name}' not found"
            }
        
        rule_set = self.rule_sets[rule_set_name]
        
        # Analyze performance data and suggest optimizations
        suggestions = []
        
        # Check for high-cardinality exact matches
        for rule in rule_set.rules:
            if rule.strategy == BlockingStrategy.EXACT:
                field_cardinality = performance_data.get(f"{rule.field}_cardinality", 0)
                if field_cardinality > 10000:
                    suggestions.append({
                        "rule": rule.name,
                        "suggestion": f"Consider using FIRST_N or SOUNDEX for high-cardinality field '{rule.field}'"
                    })
        
        # Check for missing indexes
        for rule in rule_set.rules:
            if rule.field not in performance_data.get("indexed_fields", []):
                suggestions.append({
                    "rule": rule.name,
                    "suggestion": f"Consider adding index on field '{rule.field}'"
                })
        
        # Suggest reordering based on selectivity
        selectivity_scores = []
        for rule in rule_set.rules:
            selectivity = performance_data.get(f"{rule.field}_selectivity", 0.5)
            selectivity_scores.append((rule.name, selectivity))
        
        selectivity_scores.sort(key=lambda x: x[1], reverse=True)
        
        if selectivity_scores:
            suggestions.append({
                "type": "reordering",
                "suggestion": "Optimal rule order based on selectivity",
                "order": [name for name, _ in selectivity_scores]
            })
        
        return {
            "success": True,
            "suggestions": suggestions,
            "message": f"Generated {len(suggestions)} optimization suggestions"
        }

# Pre-defined rule templates
RULE_TEMPLATES = {
    "person_matching": BlockingRuleSet(
        name="person_matching_template",
        description="Template for matching person records",
        rules=[
            BlockingRule(
                name="exact_ssn",
                field="ssn",
                strategy=BlockingStrategy.EXACT,
                priority=10
            ),
            BlockingRule(
                name="soundex_lastname",
                field="last_name",
                strategy=BlockingStrategy.SOUNDEX,
                priority=8
            ),
            BlockingRule(
                name="first3_firstname",
                field="first_name",
                strategy=BlockingStrategy.FIRST_N,
                parameters={"n": 3},
                priority=7
            ),
            BlockingRule(
                name="birth_year",
                field="birth_date",
                strategy=BlockingStrategy.DATE_RANGE,
                parameters={"days": 365},
                priority=6
            )
        ],
        combination_logic="OR"
    ),
    
    "address_matching": BlockingRuleSet(
        name="address_matching_template",
        description="Template for matching address records",
        rules=[
            BlockingRule(
                name="exact_zipcode",
                field="zipcode",
                strategy=BlockingStrategy.EXACT,
                priority=9
            ),
            BlockingRule(
                name="first5_street",
                field="street_address",
                strategy=BlockingStrategy.FIRST_N,
                parameters={"n": 5},
                priority=7
            ),
            BlockingRule(
                name="soundex_city",
                field="city",
                strategy=BlockingStrategy.SOUNDEX,
                priority=6
            )
        ],
        combination_logic="AND"
    ),
    
    "company_matching": BlockingRuleSet(
        name="company_matching_template",
        description="Template for matching company records",
        rules=[
            BlockingRule(
                name="exact_tax_id",
                field="tax_id",
                strategy=BlockingStrategy.EXACT,
                priority=10
            ),
            BlockingRule(
                name="token_company_name",
                field="company_name",
                strategy=BlockingStrategy.TOKEN_SET,
                priority=8
            ),
            BlockingRule(
                name="exact_industry",
                field="industry_code",
                strategy=BlockingStrategy.EXACT,
                priority=5
            )
        ],
        combination_logic="OR"
    )
}

def get_template(template_name: str) -> Optional[BlockingRuleSet]:
    """Get a pre-defined rule template"""
    return RULE_TEMPLATES.get(template_name)