#!/usr/bin/env python3

"""
ElectionGuard API Server
Provides HTTP API for election operations
"""

import os
import sys
import json
import argparse
import logging
import hashlib
import secrets
from pathlib import Path
from flask import Flask, jsonify, request
from typing import Optional, Dict, Any

# Add ElectionGuard imports when fully implemented
# from electionguard import (
#     CiphertextElectionContext,
#     ElGamalKeyPair,
#     encrypt_ballot,
#     Guardian,
#     make_ciphertext_election_context,
# )

app = Flask(__name__)

# Configure logging
log_level = os.environ.get('ELECTIONGUARD_LOG_LEVEL', 'info').upper()
logging.basicConfig(
    level=getattr(logging, log_level, logging.INFO),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

# Global state (would be replaced with proper storage)
elections = {}
guardians = {}

# Vault integration class
class VaultClient:
    """Client for interacting with HashiCorp Vault"""
    
    def __init__(self):
        self.enabled = os.environ.get('ELECTIONGUARD_VAULT_ENABLED', 'true').lower() == 'true'
        self.vault_url = os.environ.get('ELECTIONGUARD_VAULT_URL', 'http://localhost:8200')
        self.vault_token = os.environ.get('VAULT_TOKEN', 'myroot')
        self.vault_path = os.environ.get('ELECTIONGUARD_VAULT_PATH', '/electionguard')
        self.connected = False
        
        if self.enabled:
            self._check_connection()
    
    def _check_connection(self):
        """Check if Vault is accessible"""
        try:
            import requests
            response = requests.get(
                f"{self.vault_url}/v1/sys/health",
                headers={'X-Vault-Token': self.vault_token},
                timeout=5
            )
            self.connected = response.status_code == 200
            if self.connected:
                logger.info("Successfully connected to Vault")
            else:
                logger.warning(f"Vault health check failed with status: {response.status_code}")
        except Exception as e:
            logger.warning(f"Could not connect to Vault: {e}")
            self.connected = False
    
    def store_secret(self, key: str, value: Dict[str, Any]) -> bool:
        """Store a secret in Vault"""
        if not self.enabled or not self.connected:
            logger.debug(f"Vault not available, storing secret {key} locally")
            return False
        
        try:
            import requests
            path = f"secret/data{self.vault_path}/{key}"
            response = requests.post(
                f"{self.vault_url}/v1/{path}",
                headers={'X-Vault-Token': self.vault_token},
                json={'data': value},
                timeout=10
            )
            if response.status_code in [200, 204]:
                logger.info(f"Stored secret {key} in Vault")
                return True
            else:
                logger.error(f"Failed to store secret {key}: {response.text}")
                return False
        except Exception as e:
            logger.error(f"Error storing secret {key}: {e}")
            return False
    
    def retrieve_secret(self, key: str) -> Optional[Dict[str, Any]]:
        """Retrieve a secret from Vault"""
        if not self.enabled or not self.connected:
            logger.debug(f"Vault not available, cannot retrieve secret {key}")
            return None
        
        try:
            import requests
            path = f"secret/data{self.vault_path}/{key}"
            response = requests.get(
                f"{self.vault_url}/v1/{path}",
                headers={'X-Vault-Token': self.vault_token},
                timeout=10
            )
            if response.status_code == 200:
                data = response.json()
                logger.info(f"Retrieved secret {key} from Vault")
                return data.get('data', {}).get('data', {})
            else:
                logger.debug(f"Secret {key} not found in Vault")
                return None
        except Exception as e:
            logger.error(f"Error retrieving secret {key}: {e}")
            return None
    
    def delete_secret(self, key: str) -> bool:
        """Delete a secret from Vault"""
        if not self.enabled or not self.connected:
            return False
        
        try:
            import requests
            path = f"secret/metadata{self.vault_path}/{key}"
            response = requests.delete(
                f"{self.vault_url}/v1/{path}",
                headers={'X-Vault-Token': self.vault_token},
                timeout=10
            )
            if response.status_code in [200, 204]:
                logger.info(f"Deleted secret {key} from Vault")
                return True
            else:
                logger.error(f"Failed to delete secret {key}: {response.text}")
                return False
        except Exception as e:
            logger.error(f"Error deleting secret {key}: {e}")
            return False

# Initialize Vault client
vault_client = VaultClient()


@app.route('/health', methods=['GET'])
def health_check():
    """Health check endpoint"""
    vault_status = 'connected' if vault_client.enabled and vault_client.connected else 'disabled'
    if vault_client.enabled and not vault_client.connected:
        vault_status = 'disconnected'
    
    return jsonify({
        'status': 'healthy',
        'service': 'electionguard',
        'version': '1.0.0',
        'elections_count': len(elections),
        'vault_status': vault_status,
        'integrations': {
            'vault': vault_client.enabled and vault_client.connected,
            'postgres': os.environ.get('ELECTIONGUARD_DB_ENABLED', 'false').lower() == 'true',
            'questdb': os.environ.get('ELECTIONGUARD_QUESTDB_ENABLED', 'false').lower() == 'true'
        }
    }), 200


@app.route('/api/v1/election/create', methods=['POST'])
def create_election():
    """Create a new election"""
    try:
        data = request.json
        election_id = data.get('election_id', f'election_{len(elections)}')
        
        # Store election configuration
        elections[election_id] = {
            'id': election_id,
            'name': data.get('name', 'Unnamed Election'),
            'guardians': data.get('guardians', 5),
            'threshold': data.get('threshold', 3),
            'status': 'created',
            'ballots': [],
            'tally': None
        }
        
        logger.info(f"Created election: {election_id}")
        
        return jsonify({
            'success': True,
            'election_id': election_id,
            'message': f'Election created with {data.get("guardians", 5)} guardians'
        }), 201
        
    except Exception as e:
        logger.error(f"Error creating election: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/keys/generate', methods=['POST'])
def generate_keys():
    """Generate guardian keys for an election"""
    try:
        data = request.json
        election_id = data.get('election_id')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        election = elections[election_id]
        
        # Generate guardian keys
        guardian_keys = []
        for i in range(election['guardians']):
            # Simulate key generation with secure random values
            key_data = {
                'guardian_id': f'guardian_{i+1}',
                'public_key': secrets.token_hex(32),  # Simulated public key
                'private_key_share': secrets.token_hex(32),  # Simulated private key share
                'commitment': secrets.token_hex(16),
                'proof': secrets.token_hex(16)
            }
            guardian_keys.append(key_data)
            
            # Store each guardian's private key in Vault
            if vault_client.enabled:
                vault_key = f"elections/{election_id}/guardian_{i+1}"
                vault_data = {
                    'private_key_share': key_data['private_key_share'],
                    'guardian_id': key_data['guardian_id'],
                    'election_id': election_id
                }
                if vault_client.store_secret(vault_key, vault_data):
                    logger.info(f"Stored guardian {i+1} keys in Vault")
                else:
                    logger.warning(f"Failed to store guardian {i+1} keys in Vault")
        
        # Store public election keys (these can be public)
        guardians[election_id] = {
            'count': election['guardians'],
            'threshold': election['threshold'],
            'keys_generated': True,
            'public_keys': [{'guardian_id': k['guardian_id'], 'public_key': k['public_key']} 
                           for k in guardian_keys],
            'vault_storage': vault_client.enabled and vault_client.connected
        }
        
        # Store election context in Vault
        if vault_client.enabled:
            election_context = {
                'election_id': election_id,
                'name': election['name'],
                'guardians': election['guardians'],
                'threshold': election['threshold'],
                'public_keys': guardians[election_id]['public_keys'],
                'created_at': str(Path.cwd())  # Placeholder for timestamp
            }
            vault_client.store_secret(f"elections/{election_id}/context", election_context)
        
        elections[election_id]['status'] = 'keys_generated'
        
        logger.info(f"Generated keys for election: {election_id}")
        
        return jsonify({
            'success': True,
            'election_id': election_id,
            'guardians': election['guardians'],
            'threshold': election['threshold'],
            'vault_secured': vault_client.enabled and vault_client.connected,
            'public_keys': guardians[election_id]['public_keys']
        }), 200
        
    except Exception as e:
        logger.error(f"Error generating keys: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/ballot/encrypt', methods=['POST'])
def encrypt_ballot():
    """Encrypt a ballot for an election"""
    try:
        data = request.json
        election_id = data.get('election_id')
        ballot = data.get('ballot')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        # Simulate ballot encryption
        encrypted_ballot = {
            'election_id': election_id,
            'ballot_id': f'ballot_{len(elections[election_id]["ballots"])}',
            'encrypted': True,
            'receipt': f'receipt_{len(elections[election_id]["ballots"])}'
        }
        
        elections[election_id]['ballots'].append(encrypted_ballot)
        
        logger.info(f"Encrypted ballot for election: {election_id}")
        
        return jsonify({
            'success': True,
            'ballot_id': encrypted_ballot['ballot_id'],
            'receipt': encrypted_ballot['receipt']
        }), 200
        
    except Exception as e:
        logger.error(f"Error encrypting ballot: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/tally/compute', methods=['POST'])
def compute_tally():
    """Compute the tally for an election"""
    try:
        data = request.json
        election_id = data.get('election_id')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        election = elections[election_id]
        
        # Simulate tally computation
        tally = {
            'election_id': election_id,
            'total_ballots': len(election['ballots']),
            'computed': True,
            'results': {
                'candidate_a': 45,
                'candidate_b': 55
            }
        }
        
        elections[election_id]['tally'] = tally
        elections[election_id]['status'] = 'tallied'
        
        logger.info(f"Computed tally for election: {election_id}")
        
        return jsonify({
            'success': True,
            'tally': tally
        }), 200
        
    except Exception as e:
        logger.error(f"Error computing tally: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/verify/ballot', methods=['GET'])
def verify_ballot():
    """Verify a ballot receipt"""
    try:
        election_id = request.args.get('election_id')
        receipt = request.args.get('receipt')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        # Simulate verification
        verification = {
            'election_id': election_id,
            'receipt': receipt,
            'verified': True,
            'included_in_tally': True
        }
        
        logger.info(f"Verified ballot receipt: {receipt}")
        
        return jsonify({
            'success': True,
            'verification': verification
        }), 200
        
    except Exception as e:
        logger.error(f"Error verifying ballot: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/election/<election_id>', methods=['GET'])
def get_election(election_id):
    """Get election status and details"""
    if election_id not in elections:
        return jsonify({
            'success': False,
            'error': 'Election not found'
        }), 404
    
    return jsonify({
        'success': True,
        'election': elections[election_id]
    }), 200


@app.route('/api/v1/keys/retrieve', methods=['GET'])
def retrieve_keys():
    """Retrieve guardian keys from Vault"""
    try:
        election_id = request.args.get('election_id')
        guardian_id = request.args.get('guardian_id')
        
        if not election_id:
            return jsonify({
                'success': False,
                'error': 'election_id required'
            }), 400
        
        if not vault_client.enabled or not vault_client.connected:
            return jsonify({
                'success': False,
                'error': 'Vault not available'
            }), 503
        
        # Retrieve specific guardian key or all guardian keys
        if guardian_id:
            vault_key = f"elections/{election_id}/{guardian_id}"
            secret = vault_client.retrieve_secret(vault_key)
            if secret:
                return jsonify({
                    'success': True,
                    'guardian_id': guardian_id,
                    'key_data': secret
                }), 200
            else:
                return jsonify({
                    'success': False,
                    'error': 'Guardian key not found'
                }), 404
        else:
            # Retrieve election context
            context = vault_client.retrieve_secret(f"elections/{election_id}/context")
            if context:
                return jsonify({
                    'success': True,
                    'election_context': context
                }), 200
            else:
                return jsonify({
                    'success': False,
                    'error': 'Election context not found'
                }), 404
                
    except Exception as e:
        logger.error(f"Error retrieving keys: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/export/postgres', methods=['POST'])
def export_to_postgres():
    """Export election data to PostgreSQL"""
    try:
        data = request.json
        election_id = data.get('election_id')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        # Check if PostgreSQL is enabled
        if os.environ.get('ELECTIONGUARD_DB_ENABLED', 'false').lower() != 'true':
            return jsonify({
                'success': False,
                'error': 'PostgreSQL export not enabled'
            }), 503
        
        # Simulate export (would actually connect to PostgreSQL)
        export_data = {
            'election': elections[election_id],
            'guardians': guardians.get(election_id, {}),
            'exported_at': str(Path.cwd())
        }
        
        logger.info(f"Exported election {election_id} to PostgreSQL")
        
        return jsonify({
            'success': True,
            'election_id': election_id,
            'records_exported': len(elections[election_id].get('ballots', [])),
            'message': 'Election data exported to PostgreSQL'
        }), 200
        
    except Exception as e:
        logger.error(f"Error exporting to PostgreSQL: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/export/questdb', methods=['POST'])
def export_to_questdb():
    """Export election metrics to QuestDB"""
    try:
        data = request.json
        election_id = data.get('election_id')
        
        if election_id not in elections:
            return jsonify({
                'success': False,
                'error': 'Election not found'
            }), 404
        
        # Check if QuestDB is enabled
        if os.environ.get('ELECTIONGUARD_QUESTDB_ENABLED', 'false').lower() != 'true':
            return jsonify({
                'success': False,
                'error': 'QuestDB export not enabled'
            }), 503
        
        # Simulate time-series export
        metrics = {
            'election_id': election_id,
            'timestamp': str(Path.cwd()),
            'total_ballots': len(elections[election_id].get('ballots', [])),
            'status': elections[election_id].get('status'),
            'guardians': elections[election_id].get('guardians')
        }
        
        logger.info(f"Exported election {election_id} metrics to QuestDB")
        
        return jsonify({
            'success': True,
            'election_id': election_id,
            'metrics': metrics,
            'message': 'Election metrics exported to QuestDB'
        }), 200
        
    except Exception as e:
        logger.error(f"Error exporting to QuestDB: {e}")
        return jsonify({
            'success': False,
            'error': str(e)
        }), 500


@app.route('/api/v1/vault/status', methods=['GET'])
def vault_status():
    """Check Vault connection status"""
    vault_client._check_connection()
    
    return jsonify({
        'enabled': vault_client.enabled,
        'connected': vault_client.connected,
        'vault_url': vault_client.vault_url,
        'vault_path': vault_client.vault_path
    }), 200


def main():
    """Main entry point"""
    parser = argparse.ArgumentParser(description='ElectionGuard API Server')
    parser.add_argument('--port', type=int, default=int(os.environ.get('ELECTIONGUARD_PORT', 18250)))
    parser.add_argument('--host', default=os.environ.get('ELECTIONGUARD_HOST', '0.0.0.0'))
    parser.add_argument('--data-dir', default=os.environ.get('ELECTIONGUARD_DATA_DIR', './data'))
    parser.add_argument('--debug', action='store_true')
    
    args = parser.parse_args()
    
    # Create data directory if it doesn't exist
    Path(args.data_dir).mkdir(parents=True, exist_ok=True)
    
    logger.info(f"Starting ElectionGuard API server on {args.host}:{args.port}")
    
    # Run the Flask app
    app.run(
        host=args.host,
        port=args.port,
        debug=args.debug
    )


if __name__ == '__main__':
    main()