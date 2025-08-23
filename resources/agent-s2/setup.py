#!/usr/bin/env python3
"""Setup script for Agent S2 Python package"""

from setuptools import setup, find_packages
import os

# Read README for long description
here = os.path.abspath(os.path.dirname(__file__))
with open(os.path.join(here, 'README.md'), encoding='utf-8') as f:
    long_description = f.read()

setup(
    name="agent-s2",
    version="1.0.0",
    description="Agent S2 - Autonomous Computer Interaction Service",
    long_description=long_description,
    long_description_content_type="text/markdown",
    author="Vrooli Team",
    author_email="dev@vrooli.com",
    url="https://github.com/Vrooli/Vrooli",
    packages=find_packages(exclude=["tests", "examples", "docker"]),
    install_requires=[
        "fastapi>=0.104.1",
        "uvicorn[standard]>=0.24.0",
        "pydantic>=2.5.0",
        "pyautogui>=0.9.54",
        "pillow>=10.1.0",
        "requests>=2.31.0",
        "aiofiles>=23.2.1",
        "python-multipart>=0.0.6",
    ],
    extras_require={
        "dev": [
            "pytest>=7.4.3",
            "pytest-asyncio>=0.21.1",
            "pytest-cov>=4.1.0",
            "black>=23.11.0",
            "flake8>=6.1.0",
            "mypy>=1.7.0",
        ],
        "ai": [
            "anthropic>=0.7.0",
            "openai>=1.3.0",
        ]
    },
    python_requires=">=3.8",
    classifiers=[
        "Development Status :: 4 - Beta",
        "Intended Audience :: Developers",
        "Topic :: Software Development :: Libraries :: Python Modules",
        "License :: OSI Approved :: MIT License",
        "Programming Language :: Python :: 3",
        "Programming Language :: Python :: 3.8",
        "Programming Language :: Python :: 3.9",
        "Programming Language :: Python :: 3.10",
        "Programming Language :: Python :: 3.11",
    ],
    keywords="automation gui screenshot ai agent",
    project_urls={
        "Bug Reports": "https://github.com/Vrooli/Vrooli/issues",
        "Source": "https://github.com/Vrooli/Vrooli",
    },
)