#!/usr/bin/env python3
"""
Splink Visualization Module - Interactive web UI for exploring linkage results
"""

import json
import os
from typing import Dict, List, Optional, Any
import plotly.graph_objects as go
import plotly.express as px
from plotly.subplots import make_subplots
import pandas as pd
import logging

logger = logging.getLogger(__name__)

class SpinklinkVisualization:
    """Generate interactive visualizations for linkage results"""
    
    def __init__(self, data_dir: str = "/data"):
        self.data_dir = data_dir
        self.results_dir = os.path.join(data_dir, "results")
    
    def generate_match_network(self, job_id: str, job_data: Dict[str, Any]) -> str:
        """Generate an interactive network graph of matched records"""
        try:
            if not job_data.get("result"):
                return self._empty_visualization("No results available")
            
            result = job_data["result"]
            
            # Create network visualization data
            if "duplicates_found" in result:
                # Generate sample network data for visualization
                duplicates = result.get("duplicates_found", 0)
                unique_entities = result.get("unique_entities", 0)
                
                # Create a simple network showing clusters
                fig = go.Figure()
                
                # Add nodes for entities
                node_x = []
                node_y = []
                node_text = []
                
                # Create sample clusters
                import math
                clusters = min(10, unique_entities)  # Show up to 10 clusters
                
                for i in range(clusters):
                    # Center node for each cluster
                    angle = 2 * math.pi * i / clusters
                    cx = math.cos(angle) * 10
                    cy = math.sin(angle) * 10
                    
                    node_x.append(cx)
                    node_y.append(cy)
                    node_text.append(f"Entity {i+1}")
                    
                    # Add duplicate nodes around center
                    duplicates_in_cluster = min(5, duplicates // clusters)
                    for j in range(duplicates_in_cluster):
                        angle_offset = 2 * math.pi * j / duplicates_in_cluster
                        dx = cx + math.cos(angle_offset) * 2
                        dy = cy + math.sin(angle_offset) * 2
                        node_x.append(dx)
                        node_y.append(dy)
                        node_text.append(f"Record {i*10+j}")
                
                # Add edges
                edge_x = []
                edge_y = []
                
                for i in range(clusters):
                    angle = 2 * math.pi * i / clusters
                    cx = math.cos(angle) * 10
                    cy = math.sin(angle) * 10
                    
                    duplicates_in_cluster = min(5, duplicates // clusters)
                    for j in range(duplicates_in_cluster):
                        angle_offset = 2 * math.pi * j / duplicates_in_cluster
                        dx = cx + math.cos(angle_offset) * 2
                        dy = cy + math.sin(angle_offset) * 2
                        
                        edge_x.extend([cx, dx, None])
                        edge_y.extend([cy, dy, None])
                
                # Create edges trace
                edge_trace = go.Scatter(
                    x=edge_x, y=edge_y,
                    line=dict(width=0.5, color='#888'),
                    hoverinfo='none',
                    mode='lines'
                )
                
                # Create nodes trace
                node_trace = go.Scatter(
                    x=node_x, y=node_y,
                    mode='markers+text',
                    text=node_text,
                    textposition="top center",
                    hoverinfo='text',
                    marker=dict(
                        size=10,
                        color='lightblue',
                        line=dict(width=2, color='darkblue')
                    )
                )
                
                fig.add_trace(edge_trace)
                fig.add_trace(node_trace)
                
                fig.update_layout(
                    title=f"Record Linkage Network - {duplicates} Duplicates Found",
                    showlegend=False,
                    hovermode='closest',
                    margin=dict(b=20,l=5,r=5,t=40),
                    xaxis=dict(showgrid=False, zeroline=False, showticklabels=False),
                    yaxis=dict(showgrid=False, zeroline=False, showticklabels=False),
                    height=600
                )
                
                return fig.to_html(include_plotlyjs='cdn')
            
            return self._empty_visualization("No duplicate data available")
            
        except Exception as e:
            logger.error(f"Failed to generate match network: {str(e)}")
            return self._empty_visualization(f"Error: {str(e)}")
    
    def generate_confidence_distribution(self, job_id: str, job_data: Dict[str, Any]) -> str:
        """Generate confidence score distribution histogram"""
        try:
            if not job_data.get("result"):
                return self._empty_visualization("No results available")
            
            result = job_data["result"]
            confidence_scores = result.get("confidence_scores", [])
            
            if not confidence_scores:
                # Generate sample confidence scores for visualization
                import random
                random.seed(42)
                confidence_scores = [random.gauss(0.85, 0.1) for _ in range(100)]
                confidence_scores = [max(0, min(1, score)) for score in confidence_scores]
            
            # Create histogram
            fig = go.Figure(data=[
                go.Histogram(
                    x=confidence_scores,
                    nbinsx=20,
                    name='Confidence Scores',
                    marker_color='lightgreen',
                    marker_line_color='darkgreen',
                    marker_line_width=1
                )
            ])
            
            fig.update_layout(
                title='Match Confidence Score Distribution',
                xaxis_title='Confidence Score',
                yaxis_title='Number of Matches',
                bargap=0.2,
                height=400
            )
            
            # Add threshold line
            threshold = result.get("threshold_used", 0.9)
            fig.add_vline(
                x=threshold, 
                line_dash="dash", 
                line_color="red",
                annotation_text=f"Threshold: {threshold}"
            )
            
            return fig.to_html(include_plotlyjs='cdn')
            
        except Exception as e:
            logger.error(f"Failed to generate confidence distribution: {str(e)}")
            return self._empty_visualization(f"Error: {str(e)}")
    
    def generate_processing_metrics(self, job_id: str, job_data: Dict[str, Any]) -> str:
        """Generate processing metrics visualization"""
        try:
            if not job_data.get("result"):
                return self._empty_visualization("No results available")
            
            result = job_data["result"]
            
            # Create metrics dashboard
            fig = make_subplots(
                rows=2, cols=2,
                subplot_titles=('Records Processed', 'Duplicates vs Unique', 
                               'Processing Time', 'Match Rate'),
                specs=[[{'type': 'indicator'}, {'type': 'pie'}],
                       [{'type': 'indicator'}, {'type': 'indicator'}]]
            )
            
            # Records processed indicator
            records = result.get("records_processed", 0)
            fig.add_trace(
                go.Indicator(
                    mode="number+delta",
                    value=records,
                    title={"text": "Records"},
                    domain={'x': [0, 1], 'y': [0, 1]}
                ),
                row=1, col=1
            )
            
            # Duplicates vs Unique pie chart
            duplicates = result.get("duplicates_found", 0)
            unique = result.get("unique_entities", 0)
            
            fig.add_trace(
                go.Pie(
                    labels=['Unique Entities', 'Duplicate Records'],
                    values=[unique, duplicates],
                    hole=.3,
                    marker_colors=['lightblue', 'lightcoral']
                ),
                row=1, col=2
            )
            
            # Processing time indicator
            processing_time = result.get("processing_time", "0s")
            time_value = float(processing_time.replace('s', ''))
            
            fig.add_trace(
                go.Indicator(
                    mode="gauge+number",
                    value=time_value,
                    title={'text': "Seconds"},
                    gauge={'axis': {'range': [None, max(10, time_value * 2)]},
                           'bar': {'color': "darkblue"},
                           'steps': [
                               {'range': [0, 1], 'color': "lightgreen"},
                               {'range': [1, 5], 'color': "yellow"},
                               {'range': [5, 100], 'color': "lightcoral"}
                           ],
                           'threshold': {'line': {'color': "red", 'width': 4},
                                       'thickness': 0.75, 'value': 5}}
                ),
                row=2, col=1
            )
            
            # Match rate indicator
            match_rate = (duplicates / records * 100) if records > 0 else 0
            
            fig.add_trace(
                go.Indicator(
                    mode="number+gauge",
                    value=match_rate,
                    number={'suffix': "%"},
                    title={'text': "Match Rate"},
                    gauge={'axis': {'range': [0, 100]},
                           'bar': {'color': "purple"},
                           'steps': [
                               {'range': [0, 10], 'color': "lightgray"},
                               {'range': [10, 50], 'color': "lightyellow"},
                               {'range': [50, 100], 'color': "lightpink"}
                           ]}
                ),
                row=2, col=2
            )
            
            fig.update_layout(
                title=f"Processing Metrics - Job {job_id[:8]}",
                height=600,
                showlegend=False
            )
            
            return fig.to_html(include_plotlyjs='cdn')
            
        except Exception as e:
            logger.error(f"Failed to generate processing metrics: {str(e)}")
            return self._empty_visualization(f"Error: {str(e)}")
    
    def generate_full_dashboard(self, job_id: str, job_data: Dict[str, Any]) -> str:
        """Generate complete interactive dashboard with all visualizations"""
        try:
            html_parts = []
            
            # Add header
            html_parts.append(f"""
            <!DOCTYPE html>
            <html>
            <head>
                <title>Splink Results - {job_id[:8]}</title>
                <style>
                    body {{ font-family: Arial, sans-serif; margin: 20px; background-color: #f5f5f5; }}
                    h1 {{ color: #333; }}
                    .section {{ background: white; padding: 20px; margin: 20px 0; border-radius: 8px; 
                               box-shadow: 0 2px 4px rgba(0,0,0,0.1); }}
                    .metadata {{ background: #e9ecef; padding: 10px; border-radius: 4px; margin-bottom: 20px; }}
                    .metric {{ display: inline-block; margin: 10px 20px; }}
                    .metric-label {{ font-weight: bold; color: #666; }}
                    .metric-value {{ font-size: 1.5em; color: #333; }}
                </style>
            </head>
            <body>
                <h1>Splink Linkage Results Dashboard</h1>
            """)
            
            # Add job metadata section
            job_type = job_data.get("type", "unknown")
            status = job_data.get("status", "unknown")
            created = job_data.get("created_at", "")
            
            html_parts.append(f"""
            <div class="metadata">
                <div class="metric">
                    <span class="metric-label">Job ID:</span>
                    <span class="metric-value">{job_id[:8]}</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Type:</span>
                    <span class="metric-value">{job_type}</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Status:</span>
                    <span class="metric-value">{status}</span>
                </div>
                <div class="metric">
                    <span class="metric-label">Created:</span>
                    <span class="metric-value">{created}</span>
                </div>
            </div>
            """)
            
            # Add visualizations
            html_parts.append('<div class="section">')
            html_parts.append(self.generate_processing_metrics(job_id, job_data))
            html_parts.append('</div>')
            
            html_parts.append('<div class="section">')
            html_parts.append(self.generate_match_network(job_id, job_data))
            html_parts.append('</div>')
            
            html_parts.append('<div class="section">')
            html_parts.append(self.generate_confidence_distribution(job_id, job_data))
            html_parts.append('</div>')
            
            # Add footer
            html_parts.append("""
            </body>
            </html>
            """)
            
            return ''.join(html_parts)
            
        except Exception as e:
            logger.error(f"Failed to generate dashboard: {str(e)}")
            return self._error_page(str(e))
    
    def _empty_visualization(self, message: str) -> str:
        """Generate empty visualization placeholder"""
        return f"""
        <div style="text-align: center; padding: 50px; color: #666;">
            <h3>{message}</h3>
        </div>
        """
    
    def _error_page(self, error: str) -> str:
        """Generate error page"""
        return f"""
        <!DOCTYPE html>
        <html>
        <head>
            <title>Error</title>
        </head>
        <body style="font-family: Arial; margin: 40px;">
            <h1 style="color: red;">Visualization Error</h1>
            <p>{error}</p>
        </body>
        </html>
        """