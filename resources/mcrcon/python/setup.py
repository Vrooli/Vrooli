#!/usr/bin/env python3
"""
Setup script for MCRcon Python library
"""

from setuptools import setup, find_packages
from pathlib import Path

# Read README if it exists
readme_path = Path(__file__).parent.parent / 'README.md'
long_description = ''
if readme_path.exists():
    long_description = readme_path.read_text()

setup(
    name='vrooli-mcrcon',
    version='1.0.0',
    description='Python library for Minecraft RCON protocol',
    long_description=long_description,
    long_description_content_type='text/markdown',
    author='Vrooli',
    author_email='support@vrooli.com',
    url='https://github.com/vrooli/mcrcon',
    py_modules=['mcrcon'],
    python_requires='>=3.6',
    install_requires=[],
    classifiers=[
        'Development Status :: 4 - Beta',
        'Intended Audience :: Developers',
        'License :: OSI Approved :: MIT License',
        'Programming Language :: Python :: 3',
        'Programming Language :: Python :: 3.6',
        'Programming Language :: Python :: 3.7',
        'Programming Language :: Python :: 3.8',
        'Programming Language :: Python :: 3.9',
        'Programming Language :: Python :: 3.10',
        'Programming Language :: Python :: 3.11',
        'Topic :: Games/Entertainment',
        'Topic :: Software Development :: Libraries :: Python Modules',
    ],
    keywords='minecraft rcon remote console automation',
    project_urls={
        'Bug Reports': 'https://github.com/vrooli/mcrcon/issues',
        'Source': 'https://github.com/vrooli/mcrcon',
    },
)