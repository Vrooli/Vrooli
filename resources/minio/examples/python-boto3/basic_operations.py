#!/usr/bin/env python3
"""
MinIO Basic Operations Example using boto3

This example demonstrates fundamental MinIO operations using the boto3 library:
- Connecting to MinIO
- Creating and listing buckets
- Uploading and downloading files
- Error handling

Prerequisites:
- MinIO running locally (resource-minio manage install)
- Python 3.6+
- boto3 installed (pip install boto3)
"""

import boto3
import os
import tempfile
from botocore.client import Config
from botocore.exceptions import ClientError, NoCredentialsError

# Configuration - Update these based on your MinIO setup
MINIO_ENDPOINT = os.getenv('MINIO_ENDPOINT', 'http://localhost:9000')
MINIO_ACCESS_KEY = os.getenv('MINIO_ACCESS_KEY', 'minioadmin')
MINIO_SECRET_KEY = os.getenv('MINIO_SECRET_KEY', 'minio123')
MINIO_REGION = os.getenv('MINIO_REGION', 'us-east-1')


def create_minio_client():
    """Create and configure MinIO client using boto3."""
    try:
        client = boto3.client(
            's3',
            endpoint_url=MINIO_ENDPOINT,
            aws_access_key_id=MINIO_ACCESS_KEY,
            aws_secret_access_key=MINIO_SECRET_KEY,
            config=Config(signature_version='s3v4'),
            region_name=MINIO_REGION
        )
        
        # Test connection
        client.list_buckets()
        print(f"✅ Successfully connected to MinIO at {MINIO_ENDPOINT}")
        return client
    
    except NoCredentialsError:
        print("❌ Error: Invalid credentials")
        print("Run 'resource-minio credentials' to get correct credentials")
        return None
    except Exception as e:
        print(f"❌ Error connecting to MinIO: {e}")
        print("Make sure MinIO is running: resource-minio status")
        return None


def list_buckets(client):
    """List all available buckets."""
    try:
        response = client.list_buckets()
        print("\n📦 Available buckets:")
        for bucket in response['Buckets']:
            print(f"   • {bucket['Name']} (Created: {bucket['CreationDate']})")
        return [bucket['Name'] for bucket in response['Buckets']]
    except ClientError as e:
        print(f"❌ Error listing buckets: {e}")
        return []


def create_bucket(client, bucket_name):
    """Create a new bucket if it doesn't exist."""
    try:
        client.head_bucket(Bucket=bucket_name)
        print(f"📦 Bucket '{bucket_name}' already exists")
        return True
    except ClientError as e:
        if e.response['Error']['Code'] == '404':
            try:
                client.create_bucket(Bucket=bucket_name)
                print(f"✅ Created bucket '{bucket_name}'")
                return True
            except ClientError as create_error:
                print(f"❌ Error creating bucket: {create_error}")
                return False
        else:
            print(f"❌ Error checking bucket: {e}")
            return False


def upload_file(client, bucket_name, local_file, object_name=None):
    """Upload a file to MinIO bucket."""
    if object_name is None:
        object_name = os.path.basename(local_file)
    
    try:
        client.upload_file(local_file, bucket_name, object_name)
        print(f"✅ Uploaded '{local_file}' as '{object_name}' to bucket '{bucket_name}'")
        return True
    except ClientError as e:
        print(f"❌ Error uploading file: {e}")
        return False


def download_file(client, bucket_name, object_name, local_file):
    """Download a file from MinIO bucket."""
    try:
        client.download_file(bucket_name, object_name, local_file)
        print(f"✅ Downloaded '{object_name}' from bucket '{bucket_name}' to '{local_file}'")
        return True
    except ClientError as e:
        print(f"❌ Error downloading file: {e}")
        return False


def list_objects(client, bucket_name):
    """List all objects in a bucket."""
    try:
        response = client.list_objects_v2(Bucket=bucket_name)
        
        if 'Contents' not in response:
            print(f"📁 Bucket '{bucket_name}' is empty")
            return []
        
        print(f"\n📁 Objects in bucket '{bucket_name}':")
        objects = []
        for obj in response['Contents']:
            size_mb = obj['Size'] / (1024 * 1024)
            print(f"   • {obj['Key']} ({size_mb:.2f} MB, Modified: {obj['LastModified']})")
            objects.append(obj['Key'])
        
        return objects
    except ClientError as e:
        print(f"❌ Error listing objects: {e}")
        return []


def delete_object(client, bucket_name, object_name):
    """Delete an object from bucket."""
    try:
        client.delete_object(Bucket=bucket_name, Key=object_name)
        print(f"✅ Deleted object '{object_name}' from bucket '{bucket_name}'")
        return True
    except ClientError as e:
        print(f"❌ Error deleting object: {e}")
        return False


def main():
    """Main example function demonstrating MinIO operations."""
    print("🚀 MinIO Basic Operations Example")
    print("=" * 40)
    
    # Create MinIO client
    client = create_minio_client()
    if not client:
        return
    
    # List existing buckets
    buckets = list_buckets(client)
    
    # Use test bucket
    test_bucket = 'python-example-bucket'
    
    # Create test bucket
    if create_bucket(client, test_bucket):
        
        # Create a temporary test file
        with tempfile.NamedTemporaryFile(mode='w', suffix='.txt', delete=False) as f:
            f.write("Hello from MinIO Python example!\n")
            f.write(f"This file was created for testing MinIO operations.\n")
            f.write(f"Timestamp: {os.popen('date').read().strip()}\n")
            temp_file = f.name
        
        print(f"\n📝 Created temporary test file: {temp_file}")
        
        # Upload the file
        if upload_file(client, test_bucket, temp_file, 'test-file.txt'):
            
            # List objects in bucket
            list_objects(client, test_bucket)
            
            # Download the file to a new location
            download_path = temp_file + '.downloaded'
            if download_file(client, test_bucket, 'test-file.txt', download_path):
                
                # Verify download
                with open(download_path, 'r') as f:
                    content = f.read()
                    print(f"\n📄 Downloaded file content:")
                    print(content)
                
                # Clean up local files
                os.unlink(temp_file)
                os.unlink(download_path)
                print("🧹 Cleaned up local temporary files")
            
            # Delete the object from MinIO
            delete_object(client, test_bucket, 'test-file.txt')
        
        # Optional: Delete the test bucket (uncomment if desired)
        # try:
        #     client.delete_bucket(Bucket=test_bucket)
        #     print(f"✅ Deleted test bucket '{test_bucket}'")
        # except ClientError as e:
        #     print(f"❌ Error deleting bucket: {e}")
    
    print("\n🎉 Example completed successfully!")
    print("\n💡 Next steps:")
    print("   - Modify this script for your specific use case")
    print("   - Check out other examples in the examples/ directory")
    print("   - Read the API documentation in docs/API.md")


if __name__ == "__main__":
    main()