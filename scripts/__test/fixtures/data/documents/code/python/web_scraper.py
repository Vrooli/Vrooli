#!/usr/bin/env python3
"""
Web Scraper Test Module
Test fixture for code analysis and processing workflows
"""

import requests
import time
import json
import logging
from typing import Dict, List, Optional, Any
from urllib.parse import urljoin, urlparse
from dataclasses import dataclass, asdict
from datetime import datetime, timedelta

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

@dataclass
class ScrapedData:
    """Data structure for scraped content."""
    url: str
    title: str
    content: str
    timestamp: datetime
    metadata: Dict[str, Any]
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dictionary for JSON serialization."""
        data = asdict(self)
        data['timestamp'] = self.timestamp.isoformat()
        return data

class WebScraper:
    """
    A web scraper for extracting content from websites.
    
    This is a test fixture designed for:
    - Code analysis by AI systems
    - Document processing workflows
    - Automation testing scenarios
    """
    
    def __init__(self, delay: float = 1.0, timeout: int = 30):
        """
        Initialize the web scraper.
        
        Args:
            delay: Delay between requests in seconds
            timeout: Request timeout in seconds
        """
        self.delay = delay
        self.timeout = timeout
        self.session = requests.Session()
        self.session.headers.update({
            'User-Agent': 'Vrooli-Test-Scraper/1.0 (Test Fixture)'
        })
        
    def scrape_url(self, url: str) -> Optional[ScrapedData]:
        """
        Scrape content from a single URL.
        
        Args:
            url: The URL to scrape
            
        Returns:
            ScrapedData object or None if scraping failed
        """
        try:
            logger.info(f"Scraping URL: {url}")
            
            # Add delay to be respectful
            time.sleep(self.delay)
            
            response = self.session.get(url, timeout=self.timeout)
            response.raise_for_status()
            
            # Extract basic content (simplified for test fixture)
            content = response.text
            title = self._extract_title(content)
            
            # Create metadata
            metadata = {
                'status_code': response.status_code,
                'content_type': response.headers.get('content-type', ''),
                'content_length': len(content),
                'response_time': response.elapsed.total_seconds(),
                'final_url': response.url
            }
            
            return ScrapedData(
                url=url,
                title=title,
                content=content[:5000],  # Truncate for testing
                timestamp=datetime.now(),
                metadata=metadata
            )
            
        except requests.RequestException as e:
            logger.error(f"Failed to scrape {url}: {e}")
            return None
        except Exception as e:
            logger.error(f"Unexpected error scraping {url}: {e}")
            return None
    
    def _extract_title(self, html_content: str) -> str:
        """Extract title from HTML content (simplified)."""
        import re
        title_match = re.search(r'<title[^>]*>([^<]+)</title>', html_content, re.IGNORECASE)
        return title_match.group(1).strip() if title_match else "No Title"
    
    def scrape_multiple(self, urls: List[str]) -> List[ScrapedData]:
        """
        Scrape multiple URLs.
        
        Args:
            urls: List of URLs to scrape
            
        Returns:
            List of successfully scraped data
        """
        results = []
        
        for url in urls:
            try:
                data = self.scrape_url(url)
                if data:
                    results.append(data)
                    logger.info(f"Successfully scraped: {url}")
                else:
                    logger.warning(f"Failed to scrape: {url}")
                    
            except KeyboardInterrupt:
                logger.info("Scraping interrupted by user")
                break
            except Exception as e:
                logger.error(f"Error processing {url}: {e}")
                continue
                
        return results
    
    def save_results(self, results: List[ScrapedData], filename: str):
        """Save scraping results to JSON file."""
        try:
            data = [result.to_dict() for result in results]
            
            with open(filename, 'w', encoding='utf-8') as f:
                json.dump({
                    'scraped_at': datetime.now().isoformat(),
                    'total_results': len(data),
                    'results': data
                }, f, indent=2, ensure_ascii=False)
                
            logger.info(f"Saved {len(results)} results to {filename}")
            
        except Exception as e:
            logger.error(f"Failed to save results: {e}")

def main():
    """Main function for testing the scraper."""
    # Test URLs (using local/safe URLs for testing)
    test_urls = [
        "http://localhost:3000",
        "http://localhost:1880",
        "http://localhost:5678"
    ]
    
    # Initialize scraper
    scraper = WebScraper(delay=0.5, timeout=10)
    
    # Scrape URLs
    results = scraper.scrape_multiple(test_urls)
    
    # Display results
    print(f"\nScraping completed. {len(results)} successful results:")
    for result in results:
        print(f"- {result.url}: {result.title}")
    
    # Save results
    if results:
        scraper.save_results(results, 'scraping_results.json')

if __name__ == "__main__":
    main()